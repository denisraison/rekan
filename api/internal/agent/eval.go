package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"gopkg.in/yaml.v3"
)

// TestCase represents a single eval test case from YAML.
type TestCase struct {
	ID                  string   `yaml:"id"`
	Message             string   `yaml:"message"`
	Operator            Operator `yaml:"operator"`
	Context             string   `yaml:"context"`
	ConversationHistory string   `yaml:"conversation_history"`
	Graders             []Grader `yaml:"graders"`
}

// Operator identifies the test sender.
type Operator struct {
	Name string `yaml:"name"`
	JID  string `yaml:"jid"`
}

// Grader describes how to evaluate a test case result.
type Grader struct {
	Type     string `yaml:"type"`
	Field    string `yaml:"field"`
	Equals   string `yaml:"equals"`
	Contains string `yaml:"contains"`
	Judge    string `yaml:"judge"`
	Criteria string `yaml:"criteria"`
}

// TestSuite is the top-level YAML structure.
type TestSuite struct {
	Tests []TestCase `yaml:"tests"`
}

// TestResult holds the outcome of running a single test case.
type TestResult struct {
	ID     string
	Passed bool
	Checks []CheckResult
}

// CheckResult is the outcome of a single grader.
type CheckResult struct {
	Grader Grader
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
	Reply       string
	ToolsCalled []string
	ToolArgs    map[string]json.RawMessage // tool name -> last input
}

