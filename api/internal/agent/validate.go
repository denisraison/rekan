package agent

import (
	"errors"

	"github.com/denisraison/rekan/api/internal/service"
)

func validateCustomerCreate(p service.CreateBusinessParams) error {
	if p.Name == "" {
		return errors.New("faltou o nome da cliente")
	}
	if p.Type == "" {
		return errors.New("faltou o tipo de negócio")
	}
	if p.City == "" {
		return errors.New("faltou a cidade")
	}
	if p.Phone == "" {
		return errors.New("faltou o telefone")
	}
	return nil
}
