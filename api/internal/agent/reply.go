package agent

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// WAClient is the subset of whatsmeow.Client used by the agent.
type WAClient interface {
	SendMessage(ctx context.Context, to types.JID, msg *waE2E.Message) (whatsmeow.SendResponse, error)
	ResolveLID(ctx context.Context, jid types.JID) types.JID
	Download(ctx context.Context, msg whatsmeow.DownloadableMessage) (data []byte, err error)
	Upload(ctx context.Context, data []byte, mediaType whatsmeow.MediaType) (whatsmeow.UploadResponse, error)
}

// SendImage sends an image message to a WhatsApp chat.
func SendImage(ctx context.Context, wa WAClient, to types.JID, imageData []byte, caption string) error {
	resp, err := wa.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return err
	}
	_, err = wa.SendMessage(ctx, to, &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           &resp.URL,
			DirectPath:    &resp.DirectPath,
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Mimetype:      proto.String("image/jpeg"),
			Caption:       &caption,
		},
	})
	return err
}

// SendReply sends a text message to the WhatsApp group.
func SendReply(ctx context.Context, waClient WAClient, groupJID types.JID, text string) error {
	_, err := waClient.SendMessage(ctx, groupJID, &waE2E.Message{
		Conversation: &text,
	})
	return err
}

// ReactThumbsUp reacts to a message with a thumbs up emoji.
func ReactThumbsUp(ctx context.Context, waClient WAClient, chat types.JID, messageID string, senderJID types.JID) error {
	_, err := waClient.SendMessage(ctx, chat, &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID:   proto.String(chat.String()),
				FromMe:      proto.Bool(false),
				ID:          &messageID,
				Participant: proto.String(senderJID.String()),
			},
			Text: proto.String("\U0001F44D"),
		},
	})
	return err
}
