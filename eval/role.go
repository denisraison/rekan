package eval

import "math/rand/v2"

type Role struct {
	Name        string
	Description string
}

var RolePool = []Role{
	{"Bastidor", "algo do seu trabalho que o cliente nunca vê e ficaria surpreso de saber."},
	{"Útil", "um erro que todo mundo comete ou uma verdade incômoda sobre o seu nicho."},
	{"Pessoal", "um momento real que mudou algo no jeito que você trabalha."},
	{"Cliente", "uma história de cliente que mostra como seu serviço funciona na prática."},
	{"Opinião", "uma posição firme sobre algo do seu nicho que nem todo mundo concorda."},
	{"Dia a dia", "uma cena real da sua rotina de trabalho, sem filtro."},
	{"Antes/Depois", "uma transformação visível que mostra o resultado do seu trabalho."},
	{"Tendência", "algo que tá mudando no seu nicho e como isso afeta seus clientes."},
	{"Pergunta", "uma dúvida que seus clientes sempre fazem e a resposta que você dá."},
	{"Marco", "uma conquista, número ou momento que marca sua trajetória no negócio."},
	{"Temporada", "algo ligado à época do ano, evento local ou sazonalidade do seu serviço."},
	{"Desafio", "um problema real que você enfrentou no negócio e como resolveu."},
}

// PickRoles selects n random roles from the pool, excluding any whose Name is in the exclude list.
func PickRoles(n int, exclude []string) []Role {
	excl := make(map[string]bool, len(exclude))
	for _, name := range exclude {
		excl[name] = true
	}

	var candidates []Role
	for _, r := range RolePool {
		if !excl[r.Name] {
			candidates = append(candidates, r)
		}
	}

	if n >= len(candidates) {
		return candidates
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return candidates[:n]
}
