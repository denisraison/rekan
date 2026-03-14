package agent_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	bamltypes "github.com/denisraison/rekan/api/internal/baml/baml_client/types"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"

	_ "github.com/denisraison/rekan/api/migrations"
	"github.com/pocketbase/pocketbase/core"
)

// TestCustomerCreate_HappyPath: typed params -> confirm -> verify DB record.
func TestCustomerCreate_HappyPath(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		return bamltypes.AgentResponse{
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name: "Ana",
					Type: "Manicure",
					City: "Goiania",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "cria a Ana, manicure em Goiania", "Elenice", "5511999990000")

	// Verify confirmation message is structured (built from params)
	sent := wa.sentMessages()
	if len(sent) == 0 {
		t.Fatal("expected a reply")
	}
	confirm := sent[len(sent)-1]
	if !strings.Contains(confirm, "Nome: Ana") {
		t.Errorf("confirmation should contain 'Nome: Ana', got: %s", confirm)
	}
	if !strings.Contains(confirm, "Confirma?") {
		t.Errorf("confirmation should end with 'Confirma?', got: %s", confirm)
	}

	// Confirm
	send(a, "sim", "Elenice", "5511999990000")

	// Verify DB record
	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	var biz *core.Record
	for _, b := range allBiz {
		if b.GetString("name") == "Ana" {
			biz = b
			break
		}
	}
	if biz == nil {
		t.Fatal("business 'Ana' not found in DB")
	}
	if biz.GetString("type") != "Manicure" {
		t.Errorf("type: got %q, want %q", biz.GetString("type"), "Manicure")
	}
	if biz.GetString("city") != "Goiania" {
		t.Errorf("city: got %q, want %q", biz.GetString("city"), "Goiania")
	}

	// Verify reply contains "cadastrada"
	allSent := wa.sentMessages()
	found := false
	for _, msg := range allSent {
		if strings.Contains(msg, "cadastrada") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected reply with 'cadastrada', got: %v", allSent)
	}
}

// TestCustomerCreate_MissingRequiredField: empty name -> validation error.
func TestCustomerCreate_MissingRequiredField(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		return bamltypes.AgentResponse{
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name: "",
					Type: "Manicure",
					City: "Goiania",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "cria uma manicure em Goiania", "Elenice", "5511999990000")

	sent := wa.sentMessages()
	found := false
	for _, msg := range sent {
		if strings.Contains(msg, "faltou o nome") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected pt-BR validation error about name, got: %v", sent)
	}

	// Verify no business created
	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	if len(allBiz) > 0 {
		t.Error("no business should be created when validation fails")
	}
}

// TestCustomerCreate_DuplicateName: seed "Ana", try to create another "Ana".
func TestCustomerCreate_DuplicateName(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Ana", "Manicure", "Goiania")
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		return bamltypes.AgentResponse{
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name: "Ana",
					Type: "Salão",
					City: "SP",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "cria a Ana", "Elenice", "5511999990000")

	sent := wa.sentMessages()
	found := false
	for _, msg := range sent {
		if strings.Contains(msg, "já existe") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected duplicate detection message, got: %v", sent)
	}
}

// TestCustomerUpdate_HappyPath: seed business, update city, verify DB.
func TestCustomerUpdate_HappyPath(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Patricia", "Salão", "BH")
	wa := &fakeWAClient{}

	newCity := "Contagem"
	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := "Bruna, alterar a Patricia: cidade pra Contagem. Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_UPDATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerUpdate: &bamltypes.CustomerUpdateParams{
					Name: "Patricia",
					City: &newCity,
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "muda a Patricia pra Contagem", "Bruna", "5511999991111")
	send(a, "sim", "Bruna", "5511999991111")

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range allBiz {
		if b.GetString("name") == "Patricia" {
			if b.GetString("city") != "Contagem" {
				t.Errorf("city: got %q, want %q", b.GetString("city"), "Contagem")
			}
			return
		}
	}
	t.Fatal("Patricia not found in DB")
}

