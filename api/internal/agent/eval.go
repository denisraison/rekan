package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	baml "github.com/denisraison/rekan/api/internal/baml/baml_client"
	"gopkg.in/yaml.v3"
)

// TestCase represents a single eval test case from YAML.
type TestCase struct {
	ID                  string   `yaml:"id"`
	Message             string   `yaml:"message"`
	Operator            Operator `yaml:"operator"`
	Context             string   `yaml:"context"`
	ConversationHistory string   `yaml:"conversation_history"`
	Graders             []Grader `yaml:"graders"`
}

// Operator identifies the test sender.
type Operator struct {
	Name string `yaml:"name"`
	JID  string `yaml:"jid"`
}

// Grader describes how to evaluate a test case result.
type Grader struct {
	Type     string `yaml:"type"`
	Field    string `yaml:"field"`
	Equals   string `yaml:"equals"`
	Judge    string `yaml:"judge"`
	Criteria string `yaml:"criteria"`
}

// TestSuite is the top-level YAML structure.
type TestSuite struct {
	Tests []TestCase `yaml:"tests"`
}

// TestResult holds the outcome of running a single test case.
type TestResult struct {
	ID       string
	Passed   bool
	Checks   []CheckResult
}

// CheckResult is the outcome of a single grader.
type CheckResult struct {
	Grader Grader
	Passed bool
	Reason string
}

// LoadTestCases reads test cases from a YAML file.
func LoadTestCases(path string) ([]TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading test cases: %w", err)
	}
	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parsing test cases: %w", err)
	}
	return suite.Tests, nil
}

// RunEval runs all test cases and returns results.
func RunEval(ctx context.Context, cases []TestCase, verbose bool) ([]TestResult, error) {
	results := make([]TestResult, 0, len(cases))

	for _, tc := range cases {
		result := TestResult{ID: tc.ID, Passed: true}

		// Call BAML agent
		history := tc.ConversationHistory
		response, err := baml.AgentProcess(ctx, tc.Operator.Name, tc.Message, tc.Context, history)
		if err != nil {
			result.Passed = false
			result.Checks = append(result.Checks, CheckResult{
				Grader: Grader{Type: "error"},
				Passed: false,
				Reason: fmt.Sprintf("BAML error: %v", err),
			})
			results = append(results, result)
			continue
		}

		// Extract fields for grading
		actionType := ""
		if response.Action != nil {
			actionType = string(response.Action.ActionType)
		}
		reply := ""
		if response.Reply != nil {
			reply = *response.Reply
		}
		hasReply := "false"
		if reply != "" {
			hasReply = "true"
		}

		if verbose {
			fmt.Printf("  [%s] action=%q reply=%q\n", tc.ID, actionType, truncate(reply, 80))
		}

		// Run graders
		for _, g := range tc.Graders {
			cr := runGrader(g, actionType, reply, hasReply)
			result.Checks = append(result.Checks, cr)
			if !cr.Passed {
				result.Passed = false
			}
		}

		results = append(results, result)
	}

	return results, nil
}

func runGrader(g Grader, actionType, reply, hasReply string) CheckResult {
	switch g.Type {
	case "deterministic":
		return runDeterministicGrader(g, actionType, reply, hasReply)
	case "llm_judge":
		// LLM judges run separately (require API keys)
		return CheckResult{Grader: g, Passed: true, Reason: "llm judge (skipped in deterministic mode)"}
	default:
		return CheckResult{Grader: g, Passed: false, Reason: fmt.Sprintf("unknown grader type: %s", g.Type)}
	}
}

func runDeterministicGrader(g Grader, actionType, reply, hasReply string) CheckResult {
	var actual string
	switch g.Field {
	case "action_type":
		actual = actionType
	case "reply":
		actual = reply
	case "has_reply":
		actual = hasReply
	default:
		return CheckResult{Grader: g, Passed: false, Reason: fmt.Sprintf("unknown field: %s", g.Field)}
	}

	passed := strings.EqualFold(actual, g.Equals)
	reason := ""
	if !passed {
		reason = fmt.Sprintf("expected %q, got %q", g.Equals, actual)
	}
	return CheckResult{Grader: g, Passed: passed, Reason: reason}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
