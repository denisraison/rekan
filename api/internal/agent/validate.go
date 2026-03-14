package agent

import "fmt"

func validateCustomerCreate(p *CustomerCreateParams, operatorName string) error {
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

func validatePostApprove(p *PostApproveParams, operatorName string) error {
	if p.PostId == "" {
		return fmt.Errorf("%s, qual post você quer aprovar?", operatorName)
	}
	return nil
}

func validatePostReject(p *PostRejectParams, operatorName string) error {
	if p.PostId == "" {
		return fmt.Errorf("%s, qual post você quer rejeitar?", operatorName)
	}
	return nil
}
