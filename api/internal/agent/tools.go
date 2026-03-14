package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// agentTools is the static set of tool definitions, computed once.
var agentTools = buildToolDefs()

func buildToolDefs() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		// Read tools
		{OfTool: &anthropic.ToolParam{
			Name:        "find_customer",
			Description: anthropic.String("Busca cliente por nome (match fuzzy). Retorna detalhes: nome, tipo, cidade, telefone, público, vibe, obs, status."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"query": map[string]any{"type": "string", "description": "Nome ou parte do nome da cliente"},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "list_customers",
			Description: anthropic.String("Lista todas as clientes ativas/rascunho com nome, tipo e cidade."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "find_post",
			Description: anthropic.String("Busca um post pelo ID. Retorna legenda, cliente, status de revisão e nota."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"post_id": map[string]any{"type": "string", "description": "ID do post"},
				},
				Required: []string{"post_id"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "list_posts",
			Description: anthropic.String("Lista posts. Filtre por nome da cliente e/ou status (pending, reviewed, all). Máximo 20, mais recentes primeiro."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"customer_name": map[string]any{"type": "string", "description": "Nome da cliente (opcional)"},
					"status":        map[string]any{"type": "string", "enum": []string{"pending", "reviewed", "all"}, "description": "Filtro de status (padrão: all)"},
				},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "recent_activity",
			Description: anthropic.String("Mostra as últimas ações realizadas pelo agente."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"limit": map[string]any{"type": "integer", "description": "Número de ações (padrão: 5, máximo: 20)"},
				},
			},
		}},
		// Write tools
		{OfTool: &anthropic.ToolParam{
			Name:        "create_customer",
			Description: anthropic.String("Cadastra nova cliente. Requer confirmação da operadora. Campos obrigatórios: name, type, city, phone."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"name":            map[string]any{"type": "string", "description": "Nome da cliente"},
					"type":            map[string]any{"type": "string", "description": "Tipo de negócio (ex: Salão de Beleza, Confeitaria)"},
					"city":            map[string]any{"type": "string", "description": "Cidade"},
					"phone":           map[string]any{"type": "string", "description": "Telefone da cliente"},
					"target_audience": map[string]any{"type": "string", "description": "Público-alvo (opcional)"},
					"brand_vibe":      map[string]any{"type": "string", "description": "Vibe da marca (opcional)"},
					"quirks":          map[string]any{"type": "string", "description": "Observações especiais (opcional)"},
				},
				Required: []string{"name", "type", "city", "phone"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "update_customer",
			Description: anthropic.String("Altera dados de uma cliente existente. Requer confirmação. Apenas name é obrigatório (identifica a cliente)."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"name":            map[string]any{"type": "string", "description": "Nome da cliente (para identificação)"},
					"new_name":        map[string]any{"type": "string", "description": "Novo nome do negócio (se quiser renomear)"},
					"type":            map[string]any{"type": "string", "description": "Novo tipo de negócio"},
					"city":            map[string]any{"type": "string", "description": "Nova cidade"},
					"phone":           map[string]any{"type": "string", "description": "Novo telefone"},
					"target_audience": map[string]any{"type": "string", "description": "Novo público-alvo"},
					"brand_vibe":      map[string]any{"type": "string", "description": "Nova vibe da marca"},
					"quirks":          map[string]any{"type": "string", "description": "Novas observações"},
				},
				Required: []string{"name"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "pause_customer",
			Description: anthropic.String("Pausa uma cliente. Requer confirmação."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"name":   map[string]any{"type": "string", "description": "Nome da cliente"},
					"reason": map[string]any{"type": "string", "description": "Motivo da pausa (opcional)"},
				},
				Required: []string{"name"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "generate_post",
			Description: anthropic.String("Gera posts para uma cliente. Requer confirmação."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"customer_name": map[string]any{"type": "string", "description": "Nome da cliente"},
				},
				Required: []string{"customer_name"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "approve_post",
			Description: anthropic.String("Aprova um post pendente. Requer confirmação."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"post_id":       map[string]any{"type": "string", "description": "ID do post"},
					"customer_name": map[string]any{"type": "string", "description": "Nome da cliente"},
				},
				Required: []string{"post_id"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "reject_post",
			Description: anthropic.String("Rejeita um post com feedback. Requer confirmação."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"post_id":       map[string]any{"type": "string", "description": "ID do post"},
					"customer_name": map[string]any{"type": "string", "description": "Nome da cliente"},
					"feedback":      map[string]any{"type": "string", "description": "Feedback sobre o que melhorar"},
				},
				Required: []string{"post_id", "feedback"},
			},
		}},
	}
}

// ToolExecutor handles tool call execution for the agent loop.
type ToolExecutor struct {
	App         core.App
	State       *OperatorState
	OperatorJID string
	Generate    content.GenerateFunc
	businesses  []*core.Record // cached on first access
	Posts       []*core.Record // posts referenced during execution, appended to reply programmatically
}

// loadBusinesses returns cached businesses, querying once per executor lifetime.
func (te *ToolExecutor) loadBusinesses() []*core.Record {
	if te.businesses != nil {
		return te.businesses
	}
	te.businesses = loadActiveBusinesses(te.App)
	return te.businesses
}

// bizNameMap returns a map of business ID to display name from the cached businesses.
func (te *ToolExecutor) bizNameMap() map[string]string {
	m := make(map[string]string, len(te.loadBusinesses()))
	for _, biz := range te.businesses {
		m[biz.Id] = biz.GetString("name")
	}
	return m
}

// toolResult is returned by executeTool to signal both the result text and whether a write was triggered.
type toolResult struct {
	Text     string
	ToolName string
	IsWrite  bool
}

// executeTool dispatches a tool call and returns the result.
func (te *ToolExecutor) executeTool(name string, input json.RawMessage, operatorName string) toolResult {
	switch name {
	case "find_customer":
		return toolResult{Text: te.findCustomer(input), ToolName: name}
	case "list_customers":
		return toolResult{Text: te.listCustomers(), ToolName: name}
	case "find_post":
		return toolResult{Text: te.findPost(input), ToolName: name}
	case "list_posts":
		return toolResult{Text: te.listPosts(input), ToolName: name}
	case "recent_activity":
		return toolResult{Text: te.recentActivity(input), ToolName: name}
	case "create_customer":
		return te.createCustomer(input, operatorName)
	case "update_customer":
		return te.updateCustomer(input)
	case "pause_customer":
		return te.pauseCustomer(input)
	case "generate_post":
		return te.generatePost(input)
	case "approve_post":
		return te.approvePost(input, operatorName)
	case "reject_post":
		return te.rejectPost(input, operatorName)
	default:
		return toolResult{Text: "Ferramenta desconhecida: " + name, ToolName: name}
	}
}

// formatPostDetails builds a WhatsApp-friendly block with full post content.
// Deduplicates by post ID in case the same post was referenced multiple times.
// bizNames maps business ID to display name.
func formatPostDetails(bizNames map[string]string, posts []*core.Record) string {
	seen := map[string]bool{}
	var b strings.Builder
	for _, p := range posts {
		if seen[p.Id] {
			continue
		}
		seen[p.Id] = true

		bizID := p.GetString("business")
		name := bizNames[bizID]
		if name == "" {
			name = bizID
		}

		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "*Post %s* (%s)\n", p.Id, name)
		appendPostFieldsJSON(&b, p.GetString("caption"), p.GetString("hashtags"), p.GetString("production_note"))
	}
	return b.String()
}

// appendPostFields writes caption, hashtags, and production note to b.
func appendPostFields(b *strings.Builder, caption string, hashtags []string, productionNote string) {
	fmt.Fprintf(b, "Legenda: %s\n", caption)
	if len(hashtags) > 0 {
		fmt.Fprintf(b, "Hashtags: %s\n", strings.Join(hashtags, " "))
	}
	if productionNote != "" {
		fmt.Fprintf(b, "Nota de produção: %s\n", productionNote)
	}
}

// appendPostFieldsJSON is like appendPostFields but decodes hashtags from a
// raw JSON string (e.g. `["#foo","#bar"]`).
func appendPostFieldsJSON(b *strings.Builder, caption, hashtagsJSON, productionNote string) {
	var tags []string
	if hashtagsJSON != "" {
		json.Unmarshal([]byte(hashtagsJSON), &tags) //nolint:errcheck
	}
	appendPostFields(b, caption, tags, productionNote)
}

// --- Read tool implementations ---

func (te *ToolExecutor) findCustomer(input json.RawMessage) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &args); err != nil || args.Query == "" {
		return "Parâmetro 'query' obrigatório."
	}

	matches := findBusinessRecords(te.loadBusinesses(), args.Query)
	if len(matches) == 0 {
		return fmt.Sprintf("Nenhuma cliente encontrada com '%s'.", args.Query)
	}

	var b strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&b, "Nome: %s\n", m.GetString("name"))
		fmt.Fprintf(&b, "Tipo: %s\n", m.GetString("type"))
		fmt.Fprintf(&b, "Cidade: %s\n", m.GetString("city"))
		if phone := m.GetString("phone"); phone != "" {
			fmt.Fprintf(&b, "Tel: %s\n", phone)
		}
		if ta := m.GetString("target_audience"); ta != "" {
			fmt.Fprintf(&b, "Público: %s\n", ta)
		}
		if bv := m.GetString("brand_vibe"); bv != "" {
			fmt.Fprintf(&b, "Vibe: %s\n", bv)
		}
		if q := m.GetString("quirks"); q != "" {
			fmt.Fprintf(&b, "Obs: %s\n", q)
		}
		fmt.Fprintf(&b, "Status: %s\n", m.GetString("invite_status"))
		b.WriteString("---\n")
	}
	return b.String()
}

