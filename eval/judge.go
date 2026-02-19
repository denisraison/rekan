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
	type judgeOut struct {
		idx    int
		result JudgeResult
		err    error
	}

	ch := make(chan judgeOut, len(judgeNames))
	for i, name := range judgeNames {
		go func(i int, name string) {
			r, err := RunJudge(ctx, name, profile, content)
			ch <- judgeOut{idx: i, result: r, err: err}
		}(i, name)
	}

	results := make([]JudgeResult, len(judgeNames))
	for range judgeNames {
		out := <-ch
		if out.err != nil {
			return nil, out.err
		}
		results[out.idx] = out.result
	}
	return results, nil
}
