package agent

import (
	"fmt"

	"github.com/denisraison/rekan/api/internal/service"
)

func validateCustomerCreate(p service.CreateBusinessParams, operatorName string) error {
	if p.Name == "" {
		return fmt.Errorf("%s, faltou o nome da cliente, pode repetir?", operatorName)
	}
	if p.Type == "" {
		return fmt.Errorf("%s, faltou o tipo de negócio, pode repetir?", operatorName)
	}
	if p.City == "" {
		return fmt.Errorf("%s, faltou a cidade, pode repetir?", operatorName)
	}
	if p.Phone == "" {
		return fmt.Errorf("%s, faltou o telefone da cliente, pode repetir?", operatorName)
	}
	return nil
}