func (te *ToolExecutor) listCustomers() string {
	businesses := te.loadBusinesses()

	if len(businesses) == 0 {
		return "Nenhuma cliente ativa no momento."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Clientes ativas: %d\n", len(businesses))
	for _, biz := range businesses {
		fmt.Fprintf(&b, "- %s (%s, %s)\n", biz.GetString("name"), biz.GetString("type"), biz.GetString("city"))
	}
	return b.String()
}

func (te *ToolExecutor) findPost(input json.RawMessage) string {
	var args struct {
		PostID string `json:"post_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil || args.PostID == "" {
		return "Parâmetro 'post_id' obrigatório."
	}

	record, err := te.App.FindRecordById(domain.CollPosts, args.PostID)
	if err != nil {
		return fmt.Sprintf("Post %s não encontrado.", args.PostID)
	}

	bizName := resolveBusinessName(te.App, record, record.GetString("business"))

	te.Posts = append(te.Posts, record)

	status := "pendente"
	if record.GetBool("reviewed") {
		status = "revisado"
	}
	result := fmt.Sprintf("Post: %s | Cliente: %s | Status: %s", record.Id, bizName, status)
	if note := record.GetString("review_note"); note != "" {
		result += " | Nota: " + note
	}
	return result
}

func (te *ToolExecutor) listPosts(input json.RawMessage) string {
	var args struct {
		CustomerName string `json:"customer_name"`
		Status       string `json:"status"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "erro ao processar parâmetros: " + err.Error()
		}
	}

	q := te.App.RecordQuery(domain.CollPosts).OrderBy("created DESC").Limit(20)

	switch args.Status {
	case "pending":
		q = q.AndWhere(dbx.NewExp("reviewed = FALSE OR reviewed = ''"))
	case "reviewed":
		q = q.AndWhere(dbx.NewExp("reviewed = TRUE"))
	}

	businesses := te.loadBusinesses()

	if args.CustomerName != "" {
		matches := findBusinessRecords(businesses, args.CustomerName)
		if len(matches) == 0 {
			return fmt.Sprintf("Nenhuma cliente encontrada com '%s'.", args.CustomerName)
		}
		params := dbx.Params{}
		placeholders := make([]string, len(matches))
		for i, m := range matches {
			key := fmt.Sprintf("bid%d", i)
			placeholders[i] = fmt.Sprintf("{:%s}", key)
			params[key] = m.Id
		}
		q = q.AndWhere(dbx.NewExp("business IN ("+strings.Join(placeholders, ",")+") ", params))
	}

	var posts []*core.Record
	if err := q.All(&posts); err != nil {
		return "Erro ao buscar posts."
	}

	if len(posts) == 0 {
		return "Nenhum post encontrado."
	}

	bizNames := map[string]string{}
	for _, biz := range businesses {
		bizNames[biz.Id] = biz.GetString("name")
	}

	te.Posts = append(te.Posts, posts...)

	var b strings.Builder
	fmt.Fprintf(&b, "Posts: %d\n", len(posts))
	for _, p := range posts {
		name := bizNames[p.GetString("business")]
		if name == "" {
			name = p.GetString("business")
		}
		status := "pendente"
		if p.GetBool("reviewed") {
			status = "revisado"
		}
		fmt.Fprintf(&b, "- %s (%s) [%s]\n", name, p.Id, status)
	}
	b.WriteString("O conteúdo completo dos posts será exibido automaticamente. Não inclua legendas, hashtags ou notas de produção na sua resposta.")
	return b.String()
}

