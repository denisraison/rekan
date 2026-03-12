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
}

// SendReply sends a text message to the WhatsApp group.
func SendReply(ctx context.Context, waClient WAClient, groupJID types.JID, text string) error {
	_, err := waClient.SendMessage(ctx, groupJID, &waE2E.Message{
		Conversation: &text,
	})
	return err
}

// ReactThumbsUp reacts to a message with a thumbs up emoji.
func ReactThumbsUp(ctx context.Context, waClient WAClient, chat types.JID, messageID string, senderJID types.JID) {
	waClient.SendMessage(ctx, chat, &waE2E.Message{
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
}
