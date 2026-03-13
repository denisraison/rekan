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

// LoadRecentAndPrune loads the last n messages (oldest first) and deletes any overflow in a single pass.
func LoadRecentAndPrune(app core.App, n int) ([]ConversationMessage, error) {
	records, err := queryRecentConversations(app, 0)
	if err != nil {
		return nil, err
	}

	// Delete overflow (records are newest-first from SQL)
	if n > 0 && len(records) > n {
		for _, r := range records[n:] {
			_ = app.Delete(r)
		}
		records = records[:n]
	}

	return toMessages(records), nil
}

// LoadRecent loads the last n messages from the conversation buffer, oldest first.
func LoadRecent(app core.App, n int) ([]ConversationMessage, error) {
	records, err := queryRecentConversations(app, int64(n))
	if err != nil {
		return nil, err
	}
	return toMessages(records), nil
}

// queryRecentConversations returns conversations ordered newest-first.
// Pass limit=0 for all records.
func queryRecentConversations(app core.App, limit int64) ([]*core.Record, error) {
	q := app.RecordQuery(domain.CollAgentConversations).
		OrderBy("timestamp DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var records []*core.Record
	if err := q.All(&records); err != nil {
		return nil, fmt.Errorf("load conversations: %w", err)
	}
	return records, nil
}

// toMessages converts newest-first records to oldest-first ConversationMessages.
func toMessages(records []*core.Record) []ConversationMessage {
	msgs := make([]ConversationMessage, len(records))
	for i, r := range records {
		msgs[len(records)-1-i] = ConversationMessage{
			OperatorName: r.GetString("operator_name"),
			Role:         r.GetString("role"),
			Content:      r.GetString("content"),
		}
	}
	return msgs
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
