package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// HandlerDeps holds dependencies for the message event handler.
type HandlerDeps struct {
	Client     *Client
	App        core.App
	Logger     *slog.Logger
	Transcribe *transcribe.Client // nil if GEMINI_API_KEY not set
}

// RegisterMessageHandler wires incoming WhatsApp messages to PocketBase storage.
func RegisterMessageHandler(deps HandlerDeps) {
	deps.Client.AddEventHandler(func(evt any) {
		switch v := evt.(type) {
		case *events.Message:
			handleMessage(deps, v)
		}
	})
}

func handleMessage(deps HandlerDeps, evt *events.Message) {
	if evt.Info.IsGroup {
		return
	}

	msg := evt.Message
	if msg == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// For messages sent from self, the conversation partner is in Chat.
	// LID JIDs (server == "lid") must be resolved to phone numbers via the LID map.
	direction := domain.DirectionIncoming
	var phone, pushName string
	if evt.Info.IsFromMe {
		jid := deps.Client.ResolveLID(ctx, evt.Info.Chat)
		if jid.IsEmpty() {
			deps.Logger.Warn("whatsapp: skipping unresolvable LID outgoing event", "chat", evt.Info.Chat)
			return
		}
		direction = domain.DirectionOutgoing
		phone = jid.User
	} else {
		jid := deps.Client.ResolveLID(ctx, evt.Info.Sender)
		if jid.IsEmpty() {
			deps.Logger.Warn("whatsapp: skipping unresolvable LID incoming event", "sender", evt.Info.Sender)
			return
		}
		phone = jid.User
		pushName = evt.Info.PushName
	}

	waMessageID := string(evt.Info.ID)
	waTimestamp := evt.Info.Timestamp

	var msgType, content, extraCaption string
	var mediaFile *filesystem.File

	switch {
	case msg.GetConversation() != "":
		msgType = domain.MsgTypeText
		content = msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil:
		msgType = domain.MsgTypeText
		content = msg.GetExtendedTextMessage().GetText()
	case msg.GetAudioMessage() != nil:
		msgType = domain.MsgTypeAudio
		content = transcribeAudio(ctx, deps, evt)
	case msg.GetImageMessage() != nil:
		msgType = domain.MsgTypeImage
		var caption string
		content, caption, mediaFile = processImage(ctx, deps, evt)
		if caption != "" {
			extraCaption = caption
		}
	case msg.GetVideoMessage() != nil:
		msgType = domain.MsgTypeVideo
		var caption string
		content, caption, mediaFile = processVideo(ctx, deps, evt)
		if caption != "" {
			extraCaption = caption
		}
	default:
		return
	}

	// Deduplicate
	existing, _ := deps.App.FindFirstRecordByFilter(domain.CollMessages, "wa_message_id = {:id}", map[string]any{"id": waMessageID})
	if existing != nil {
		return
	}

	// Find or create a business for this phone number.
	businessID := findOrCreateBusiness(deps, phone, pushName)

	// Refresh profile picture for incoming messages (we know the sender JID).
	if direction == domain.DirectionIncoming && deps.Client != nil {
		senderJID := types.JID{User: phone, Server: "s.whatsapp.net"}
		go refreshProfilePicture(deps, businessID, senderJID)
	}

	collection, err := deps.App.FindCachedCollectionByNameOrId(domain.CollMessages)
	if err != nil {
		deps.Logger.Error("whatsapp: messages collection not found", "error", err)
		return
	}

	record := core.NewRecord(collection)
	if businessID != "" {
		record.Set("business", businessID)
	}
	record.Set("phone", phone)
	record.Set("type", msgType)
	record.Set("content", content)
	record.Set("direction", direction)
	record.Set("wa_timestamp", waTimestamp.UTC().Format(time.RFC3339))
	record.Set("wa_message_id", waMessageID)

	if mediaFile != nil {
		record.Set("media", mediaFile)
	}

	if err := deps.App.Save(record); err != nil {
		deps.Logger.Error("whatsapp: failed to save message", "error", err)
	}

	// If the media message carried a caption, save it as a separate text message
	// so the operator sees both the media and the client's own words.
	if extraCaption != "" {
		captionRecord := core.NewRecord(collection)
		if businessID != "" {
			captionRecord.Set("business", businessID)
		}
		captionRecord.Set("phone", phone)
		captionRecord.Set("type", domain.MsgTypeText)
		captionRecord.Set("content", extraCaption)
		captionRecord.Set("direction", direction)
		captionRecord.Set("wa_timestamp", waTimestamp.UTC().Format(time.RFC3339))
		captionRecord.Set("wa_message_id", waMessageID+"_caption")
		if err := deps.App.Save(captionRecord); err != nil {
			deps.Logger.Error("whatsapp: failed to save caption message", "error", err)
		}
	}
}

