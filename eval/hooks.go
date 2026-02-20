package eval

import "strings"

// ExtractHooks pulls the first sentence of each post's caption.
func ExtractHooks(posts []Post) []string {
	hooks := make([]string, 0, len(posts))
	for _, p := range posts {
		if h := firstSentence(p.Caption); h != "" {
			hooks = append(hooks, h)
		}
	}
	return hooks
}

func firstSentence(text string) string {
	// Take only the first paragraph to avoid matching sentence-enders in hashtags.
	lines := strings.SplitN(text, "\n\n", 2)
	paragraph := strings.TrimSpace(lines[0])
	paragraph = strings.ReplaceAll(paragraph, "\n", " ")

	for i, r := range paragraph {
		if r == '.' || r == '!' || r == '?' {
			return paragraph[:i+1]
		}
	}
	return paragraph
}
