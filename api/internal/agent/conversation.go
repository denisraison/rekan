package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

// StoreMessage saves a message to the agent_conversations collection.
func StoreMessage(app core.App, operatorName, operatorJID, role, content, mediaType string) error {
	col, err := app.FindCachedCollectionByNameOrId(domain.CollAgentConversations)
	if err != nil {
		return fmt.Errorf("agent_conversations collection: %w", err)
	}
	record := core.NewRecord(col)
	record.Set("operator_name", operatorName)
	record.Set("operator_jid", operatorJID)
	record.Set("role", role)
	record.Set("content", content)
	record.Set("media_type", mediaType)
	record.Set("timestamp", time.Now().UTC().Format(time.RFC3339))
	return app.Save(record)
}

// ConversationMessage represents a single message in the conversation buffer.
type ConversationMessage struct {
	OperatorName string
	Role         string
	Content      string
}

// LoadRecent loads the last n messages from the conversation buffer, oldest first.
func LoadRecent(app core.App, n int) ([]ConversationMessage, error) {
	records, err := app.FindRecordsByFilter(
		domain.CollAgentConversations,
		"1=1",
		"-timestamp",
		0,
		n,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("load conversations: %w", err)
	}
	msgs := make([]ConversationMessage, len(records))
	for i, r := range records {
		// Reverse order: records come newest-first, we want oldest-first
		msgs[len(records)-1-i] = ConversationMessage{
			OperatorName: r.GetString("operator_name"),
			Role:         r.GetString("role"),
			Content:      r.GetString("content"),
		}
	}
	return msgs, nil
}

// FormatConversation renders messages as a text block for the BAML prompt.
func FormatConversation(msgs []ConversationMessage) string {
	if len(msgs) == 0 {
		return "(sem histórico)"
	}
	var b strings.Builder
	for _, m := range msgs {
		if m.Role == "assistant" {
			b.WriteString("[Rekan]: ")
		} else {
			b.WriteString(m.OperatorName + ": ")
		}
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

// Prune deletes messages beyond the keep limit, keeping the most recent ones.
func Prune(app core.App, keep int) error {
	// Fetch only the records past the keep window using offset.
	records, err := app.FindRecordsByFilter(
		domain.CollAgentConversations,
		"1=1",
		"-timestamp",
		keep,
		0,
		nil,
	)
	if err != nil || len(records) == 0 {
		return err
	}
	for _, r := range records {
		if err := app.Delete(r); err != nil {
			return err
		}
	}
	return nil
}
