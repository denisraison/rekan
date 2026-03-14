package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
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
			Description: anthropic.String("Cadastra nova cliente. Campos obrigatórios: name, type, city, phone."),
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
			Description: anthropic.String("Altera dados de uma cliente existente. Apenas name é obrigatório (identifica a cliente)."),
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
			Description: anthropic.String("Pausa uma cliente."),
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
			Description: anthropic.String("Gera posts para uma cliente."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"customer_name": map[string]any{"type": "string", "description": "Nome da cliente"},
				},
				Required: []string{"customer_name"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "approve_post",
			Description: anthropic.String("Aprova um post pendente."),
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
			Description: anthropic.String("Rejeita um post com feedback."),
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
	Ctx        context.Context
	App        core.App
	WAClient   WAClient
	Generate   content.GenerateFunc
	businesses []*core.Record // cached on first access
	Posts      []*core.Record // posts referenced during execution, appended to reply programmatically
}

// loadBusinesses returns cached businesses, querying once per executor lifetime.
func (te *ToolExecutor) loadBusinesses() []*core.Record {
	if te.businesses != nil {
		return te.businesses
	}
	te.businesses = service.ListActiveBusinesses(te.App)
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
	Text    string
	IsWrite bool
}

// executeTool dispatches a tool call and returns the result.
func (te *ToolExecutor) executeTool(name string, input json.RawMessage, operatorName string) toolResult {
	switch name {
	case "find_customer":
		return toolResult{Text: te.findCustomer(input)}
	case "list_customers":
		return toolResult{Text: te.listCustomers()}
	case "find_post":
		return toolResult{Text: te.findPost(input)}
	case "list_posts":
		return toolResult{Text: te.listPosts(input)}
	case "recent_activity":
		return toolResult{Text: te.recentActivity(input)}
	case "create_customer":
		return te.createCustomer(input, operatorName)
	case "update_customer":
		return te.updateCustomer(input, operatorName)
	case "pause_customer":
		return te.pauseCustomer(input, operatorName)
	case "generate_post":
		return te.generatePost(input, operatorName)
	case "approve_post":
		return te.approvePost(input, operatorName)
	case "reject_post":
		return te.rejectPost(input, operatorName)
	default:
		return toolResult{Text: "Ferramenta desconhecida: " + name}
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

// decodeHashtags parses a JSON array string (e.g. `["#foo","#bar"]`) into a slice.
func decodeHashtags(hashtagsJSON string) []string {
	if hashtagsJSON == "" {
		return nil
	}
	var tags []string
	json.Unmarshal([]byte(hashtagsJSON), &tags) //nolint:errcheck
	return tags
}

// appendPostFieldsJSON is like appendPostFields but decodes hashtags from a
// raw JSON string (e.g. `["#foo","#bar"]`).
func appendPostFieldsJSON(b *strings.Builder, caption, hashtagsJSON, productionNote string) {
	appendPostFields(b, caption, decodeHashtags(hashtagsJSON), productionNote)
}

var fieldLabels = map[string]string{
	"name":            "nome",
	"type":            "tipo",
	"city":            "cidade",
	"phone":           "telefone",
	"target_audience": "público",
	"brand_vibe":      "vibe",
	"quirks":          "obs",
}

func fieldLabel(key string) string {
	if label, ok := fieldLabels[key]; ok {
		return label
	}
	return key
}

// resolveBizName returns the business display name for a post record, using the cached businesses.
func (te *ToolExecutor) resolveBizName(postRecord *core.Record) string {
	bizID := postRecord.GetString("business")
	if name := te.bizNameMap()[bizID]; name != "" {
		return name
	}
	return bizID
}

// resolveCustomer finds exactly one business by name, returning an error toolResult on 0 or 2+ matches.
func (te *ToolExecutor) resolveCustomer(name, operatorName string) (*core.Record, *toolResult) {
	matches := service.FindBusinessByName(te.loadBusinesses(), name)
	if len(matches) == 0 {
		return nil, &toolResult{Text: fmt.Sprintf("%s, não encontrei cliente '%s'.", operatorName, name), IsWrite: true}
	}
	if len(matches) > 1 {
		return nil, &toolResult{Text: disambiguate(operatorName, matches), IsWrite: true}
	}
	return matches[0], nil
}

// --- Read tool implementations ---

func (te *ToolExecutor) findCustomer(input json.RawMessage) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &args); err != nil || args.Query == "" {
		return "Parâmetro 'query' obrigatório."
	}

	matches := service.FindBusinessByName(te.loadBusinesses(), args.Query)
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

	bizName := te.resolveBizName(record)
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

	filter := service.ListPostsFilter{Status: args.Status}

	if args.CustomerName != "" {
		matches := service.FindBusinessByName(te.loadBusinesses(), args.CustomerName)
		if len(matches) == 0 {
			return fmt.Sprintf("Nenhuma cliente encontrada com '%s'.", args.CustomerName)
		}
		for _, m := range matches {
			filter.BusinessIDs = append(filter.BusinessIDs, m.Id)
		}
	}

	posts, err := service.ListPosts(te.App, filter)
	if err != nil {
		return "Erro ao buscar posts."
	}
	if len(posts) == 0 {
		return "Nenhum post encontrado."
	}

	te.Posts = append(te.Posts, posts...)

	bizNames := te.bizNameMap()
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

	actions, err := service.RecentActions(te.App, limit)
	if err != nil {
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
		return toolResult{Text: "Erro ao ler parâmetros.", IsWrite: true}
	}

	p := service.CreateBusinessParams{
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
		return toolResult{Text: err.Error(), IsWrite: true}
	}

	if dup := service.FindDuplicate(te.loadBusinesses(), p.Name); dup != nil {
		return toolResult{
			Text:    fmt.Sprintf("%s já existe (%s, %s).", dup.GetString("name"), dup.GetString("type"), dup.GetString("city")),
			IsWrite: true,
		}
	}

	record, err := service.CreateBusiness(te.App, p)
	if err != nil {
		return toolResult{Text: "Erro ao cadastrar: " + err.Error(), IsWrite: true}
	}
	te.businesses = nil // invalidate cache so subsequent tools see the new customer
	return toolResult{
		Text:    fmt.Sprintf("%s, %s cadastrada! (%s, %s)", operatorName, record.GetString("name"), record.GetString("type"), record.GetString("city")),
		IsWrite: true,
	}
}