// RunEval runs all test cases through the tool-use loop and returns results.
func RunEval(ctx context.Context, cases []TestCase, verbose bool) ([]TestResult, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	results := make([]TestResult, 0, len(cases))

	for _, tc := range cases {
		result := TestResult{ID: tc.ID, Passed: true}

		er, err := runEvalCase(ctx, client, tc)
		if err != nil {
			result.Passed = false
			result.Checks = append(result.Checks, CheckResult{
				Grader: Grader{Type: "error"},
				Passed: false,
				Reason: fmt.Sprintf("tool-use error: %v", err),
			})
			results = append(results, result)
			continue
		}

		toolsCalledStr := strings.Join(er.ToolsCalled, ",")
		hasReply := "false"
		if er.Reply != "" {
			hasReply = "true"
		}

		if verbose {
			fmt.Printf("  [%s] tools=%q reply=%q\n", tc.ID, toolsCalledStr, truncate(er.Reply, 80))
		}

		for _, g := range tc.Graders {
			cr := runGrader(g, toolsCalledStr, er.Reply, hasReply, er.ToolArgs)
			result.Checks = append(result.Checks, cr)
			if !cr.Passed {
				result.Passed = false
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// runEvalCase runs a single test case through the Claude tool-use loop with mock data.
func runEvalCase(ctx context.Context, client anthropic.Client, tc TestCase) (*evalResult, error) {
	systemPrompt := buildSystemPrompt(tc.Operator.Name)
	// Inject test context into system prompt so Claude has data to work with
	if tc.Context != "" {
		systemPrompt += "\n\nContexto do sistema:\n" + tc.Context
	}

	// Build messages from conversation history
	var messages []anthropic.MessageParam
	if tc.ConversationHistory != "" {
		messages = parseConversationHistory(tc.ConversationHistory)
	}
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(tc.Message)))

	tools := agentTools
	er := &evalResult{ToolArgs: make(map[string]json.RawMessage)}
	writeUsed := false

	for range maxToolRoundTrips {
		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeSonnet4_6,
			MaxTokens: 1024,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  messages,
			Tools:     tools,
		})
		if err != nil {
			return nil, err
		}

		messages = append(messages, resp.ToParam())

		var toolResults []anthropic.ContentBlockParamUnion
		hasWrite := false

		for _, block := range resp.Content {
			switch v := block.AsAny().(type) {
			case anthropic.TextBlock:
				er.Reply = v.Text
			case anthropic.ToolUseBlock:
				er.ToolsCalled = append(er.ToolsCalled, v.Name)
				er.ToolArgs[v.Name] = v.Input

				// Return mock data from context
				mockResult := mockToolResult(v.Name, v.Input, tc.Context)
				toolResults = append(toolResults, anthropic.NewToolResultBlock(v.ID, mockResult, false))

				if isWriteTool(v.Name) {
					hasWrite = true
				}
			}
		}

		if hasWrite {
			writeUsed = true
		}

		if len(toolResults) == 0 || hasWrite {
			break
		}

		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	// If a write tool was called but Claude produced no text, use a fallback
	if er.Reply == "" && writeUsed {
		er.Reply = tc.Operator.Name + fallbackWriteReply
	}

	return er, nil
}

// mockToolResult returns mock data for a tool call based on the test context.
func mockToolResult(name string, input json.RawMessage, testContext string) string {
	switch name {
	case "find_customer":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(input, &args); err != nil {
			return "erro: " + err.Error()
		}
		return findInContext(testContext, args.Query)
	case "list_customers":
		return extractSection(testContext, "Clientes ativas")
	case "find_post":
		var args struct {
			PostID string `json:"post_id"`
		}
		if err := json.Unmarshal(input, &args); err != nil {
			return "erro: " + err.Error()
		}
		return findPostInContext(testContext, args.PostID)
	case "list_posts":
		var args struct {
			CustomerName string `json:"customer_name"`
		}
		if err := json.Unmarshal(input, &args); err != nil {
			return "erro: " + err.Error()
		}
		if args.CustomerName != "" {
			return findPostsForCustomer(testContext, args.CustomerName)
		}
		return extractSection(testContext, "Posts pendentes")
	case "recent_activity":
		return extractSection(testContext, "Últimas ações")
	default:
		// Write tools in eval just confirm
		return "Confirmação necessária. Aguardando resposta da operadora."
	}
}

// findInContext searches test context for a customer matching the query.
func findInContext(ctx, query string) string {
	if query == "" {
		return "Nenhuma cliente encontrada."
	}
	normalizedQuery := normalizeForMatch(query)
	var found []string
	for line := range strings.SplitSeq(ctx, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		entry := strings.TrimPrefix(trimmed, "- ")
		if strings.Contains(normalizeForMatch(entry), normalizedQuery) {
			// Parse "Name (Type, City)" format
			if idx := strings.Index(entry, " ("); idx > 0 {
				name := entry[:idx]
				rest := strings.TrimSuffix(entry[idx+2:], ")")
				parts := strings.SplitN(rest, ", ", 2)
				if len(parts) == 2 {
					found = append(found, fmt.Sprintf("Nome: %s\nTipo: %s\nCidade: %s\nStatus: active", name, parts[0], parts[1]))
				}
			}
		}
	}
	if len(found) == 0 {
		return fmt.Sprintf("Nenhuma cliente encontrada com '%s'.", query)
	}
	return strings.Join(found, "\n---\n")
}

// findPostInContext searches test context for a post matching the ID.
func findPostInContext(ctx, postID string) string {
	for line := range strings.SplitSeq(ctx, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "("+postID+")") {
			// Parse "- Name: "caption..." (post_id)" format
			entry := strings.TrimPrefix(trimmed, "- ")
			if colonIdx := strings.Index(entry, ": \""); colonIdx > 0 {
				name := entry[:colonIdx]
				rest := entry[colonIdx+3:]
				if quoteEnd := strings.Index(rest, "\""); quoteEnd > 0 {
					caption := rest[:quoteEnd]
					return fmt.Sprintf("Post: %s\nCliente: %s\nLegenda: %s\nStatus: pendente", postID, name, caption)
				}
			}
		}
	}
	return fmt.Sprintf("Post %s não encontrado.", postID)
}

// findPostsForCustomer returns posts matching a customer name from context.
func findPostsForCustomer(ctx, customerName string) string {
	normalizedName := normalizeForMatch(customerName)
	var found []string
	for line := range strings.SplitSeq(ctx, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		entry := strings.TrimPrefix(trimmed, "- ")
		if colonIdx := strings.Index(entry, ": \""); colonIdx > 0 {
			name := entry[:colonIdx]
			if strings.Contains(normalizeForMatch(name), normalizedName) {
				found = append(found, trimmed)
			}
		}
	}
	if len(found) == 0 {
		return fmt.Sprintf("Nenhum post encontrado para '%s'.", customerName)
	}
	return strings.Join(found, "\n")
}

