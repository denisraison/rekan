package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/denisraison/rekan/api/internal/service"
	"gopkg.in/yaml.v3"
)

// TestCase represents a single eval test case from YAML.
type TestCase struct {
	ID       string      `yaml:"id"`
	Message  string      `yaml:"message"`
	Operator Operator    `yaml:"operator"`
	Fixtures Fixtures    `yaml:"fixtures"`
	Assert   []Assertion `yaml:"assert"`
}

// Operator identifies the test sender.
type Operator struct {
	Name string `yaml:"name"`
	JID  string `yaml:"jid"`
}

// Fixtures holds structured mock data for a test case.
type Fixtures struct {
	Customers []MockCustomer `yaml:"customers"`
	Posts     []MockPost     `yaml:"posts"`
}

// MockCustomer is a customer fixture.
type MockCustomer struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
	City  string `yaml:"city"`
	Phone string `yaml:"phone"`
}

// MockPost is a post fixture.
type MockPost struct {
	ID             string   `yaml:"id"`
	Business       string   `yaml:"business"`
	Caption        string   `yaml:"caption"`
	Hashtags       []string `yaml:"hashtags"`
	ProductionNote string   `yaml:"production_note"`
	Reviewed       bool     `yaml:"reviewed"`
}

// Assertion describes a single check on the eval result.
type Assertion struct {
	ToolCalled      string      `yaml:"tool_called"`
	ToolNotCalled   string      `yaml:"tool_not_called"`
	ToolArg         *ToolArgDef `yaml:"tool_arg"`
	ReplyContains   string      `yaml:"reply_contains"`
	ReplyNotContain string      `yaml:"reply_not_contains"`
	NoEmptyPromise  bool        `yaml:"no_empty_promise"`
	MaxToolCalls    int         `yaml:"max_tool_calls"`
}

// ToolArgDef checks that a tool was called with a specific argument value.
type ToolArgDef struct {
	Tool     string `yaml:"tool"`
	Key      string `yaml:"key"`
	Contains string `yaml:"contains"`
}

// TestSuite is the top-level YAML structure.
type TestSuite struct {
	Tests []TestCase `yaml:"tests"`
}

// TestResult holds the outcome of running a single test case.
type TestResult struct {
	ID             string
	Passed         bool
	Checks         []CheckResult
	Reply          string
	ToolsCalled    []string
	ToolLog        []toolCallEntry
	InputTokens    int
	OutputTokens   int
	WallTimeMs     int64
	ToolRoundTrips int
	Error          string
}

// CheckResult is the outcome of a single assertion.
type CheckResult struct {
	Name   string
	Passed bool
	Reason string
}

// LoadTestCases reads test cases from a YAML file.
func LoadTestCases(path string) ([]TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading test cases: %w", err)
	}
	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parsing test cases: %w", err)
	}
	return suite.Tests, nil
}

// evalResult holds the output of running a single eval case through the tool-use loop.
type evalResult struct {
	Reply          string
	ToolsCalled    []string
	ToolArgs       map[string][]json.RawMessage // tool name -> all invocations' input
	ToolLog        []toolCallEntry
	InputTokens    int
	OutputTokens   int
	ToolRoundTrips int
}

// RunEval runs all test cases in parallel (max 5 concurrent) and returns results.
func RunEval(ctx context.Context, cases []TestCase) []TestResult {
	client := NewClient()

	results := make([]TestResult, len(cases))
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for i, tc := range cases {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			results[i] = runAndGrade(ctx, client, tc)
		})
	}

	wg.Wait()
	return results
}

func runAndGrade(ctx context.Context, client *Client, tc TestCase) TestResult {
	start := timeNowMs()
	er, err := runEvalCase(ctx, client, tc)
	elapsed := timeNowMs() - start

	result := TestResult{
		ID:         tc.ID,
		Passed:     true,
		WallTimeMs: elapsed,
	}

	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		result.Checks = append(result.Checks, CheckResult{
			Name:   "api_call",
			Passed: false,
			Reason: err.Error(),
		})
		return result
	}

	result.Reply = er.Reply
	result.ToolsCalled = er.ToolsCalled
	result.ToolLog = er.ToolLog
	result.InputTokens = er.InputTokens
	result.OutputTokens = er.OutputTokens
	result.ToolRoundTrips = er.ToolRoundTrips

	for _, a := range tc.Assert {
		cr := runAssertion(a, er)
		result.Checks = append(result.Checks, cr)
		if !cr.Passed {
			result.Passed = false
		}
	}

	return result
}