func (te *ToolExecutor) recentActivity(input json.RawMessage) string {
	limit := 5
	if len(input) > 0 {
		var args struct {
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal(input, &args); err == nil && args.Limit > 0 {
			limit = min(args.Limit, 20)
		}
	}

	var actions []*core.Record
	if err := te.App.RecordQuery(domain.CollAgentActionLog).
		OrderBy("created DESC").
		Limit(int64(limit)).
		All(&actions); err != nil {
		return "Erro ao buscar ações recentes."
	}

	if len(actions) == 0 {
		return "Nenhuma ação recente."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Últimas %d ações:\n", len(actions))
	for _, a := range actions {
		result := a.GetString("result")
		if len(result) > 80 {
			result = result[:80] + "..."
		}
		fmt.Fprintf(&b, "- %s: %s (%s)\n", a.GetString("operator_name"), a.GetString("action_type"), result)
	}
	return b.String()
}

// --- Write tool implementations ---
// These validate, store confirmation state, and return a confirmation message.
// The actual execution happens via handleStatefulMessage when the operator confirms.

func (te *ToolExecutor) createCustomer(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		Name           string `json:"name"`
		Type           string `json:"type"`
		City           string `json:"city"`
		Phone          string `json:"phone"`
		TargetAudience string `json:"target_audience"`
		BrandVibe      string `json:"brand_vibe"`
		Quirks         string `json:"quirks"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", ToolName: "create_customer", IsWrite: true}
	}

	p := &CustomerCreateParams{
		Name:  args.Name,
		Type:  args.Type,
		City:  args.City,
		Phone: args.Phone,
	}
	if args.TargetAudience != "" {
		p.TargetAudience = &args.TargetAudience
	}
	if args.BrandVibe != "" {
		p.BrandVibe = &args.BrandVibe
	}
	if args.Quirks != "" {
		p.Quirks = &args.Quirks
	}

	if err := validateCustomerCreate(p, operatorName); err != nil {
		return toolResult{Text: err.Error(), ToolName: "create_customer", IsWrite: true}
	}

	if dup := findDuplicate(te.loadBusinesses(), p.Name); dup != nil {
		return toolResult{
			Text:     fmt.Sprintf("%s já existe (%s, %s).", dup.GetString("name"), dup.GetString("type"), dup.GetString("city")),
			ToolName: "create_customer",
			IsWrite:  true,
		}
	}

	if who, conflict := HasPendingAction(te.App, te.OperatorJID, p.Name); conflict {
		return toolResult{
			Text:     fmt.Sprintf("Outro operador (%s) já tem uma ação pendente pra essa cliente.", who),
			ToolName: "create_customer",
			IsWrite:  true,
		}
	}

	if err := SetConfirming(te.App, te.State, te.OperatorJID, ActionCustomerCreate, p); err != nil {
		return toolResult{Text: "Erro ao salvar estado.", ToolName: "create_customer", IsWrite: true}
	}

	return toolResult{
		Text:     fmt.Sprintf("Confirmação necessária. Dados: %s (%s, %s). Aguardando resposta da operadora.", p.Name, p.Type, p.City),
		ToolName: "create_customer",
		IsWrite:  true,
	}
}

func (te *ToolExecutor) updateCustomer(input json.RawMessage) toolResult {
	var args struct {
		Name           string `json:"name"`
		NewName        string `json:"new_name"`
		Type           string `json:"type"`
		City           string `json:"city"`
		Phone          string `json:"phone"`
		TargetAudience string `json:"target_audience"`
		BrandVibe      string `json:"brand_vibe"`
		Quirks         string `json:"quirks"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", ToolName: "update_customer", IsWrite: true}
	}

	p := &CustomerUpdateParams{Name: args.Name}
	if args.NewName != "" {
		p.NewName = &args.NewName
	}
	if args.Type != "" {
		p.Type = &args.Type
	}
	if args.City != "" {
		p.City = &args.City
	}
	if args.Phone != "" {
		p.Phone = &args.Phone
	}
	if args.TargetAudience != "" {
		p.TargetAudience = &args.TargetAudience
	}
	if args.BrandVibe != "" {
		p.BrandVibe = &args.BrandVibe
	}
	if args.Quirks != "" {
		p.Quirks = &args.Quirks
	}

	if err := SetConfirming(te.App, te.State, te.OperatorJID, ActionCustomerUpdate, p); err != nil {
		return toolResult{Text: "Erro ao salvar estado.", ToolName: "update_customer", IsWrite: true}
	}

	return toolResult{
		Text:     fmt.Sprintf("Confirmação necessária. Alterar dados da %s. Aguardando resposta da operadora.", args.Name),
		ToolName: "update_customer",
		IsWrite:  true,
	}
}

func (te *ToolExecutor) pauseCustomer(input json.RawMessage) toolResult {
	var args struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", ToolName: "pause_customer", IsWrite: true}
	}

	p := &CustomerPauseParams{Name: args.Name}
	if args.Reason != "" {
		p.Reason = &args.Reason
	}

	if err := SetConfirming(te.App, te.State, te.OperatorJID, ActionCustomerPause, p); err != nil {
		return toolResult{Text: "Erro ao salvar estado.", ToolName: "pause_customer", IsWrite: true}
	}

	return toolResult{
		Text:     fmt.Sprintf("Confirmação necessária. Pausar %s. Aguardando resposta da operadora.", args.Name),
		ToolName: "pause_customer",
		IsWrite:  true,
	}
}

