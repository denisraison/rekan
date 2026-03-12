package whatsapp

import "go.mau.fi/whatsmeow/types/events"

func handleGroupMessage(deps HandlerDeps, evt *events.Message) {
	if deps.HandleGroupMsg == nil {
		deps.Logger.Debug("whatsapp: group message ignored (agent not configured)")
		return
	}

	if deps.AgentGroupJID != "" && evt.Info.Chat.User != deps.AgentGroupJID {
		return
	}

	deps.HandleGroupMsg(evt)
}
