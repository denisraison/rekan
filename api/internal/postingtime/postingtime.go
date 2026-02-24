package postingtime

import "strings"

type Window struct {
	Primary   string
	Secondary string
}

var categories = map[string]Window{
	"food":    {Primary: "10h-14h", Secondary: "17h-19h"},
	"beauty":  {Primary: "10h-13h", Secondary: "19h-21h"},
	"fashion": {Primary: "11h-14h", Secondary: "19h-21h"},
	"fitness": {Primary: "6h-8h", Secondary: "17h-19h"},
	"pet":     {Primary: "11h-13h", Secondary: "18h-20h"},
}

var fallback = Window{Primary: "11h-13h", Secondary: "18h-20h"}

// keywords maps substrings found in business types to category keys.
var keywords = map[string]string{
	// food
	"hamburguer":   "food",
	"burger":       "food",
	"pizz":         "food",
	"restaurante":  "food",
	"lanchonete":   "food",
	"confeitaria":  "food",
	"doceria":      "food",
	"padaria":      "food",
	"sorveteria":   "food",
	"acai":         "food",
	"açaí":         "food",
	"cafe":         "food",
	"café":         "food",
	"bar ":         "food",
	"boteco":       "food",
	"churrascaria": "food",
	"marmita":      "food",
	"delivery":     "food",
	"comida":       "food",
	"culinaria":    "food",
	"culinária":    "food",
	"gastronomia":  "food",
	"cozinha":      "food",
	"salgado":      "food",
	"bolo":         "food",
	"brownie":      "food",
	"brigadeiro":   "food",
	// beauty
	"salao":        "beauty",
	"salão":        "beauty",
	"barbearia":    "beauty",
	"manicure":     "beauty",
	"cabeleir":     "beauty",
	"estetica":     "beauty",
	"estética":     "beauty",
	"maquiagem":    "beauty",
	"sobrancelha":  "beauty",
	"depilacao":    "beauty",
	"depilação":    "beauty",
	"nail":         "beauty",
	"unha":         "beauty",
	"lash":         "beauty",
	"cilios":       "beauty",
	"cílios":       "beauty",
	"micropigment": "beauty",
	"tatuag":       "beauty",
	// fashion
	"roupa":    "fashion",
	"moda":     "fashion",
	"boutique": "fashion",
	"brechó":   "fashion",
	"brecho":   "fashion",
	"calçado":  "fashion",
	"calcado":  "fashion",
	"sapato":   "fashion",
	"joia":     "fashion",
	"jóia":     "fashion",
	"bijuteria": "fashion",
	"acessorio": "fashion",
	"acessório": "fashion",
	// fitness
	"personal":  "fitness",
	"academia":  "fitness",
	"pilates":   "fitness",
	"yoga":      "fitness",
	"crossfit":  "fitness",
	"funcional": "fitness",
	"fitness":   "fitness",
	"treino":    "fitness",
	"nutri":     "fitness",
	// pet
	"pet":        "pet",
	"veterinar":  "pet",
	"banho e tosa": "pet",
}

// ForBusinessType returns the best posting time windows for a given
// free-text business type. Falls back to general Brazilian prime time
// if no category matches.
func ForBusinessType(businessType string) Window {
	lower := strings.ToLower(businessType)
	for kw, cat := range keywords {
		if strings.Contains(lower, kw) {
			return categories[cat]
		}
	}
	return fallback
}

// Tip formats a WhatsApp-ready posting time suggestion.
func Tip(businessType string) string {
	w := ForBusinessType(businessType)
	return "*Melhor horário pra postar:* entre " + w.Primary + " ou entre " + w.Secondary
}
