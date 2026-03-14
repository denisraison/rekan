package agent_test

import (
	"context"
	"strings"
	"testing"

	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"

	"github.com/denisraison/rekan/api/internal/agent"
	_ "github.com/denisraison/rekan/api/migrations"
	"github.com/pocketbase/pocketbase/core"
)

// TestCustomerCreate_HappyPath: set confirming state -> confirm -> verify DB record.
func TestCustomerCreate_HappyPath(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	setConfirmingState(t, app, "5511999990000", agent.ActionCustomerCreate, &agent.CustomerCreateParams{
		Name:  "Ana",
		Type:  "Manicure",
		City:  "Goiania",
		Phone: "62999990000",
	})

	a := newAgent(t, app, wa)

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

// TestCustomerUpdate_HappyPath: seed business, set confirming state, confirm, verify DB.
func TestCustomerUpdate_HappyPath(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Patricia", "Salão", "BH")
	wa := &fakeWAClient{}

	newCity := "Contagem"
	setConfirmingState(t, app, "5511999991111", agent.ActionCustomerUpdate, &agent.CustomerUpdateParams{
		Name: "Patricia",
		City: &newCity,
	})

	a := newAgent(t, app, wa)
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
	setConfirmingState(t, app, "5511999991111", agent.ActionCustomerUpdate, &agent.CustomerUpdateParams{
		Name: "Inexistente",
		City: &newCity,
	})

	a := newAgent(t, app, wa)
	send(a, "sim", "Bruna", "5511999991111")

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

// TestCustomerPause_HappyPath: seed, set confirming state, confirm, verify invite_status.
func TestCustomerPause_HappyPath(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Joana", "Loja", "RJ")
	wa := &fakeWAClient{}

	reason := "vai viajar"
	setConfirmingState(t, app, "5511999991111", agent.ActionCustomerPause, &agent.CustomerPauseParams{
		Name:   "Joana",
		Reason: &reason,
	})

	a := newAgent(t, app, wa)
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

	setConfirmingState(t, app, "5511999990000", agent.ActionPostGenerate, &agent.PostGenerateParams{
		Name: "Patricia",
	})

	fakeGenerate := func(_ context.Context, _ content.BusinessProfile, _ []content.Role, _ []string) ([]content.Post, error) {
		return []content.Post{{
			Caption:        "Post gerado de teste",
			Hashtags:       []string{"#teste"},
			ProductionNote: "nota",
		}}, nil
	}

	a := newAgent(t, app, wa)
	a.Generate = fakeGenerate

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

// TestPostApprove_HappyPath: seed business + post, confirm approve, verify reviewed=true.
func TestPostApprove_HappyPath(t *testing.T) {
	app := newTestApp(t)
	biz := seedBusiness(t, app, "Patricia", "Salão", "BH")
	post := seedPost(t, app, biz.Id, "Hoje no salão foi dia de transformação...")
	wa := &fakeWAClient{}

	setConfirmingState(t, app, "5511999991111", agent.ActionPostApprove, &agent.PostApproveParams{
		PostId: post.Id,
		Name:   "Patricia",
	})

	a := newAgent(t, app, wa)
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

	setConfirmingState(t, app, "5511999990000", agent.ActionPostReject, &agent.PostRejectParams{
		PostId:   post.Id,
		Name:     "Maria",
		Feedback: "muito genérico",
	})

	a := newAgent(t, app, wa)
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

// TestDoubleConfirmation_Idempotent: send "sim" twice, only one DB record.
func TestDoubleConfirmation_Idempotent(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	setConfirmingState(t, app, "5511999990000", agent.ActionCustomerCreate, &agent.CustomerCreateParams{
		Name:  "Ana",
		Type:  "Manicure",
		City:  "SP",
		Phone: "11999990000",
	})

	a := newAgent(t, app, wa)
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
