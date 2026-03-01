package whatsapp

import (
	"context"
	"log"
	"time"

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
	Transcribe *transcribe.Client // nil if OPENAI_API_KEY not set
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
			log.Printf("whatsapp: skipping unresolvable LID outgoing event (chat=%s)", evt.Info.Chat)
			return
		}
		direction = domain.DirectionOutgoing
		phone = jid.User
	} else {
		jid := deps.Client.ResolveLID(ctx, evt.Info.Sender)
		if jid.IsEmpty() {
			log.Printf("whatsapp: skipping unresolvable LID incoming event (sender=%s)", evt.Info.Sender)
			return
		}
		phone = jid.User
		pushName = evt.Info.PushName
	}

	waMessageID := string(evt.Info.ID)
	waTimestamp := evt.Info.Timestamp

	var msgType, content string
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
		if msg.GetImageMessage().GetCaption() != "" {
			content = msg.GetImageMessage().GetCaption()
		}
		mediaFile = downloadImage(ctx, deps, evt)
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

	collection, err := deps.App.FindCollectionByNameOrId(domain.CollMessages)
	if err != nil {
		log.Printf("whatsapp: messages collection not found: %v", err)
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
		log.Printf("whatsapp: failed to save message: %v", err)
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
				log.Printf("whatsapp: failed to update placeholder name for %s: %v", phone, err)
			}
		}
		return business.Id
	}

	collection, err := deps.App.FindCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		log.Printf("whatsapp: businesses collection not found: %v", err)
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
		log.Printf("whatsapp: failed to create placeholder business for %s: %v", phone, err)
		return ""
	}

	log.Printf("whatsapp: created placeholder business for %s (%s)", phone, name)
	return record.Id
}

func transcribeAudio(ctx context.Context, deps HandlerDeps, evt *events.Message) string {
	if deps.Transcribe == nil {
		log.Printf("whatsapp: audio received but no transcription client configured")
		return ""
	}

	audio := evt.Message.GetAudioMessage()
	if audio == nil {
		return ""
	}

	data, err := deps.Client.Download(ctx, audio)
	if err != nil {
		log.Printf("whatsapp: failed to download audio: %v", err)
		return ""
	}

	text, err := deps.Transcribe.Transcribe(ctx, data)
	if err != nil {
		log.Printf("whatsapp: transcription failed: %v", err)
		return ""
	}

	return text
}

func downloadImage(ctx context.Context, deps HandlerDeps, evt *events.Message) *filesystem.File {
	img := evt.Message.GetImageMessage()
	if img == nil {
		return nil
	}

	data, err := deps.Client.Download(ctx, img)
	if err != nil {
		log.Printf("whatsapp: failed to download image: %v", err)
		return nil
	}

	ext := ".jpg"
	switch img.GetMimetype() {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}

	filename := string(evt.Info.ID) + ext
	f, err := filesystem.NewFileFromBytes(data, filename)
	if err != nil {
		log.Printf("whatsapp: failed to create file from bytes: %v", err)
		return nil
	}

	return f
}
