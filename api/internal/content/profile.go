package content

import (
	"context"
	"fmt"

	baml "github.com/denisraison/rekan/api/internal/content/baml_client/baml_client"
)

// PartialService is a service extracted from a voice transcript.
// PriceBRL is nil when the price was not mentioned.
type PartialService struct {
	Name     string
	PriceBRL *float64
}

// PartialBusinessProfile is the result of extracting profile fields from a transcript.
// All fields are nil/empty when not mentioned in the audio.
type PartialBusinessProfile struct {
	Services       []PartialService
	TargetAudience *string
	BrandVibe      *string
	Quirks         []string
}

// ExtractFromAudioFunc is the signature for the combined audio transcription + profile extraction pipeline.
type ExtractFromAudioFunc func(ctx context.Context, audioBytes []byte, mimeType string, businessType string) (PartialBusinessProfile, error)

// ProfileSignal is a single profile-relevant signal extracted from a WhatsApp message.
// Field is one of: "services", "quirks", "target_audience", "brand_vibe".
// For services, Value is "Name|price_brl" (e.g. "Selagem|150.0"). For others, Value is plain text.
type ProfileSignal struct {
	Field string
	Value string
}

// ExtractSignalFunc checks whether a WhatsApp message contains profile-relevant information.
type ExtractSignalFunc func(ctx context.Context, message, businessType string) (*ProfileSignal, error)

// ExtractBusinessProfile calls Gemini to extract structured profile fields from a transcript.
func ExtractBusinessProfile(ctx context.Context, transcript string, businessType string) (PartialBusinessProfile, error) {
	result, err := baml.ExtractBusinessProfile(ctx, transcript, businessType)
	if err != nil {
		return PartialBusinessProfile{}, fmt.Errorf("extract business profile: %w", err)
	}

	profile := PartialBusinessProfile{
		TargetAudience: result.TargetAudience,
		BrandVibe:      result.BrandVibe,
	}

	if result.Services != nil {
		for _, s := range *result.Services {
			profile.Services = append(profile.Services, PartialService{
				Name:     s.Name,
				PriceBRL: s.PriceBRL,
			})
		}
	}

	if result.Quirks != nil {
		profile.Quirks = *result.Quirks
	}

	return profile, nil
}

// ExtractProfileSignal checks if a WhatsApp message contains profile-relevant information.
// Returns nil when the message has no useful profile signal.
func ExtractProfileSignal(ctx context.Context, message, businessType string) (*ProfileSignal, error) {
	result, err := baml.ExtractProfileSignal(ctx, message, businessType)
	if err != nil {
		return nil, fmt.Errorf("extract profile signal: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	return &ProfileSignal{
		Field: result.Field,
		Value: result.Value,
	}, nil
}
