package eval

import (
	"context"
	"fmt"

	baml "github.com/denisraison/rekan/eval/baml_client/baml_client"
	"github.com/denisraison/rekan/eval/baml_client/baml_client/types"
)

var judgeNames = []string{
	"naturalidade",
	"especificidade",
	"acionavel",
	"variedade",
	"engajamento",
}

type JudgeResult struct {
	Name      string
	Reasoning string
	Verdict   bool
}

func toBamlProfile(p BusinessProfile) types.BusinessProfile {
	services := make([]types.Service, len(p.Services))
	for i, s := range p.Services {
		services[i] = types.Service{Name: s.Name, PriceBRL: s.PriceBRL}
	}
	return types.BusinessProfile{
		BusinessName:   p.BusinessName,
		BusinessType:   p.BusinessType,
		City:           p.City,
		Neighbourhood:  p.Neighbourhood,
		Services:       services,
		TargetAudience: p.TargetAudience,
		BrandVibe:      p.BrandVibe,
		Quirks:         p.Quirks,
	}
}

func RunJudge(ctx context.Context, name string, profile BusinessProfile, content string) (JudgeResult, error) {
	bp := toBamlProfile(profile)
	var res types.JudgeResult
	var err error

	switch name {
	case "naturalidade":
		res, err = baml.JudgeNaturalidade(ctx, bp, content)
	case "especificidade":
		res, err = baml.JudgeEspecificidade(ctx, bp, content)
	case "acionavel":
		res, err = baml.JudgeAcionavel(ctx, bp, content)
	case "variedade":
		res, err = baml.JudgeVariedade(ctx, bp, content)
	case "engajamento":
		res, err = baml.JudgeEngajamento(ctx, bp, content)
	default:
		return JudgeResult{}, fmt.Errorf("unknown judge: %s", name)
	}
	if err != nil {
		return JudgeResult{}, fmt.Errorf("judge %s: %w", name, err)
	}

	return JudgeResult{
		Name:      name,
		Reasoning: res.Reasoning,
		Verdict:   res.Verdict,
	}, nil
}

func RunAllJudges(ctx context.Context, profile BusinessProfile, content string) ([]JudgeResult, error) {
	results := make([]JudgeResult, 0, len(judgeNames))
	for _, name := range judgeNames {
		r, err := RunJudge(ctx, name, profile, content)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}
