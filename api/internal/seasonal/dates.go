package seasonal

// Date represents a seasonal marketing opportunity.
// Year is non-zero for moveable holidays (Carnaval, Páscoa, Dia das Mães) that
// don't fall on the same date each year. Fixed calendar dates leave Year as 0.
type Date struct {
	Year     int // 0 = recurs every year; non-zero = specific year only
	Month    int
	Day      int
	Label    string
	Niches   []string // empty = all niches
	Template string   // {name} is replaced with the business name
}

// Dates mirrors the SEASONAL_DATES constant in the frontend operador page.
// Moveable holidays (Carnaval, Páscoa, Dia das Mães) are hardcoded for 2026.
var Dates = []Date{
	{
		Year:     2026,
		Month:    2,
		Day:      14,
		Label:    "Carnaval",
		Niches:   []string{"Salão de Beleza", "Barbearia", "Personal Trainer", "Nail Designer"},
		Template: "{name}, Carnaval tá chegando! Vamos preparar posts especiais?",
	},
	{
		Month:    3,
		Day:      8,
		Label:    "Dia da Mulher",
		Niches:   []string{"Salão de Beleza", "Nail Designer", "Confeitaria", "Loja de Roupas"},
		Template: "{name}, Dia da Mulher vem ai! Que tal um post com promo especial?",
	},
	{
		Year:     2026,
		Month:    4,
		Day:      5,
		Label:    "Páscoa",
		Niches:   []string{"Confeitaria", "Restaurante", "Hamburgueria", "Loja de Açaí"},
		Template: "{name}, Páscoa tá chegando! Vamos montar os posts das encomendas?",
	},
	{
		Year:     2026,
		Month:    5,
		Day:      10,
		Label:    "Dia das Mães",
		Niches:   []string{"Salão de Beleza", "Confeitaria", "Nail Designer", "Loja de Roupas", "Restaurante"},
		Template: "{name}, Dia das Mães daqui a pouco! Bora preparar posts de presente e promo?",
	},
	{
		Month:    6,
		Day:      12,
		Label:    "Dia dos Namorados",
		Niches:   []string{"Confeitaria", "Restaurante", "Hamburgueria", "Salão de Beleza", "Loja de Roupas"},
		Template: "{name}, Dia dos Namorados vem ai! Vamos criar posts romanticos pro seu negocio?",
	},
	{
		Month:    6,
		Day:      13,
		Label:    "Festas Juninas",
		Niches:   []string{"Confeitaria", "Restaurante", "Hamburgueria", "Banda Musical"},
		Template: "{name}, Junho ta ai! Vamos postar algo com tema junino?",
	},
	{
		Month:    9,
		Day:      1,
		Label:    "Dia do Educador Físico",
		Niches:   []string{"Personal Trainer"},
		Template: "{name}, vem ai o Dia do Educador Fisico! Bora fazer um post especial?",
	},
	{
		Month:    10,
		Day:      1,
		Label:    "Início do Verão",
		Niches:   []string{"Personal Trainer", "Loja de Açaí"},
		Template: "{name}, verao chegando! Momento perfeito pra postar sobre preparacao e resultados.",
	},
	{
		Month:    10,
		Day:      12,
		Label:    "Dia das Crianças",
		Niches:   []string{"Confeitaria", "Pet Shop", "Loja de Roupas"},
		Template: "{name}, Dia das Criancas ta perto! Vamos criar posts com ofertas kids?",
	},
	{
		Month:    12,
		Day:      19,
		Label:    "Dia do Cabeleireiro",
		Niches:   []string{"Salão de Beleza", "Barbearia"},
		Template: "{name}, Dia do Cabeleireiro chegando! Que tal um post especial celebrando a profissao?",
	},
	{
		Month:    12,
		Day:      25,
		Label:    "Natal",
		Niches:   []string{},
		Template: "{name}, Natal chegando! Vamos preparar posts com ofertas e mensagem de final de ano?",
	},
	{
		Month:    12,
		Day:      31,
		Label:    "Réveillon",
		Niches:   []string{"Salão de Beleza", "Barbearia", "Nail Designer", "Personal Trainer", "Loja de Roupas"},
		Template: "{name}, Réveillon vem aí! Bora postar sobre agendamento e preparação?",
	},
}
