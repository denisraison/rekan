package whatsapp

import (
	"context"
	"errors"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// findOrCreateBusiness returns the business ID, invite status, and type for the given
// phone number, creating a placeholder business if none exists yet. pushName is the
// sender's WhatsApp display name (empty for outgoing messages).
func findOrCreateBusiness(deps HandlerDeps, phone, pushName string) (id, inviteStatus, businessType string) {
	business, _ := deps.App.FindFirstRecordByFilter(domain.CollBusinesses, "phone = {:phone}", map[string]any{"phone": phone})
	if business != nil {
		// Update name if the placeholder still uses the raw phone and we now have a real name.
		if pushName != "" && business.GetString("name") == "+"+phone {
			business.Set("name", pushName)
			business.Set("client_name", pushName)
			if err := deps.App.Save(business); err != nil {
				deps.Logger.Error("whatsapp: failed to update placeholder name", "phone", phone, "error", err)
			}
		}
		return business.Id, business.GetString("invite_status"), business.GetString("type")
	}

	collection, err := deps.App.FindCachedCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		deps.Logger.Error("whatsapp: businesses collection not found", "error", err)
		return "", "", ""
	}

	name := "+" + phone
	if pushName != "" {
		name = pushName
	}

	record := core.NewRecord(collection)
	record.Set("phone", phone)
	record.Set("name", name)
	record.Set("client_name", pushName)
	record.Set("type", "Desconhecido")
	record.Set("city", "-")
	record.Set("state", "-")

	if err := deps.App.Save(record); err != nil {
		deps.Logger.Error("whatsapp: failed to create placeholder business", "phone", phone, "error", err)
		return "", "", ""
	}

	deps.Logger.Info("whatsapp: created placeholder business", "phone", phone, "name", name)
	return record.Id, "", "Desconhecido"
}

// extractAndSaveSignal checks whether the message content contains profile-relevant
// information and saves a profile_suggestions row when it does. Runs in a goroutine.
func extractAndSaveSignal(deps HandlerDeps, businessID, businessType, content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	signal, err := deps.ExtractSignal(ctx, content, businessType)
	if err != nil {
		deps.Logger.Warn("whatsapp: profile signal extraction failed", "error", err)
		return
	}
	if signal == nil {
		return
	}

	collection, err := deps.App.FindCachedCollectionByNameOrId(domain.CollProfileSuggestions)
	if err != nil {
		deps.Logger.Error("whatsapp: profile_suggestions collection not found", "error", err)
		return
	}

	record := core.NewRecord(collection)
	record.Set("business", businessID)
	record.Set("field", signal.Field)
	record.Set("suggestion", signal.Value)

	if err := deps.App.Save(record); err != nil {
		deps.Logger.Error("whatsapp: failed to save profile suggestion", "error", err)
	}
}

// refreshProfilePicture fetches the WhatsApp profile picture for jid and stores
// it on the business record. It skips the fetch if the picture was updated less
// than 7 days ago and hasn't changed on WhatsApp. Runs in a goroutine.
func refreshProfilePicture(deps HandlerDeps, businessID string, jid types.JID) {
	if businessID == "" || deps.Client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	business, err := deps.App.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return
	}

	// Skip if refreshed within the last 7 days.
	updatedAt := business.GetDateTime("profile_picture_updated")
	if !updatedAt.IsZero() && time.Since(updatedAt.Time()) < 7*24*time.Hour {
		return
	}

	existingID := business.GetString("profile_picture_id")
	info, err := deps.Client.GetProfilePicture(ctx, jid, existingID)
	if err != nil {
		// Not-set and unauthorized are expected; anything else is worth logging.
		if !errors.Is(err, whatsmeow.ErrProfilePictureNotSet) && !errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
			deps.Logger.Warn("whatsapp: could not fetch profile picture", "phone", jid.User, "error", err)
		}
		// Still update the timestamp so we don't hammer the API on every message.
		business.Set("profile_picture_updated", time.Now().UTC().Format(time.RFC3339))
		_ = deps.App.Save(business)
		return
	}
	if info == nil {
		// Picture unchanged since last fetch.
		return
	}

	data, err := deps.Client.DownloadURL(ctx, info.URL)
	if err != nil {
		deps.Logger.Warn("whatsapp: could not download profile picture", "phone", jid.User, "error", err)
		return
	}

	f, err := filesystem.NewFileFromBytes(data, jid.User+"_avatar.jpg")
	if err != nil {
		deps.Logger.Error("whatsapp: failed to create profile picture file", "error", err)
		return
	}

	business.Set("profile_picture", f)
	business.Set("profile_picture_id", info.ID)
	business.Set("profile_picture_updated", time.Now().UTC().Format(time.RFC3339))

	if err := deps.App.Save(business); err != nil {
		deps.Logger.Error("whatsapp: failed to save profile picture", "phone", jid.User, "error", err)
	}
}