func (te *ToolExecutor) updateCustomer(input json.RawMessage, operatorName string) toolResult {
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
		return toolResult{Text: "Erro ao ler parâmetros.", IsWrite: true}
	}

	record, errResult := te.resolveCustomer(args.Name, operatorName)
	if errResult != nil {
		return *errResult
	}

	p := service.UpdateBusinessParams{}
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

	updatedKeys, err := service.UpdateBusiness(te.App, record, p)
	if err != nil {
		return toolResult{Text: "Erro ao alterar: " + err.Error(), IsWrite: true}
	}
	if len(updatedKeys) == 0 {
		return toolResult{Text: fmt.Sprintf("%s, nenhum campo pra atualizar na %s.", operatorName, args.Name), IsWrite: true}
	}

	labels := make([]string, len(updatedKeys))
	for i, key := range updatedKeys {
		labels[i] = fieldLabel(key)
	}

	te.businesses = nil // invalidate cache so subsequent tools see the update

	displayName := args.Name
	if args.NewName != "" {
		displayName = args.NewName
	}
	return toolResult{
		Text:    fmt.Sprintf("%s, %s atualizada! Campos: %s.", operatorName, displayName, strings.Join(labels, ", ")),
		IsWrite: true,
	}
}

func (te *ToolExecutor) pauseCustomer(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", IsWrite: true}
	}

	record, errResult := te.resolveCustomer(args.Name, operatorName)
	if errResult != nil {
		return *errResult
	}

	if err := service.PauseBusiness(te.App, record); err != nil {
		return toolResult{Text: "Erro ao pausar: " + err.Error(), IsWrite: true}
	}
	te.businesses = nil // invalidate cache so subsequent tools see the pause

	msg := fmt.Sprintf("%s, %s pausada.", operatorName, args.Name)
	if args.Reason != "" {
		msg = fmt.Sprintf("%s, %s pausada. Motivo: %s.", operatorName, args.Name, args.Reason)
	}
	return toolResult{Text: msg, IsWrite: true}
}

