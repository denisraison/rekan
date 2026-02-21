package whatsapp

import (
	"context"
	"log"
	"time"

	"go.mau.fi/whatsmeow/types/events"

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
	if evt.Info.IsFromMe {
		return
	}
	if evt.Info.IsGroup {
		return
	}

	msg := evt.Message
	if msg == nil {
		return
	}

	phone := evt.Info.Sender.User // phone number without @s.whatsapp.net
	waMessageID := string(evt.Info.ID)
	waTimestamp := evt.Info.Timestamp

	var msgType, content string
	var mediaFile *filesystem.File

	switch {
	case msg.GetConversation() != "":
		msgType = "text"
		content = msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil:
		msgType = "text"
		content = msg.GetExtendedTextMessage().GetText()
	case msg.GetAudioMessage() != nil:
		msgType = "audio"
		content = transcribeAudio(deps, evt)
	case msg.GetImageMessage() != nil:
		msgType = "image"
		if msg.GetImageMessage().GetCaption() != "" {
			content = msg.GetImageMessage().GetCaption()
		}
		mediaFile = downloadImage(deps, evt)
	default:
		return
	}

	// Deduplicate
	existing, _ := deps.App.FindFirstRecordByFilter("messages", "wa_message_id = {:id}", map[string]any{"id": waMessageID})
	if existing != nil {
		return
	}

	// Match phone to business
	businessID := ""
	business, _ := deps.App.FindFirstRecordByFilter("businesses", "phone = {:phone}", map[string]any{"phone": phone})
	if business != nil {
		businessID = business.Id
	}

	collection, err := deps.App.FindCollectionByNameOrId("messages")
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
	record.Set("direction", "incoming")
	record.Set("wa_timestamp", waTimestamp.UTC().Format(time.RFC3339))
	record.Set("wa_message_id", waMessageID)

	if mediaFile != nil {
		record.Set("media", mediaFile)
	}

	if err := deps.App.Save(record); err != nil {
		log.Printf("whatsapp: failed to save message: %v", err)
	}
}

func transcribeAudio(deps HandlerDeps, evt *events.Message) string {
	if deps.Transcribe == nil {
		log.Printf("whatsapp: audio received but no transcription client configured")
		return ""
	}

	audio := evt.Message.GetAudioMessage()
	if audio == nil {
		return ""
	}

	data, err := deps.Client.Download(context.Background(), audio)
	if err != nil {
		log.Printf("whatsapp: failed to download audio: %v", err)
		return ""
	}

	text, err := deps.Transcribe.Transcribe(data)
	if err != nil {
		log.Printf("whatsapp: transcription failed: %v", err)
		return ""
	}

	return text
}

func downloadImage(deps HandlerDeps, evt *events.Message) *filesystem.File {
	img := evt.Message.GetImageMessage()
	if img == nil {
		return nil
	}

	data, err := deps.Client.Download(context.Background(), img)
	if err != nil {
		log.Printf("whatsapp: failed to download image: %v", err)
		return nil
	}

	mime := img.GetMimetype()
	ext := ".jpg"
	if mime == "image/png" {
		ext = ".png"
	} else if mime == "image/webp" {
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
