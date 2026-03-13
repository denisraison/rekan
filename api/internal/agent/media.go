package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/denisraison/rekan/api/internal/transcribe"
)

// MediaResult holds the preprocessed content from a media message.
type MediaResult struct {
	Text      string // text to send to BAML (includes media description)
	MediaType string // "image", "audio", "video", "sticker", "contact", "document"
}

// ExtractMedia processes non-text content from a WhatsApp group message.
// Returns the text representation and media type, or empty if unsupported.
func ExtractMedia(ctx context.Context, wa WAClient, tc *transcribe.Client, evt *events.Message) MediaResult {
	msg := evt.Message

	// Sticker: thumbs up = "sim"
	if sticker := msg.GetStickerMessage(); sticker != nil {
		return MediaResult{Text: "", MediaType: "sticker"}
	}

	// Audio/voice note
	if audio := msg.GetAudioMessage(); audio != nil {
		return processAudio(ctx, wa, tc, audio)
	}

	// Image
	if img := msg.GetImageMessage(); img != nil {
		return processImageForAgent(ctx, wa, tc, img)
	}

	// Video: pass metadata as text
	if vid := msg.GetVideoMessage(); vid != nil {
		caption := vid.GetCaption()
		if caption != "" {
			return MediaResult{Text: fmt.Sprintf("[Vídeo com legenda: %s]", caption), MediaType: "video"}
		}
		return MediaResult{Text: "[Vídeo recebido]", MediaType: "video"}
	}

	// Contact card (vCard)
	if contact := msg.GetContactMessage(); contact != nil {
		return processContact(contact.GetDisplayName(), contact.GetVcard())
	}

	return MediaResult{}
}

func processAudio(ctx context.Context, wa WAClient, tc *transcribe.Client, audio whatsmeow.DownloadableMessage) MediaResult {
	if tc == nil {
		return MediaResult{Text: "Não consegui entender o áudio. Pode mandar por texto?", MediaType: "audio"}
	}

	data, err := wa.Download(ctx, audio)
	if err != nil {
		return MediaResult{Text: "Não consegui entender o áudio. Pode mandar por texto?", MediaType: "audio"}
	}

	text, err := tc.Transcribe(ctx, data, "audio/ogg")
	if err != nil || strings.TrimSpace(text) == "" {
		return MediaResult{Text: "Não consegui entender o áudio. Pode mandar por texto?", MediaType: "audio"}
	}

	return MediaResult{Text: text, MediaType: "audio"}
}

func processImageForAgent(ctx context.Context, wa WAClient, tc *transcribe.Client, img *waE2E.ImageMessage) MediaResult {
	caption := img.GetCaption()

	if tc == nil {
		if caption != "" {
			return MediaResult{Text: fmt.Sprintf("[Imagem com legenda: %s]", caption), MediaType: "image"}
		}
		return MediaResult{Text: "[Imagem recebida]", MediaType: "image"}
	}

	mimeType := img.GetMimetype()
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	data, err := wa.Download(ctx, img)
	if err != nil {
		if caption != "" {
			return MediaResult{Text: fmt.Sprintf("[Imagem com legenda: %s]", caption), MediaType: "image"}
		}
		return MediaResult{Text: "[Imagem recebida]", MediaType: "image"}
	}

	desc, err := tc.DescribeImage(ctx, data, mimeType, caption)
	if err != nil || strings.TrimSpace(desc) == "" {
		if caption != "" {
			return MediaResult{Text: fmt.Sprintf("[Imagem com legenda: %s]", caption), MediaType: "image"}
		}
		return MediaResult{Text: "[Imagem recebida]", MediaType: "image"}
	}

	text := fmt.Sprintf("[Imagem: %s]", desc)
	if caption != "" {
		text += " " + caption
	}
	return MediaResult{Text: text, MediaType: "image"}
}

func processContact(displayName, vcard string) MediaResult {
	if vcard == "" && displayName == "" {
		return MediaResult{}
	}

	var parts []string
	if displayName != "" {
		parts = append(parts, fmt.Sprintf("Nome: %s", displayName))
	}

	// Extract phone from vCard TEL field
	if phone := extractVCardPhone(vcard); phone != "" {
		parts = append(parts, fmt.Sprintf("Tel: %s", phone))
	}

	return MediaResult{
		Text:      fmt.Sprintf("[Contato: %s]", strings.Join(parts, ", ")),
		MediaType: "contact",
	}
}

var vCardPhoneRe = regexp.MustCompile(`TEL[^:]*:([+\d\s()-]+)`)

func extractVCardPhone(vcard string) string {
	m := vCardPhoneRe.FindStringSubmatch(vcard)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}