func (te *ToolExecutor) generatePost(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		CustomerName string `json:"customer_name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", IsWrite: true}
	}

	if args.CustomerName == "" {
		return toolResult{Text: operatorName + ", pra qual cliente você quer gerar post?", IsWrite: true}
	}

	biz, errResult := te.resolveCustomer(args.CustomerName, operatorName)
	if errResult != nil {
		return *errResult
	}
	if te.Generate == nil {
		return toolResult{Text: operatorName + ", geração de posts não está configurada.", IsWrite: true}
	}

	result, err := service.GeneratePosts(te.Ctx, te.App, te.Generate, biz.Id)
	if err != nil {
		return toolResult{Text: "Erro ao gerar: " + err.Error(), IsWrite: true}
	}
	if len(result.Posts) == 0 {
		return toolResult{Text: fmt.Sprintf("%s, não consegui gerar post pra %s.", operatorName, biz.GetString("name")), IsWrite: true}
	}

	post := result.Posts[0]
	var b strings.Builder
	fmt.Fprintf(&b, "%s, post gerado pra %s!\n\n", operatorName, biz.GetString("name"))
	appendPostFields(&b, post.Caption, post.Hashtags, post.ProductionNote)
	fmt.Fprintf(&b, "ID: %s", post.ID)

	for _, p := range result.Posts {
		if record, findErr := te.App.FindRecordById(domain.CollPosts, p.ID); findErr == nil {
			te.Posts = append(te.Posts, record)
		}
	}
	return toolResult{Text: b.String(), IsWrite: true}
}

func (te *ToolExecutor) approvePost(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		PostID       string `json:"post_id"`
		CustomerName string `json:"customer_name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", IsWrite: true}
	}

	if args.PostID == "" {
		return toolResult{Text: operatorName + ", qual post você quer aprovar?", IsWrite: true}
	}

	record, err := service.ApprovePost(te.App, args.PostID)
	if err != nil {
		return toolResult{Text: "Erro ao aprovar: " + err.Error(), IsWrite: true}
	}

	te.Posts = append(te.Posts, record)
	bizName := te.resolveBizName(record)
	result := fmt.Sprintf("%s, post da %s aprovado!", operatorName, bizName)

	if te.WAClient != nil {
		if sendErr := te.sendPostToClient(record); sendErr != nil {
			return toolResult{Text: result + " (mas não consegui enviar pro cliente: " + sendErr.Error() + ")", IsWrite: true}
		}
		result += " Enviado pro cliente!"
	}

	return toolResult{Text: result, IsWrite: true}
}

func (te *ToolExecutor) rejectPost(input json.RawMessage, operatorName string) toolResult {
	var args struct {
		PostID       string `json:"post_id"`
		CustomerName string `json:"customer_name"`
		Feedback     string `json:"feedback"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return toolResult{Text: "Erro ao ler parâmetros.", IsWrite: true}
	}

	if args.PostID == "" {
		return toolResult{Text: operatorName + ", qual post você quer rejeitar?", IsWrite: true}
	}

	record, err := service.RejectPost(te.App, args.PostID, args.Feedback)
	if err != nil {
		return toolResult{Text: "Erro ao rejeitar: " + err.Error(), IsWrite: true}
	}

	te.Posts = append(te.Posts, record)
	bizName := te.resolveBizName(record)

	msg := fmt.Sprintf("%s, post da %s rejeitado.", operatorName, bizName)
	if args.Feedback != "" {
		msg = fmt.Sprintf("%s, post da %s rejeitado. Feedback: %s.", operatorName, bizName, args.Feedback)
	}
	return toolResult{Text: msg, IsWrite: true}
}

// sendPostToClient sends the post content to the client's WhatsApp.
func (te *ToolExecutor) sendPostToClient(post *core.Record) error {
	return service.SendTextMessage(te.Ctx, te.App, te.WAClient, service.SendTextParams{
		BusinessID:     post.GetString("business"),
		Caption:        post.GetString("caption"),
		Hashtags:       strings.Join(decodeHashtags(post.GetString("hashtags")), " "),
		ProductionNote: post.GetString("production_note"),
	})
}
