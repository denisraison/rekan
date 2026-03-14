package service

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/pricing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NormalizeForMatch strips accents, lowercases, and trims for fuzzy comparison.
func NormalizeForMatch(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(s))
	}
	return strings.ToLower(strings.TrimSpace(result))
}

// FindBusinessByName returns business records whose names fuzzy-match the query.
func FindBusinessByName(businesses []*core.Record, query string) []*core.Record {
	normalized := NormalizeForMatch(query)
	if normalized == "" {
		return nil
	}

	var matches []*core.Record
	for _, biz := range businesses {
		name := NormalizeForMatch(biz.GetString("name"))
		if name == normalized || strings.Contains(name, normalized) || strings.Contains(normalized, name) {
			matches = append(matches, biz)
		}
	}
	return matches
}

// FindDuplicate checks if a business with the same normalized name already exists.
func FindDuplicate(businesses []*core.Record, name string) *core.Record {
	normalized := NormalizeForMatch(name)
	for _, biz := range businesses {
		if NormalizeForMatch(biz.GetString("name")) == normalized {
			return biz
		}
	}
	return nil
}

// ListActiveBusinesses returns all active and draft businesses ordered by name.
func ListActiveBusinesses(app core.App) []*core.Record {
	var businesses []*core.Record
	if err := app.RecordQuery(domain.CollBusinesses).
		AndWhere(dbx.NewExp("invite_status IN ('active', 'draft')")).
		OrderBy("name ASC").
		All(&businesses); err != nil {
		return nil
	}
	return businesses
}

// CreateBusinessParams holds fields for creating a new business.
type CreateBusinessParams struct {
	Name           string
	Type           string
	City           string
	Phone          string
	TargetAudience *string
	BrandVibe      *string
	Quirks         *string
}

// CreateBusiness creates a new business record in draft status.
func CreateBusiness(app core.App, p CreateBusinessParams) (*core.Record, error) {
	col, err := app.FindCachedCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		return nil, fmt.Errorf("businesses collection: %w", err)
	}
	record := core.NewRecord(col)
	record.Set("name", p.Name)
	record.Set("type", p.Type)
	record.Set("city", p.City)
	record.Set("phone", p.Phone)
	if p.TargetAudience != nil {
		record.Set("target_audience", *p.TargetAudience)
	}
	if p.BrandVibe != nil {
		record.Set("brand_vibe", *p.BrandVibe)
	}
	if p.Quirks != nil {
		record.Set("quirks", *p.Quirks)
	}
	record.Set("invite_status", domain.InviteStatusDraft)

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("creating business: %w", err)
	}
	return record, nil
}

// UpdateBusinessParams holds optional fields for updating a business.
type UpdateBusinessParams struct {
	NewName        *string
	Type           *string
	City           *string
	Phone          *string
	TargetAudience *string
	BrandVibe      *string
	Quirks         *string
}

// UpdateBusiness applies the given fields to the record and saves.
// Returns the list of field keys that were updated.
func UpdateBusiness(app core.App, record *core.Record, p UpdateBusinessParams) ([]string, error) {
	var updated []string
	if p.NewName != nil {
		record.Set("name", *p.NewName)
		updated = append(updated, "name")
	}
	if p.Type != nil {
		record.Set("type", *p.Type)
		updated = append(updated, "type")
	}
	if p.City != nil {
		record.Set("city", *p.City)
		updated = append(updated, "city")
	}
	if p.Phone != nil {
		record.Set("phone", *p.Phone)
		updated = append(updated, "phone")
	}
	if p.TargetAudience != nil {
		record.Set("target_audience", *p.TargetAudience)
		updated = append(updated, "target_audience")
	}
	if p.BrandVibe != nil {
		record.Set("brand_vibe", *p.BrandVibe)
		updated = append(updated, "brand_vibe")
	}
	if p.Quirks != nil {
		record.Set("quirks", *p.Quirks)
		updated = append(updated, "quirks")
	}

	if len(updated) == 0 {
		return nil, nil
	}

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("updating business: %w", err)
	}
	return updated, nil
}

// PauseBusiness sets a business to cancelled status.
func PauseBusiness(app core.App, record *core.Record) error {
	record.Set("invite_status", domain.InviteStatusCancelled)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("pausing business: %w", err)
	}
	return nil
}

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
