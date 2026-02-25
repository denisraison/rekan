package pricing

type Tier string

const (
	Basico       Tier = "basico"
	Parceiro     Tier = "parceiro"
	Profissional Tier = "profissional"
)

type Commitment string

const (
	Mensal     Commitment = "mensal"
	Trimestral Commitment = "trimestral"
)

type key struct {
	Tier       Tier
	Commitment Commitment
}

// Prices maps (tier, commitment) to the total charge amount for the period.
var Prices = map[key]float64{
	{Basico, Mensal}:     69.90,
	{Basico, Trimestral}: 179.70,

	{Parceiro, Mensal}:     108.90,
	{Parceiro, Trimestral}: 299.70,

	{Profissional, Mensal}:     249.90,
	{Profissional, Trimestral}: 599.70,
}

// Months maps a commitment to the number of months in the billing cycle.
var Months = map[Commitment]int{
	Mensal:     1,
	Trimestral: 3,
}

// Price returns the total charge amount for a tier+commitment pair.
// Returns 0 and false if the combination is invalid.
func Price(tier Tier, commitment Commitment) (float64, bool) {
	v, ok := Prices[key{tier, commitment}]
	return v, ok
}

func ValidTier(s string) bool {
	switch Tier(s) {
	case Basico, Parceiro, Profissional:
		return true
	}
	return false
}

func ValidCommitment(s string) bool {
	switch Commitment(s) {
	case Mensal, Trimestral:
		return true
	}
	return false
}

// AsaasFrequency maps a commitment to the Asaas Pix Automatico frequency value.
var AsaasFrequency = map[Commitment]string{
	Mensal:     "MONTHLY",
	Trimestral: "QUARTERLY",
}