// findOrCreateBusiness returns the business ID for the given phone number,
// creating a placeholder business if none exists yet. pushName is the sender's
// WhatsApp display name (empty for outgoing messages).
func findOrCreateBusiness(deps HandlerDeps, phone, pushName string) string {
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
		return business.Id
	}

	collection, err := deps.App.FindCachedCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		deps.Logger.Error("whatsapp: businesses collection not found", "error", err)
		return ""
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
		return ""
	}

	deps.Logger.Info("whatsapp: created placeholder business", "phone", phone, "name", name)
	return record.Id
}

func transcribeAudio(ctx context.Context, deps HandlerDeps, evt *events.Message) string {
	if deps.Transcribe == nil {
		deps.Logger.Warn("whatsapp: audio received but no transcription client configured")
		return ""
	}

	audio := evt.Message.GetAudioMessage()
	if audio == nil {
		return ""
	}

	data, err := deps.Client.Download(ctx, audio)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to download audio", "error", err)
		return ""
	}

	text, err := deps.Transcribe.Transcribe(ctx, data, "audio/ogg")
	if err != nil {
		deps.Logger.Error("whatsapp: transcription failed", "error", err)
		return ""
	}

	return text
}

func processVideo(ctx context.Context, deps HandlerDeps, evt *events.Message) (description, caption string, file *filesystem.File) {
	vid := evt.Message.GetVideoMessage()
	if vid == nil {
		return "", "", nil
	}

	data, err := deps.Client.Download(ctx, vid)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to download video", "error", err)
		return "", "", nil
	}

	mimeType := vid.GetMimetype()
	ext := ".mp4"
	if mimeType == "video/3gpp" {
		ext = ".3gp"
	}

	filename := string(evt.Info.ID) + ext
	f, err := filesystem.NewFileFromBytes(data, filename)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to create file from bytes", "error", err)
		return "", "", nil
	}

	caption = vid.GetCaption()
	description = caption // fallback if Gemini is unavailable

	if deps.Transcribe != nil {
		desc, err := deps.Transcribe.DescribeVideo(ctx, data, mimeType, caption)
		if err != nil {
			deps.Logger.Error("whatsapp: failed to describe video", "error", err)
		} else {
			description = desc
		}
	}

	return description, caption, f
}

func processImage(ctx context.Context, deps HandlerDeps, evt *events.Message) (description, caption string, file *filesystem.File) {
	img := evt.Message.GetImageMessage()
	if img == nil {
		return "", "", nil
	}

	data, err := deps.Client.Download(ctx, img)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to download image", "error", err)
		return "", "", nil
	}

	mimeType := img.GetMimetype()
	ext := ".jpg"
	switch mimeType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}

	filename := string(evt.Info.ID) + ext
	f, err := filesystem.NewFileFromBytes(data, filename)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to create file from bytes", "error", err)
		return "", "", nil
	}

	caption = img.GetCaption()
	description = caption // fallback if Gemini is unavailable

	if deps.Transcribe != nil {
		desc, err := deps.Transcribe.DescribeImage(ctx, data, mimeType, caption)
		if err != nil {
			deps.Logger.Error("whatsapp: failed to describe image", "error", err)
		} else {
			description = desc
		}
	}

	return description, caption, f
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