// TestCustomerUpdate_NotFound: update non-existent customer.
func TestCustomerUpdate_NotFound(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	newCity := "SP"
	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		return bamltypes.AgentResponse{
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_UPDATE,
				ActionStatus: bamltypes.AgentActionStatusEXECUTE,
				CustomerUpdate: &bamltypes.CustomerUpdateParams{
					Name: "Inexistente",
					City: &newCity,
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "muda a Inexistente pra SP", "Bruna", "5511999991111")

	sent := wa.sentMessages()
	found := false
	for _, msg := range sent {
		if strings.Contains(msg, "não encontrei") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'not found' message, got: %v", sent)
	}
}

// TestCustomerPause_HappyPath: seed, pause, verify invite_status.
func TestCustomerPause_HappyPath(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Joana", "Loja", "RJ")
	wa := &fakeWAClient{}

	reason := "vai viajar"
	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := "Bruna, pausar a Joana? Motivo: vai viajar. Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_PAUSE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerPause: &bamltypes.CustomerPauseParams{
					Name:   "Joana",
					Reason: &reason,
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "pausa a Joana, vai viajar", "Bruna", "5511999991111")
	send(a, "sim", "Bruna", "5511999991111")

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range allBiz {
		if b.GetString("name") == "Joana" {
			if b.GetString("invite_status") != domain.InviteStatusCancelled {
				t.Errorf("invite_status: got %q, want %q", b.GetString("invite_status"), domain.InviteStatusCancelled)
			}
			return
		}
	}
	t.Fatal("Joana not found in DB")
}

// TestPostGenerate_HappyPath: seed business, confirm generate, verify post created.
func TestPostGenerate_HappyPath(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Patricia", "Salão", "BH")
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := "Elenice, gerar post pra Patricia? Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypePOST_GENERATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				PostGenerate: &bamltypes.PostGenerateParams{
					Name: "Patricia",
				},
			},
		}, nil
	}

	fakeGenerate := func(_ context.Context, _ content.BusinessProfile, _ []content.Role, _ []string) ([]content.Post, error) {
		return []content.Post{{
			Caption:        "Post gerado de teste",
			Hashtags:       []string{"#teste"},
			ProductionNote: "nota",
		}}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	a.Generate = fakeGenerate

	send(a, "gera post pra Patricia", "Elenice", "5511999990000")
	send(a, "sim", "Elenice", "5511999990000")

	// Verify a post was created
	posts, err := app.FindAllRecords(domain.CollPosts)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range posts {
		if strings.Contains(p.GetString("caption"), "Post gerado de teste") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected post to be created after generate confirmation")
	}

	// Verify reply mentions the generated post
	sent := wa.sentMessages()
	postReply := false
	for _, msg := range sent {
		if strings.Contains(msg, "post gerado") {
			postReply = true
			break
		}
	}
	if !postReply {
		t.Errorf("expected reply about generated post, got: %v", sent)
	}
}

// TestPostApprove_HappyPath: seed business + post, approve, verify reviewed=true.
func TestPostApprove_HappyPath(t *testing.T) {
	app := newTestApp(t)
	biz := seedBusiness(t, app, "Patricia", "Salão", "BH")
	post := seedPost(t, app, biz.Id, "Hoje no salão foi dia de transformação...")
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := "Bruna, aprovar post da Patricia? Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypePOST_APPROVE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				PostApprove: &bamltypes.PostApproveParams{
					PostId: post.Id,
					Name:   "Patricia",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "aprova o post da Patricia", "Bruna", "5511999991111")
	send(a, "sim", "Bruna", "5511999991111")

	updated, err := app.FindRecordById(domain.CollPosts, post.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.GetBool("reviewed") {
		t.Error("post should be reviewed=true after approval")
	}
}

