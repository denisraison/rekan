package service

import (
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/pricing"
	"github.com/pocketbase/pocketbase/core"
)

type InviteInfo struct {
	BusinessName     string
	ClientName       string
	InviteStatus     string
	Tier             string
	Commitment       string
	Price            float64
	CommitmentMonths int
	QRPayload        string
	SentAt           time.Time
}

func GetInviteInfo(app core.App, token string) (*InviteInfo, error) {
	business, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
	if err != nil {
		return nil, err
	}

	tier := pricing.Tier(business.GetString("tier"))
	commitment := pricing.Commitment(business.GetString("commitment"))
	price, _ := pricing.Price(tier, commitment)
	months := pricing.Months[commitment]

	info := &InviteInfo{
		BusinessName:     business.GetString("name"),
		ClientName:       business.GetString("client_name"),
		InviteStatus:     business.GetString("invite_status"),
		Tier:             string(tier),
		Commitment:       string(commitment),
		Price:            price,
		CommitmentMonths: months,
		SentAt:           business.GetDateTime("invite_sent_at").Time(),
	}

	if business.GetString("invite_status") == domain.InviteStatusAccepted {
		info.QRPayload = business.GetString("qr_payload")
	}

	return info, nil
}