// runEvalCase runs a single test case through the tool-use loop with mock data.
func runEvalCase(ctx context.Context, client *Client, tc TestCase) (*evalResult, error) {
	er := &evalResult{ToolArgs: make(map[string][]json.RawMessage)}
	mock := &MockExecutor{Fixtures: tc.Fixtures, OperatorName: tc.Operator.Name}

	// Build mock tools that record calls in evalResult
	mockTools := buildMockTools(mock, er)

	systemPrompt := buildSystemPrompt(tc.Operator.Name)
	runResult, err := client.Run(ctx, RunConfig{
		System:    systemPrompt,
		Messages:  []Message{NewUserMessage(NewTextBlock(tc.Message))},
		Tools:     mockTools,
		MaxTurns:  maxToolRoundTrips,
		MaxTokens: 2048,
	})
	if err != nil {
		return nil, err
	}

	er.Reply = runResult.Reply
	for _, trace := range runResult.Traces {
		er.InputTokens += trace.InputTokens
		er.OutputTokens += trace.OutputTokens
		if len(trace.ToolCalls) > 0 {
			er.ToolRoundTrips++
		}
	}

	return er, nil
}

// buildMockTools creates Tool values wrapping MockExecutor that record calls for eval grading.
func buildMockTools(mock *MockExecutor, er *evalResult) []Tool {
	// We need the same tool schemas as production. Build a dummy executor just for schemas.
	prodTools := buildTools(&ToolExecutor{Ctx: context.Background()}, mock.OperatorName)

	mockTools := make([]Tool, len(prodTools))
	for i, pt := range prodTools {
		toolName := pt.Name
		mockTools[i] = Tool{
			Name:        pt.Name,
			Description: pt.Description,
			InputSchema: pt.InputSchema,
			Execute: func(_ context.Context, input json.RawMessage) (string, error) {
				er.ToolsCalled = append(er.ToolsCalled, toolName)
				er.ToolArgs[toolName] = append(er.ToolArgs[toolName], input)

				mockResult := mock.Execute(toolName, input)
				er.ToolLog = append(er.ToolLog, toolCallEntry{
					Name:   toolName,
					Args:   truncate(string(input), 80),
					Result: truncate(mockResult, 60),
				})
				return mockResult, nil
			},
		}
	}
	return mockTools
}

// MockExecutor implements tool dispatch using structured fixtures.
type MockExecutor struct {
	Fixtures     Fixtures
	OperatorName string
}

// Execute dispatches a mock tool call and returns the result string.
func (m *MockExecutor) Execute(name string, input json.RawMessage) string {
	switch name {
	case "search_customers":
		return m.searchCustomers(input)
	case "search_posts":
		return m.searchPosts(input)
	case "create_customer":
		return m.createCustomer(input)
	case "update_customer":
		return m.updateCustomer(input)
	case "generate_post":
		return m.generatePost(input)
	case "approve_post":
		return m.approvePost(input)
	case "reject_post":
		return m.rejectPost(input)
	default:
		return "Ferramenta desconhecida: " + name
	}
}

