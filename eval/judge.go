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

// JudgeClients are the BAML client names for the multi-model panel.
// Each judge criterion runs on all clients; final verdict is majority vote.
// Override to a single client for faster/cheaper optimization runs.
var JudgeClients = []string{
	"JudgeClient",         // Google Gemini 3 Flash
	"JudgeClientClaude",   // Anthropic Claude Haiku 4.5
	"JudgeClientDeepSeek", // DeepSeek V3.2
}

type Vote struct {
	Client    string
	Verdict   bool
	Reasoning string
	Error     string // non-empty if this model failed to produce a usable response
}

type JudgeResult struct {
	Name      string
	Reasoning string
	Verdict   bool
	Votes     []Vote
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

func toBamlRoles(roles []Role) []types.ContentRole {
	out := make([]types.ContentRole, len(roles))
	for i, r := range roles {
		out[i] = types.ContentRole{Name: r.Name, Description: r.Description}
	}
	return out
}

func runJudgeSingle(ctx context.Context, name string, bp types.BusinessProfile, content string, client string) (Vote, error) {
	var res types.JudgeResult
	var err error
	opt := baml.WithClient(client)

	switch name {
	case "naturalidade":
		res, err = baml.JudgeNaturalidade(ctx, bp, content, opt)
	case "especificidade":
		res, err = baml.JudgeEspecificidade(ctx, bp, content, opt)
	case "acionavel":
		res, err = baml.JudgeAcionavel(ctx, bp, content, opt)
	case "variedade":
		res, err = baml.JudgeVariedade(ctx, bp, content, opt)
	case "engajamento":
		res, err = baml.JudgeEngajamento(ctx, bp, content, opt)
	default:
		return Vote{}, fmt.Errorf("unknown judge: %s", name)
	}
	if err != nil {
		return Vote{}, fmt.Errorf("judge %s (%s): %w", name, client, err)
	}

	return Vote{
		Client:    client,
		Verdict:   res.Verdict,
		Reasoning: res.Reasoning,
	}, nil
}

// RunJudge runs a single judge criterion across all panel models and returns a majority vote.
func RunJudge(ctx context.Context, name string, profile BusinessProfile, content string) (JudgeResult, error) {
	bp := toBamlProfile(profile)

	type voteOut struct {
		idx  int
		vote Vote
		err  error
	}

	ch := make(chan voteOut, len(JudgeClients))
	for i, client := range JudgeClients {
		go func(i int, client string) {
			v, err := runJudgeSingle(ctx, name, bp, content, client)
			ch <- voteOut{idx: i, vote: v, err: err}
		}(i, client)
	}

	votes := make([]Vote, 0, len(JudgeClients))
	for range JudgeClients {
		out := <-ch
		if out.err != nil {
			votes = append(votes, Vote{
				Client: JudgeClients[out.idx],
				Error:  out.err.Error(),
			})
			continue
		}
		votes = append(votes, out.vote)
	}

	// Count successful votes only.
	var successful []Vote
	for _, v := range votes {
		if v.Error == "" {
			successful = append(successful, v)
		}
	}
	if len(successful) == 0 {
		return JudgeResult{}, fmt.Errorf("judge %s: all models failed", name)
	}

	trueCount := 0
	for _, v := range successful {
		if v.Verdict {
			trueCount++
		}
	}
	majority := trueCount > len(successful)/2

	// Pick reasoning from the dissenting vote (most informative for debugging).
	// If unanimous, use the first successful vote's reasoning.
	reasoning := successful[0].Reasoning
	for _, v := range successful {
		if v.Verdict != majority {
			reasoning = v.Reasoning
			break
		}
	}

	return JudgeResult{
		Name:      name,
		Reasoning: reasoning,
		Verdict:   majority,
		Votes:     votes,
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