// extractSection returns a section of text starting with the given prefix.
func extractSection(ctx, prefix string) string {
	lines := strings.Split(ctx, "\n")
	var section []string
	capturing := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			capturing = true
			section = append(section, trimmed)
			continue
		}
		if capturing {
			if trimmed == "" || (!strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, " ")) {
				break
			}
			section = append(section, trimmed)
		}
	}
	if len(section) == 0 {
		return "Nenhum dado encontrado."
	}
	return strings.Join(section, "\n")
}

// parseConversationHistory converts a conversation history string into Claude messages.
func parseConversationHistory(history string) []anthropic.MessageParam {
	var messages []anthropic.MessageParam
	for line := range strings.SplitSeq(strings.TrimSpace(history), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if after, ok := strings.CutPrefix(line, "[Rekan]:"); ok {
			text := strings.TrimSpace(after)
			messages = append(messages, anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleAssistant,
				Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(text)},
			})
		} else if idx := strings.Index(line, ":"); idx > 0 {
			text := strings.TrimSpace(line[idx+1:])
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		}
	}
	return mergeConsecutiveRoles(messages)
}

// isWriteTool returns true if the tool modifies data and requires confirmation.
// Derived from toolNameToActionType to keep a single source of truth.
func isWriteTool(name string) bool {
	switch name {
	case "find_customer", "list_customers", "find_post", "list_posts", "recent_activity":
		return false
	}
	return toolNameToActionType(name) != ""
}

func runGrader(g Grader, toolsCalled, reply, hasReply string, toolArgs map[string]json.RawMessage) CheckResult {
	switch g.Type {
	case "deterministic":
		return runDeterministicGrader(g, toolsCalled, reply, hasReply, toolArgs)
	case "llm_judge":
		return CheckResult{Grader: g, Passed: true, Reason: "llm judge (skipped in deterministic mode)"}
	default:
		return CheckResult{Grader: g, Passed: false, Reason: "unknown grader type: " + g.Type}
	}
}

func runDeterministicGrader(g Grader, toolsCalled, reply, hasReply string, toolArgs map[string]json.RawMessage) CheckResult {
	var actual string
	switch g.Field {
	case "action_type":
		// Prefer write tools for action_type (write tools are the primary action).
		// Fall back to last read tool if no write tool was called.
		if toolsCalled != "" {
			tools := strings.Split(toolsCalled, ",")
			for _, t := range tools {
				if isWriteTool(t) {
					actual = toolNameToActionType(t)
					break
				}
			}
			if actual == "" {
				for i := len(tools) - 1; i >= 0; i-- {
					if at := toolNameToActionType(tools[i]); at != "" {
						actual = at
						break
					}
				}
			}
		}
	case "tools_called":
		actual = toolsCalled
	case "reply":
		actual = reply
	case "has_reply":
		actual = hasReply
	case "tool_args":
		// Check that a tool was called with args containing a value
		// g.Equals format: "tool_name" and g.Contains is the expected substring in args JSON
		if raw, ok := toolArgs[g.Equals]; ok {
			if g.Contains != "" {
				if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(g.Contains)) {
					return CheckResult{Grader: g, Passed: true}
				}
				return CheckResult{Grader: g, Passed: false, Reason: fmt.Sprintf("tool %s args %q doesn't contain %q", g.Equals, string(raw), g.Contains)}
			}
			return CheckResult{Grader: g, Passed: true}
		}
		return CheckResult{Grader: g, Passed: false, Reason: fmt.Sprintf("tool %q not called", g.Equals)}
	default:
		return CheckResult{Grader: g, Passed: false, Reason: "unknown field: " + g.Field}
	}

	if g.Equals != "" {
		passed := strings.EqualFold(actual, g.Equals)
		reason := ""
		if !passed {
			reason = fmt.Sprintf("expected %q, got %q", g.Equals, actual)
		}
		return CheckResult{Grader: g, Passed: passed, Reason: reason}
	}
	if g.Contains != "" {
		passed := strings.Contains(strings.ToLower(actual), strings.ToLower(g.Contains))
		reason := ""
		if !passed {
			reason = fmt.Sprintf("expected to contain %q, got %q", g.Contains, actual)
		}
		return CheckResult{Grader: g, Passed: passed, Reason: reason}
	}

	return CheckResult{Grader: g, Passed: false, Reason: "no equals or contains specified"}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