func (m *MockExecutor) searchCustomers(input json.RawMessage) string {
	var args struct {
		Query string `json:"query"`
	}
	if len(input) > 0 {
		json.Unmarshal(input, &args) //nolint:errcheck
	}

	// No query: list all
	if args.Query == "" {
		if len(m.Fixtures.Customers) == 0 {
			return "Nenhuma cliente ativa."
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d clientes ativas:\n", len(m.Fixtures.Customers))
		for _, c := range m.Fixtures.Customers {
			fmt.Fprintf(&b, "- %s (%s, %s)\n", c.Name, c.Type, c.City)
		}
		return b.String()
	}

	// With query: fuzzy search with full details
	query := service.NormalizeForMatch(args.Query)
	var matches []MockCustomer
	for _, c := range m.Fixtures.Customers {
		if strings.Contains(service.NormalizeForMatch(c.Name), query) {
			matches = append(matches, c)
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("Nenhuma cliente encontrada com '%s'.", args.Query)
	}

	var b strings.Builder
	for _, c := range matches {
		fmt.Fprintf(&b, "Nome: %s\nTipo: %s\nCidade: %s\n", c.Name, c.Type, c.City)
		if c.Phone != "" {
			fmt.Fprintf(&b, "Tel: %s\n", c.Phone)
		}
		fmt.Fprintf(&b, "Status: active\n---\n")
	}
	return b.String()
}

func (m *MockExecutor) searchPosts(input json.RawMessage) string {
	var args struct {
		PostID       string `json:"post_id"`
		CustomerName string `json:"customer_name"`
		Status       string `json:"status"`
	}
	if len(input) > 0 {
		json.Unmarshal(input, &args) //nolint:errcheck
	}

	// Single post detail view (supports prefix matching)
	if args.PostID != "" {
		match, errMsg := m.resolvePostByPrefix(args.PostID)
		if errMsg != "" {
			return errMsg
		}
		var b strings.Builder
		fmt.Fprintf(&b, "id:%s\n", match.ID)
		fmt.Fprintf(&b, "customer:%s\n", match.Business)
		fmt.Fprintf(&b, "status:%s\n", postStatus(match.Reviewed))
		fmt.Fprintf(&b, "caption:%s\n", match.Caption)
		if len(match.Hashtags) > 0 {
			fmt.Fprintf(&b, "hashtags:%s\n", strings.Join(match.Hashtags, " "))
		}
		if match.ProductionNote != "" {
			fmt.Fprintf(&b, "production_note:%s\n", match.ProductionNote)
		}
		return b.String()
	}

	// List view
	var posts []MockPost
	for _, p := range m.Fixtures.Posts {
		if args.CustomerName != "" && !strings.Contains(service.NormalizeForMatch(p.Business), service.NormalizeForMatch(args.CustomerName)) {
			continue
		}
		if args.Status == "pending" && p.Reviewed {
			continue
		}
		if args.Status == "reviewed" && !p.Reviewed {
			continue
		}
		posts = append(posts, p)
	}

	if len(posts) == 0 {
		if args.CustomerName != "" {
			return fmt.Sprintf("Nenhum post encontrado para '%s'.", args.CustomerName)
		}
		return "Nenhum post encontrado."
	}

	var b strings.Builder
	for _, p := range posts {
		preview := truncate(p.Caption, 60)
		fmt.Fprintf(&b, "id:%s customer:%s status:%s preview:\"%s\"\n", shortPostID(p.ID), p.Business, postStatus(p.Reviewed), preview)
	}
	return b.String()
}

func (m *MockExecutor) createCustomer(input json.RawMessage) string {
	var args struct {
		Name string `json:"name"`
		Type string `json:"type"`
		City string `json:"city"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	for _, c := range m.Fixtures.Customers {
		if strings.EqualFold(c.Name, args.Name) {
			return fmt.Sprintf("%s já existe (%s, %s).", c.Name, c.Type, c.City)
		}
	}

	return fmt.Sprintf("%s cadastrada (%s, %s).", args.Name, args.Type, args.City)
}

func (m *MockExecutor) updateCustomer(input json.RawMessage) string {
	var args struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}
	if args.Status == "paused" {
		return fmt.Sprintf("%s pausada.", args.Name)
	}
	if args.Status == "active" {
		return fmt.Sprintf("%s reativada.", args.Name)
	}
	return fmt.Sprintf("%s atualizada.", args.Name)
}

func (m *MockExecutor) generatePost(input json.RawMessage) string {
	var args struct {
		CustomerName string `json:"customer_name"`
		CustomerID   string `json:"customer_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	customer, errMsg := m.resolveCustomerByNameOrID(args.CustomerID, args.CustomerName)
	if errMsg != "" {
		return errMsg
	}
	customerName := customer.Name

	var b strings.Builder
	fmt.Fprintf(&b, "Post gerado pra %s.\n", customerName)
	fmt.Fprintf(&b, "ID: post_gen_001\n")
	fmt.Fprintf(&b, "Legenda: Post de exemplo para %s\n", customerName)
	b.WriteString("Hashtags: #exemplo #post\n")
	b.WriteString("Nota de produção: Foto de exemplo")
	return b.String()
}

func (m *MockExecutor) approvePost(input json.RawMessage) string {
	var args struct {
		PostID string `json:"post_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	match, errMsg := m.resolvePostByPrefix(args.PostID)
	if errMsg != "" {
		return errMsg
	}
	return fmt.Sprintf("Post da %s aprovado.", match.Business)
}

func (m *MockExecutor) rejectPost(input json.RawMessage) string {
	var args struct {
		PostID   string `json:"post_id"`
		Feedback string `json:"feedback"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	match, errMsg := m.resolvePostByPrefix(args.PostID)
	if errMsg != "" {
		return errMsg
	}
	if args.Feedback != "" {
		return fmt.Sprintf("Post da %s rejeitado. Feedback: %s.", match.Business, args.Feedback)
	}
	return fmt.Sprintf("Post da %s rejeitado.", match.Business)
}

// resolveCustomerByNameOrID resolves a mock customer by ID or fuzzy name match.
// Mock customers have no real IDs, so customer_id matches against name (the mock convention).
func (m *MockExecutor) resolveCustomerByNameOrID(id, name string) (*MockCustomer, string) {
	if id != "" {
		for i, c := range m.Fixtures.Customers {
			if c.Name == id {
				return &m.Fixtures.Customers[i], ""
			}
		}
		return nil, fmt.Sprintf("Cliente com ID '%s' não encontrada.", id)
	}
	if name == "" {
		return nil, "Pra qual cliente?"
	}
	for i, c := range m.Fixtures.Customers {
		if strings.Contains(service.NormalizeForMatch(c.Name), service.NormalizeForMatch(name)) {
			return &m.Fixtures.Customers[i], ""
		}
	}
	return nil, fmt.Sprintf("Não encontrei cliente '%s'.", name)
}

// resolvePostByPrefix finds exactly one mock post matching the given ID prefix.
func (m *MockExecutor) resolvePostByPrefix(prefix string) (*MockPost, string) {
	var match *MockPost
	for i, p := range m.Fixtures.Posts {
		if strings.HasPrefix(p.ID, prefix) {
			if match != nil {
				return nil, "Mais de um post com esse prefixo. Use um ID mais específico."
			}
			match = &m.Fixtures.Posts[i]
		}
	}
	if match == nil {
		return nil, fmt.Sprintf("Post %s não encontrado.", prefix)
	}
	return match, ""
}

// --- Assertion engine ---

func runAssertion(a Assertion, er *evalResult) CheckResult {
	switch {
	case a.ToolCalled != "":
		return assertToolCalled(a.ToolCalled, er.ToolsCalled)
	case a.ToolNotCalled != "":
		return assertToolNotCalled(a.ToolNotCalled, er.ToolsCalled)
	case a.ToolArg != nil:
		return assertToolArg(a.ToolArg, er.ToolArgs)
	case a.ReplyContains != "":
		return assertReplyContains(a.ReplyContains, er.Reply)
	case a.ReplyNotContain != "":
		return assertReplyNotContains(a.ReplyNotContain, er.Reply)
	case a.NoEmptyPromise:
		return assertNoEmptyPromise(er.Reply, er.ToolsCalled)
	case a.MaxToolCalls > 0:
		return assertMaxToolCalls(a.MaxToolCalls, er.ToolsCalled)
	default:
		return CheckResult{Name: "unknown", Passed: false, Reason: "no assertion type matched"}
	}
}

func assertToolCalled(name string, called []string) CheckResult {
	if slices.Contains(called, name) {
		return CheckResult{Name: "tool_called:" + name, Passed: true}
	}
	return CheckResult{
		Name:   "tool_called:" + name,
		Passed: false,
		Reason: name + " not called (called: " + strings.Join(called, ", ") + ")",
	}
}

func assertToolNotCalled(name string, called []string) CheckResult {
	if slices.Contains(called, name) {
		return CheckResult{
			Name:   "tool_not_called:" + name,
			Passed: false,
			Reason: name + " was called but shouldn't have been",
		}
	}
	return CheckResult{Name: "tool_not_called:" + name, Passed: true}
}

func assertToolArg(def *ToolArgDef, toolArgs map[string][]json.RawMessage) CheckResult {
	name := fmt.Sprintf("tool_arg:%s.%s~%s", def.Tool, def.Key, def.Contains)
	invocations, ok := toolArgs[def.Tool]
	if !ok {
		return CheckResult{Name: name, Passed: false, Reason: fmt.Sprintf("tool %s not called", def.Tool)}
	}

	for _, raw := range invocations {
		var args map[string]json.RawMessage
		if err := json.Unmarshal(raw, &args); err != nil {
			continue
		}
		val, ok := args[def.Key]
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(string(val)), strings.ToLower(def.Contains)) {
			return CheckResult{Name: name, Passed: true}
		}
	}

	return CheckResult{
		Name:   name,
		Passed: false,
		Reason: fmt.Sprintf("no invocation of %s has %s containing %q", def.Tool, def.Key, def.Contains),
	}
}

func assertReplyContains(substr string, reply string) CheckResult {
	name := "reply_contains:" + substr
	if strings.Contains(strings.ToLower(reply), strings.ToLower(substr)) {
		return CheckResult{Name: name, Passed: true}
	}
	return CheckResult{
		Name:   name,
		Passed: false,
		Reason: fmt.Sprintf("reply does not contain %q", substr),
	}
}

func assertReplyNotContains(substr string, reply string) CheckResult {
	name := "reply_not_contains:" + substr
	if !strings.Contains(strings.ToLower(reply), strings.ToLower(substr)) {
		return CheckResult{Name: name, Passed: true}
	}
	return CheckResult{
		Name:   name,
		Passed: false,
		Reason: fmt.Sprintf("reply contains %q but shouldn't", substr),
	}
}

// promisePatterns matches first-person Portuguese verbs that claim an action was completed.
var promisePatterns = regexp.MustCompile(`(?i)\b(cadastrei|atualizei|aprovei|rejeitei|pausei|gerei|alterei|criei|cadastrou|atualizou|aprovou|rejeitou|pausou|gerou)\b`)

// questionPattern detects sentences that are questions (contain verb near a ?).
var questionPattern = regexp.MustCompile(`(?i)\b(quer|posso|devo|gostaria|prefere|deseja)\b.*\?`)

// writeToolNames maps promise verbs to expected tools.
var writeToolNames = map[string]bool{
	"create_customer": true,
	"update_customer": true,
	"approve_post":    true,
	"reject_post":     true,
	"generate_post":   true,
}

func assertNoEmptyPromise(reply string, toolsCalled []string) CheckResult {
	name := "no_empty_promise"

	matches := promisePatterns.FindAllStringIndex(reply, -1)
	hasDeclarativePromise := false
	for _, m := range matches {
		sentence := extractSentence(reply, m[0])
		if questionPattern.MatchString(sentence) {
			continue
		}
		hasDeclarativePromise = true
		break
	}

	if !hasDeclarativePromise {
		return CheckResult{Name: name, Passed: true}
	}

	for _, t := range toolsCalled {
		if writeToolNames[t] {
			return CheckResult{Name: name, Passed: true}
		}
	}

	return CheckResult{
		Name:   name,
		Passed: false,
		Reason: "reply contains action verbs but no write tool was called",
	}
}

// extractSentence returns the sentence containing the character at position pos.
func extractSentence(text string, pos int) string {
	start := strings.LastIndexAny(text[:pos], ".!?\n") + 1
	end := strings.IndexAny(text[pos:], ".!?\n")
	if end < 0 {
		return text[start:]
	}
	return text[start : pos+end+1]
}

func assertMaxToolCalls(max int, toolsCalled []string) CheckResult {
	name := fmt.Sprintf("max_tool_calls:%d", max)
	if len(toolsCalled) <= max {
		return CheckResult{Name: name, Passed: true}
	}
	return CheckResult{
		Name:   name,
		Passed: false,
		Reason: fmt.Sprintf("called %d tools, max %d", len(toolsCalled), max),
	}
}

func timeNowMs() int64 {
	return time.Now().UnixMilli()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
