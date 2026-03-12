package whatsapp

import "go.mau.fi/whatsmeow/types/events"

func handleGroupMessage(deps HandlerDeps, evt *events.Message) {
	// PEP-023 implements this. For now, log and return.
	deps.Logger.Debug("whatsapp: group message ignored (agent not configured)")
}
