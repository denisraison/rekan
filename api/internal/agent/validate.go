package agent

import (
	"fmt"

	"github.com/denisraison/rekan/api/internal/baml/baml_client/types"
)

func validateCustomerCreate(p *types.CustomerCreateParams, operatorName string) error {
	if p.Name == "" {
		return fmt.Errorf("%s, faltou o nome da cliente, pode repetir?", operatorName)
	}
	if p.Type == "" {
		return fmt.Errorf("%s, faltou o tipo de negócio, pode repetir?", operatorName)
	}
	if p.City == "" {
		return fmt.Errorf("%s, faltou a cidade, pode repetir?", operatorName)
	}
	return nil
}

func validatePostApprove(p *types.PostApproveParams, operatorName string) error {
	if p.PostId == "" {
		return fmt.Errorf("%s, qual post você quer aprovar?", operatorName)
	}
	return nil
}

func validatePostReject(p *types.PostRejectParams, operatorName string) error {
	if p.PostId == "" {
		return fmt.Errorf("%s, qual post você quer rejeitar?", operatorName)
	}
	return nil
}
