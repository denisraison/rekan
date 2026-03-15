package agent

import (
	"fmt"

	"github.com/denisraison/rekan/api/internal/service"
)

func validateCustomerCreate(p service.CreateBusinessParams) error {
	if p.Name == "" {
		return fmt.Errorf("faltou o nome da cliente")
	}
	if p.Type == "" {
		return fmt.Errorf("faltou o tipo de negócio")
	}
	if p.City == "" {
		return fmt.Errorf("faltou a cidade")
	}
	if p.Phone == "" {
		return fmt.Errorf("faltou o telefone")
	}
	return nil
}
