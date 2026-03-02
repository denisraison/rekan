package operator

import (
	"strings"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/seasonal"
	"github.com/pocketbase/pocketbase/core"
)

// QueueSeasonalMessages runs daily and creates scheduled_messages for seasonal events
// that are 7 days away, for eligible businesses that don't already have one queued.
func QueueSeasonalMessages(app core.App) {
	now := time.Now()
	target := now.AddDate(0, 0, 7)
	year := now.Year()

	businesses, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		app.Logger().Error("seasonal_batch: list businesses", "error", err)
		return
	}

	collection, err := app.FindCollectionByNameOrId(domain.CollScheduledMessages)
	if err != nil {
		app.Logger().Error("seasonal_batch: find scheduled_messages collection", "error", err)
		return
	}

	for _, sd := range seasonal.Dates {
		dateYear := year
		if sd.Year != 0 {
			dateYear = sd.Year
		}
		date := time.Date(dateYear, time.Month(sd.Month), sd.Day, 0, 0, 0, 0, time.Local)
		// For fixed calendar dates (Year == 0), roll to next year if already past.
		// Moveable holidays (Year != 0) are only valid for their specific year; skip if past.
		if date.Before(now) {
			if sd.Year != 0 {
				continue
			}
			date = time.Date(year+1, time.Month(sd.Month), sd.Day, 0, 0, 0, 0, time.Local)
		}

		if date.Year() != target.Year() || date.Month() != target.Month() || date.Day() != target.Day() {
			continue
		}

		for _, biz := range businesses {
			bizType := biz.GetString("type")

			// Check niche eligibility.
			if len(sd.Niches) > 0 {
				eligible := false
				for _, niche := range sd.Niches {
					if niche == bizType {
						eligible = true
						break
					}
				}
				if !eligible {
					continue
				}
			}

			// Check if a non-dismissed scheduled_message already exists for this business+label.
			existing, err := app.FindRecordsByFilter(
				domain.CollScheduledMessages,
				"business = {:biz} && dismissed = false",
				"",
				0,
				0,
				map[string]any{"biz": biz.Id},
			)
			if err == nil {
				alreadyQueued := false
				for _, ex := range existing {
					if strings.Contains(ex.GetString("text"), sd.Label) {
						alreadyQueued = true
						break
					}
				}
				if alreadyQueued {
					continue
				}
			}

			// Pick name for template substitution.
			clientName := biz.GetString("client_name")
			if clientName == "" {
				clientName = biz.GetString("name")
			}
			firstName := strings.SplitN(clientName, " ", 2)[0]

			text := strings.ReplaceAll(sd.Template, "{name}", firstName)

			r := core.NewRecord(collection)
			r.Set("business", biz.Id)
			r.Set("text", text)
			r.Set("scheduled_for", date)
			r.Set("approved", false)
			r.Set("dismissed", false)

			if err := app.Save(r); err != nil {
				app.Logger().Error("seasonal_batch: save scheduled message", "business", biz.Id, "label", sd.Label, "error", err)
			}
		}
	}
}
