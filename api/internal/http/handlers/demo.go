package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
)

var priceRe = regexp.MustCompile(`R\$\s*(\d+(?:[.,]\d+)?)`)

func DemoGenerate(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			BusinessName string `json:"business_name"`
			BusinessType string `json:"business_type"`
			City         string `json:"city"`
			Services     string `json:"services"`
			Message      string `json:"message"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "corpo inválido"})
		}
		if strings.TrimSpace(body.BusinessName) == "" || strings.TrimSpace(body.Message) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "nome do negócio e mensagem são obrigatórios"})
		}

		var services []eval.Service
		for _, raw := range strings.Split(body.Services, ",") {
			name := strings.TrimSpace(raw)
			if name == "" {
				continue
			}
			var price float64
			if m := priceRe.FindStringSubmatch(name); m != nil {
				name = strings.TrimSpace(priceRe.ReplaceAllString(name, ""))
				p, _ := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
				price = p
			}
			services = append(services, eval.Service{Name: name, PriceBRL: price})
		}

		profile := eval.BusinessProfile{
			BusinessName: strings.TrimSpace(body.BusinessName),
			BusinessType: strings.TrimSpace(body.BusinessType),
			City:         strings.TrimSpace(body.City),
			Services:     services,
		}

		post, err := deps.GenerateFromMessage(e.Request.Context(), profile, strings.TrimSpace(body.Message), nil)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "Erro ao gerar conteúdo. Tente novamente.",
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"caption":         post.Caption,
			"hashtags":        post.Hashtags,
			"production_note": post.ProductionNote,
		})
	}
}
