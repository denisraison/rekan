package domain

// Collection names
const (
	CollBusinesses = "businesses"
	CollPosts      = "posts"
	CollMessages   = "messages"
)

// InviteStatus values for the invite_status field on businesses.
const (
	InviteStatusInvited       = "invited"
	InviteStatusAccepted      = "accepted"
	InviteStatusActive        = "active"
	InviteStatusCancelled     = "cancelled"
	InviteStatusPaymentFailed = "payment_failed"
)

// Asaas webhook event names.
const (
	EventPixAuthActivated  = "PIX_AUTOMATIC_RECURRING_AUTHORIZATION_ACTIVATED"
	EventPixAuthRefused    = "PIX_AUTOMATIC_RECURRING_AUTHORIZATION_REFUSED"
	EventPixAuthCancelled  = "PIX_AUTOMATIC_RECURRING_AUTHORIZATION_CANCELLED"
	EventPixAuthExpired    = "PIX_AUTOMATIC_RECURRING_AUTHORIZATION_EXPIRED"
	EventPaymentConfirmed  = "PAYMENT_CONFIRMED"
	EventPixPaymentRefused = "PIX_AUTOMATIC_RECURRING_PAYMENT_INSTRUCTION_REFUSED"
	EventPixPaymentCancelled = "PIX_AUTOMATIC_RECURRING_PAYMENT_INSTRUCTION_CANCELLED"
)

// Message direction values.
const (
	DirectionIncoming = "incoming"
	DirectionOutgoing = "outgoing"
)

// Message type values.
const (
	MsgTypeText  = "text"
	MsgTypeAudio = "audio"
	MsgTypeImage = "image"
)

// BillingType values for Asaas charges.
const (
	BillingTypePIX = "PIX"
)

// Post source values.
const (
	PostSourceOperator = "operator"
)
