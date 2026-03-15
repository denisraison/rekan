package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

// ToolExecutor handles tool call execution for the agent loop.
type ToolExecutor struct {
	Ctx        context.Context
	App        core.App
	WAClient   WAClient
	Generate   content.GenerateFunc
	businesses []*core.Record // cached on first access
	Posts      []*core.Record // posts referenced during execution, appended to reply programmatically
	WriteUsed  bool           // whether any write tool was called
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

// buildTools constructs the full set of agent tools with closures over the executor.
func buildTools(executor *ToolExecutor, operatorName string) []Tool {
	// Helper to build a JSON schema from properties and required fields
	schema := func(props map[string]any, required ...string) json.RawMessage {
		s := map[string]any{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			s["required"] = required
		}
		return marshalSchema(s)
	}

	readTool := func(name, desc string, inputSchema json.RawMessage, fn func(json.RawMessage) string) Tool {
		return Tool{
			Name:        name,
			Description: desc,
			InputSchema: inputSchema,
			Execute: func(_ context.Context, input json.RawMessage) (string, error) {
				return fn(input), nil
			},
		}
	}

	writeTool := func(name, desc string, inputSchema json.RawMessage, fn func(json.RawMessage) string) Tool {
		return Tool{
			Name:        name,
			Description: desc,
			InputSchema: inputSchema,
			Execute: func(_ context.Context, input json.RawMessage) (string, error) {
				executor.WriteUsed = true
				return fn(input), nil
			},
		}
	}

	return []Tool{
		// Read tools
		readTool("search_customers",
			"Busca clientes. Sem query: lista todas. Com query: busca por nome (match fuzzy) e retorna detalhes.",
			schema(map[string]any{
				"query": map[string]any{"type": "string", "description": "Nome ou parte do nome da cliente (opcional)"},
			}),
			func(input json.RawMessage) string { return executor.searchCustomers(input) },
		),
		readTool("search_posts",
			"Busca posts. Sem post_id: lista posts com preview. Com post_id: retorna post completo.",
			schema(map[string]any{
				"post_id":       map[string]any{"type": "string", "description": "ID do post (opcional, para busca específica)"},
				"customer_name": map[string]any{"type": "string", "description": "Nome da cliente (opcional)"},
				"status":        map[string]any{"type": "string", "enum": []string{"pending", "reviewed", "all"}, "description": "Filtro de status (padrão: all)"},
			}),
			func(input json.RawMessage) string { return executor.searchPosts(input) },
		),
		// Write tools
		writeTool("create_customer",
			"Cadastra nova cliente. Campos obrigatórios: name, type, city, phone.",
			schema(map[string]any{
				"name":            map[string]any{"type": "string", "description": "Nome da cliente"},
				"type":            map[string]any{"type": "string", "description": "Tipo de negócio (ex: Salão de Beleza, Confeitaria)"},
				"city":            map[string]any{"type": "string", "description": "Cidade"},
				"phone":           map[string]any{"type": "string", "description": "Telefone da cliente"},
				"target_audience": map[string]any{"type": "string", "description": "Público-alvo (opcional)"},
				"brand_vibe":      map[string]any{"type": "string", "description": "Vibe da marca (opcional)"},
				"quirks":          map[string]any{"type": "string", "description": "Observações especiais (opcional)"},
			}, "name", "type", "city", "phone"),
			func(input json.RawMessage) string { return executor.createCustomer(input, operatorName) },
		),
		writeTool("update_customer",
			"Altera dados de uma cliente existente. Apenas name é obrigatório (identifica a cliente).",
			schema(map[string]any{
				"name":            map[string]any{"type": "string", "description": "Nome da cliente (para identificação)"},
				"new_name":        map[string]any{"type": "string", "description": "Novo nome do negócio (se quiser renomear)"},
				"type":            map[string]any{"type": "string", "description": "Novo tipo de negócio"},
				"city":            map[string]any{"type": "string", "description": "Nova cidade"},
				"phone":           map[string]any{"type": "string", "description": "Novo telefone"},
				"target_audience": map[string]any{"type": "string", "description": "Novo público-alvo"},
				"brand_vibe":      map[string]any{"type": "string", "description": "Nova vibe da marca"},
				"quirks":          map[string]any{"type": "string", "description": "Novas observações"},
			}, "name"),
			func(input json.RawMessage) string { return executor.updateCustomer(input, operatorName) },
		),
		writeTool("pause_customer",
			"Pausa uma cliente.",
			schema(map[string]any{
				"name":   map[string]any{"type": "string", "description": "Nome da cliente"},
				"reason": map[string]any{"type": "string", "description": "Motivo da pausa (opcional)"},
			}, "name"),
			func(input json.RawMessage) string { return executor.pauseCustomer(input, operatorName) },
		),
		writeTool("generate_post",
			"Gera posts para uma cliente.",
			schema(map[string]any{
				"customer_name": map[string]any{"type": "string", "description": "Nome da cliente"},
			}, "customer_name"),
			func(input json.RawMessage) string { return executor.generatePost(input, operatorName) },
		),
		writeTool("approve_post",
			"Aprova um post pendente.",
			schema(map[string]any{
				"post_id": map[string]any{"type": "string", "description": "ID do post"},
			}, "post_id"),
			func(input json.RawMessage) string { return executor.approvePost(input, operatorName) },
		),
		writeTool("reject_post",
			"Rejeita um post com feedback.",
			schema(map[string]any{
				"post_id":  map[string]any{"type": "string", "description": "ID do post"},
				"feedback": map[string]any{"type": "string", "description": "Feedback sobre o que melhorar"},
			}, "post_id", "feedback"),
			func(input json.RawMessage) string { return executor.rejectPost(input, operatorName) },
		),
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

func postStatus(reviewed bool) string {
	if reviewed {
		return "revisado"
	}
	return "pendente"
}

func shortPostID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
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
	for _, biz := range te.loadBusinesses() {
		if biz.Id == bizID {
			return biz.GetString("name")
		}
	}
	return bizID
}

// resolveCustomer finds exactly one business by name, returning an error string on 0 or 2+ matches.
func (te *ToolExecutor) resolveCustomer(name, operatorName string) (*core.Record, string) {
	matches := service.FindBusinessByName(te.loadBusinesses(), name)
	if len(matches) == 0 {
		return nil, fmt.Sprintf("%s, não encontrei cliente '%s'.", operatorName, name)
	}
	if len(matches) > 1 {
		return nil, disambiguate(operatorName, matches)
	}
	return matches[0], ""
}

// --- Read tool implementations ---

func (te *ToolExecutor) searchCustomers(input json.RawMessage) string {
	var args struct {
		Query string `json:"query"`
	}
	if len(input) > 0 {
		json.Unmarshal(input, &args) //nolint:errcheck
	}

	// No query: list all
	if args.Query == "" {
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

	// With query: fuzzy search with full details
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

func (te *ToolExecutor) searchPosts(input json.RawMessage) string {
	var args struct {
		PostID       string `json:"post_id"`
		CustomerName string `json:"customer_name"`
		Status       string `json:"status"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "erro ao processar parâmetros: " + err.Error()
		}
	}

	// Single post detail view
	if args.PostID != "" {
		record, err := te.App.FindRecordById(domain.CollPosts, args.PostID)
		if err != nil {
			return fmt.Sprintf("Post %s não encontrado.", args.PostID)
		}
		te.Posts = append(te.Posts, record)

		bizName := te.resolveBizName(record)
		var b strings.Builder
		fmt.Fprintf(&b, "id:%s\n", record.Id)
		fmt.Fprintf(&b, "customer:%s\n", bizName)
		fmt.Fprintf(&b, "status:%s\n", postStatus(record.GetBool("reviewed")))
		fmt.Fprintf(&b, "caption:%s\n", record.GetString("caption"))
		if hashtags := record.GetString("hashtags"); hashtags != "" {
			fmt.Fprintf(&b, "hashtags:%s\n", strings.Join(decodeHashtags(hashtags), " "))
		}
		if note := record.GetString("production_note"); note != "" {
			fmt.Fprintf(&b, "production_note:%s\n", note)
		}
		if reviewNote := record.GetString("review_note"); reviewNote != "" {
			fmt.Fprintf(&b, "review_note:%s\n", reviewNote)
		}
		return b.String()
	}

	// List view
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
	for _, p := range posts {
		name := bizNames[p.GetString("business")]
		if name == "" {
			name = p.GetString("business")
		}
		preview := truncate(p.GetString("caption"), 60)
		fmt.Fprintf(&b, "id:%s customer:%s status:%s preview:\"%s\"\n", shortPostID(p.Id), name, postStatus(p.GetBool("reviewed")), preview)
	}
	return b.String()
}

// --- Write tool implementations ---

func (te *ToolExecutor) createCustomer(input json.RawMessage, operatorName string) string {
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
		return "Erro ao ler parâmetros."
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
		return err.Error()
	}

	if dup := service.FindDuplicate(te.loadBusinesses(), p.Name); dup != nil {
		return fmt.Sprintf("%s já existe (%s, %s).", dup.GetString("name"), dup.GetString("type"), dup.GetString("city"))
	}

	record, err := service.CreateBusiness(te.App, p)
	if err != nil {
		return "Erro ao cadastrar: " + err.Error()
	}
	te.businesses = nil // invalidate cache
	return fmt.Sprintf("%s, %s cadastrada! (%s, %s)", operatorName, record.GetString("name"), record.GetString("type"), record.GetString("city"))
}

func (te *ToolExecutor) updateCustomer(input json.RawMessage, operatorName string) string {
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
		return "Erro ao ler parâmetros."
	}

	record, errMsg := te.resolveCustomer(args.Name, operatorName)
	if errMsg != "" {
		return errMsg
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
		return "Erro ao alterar: " + err.Error()
	}
	if len(updatedKeys) == 0 {
		return fmt.Sprintf("%s, nenhum campo pra atualizar na %s.", operatorName, args.Name)
	}

	labels := make([]string, len(updatedKeys))
	for i, key := range updatedKeys {
		labels[i] = fieldLabel(key)
	}

	te.businesses = nil // invalidate cache

	displayName := args.Name
	if args.NewName != "" {
		displayName = args.NewName
	}
	return fmt.Sprintf("%s, %s atualizada! Campos: %s.", operatorName, displayName, strings.Join(labels, ", "))
}

func (te *ToolExecutor) pauseCustomer(input json.RawMessage, operatorName string) string {
	var args struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	record, errMsg := te.resolveCustomer(args.Name, operatorName)
	if errMsg != "" {
		return errMsg
	}

	if err := service.PauseBusiness(te.App, record); err != nil {
		return "Erro ao pausar: " + err.Error()
	}
	te.businesses = nil // invalidate cache

	msg := fmt.Sprintf("%s, %s pausada.", operatorName, args.Name)
	if args.Reason != "" {
		msg = fmt.Sprintf("%s, %s pausada. Motivo: %s.", operatorName, args.Name, args.Reason)
	}
	return msg
}

func (te *ToolExecutor) generatePost(input json.RawMessage, operatorName string) string {
	var args struct {
		CustomerName string `json:"customer_name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	if args.CustomerName == "" {
		return operatorName + ", pra qual cliente você quer gerar post?"
	}

	biz, errMsg := te.resolveCustomer(args.CustomerName, operatorName)
	if errMsg != "" {
		return errMsg
	}
	if te.Generate == nil {
		return operatorName + ", geração de posts não está configurada."
	}

	result, err := service.GeneratePosts(te.Ctx, te.App, te.Generate, biz.Id)
	if err != nil {
		return "Erro ao gerar: " + err.Error()
	}
	if len(result.Posts) == 0 {
		return fmt.Sprintf("%s, não consegui gerar post pra %s.", operatorName, biz.GetString("name"))
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
	return b.String()
}

func (te *ToolExecutor) approvePost(input json.RawMessage, operatorName string) string {
	var args struct {
		PostID string `json:"post_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	if args.PostID == "" {
		return operatorName + ", qual post você quer aprovar?"
	}

	record, err := service.ApprovePost(te.App, args.PostID)
	if err != nil {
		return "Erro ao aprovar: " + err.Error()
	}

	te.Posts = append(te.Posts, record)
	bizName := te.resolveBizName(record)
	result := fmt.Sprintf("%s, post da %s aprovado!", operatorName, bizName)

	if te.WAClient != nil {
		if sendErr := te.sendPostToClient(record); sendErr != nil {
			return result + " (mas não consegui enviar pro cliente: " + sendErr.Error() + ")"
		}
		result += " Enviado pro cliente!"
	}

	return result
}

func (te *ToolExecutor) rejectPost(input json.RawMessage, operatorName string) string {
	var args struct {
		PostID   string `json:"post_id"`
		Feedback string `json:"feedback"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "Erro ao ler parâmetros."
	}

	if args.PostID == "" {
		return operatorName + ", qual post você quer rejeitar?"
	}

	record, err := service.RejectPost(te.App, args.PostID, args.Feedback)
	if err != nil {
		return "Erro ao rejeitar: " + err.Error()
	}

	te.Posts = append(te.Posts, record)
	bizName := te.resolveBizName(record)

	msg := fmt.Sprintf("%s, post da %s rejeitado.", operatorName, bizName)
	if args.Feedback != "" {
		msg = fmt.Sprintf("%s, post da %s rejeitado. Feedback: %s.", operatorName, bizName, args.Feedback)
	}
	return msg
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