// TestPostReject_WithFeedback: reject with feedback, verify reviewed=true and review_note.
func TestPostReject_WithFeedback(t *testing.T) {
	app := newTestApp(t)
	biz := seedBusiness(t, app, "Maria", "Confeitaria", "SP")
	post := seedPost(t, app, biz.Id, "Bolo caseiro é sempre a melhor pedida...")
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := "Elenice, rejeitar post da Maria com feedback? Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypePOST_REJECT,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				PostReject: &bamltypes.PostRejectParams{
					PostId:   post.Id,
					Name:     "Maria",
					Feedback: "muito genérico",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "rejeita o post da Maria, muito genérico", "Elenice", "5511999990000")
	send(a, "sim", "Elenice", "5511999990000")

	updated, err := app.FindRecordById(domain.CollPosts, post.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.GetBool("reviewed") {
		t.Error("post should be reviewed=true after rejection")
	}
	if updated.GetString("review_note") != "muito genérico" {
		t.Errorf("review_note: got %q, want %q", updated.GetString("review_note"), "muito genérico")
	}
}

// TestConfirmationMessage_BuiltFromParams: verify structured confirmation, not LLM free text.
func TestConfirmationMessage_BuiltFromParams(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	audience := "mulheres do bairro"
	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := "Elenice, vou cadastrar a Ana!"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name:           "Ana",
					Type:           "Manicure",
					City:           "Goiania",
					TargetAudience: &audience,
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "cria a Ana, manicure, Goiania, mulheres do bairro", "Elenice", "5511999990000")

	sent := wa.sentMessages()
	if len(sent) == 0 {
		t.Fatal("expected reply")
	}

	confirm := sent[len(sent)-1]
	// Must contain exact field values from typed params
	for _, expect := range []string{"Nome: Ana", "Tipo: Manicure", "Cidade: Goiania", "Público: mulheres do bairro"} {
		if !strings.Contains(confirm, expect) {
			t.Errorf("confirmation missing %q, got: %s", expect, confirm)
		}
	}
	// Must NOT be the LLM free text reply
	if strings.Contains(confirm, "vou cadastrar a Ana!") {
		t.Error("confirmation should be built from params, not LLM reply")
	}
}

// TestValidation_ReturnsPortugueseError: validation errors address operator by name.
func TestValidation_ReturnsPortugueseError(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		return bamltypes.AgentResponse{
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name: "Ana",
					Type: "",
					City: "SP",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "cria a Ana", "Bruna", "5511999991111")

	sent := wa.sentMessages()
	found := false
	for _, msg := range sent {
		if strings.Contains(msg, "Bruna") && strings.Contains(msg, "faltou") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected pt-BR error addressing Bruna, got: %v", sent)
	}
}

// TestDoubleConfirmation_Idempotent: send "sim" twice, only one DB record.
func TestDoubleConfirmation_Idempotent(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	var callCount int
	var mu sync.Mutex
	fakeBAML := func(_ context.Context, _, _, _, _ string) (bamltypes.AgentResponse, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return bamltypes.AgentResponse{
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name: "Ana",
					Type: "Manicure",
					City: "SP",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)
	send(a, "cria a Ana, manicure em SP", "Elenice", "5511999990000")
	send(a, "sim", "Elenice", "5511999990000")
	send(a, "sim", "Elenice", "5511999990000") // second "sim" should not create duplicate

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, b := range allBiz {
		if b.GetString("name") == "Ana" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 business 'Ana', got %d", count)
	}
}

// seedPost creates a test post record.
func seedPost(t *testing.T, app core.App, businessID, caption string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(domain.CollPosts)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Set("business", businessID)
	record.Set("caption", caption)
	record.Set("hashtags", []string{"#test"})
	record.Set("production_note", "test")
	record.Set("role", "bastidor")
	record.Set("hook", "test hook")
	record.Set("reviewed", false)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}
