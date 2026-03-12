package operator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/denisraison/rekan/api/internal/domain"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/pocketbase/pocketbase/core"
)

type storedService struct {
	Name     string  `json:"name"`
	PriceBRL float64 `json:"price_brl"`
}

func BusinessToProfile(record *core.Record) (content.BusinessProfile, error) {
	var b []byte
	if s, ok := record.Get("services").(string); ok {
		b = []byte(s)
	} else {
		var err error
		b, err = json.Marshal(record.Get("services"))
		if err != nil {
			return content.BusinessProfile{}, fmt.Errorf("marshal services: %w", err)
		}
	}
	var stored []storedService
	if err := json.Unmarshal(b, &stored); err != nil {
		return content.BusinessProfile{}, fmt.Errorf("unmarshal services: %w", err)
	}
	services := make([]content.Service, len(stored))
	for i, s := range stored {
		services[i] = content.Service{Name: s.Name, PriceBRL: s.PriceBRL}
	}

	var quirks []string
	for _, line := range strings.Split(record.GetString("quirks"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			quirks = append(quirks, line)
		}
	}

	return content.BusinessProfile{
		BusinessName:   record.GetString("name"),
		BusinessType:   record.GetString("type"),
		City:           record.GetString("city"),
		Neighbourhood:  "", // collection stores state, not neighbourhood; city is sufficient for generation
		Services:       services,
		TargetAudience: record.GetString("target_audience"),
		BrandVibe:      record.GetString("brand_vibe"),
		Quirks:         quirks,
	}, nil
}

func LoadPreviousHooks(app core.App, businessID string) ([]string, error) {
	records, err := app.FindRecordsByFilter(
		domain.CollPosts,
		"business = {:business}",
		"-created",
		0,
		15,
		map[string]any{"business": businessID},
	)
	if err != nil {
		return nil, err
	}
	hooks := make([]string, 0, len(records))
	for _, r := range records {
		if h := r.GetString("hook"); h != "" {
			hooks = append(hooks, h)
		}
	}
	return hooks, nil
}
