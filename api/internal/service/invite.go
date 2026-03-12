package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	asaasclient "github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/pricing"
	"github.com/denisraison/rekan/api/internal/terms"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/pocketbase/pocketbase/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

var errAlreadyClaimed = errors.New("invite already claimed")

type SendInviteResult struct {
	InviteURL string
}

func SendInvite(ctx context.Context, app core.App, wa *whatsapp.Client, asaas *asaasclient.Client, businessID, appURL string) (*SendInviteResult, error) {
	business, err := app.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return nil, wrapNotFound(err, "negócio não encontrado")
	}
	phone := business.GetString("phone")
	if phone == "" {
		return nil, ErrNoPhone
	}

	tier := business.GetString("tier")
	commitment := business.GetString("commitment")
	if !pricing.ValidTier(tier) || !pricing.ValidCommitment(commitment) {
		return nil, errors.New("plano e compromisso devem ser definidos antes de enviar o convite")
	}

	status := business.GetString("invite_status")
	switch status {
	case domain.InviteStatusAccepted, domain.InviteStatusActive:
		return nil, fmt.Errorf("%w: convite já aceito ou assinatura ativa", ErrConflict)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	inviteURL := appURL + "/convite/" + token
	clientName := business.GetString("client_name")
	text := "Oi " + clientName + "! Segue o link pra ativar seu acesso ao Rekan: " + inviteURL

	jid := types.NewJID(phone, types.DefaultUserServer)
	_, err = wa.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: &text,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	business.Set("invite_token", token)
	business.Set("invite_status", domain.InviteStatusInvited)
	business.Set("invite_sent_at", now)
	if err := app.Save(business); err != nil {
		return nil, err
	}

	StoreOutgoingMessage(app, businessID, phone, domain.MsgTypeText, text, nil)

	return &SendInviteResult{InviteURL: inviteURL}, nil
}

func AcceptInvite(ctx context.Context, app core.App, asaas *asaasclient.Client, token, cpfCnpj string) (string, error) {
	business, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
	if err != nil {
		return "", wrapNotFound(err, "convite não encontrado")
	}

	sentAt := business.GetDateTime("invite_sent_at").Time()
	if time.Since(sentAt) > 7*24*time.Hour {
		return "", errors.New("convite expirado")
	}

	status := business.GetString("invite_status")

	// Idempotency: already accepted with authorization, return stored qr_payload
	if status == domain.InviteStatusAccepted && business.GetString("authorization_id") != "" {
		return business.GetString("qr_payload"), nil
	}

	if status == domain.InviteStatusActive {
		return "", fmt.Errorf("%w: assinatura já está ativa", ErrConflict)
	}

	if status != domain.InviteStatusInvited {
		return "", errors.New("convite não pode ser aceito neste estado")
	}

	tier := pricing.Tier(business.GetString("tier"))
	commitment := pricing.Commitment(business.GetString("commitment"))
	price, ok := pricing.Price(tier, commitment)
	if !ok {
		return "", errors.New("plano ou compromisso inválido")
	}

	// Atomically claim this invite
	err = app.RunInTransaction(func(txApp core.App) error {
		fresh, err := txApp.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
		if err != nil {
			return err
		}
		if fresh.GetString("invite_status") != domain.InviteStatusInvited {
			return errAlreadyClaimed
		}
		fresh.Set("invite_status", domain.InviteStatusAccepted)
		fresh.Set("terms_accepted_at", time.Now().UTC().Format(time.RFC3339))
		fresh.Set("terms_accepted_text", terms.Snapshot(tier, commitment, price))
		return txApp.Save(fresh)
	})
	if errors.Is(err, errAlreadyClaimed) {
		business, _ = app.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
		if business != nil && business.GetString("authorization_id") != "" {
			return business.GetString("qr_payload"), nil
		}
		return "", errors.New("convite está sendo processado")
	}
	if err != nil {
		return "", err
	}

	// Reuse existing Asaas customer on retry
	customerID := business.GetString("customer_id")
	if customerID == "" {
		clientName := business.GetString("client_name")
		clientEmail := business.GetString("client_email")
		customer, err := asaas.CreateCustomer(ctx, clientName, clientEmail, cpfCnpj)
		if err != nil {
			app.Logger().Error("asaas create customer", "error", err)
			revertToInvited(app, token)
			return "", err
		}
		customerID = customer.ID
	}

	dueDate := time.Now().Format("2006-01-02")
	auth, err := asaas.CreateAuthorization(ctx, asaasclient.CreateAuthorizationReq{
		CustomerID:  customerID,
		Description: "Rekan - " + string(tier),
		Frequency:   pricing.AsaasFrequency[commitment],
		ContractID:  business.Id,
		StartDate:   dueDate,
		ImmediateQrCode: asaasclient.ImmediateQrCodeReq{
			Value:             price,
			OriginalValue:     price,
			DueDate:           dueDate,
			ExpirationSeconds: 86400,
		},
	})
	if err != nil {
		app.Logger().Error("asaas create authorization", "error", err)
		revertToInvited(app, token)
		return "", err
	}

	qrPayload := auth.Payload

	business, err = app.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
	if err != nil {
		return "", err
	}
	business.Set("authorization_id", auth.ID)
	business.Set("customer_id", customerID)
	business.Set("qr_payload", qrPayload)
	if err := app.Save(business); err != nil {
		app.Logger().Error("save authorization details", "error", err)
		return "", err
	}

	return qrPayload, nil
}

func CancelAuthorization(ctx context.Context, app core.App, asaas *asaasclient.Client, businessID string) error {
	business, err := app.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return fmt.Errorf("%w: negócio não encontrado", ErrNotFound)
	}
	authID := business.GetString("authorization_id")
	if authID == "" || business.GetString("invite_status") != domain.InviteStatusActive {
		return errors.New("nenhuma assinatura ativa")
	}

	if err := asaas.CancelAuthorization(ctx, authID); err != nil {
		app.Logger().Error("asaas cancel authorization", "error", err)
		return err
	}

	business.Set("invite_status", domain.InviteStatusCancelled)
	return app.Save(business)
}

// revertToInvited is a best-effort rollback when Asaas calls fail after
// the status was atomically set to "accepted". Allows the user to retry.
func revertToInvited(app core.App, token string) {
	biz, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
	if err != nil {
		return
	}
	biz.Set("invite_status", domain.InviteStatusInvited)
	_ = app.Save(biz)
}
