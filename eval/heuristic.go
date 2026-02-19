package eval

import (
	"regexp"
	"strings"
	"unicode"
)

type Service struct {
	Name     string  `json:"name"`
	PriceBRL float64 `json:"priceBRL"`
}

type BusinessProfile struct {
	BusinessName   string    `json:"businessName"`
	BusinessType   string    `json:"businessType"`
	City           string    `json:"city"`
	Neighbourhood  string    `json:"neighbourhood"`
	Services       []Service `json:"services"`
	TargetAudience string    `json:"targetAudience"`
	BrandVibe      string    `json:"brandVibe"`
	Quirks         []string  `json:"quirks"`
}

type CheckResult struct {
	Name   string
	Pass   bool
	Reason string
}

func RunChecks(content string, profile BusinessProfile) []CheckResult {
	return []CheckResult{
		checkBusinessName(content, profile),
		checkLocation(content, profile),
		checkHashtags(content),
		checkCTA(content),
		checkBrazilianPortuguese(content),
		checkCaptionLength(content),
		checkProductionNote(content),
	}
}

// stripAccents removes combining diacritical marks via NFD decomposition.
func stripAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		// Skip combining marks (accents, tildes, cedillas, etc.)
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Manual NFD for Portuguese chars, avoids pulling in golang.org/x/text.
var nfdReplacer = strings.NewReplacer(
	"á", "a\u0301", "à", "a\u0300", "ã", "a\u0303", "â", "a\u0302",
	"é", "e\u0301", "è", "e\u0300", "ê", "e\u0302",
	"í", "i\u0301", "ì", "i\u0300",
	"ó", "o\u0301", "ò", "o\u0300", "õ", "o\u0303", "ô", "o\u0302",
	"ú", "u\u0301", "ù", "u\u0300",
	"ç", "c\u0327",
	"Á", "A\u0301", "À", "A\u0300", "Ã", "A\u0303", "Â", "A\u0302",
	"É", "E\u0301", "È", "E\u0300", "Ê", "E\u0302",
	"Í", "I\u0301", "Ì", "I\u0300",
	"Ó", "O\u0301", "Ò", "O\u0300", "Õ", "O\u0303", "Ô", "O\u0302",
	"Ú", "U\u0301", "Ù", "U\u0300",
	"Ç", "C\u0327",
)

// normalize decomposes to NFD so accented chars become base + combining mark,
// then stripAccents removes the combining marks.
func normalize(s string) string {
	return stripAccents(nfdReplacer.Replace(s))
}

func normalizedContains(haystack, needle string) bool {
	h := normalize(strings.ToLower(haystack))
	n := normalize(strings.ToLower(needle))
	return strings.Contains(h, n)
}

func checkBusinessName(content string, profile BusinessProfile) CheckResult {
	if normalizedContains(content, profile.BusinessName) {
		return CheckResult{Name: "business_name", Pass: true}
	}
	return CheckResult{
		Name:   "business_name",
		Reason: "business name not found in content",
	}
}

func checkLocation(content string, profile BusinessProfile) CheckResult {
	if normalizedContains(content, profile.City) {
		return CheckResult{Name: "location", Pass: true}
	}
	if profile.Neighbourhood != "" && normalizedContains(content, profile.Neighbourhood) {
		return CheckResult{Name: "location", Pass: true}
	}
	return CheckResult{
		Name:   "location",
		Reason: "neither city nor neighbourhood found in content",
	}
}

var hashtagRe = regexp.MustCompile(`#[\p{L}\p{N}_]+`)

func checkHashtags(content string) CheckResult {
	matches := hashtagRe.FindAllString(content, -1)
	if len(matches) >= 3 {
		return CheckResult{Name: "hashtags", Pass: true}
	}
	return CheckResult{
		Name:   "hashtags",
		Reason: "fewer than 3 hashtags found",
	}
}

func checkCTA(content string) CheckResult {
	lower := strings.ToLower(content)
	patterns := []string{
		"chama no dm",
		"link na bio",
		"link da bio",
		"agende",
		"entre em contato",
		"manda mensagem",
		"manda um",
		"fale conosco",
		"chama no whatsapp",
		"chama no zap",
		"garanta o seu",
		"garanta a sua",
		"aproveite",
		"reserve",
		"venha conhecer",
		"vem conhecer",
		"peça já",
		"faça seu pedido",
		"clique no link",
		"clica no link",
		"comenta aqui",
		"no direct",
		"salve no zap",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return CheckResult{Name: "cta", Pass: true}
		}
	}
	return CheckResult{
		Name:   "cta",
		Reason: "no call-to-action pattern found",
	}
}

func checkBrazilianPortuguese(content string) CheckResult {
	lower := strings.ToLower(content)

	ptBRMarkers := []string{"bora", "gente", "pra", "né", "tá"}
	hasBR := false
	for _, m := range ptBRMarkers {
		if containsWord(lower, m) {
			hasBR = true
			break
		}
	}
	if !hasBR {
		return CheckResult{
			Name:   "brazilian_portuguese",
			Reason: "no pt-BR informal markers found",
		}
	}

	ptPTMarkers := []string{"consigo", "telemóvel", "telemovel", "autocarro"}
	for _, m := range ptPTMarkers {
		if containsWord(lower, m) {
			return CheckResult{
				Name:   "brazilian_portuguese",
				Reason: "Portugal Portuguese marker found: " + m,
			}
		}
	}

	return CheckResult{Name: "brazilian_portuguese", Pass: true}
}

// containsWord checks if word appears as a standalone word in text.
// Uses unicode letter class so accented words like "né" are matched whole.
var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

func containsWord(text, word string) bool {
	nWord := normalize(word)
	for _, w := range wordRe.FindAllString(text, -1) {
		if normalize(w) == nWord {
			return true
		}
	}
	return false
}

const maxCaptionLength = 2200

// splitPosts splits multi-post output on "---" separators.
var postSepRe = regexp.MustCompile(`(?m)^-{3,}\s*$`)

func splitPosts(content string) []string {
	parts := postSepRe.Split(content, -1)
	posts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			posts = append(posts, p)
		}
	}
	return posts
}

func checkCaptionLength(content string) CheckResult {
	posts := splitPosts(content)
	for _, p := range posts {
		if len([]rune(p)) > maxCaptionLength {
			return CheckResult{
				Name:   "caption_length",
				Reason: "a post exceeds 2200 characters",
			}
		}
	}
	return CheckResult{Name: "caption_length", Pass: true}
}

func checkProductionNote(content string) CheckResult {
	lower := strings.ToLower(content)
	keywords := []string{"foto", "vídeo", "video", "imagem", "registre", "poste", "stories", "reels"}
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return CheckResult{Name: "production_note", Pass: true}
		}
	}
	return CheckResult{
		Name:   "production_note",
		Reason: "no photo/video suggestion keywords found",
	}
}