func (te *ToolExecutor) generatePost(input json.RawMessage) toolResult {
	var args struct {
		CustomerName string `json:"customer_name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", ToolName: "generate_post", IsWrite: true}
	}

	p := &PostGenerateParams{Name: args.CustomerName}

	if err := SetConfirming(te.App, te.State, te.OperatorJID, ActionPostGenerate, p); err != nil {
		return toolResult{Text: "Erro ao salvar estado.", ToolName: "generate_post", IsWrite: true}
	}

	return toolResult{
		Text:     fmt.Sprintf("Confirmação necessária. Gerar posts para %s. Aguardando resposta da operadora.", args.CustomerName),
		ToolName: "generate_post",
		IsWrite:  true,
	}
}

func (te *ToolExecutor) approvePost(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		PostID       string `json:"post_id"`
		CustomerName string `json:"customer_name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", ToolName: "approve_post", IsWrite: true}
	}

	if err := validatePostApprove(&PostApproveParams{PostId: args.PostID}, operatorName); err != nil {
		return toolResult{Text: err.Error(), ToolName: "approve_post", IsWrite: true}
	}

	p := &PostApproveParams{PostId: args.PostID, Name: args.CustomerName}

	if err := SetConfirming(te.App, te.State, te.OperatorJID, ActionPostApprove, p); err != nil {
		return toolResult{Text: "Erro ao salvar estado.", ToolName: "approve_post", IsWrite: true}
	}

	if record, err := te.App.FindRecordById(domain.CollPosts, args.PostID); err == nil {
		te.Posts = append(te.Posts, record)
	}

	return toolResult{
		Text:     fmt.Sprintf("Confirmação necessária. Aprovar post %s. O conteúdo será exibido automaticamente. Aguardando resposta da operadora.", args.PostID),
		ToolName: "approve_post",
		IsWrite:  true,
	}
}

func (te *ToolExecutor) rejectPost(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		PostID       string `json:"post_id"`
		CustomerName string `json:"customer_name"`
		Feedback     string `json:"feedback"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", ToolName: "reject_post", IsWrite: true}
	}

	if err := validatePostReject(&PostRejectParams{PostId: args.PostID, Feedback: args.Feedback}, operatorName); err != nil {
		return toolResult{Text: err.Error(), ToolName: "reject_post", IsWrite: true}
	}

	p := &PostRejectParams{PostId: args.PostID, Name: args.CustomerName, Feedback: args.Feedback}

	if err := SetConfirming(te.App, te.State, te.OperatorJID, ActionPostReject, p); err != nil {
		return toolResult{Text: "Erro ao salvar estado.", ToolName: "reject_post", IsWrite: true}
	}

	if record, err := te.App.FindRecordById(domain.CollPosts, args.PostID); err == nil {
		te.Posts = append(te.Posts, record)
	}

	return toolResult{
		Text:     fmt.Sprintf("Confirmação necessária. Rejeitar post %s. Feedback: %s. O conteúdo será exibido automaticamente. Aguardando resposta da operadora.", args.PostID, args.Feedback),
		ToolName: "reject_post",
		IsWrite:  true,
	}
}
