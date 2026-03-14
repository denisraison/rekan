package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/denisraison/rekan/api/internal/agent"
)

type runJSON struct {
	Timestamp string           `json:"timestamp"`
	Cases     []caseResultJSON `json:"cases"`
	Summary   summaryJSON      `json:"summary"`
}

type caseResultJSON struct {
	ID             string            `json:"id"`
	Passed         bool              `json:"passed"`
	Checks         []checkResultJSON `json:"checks"`
	InputTokens    int               `json:"input_tokens"`
	OutputTokens   int               `json:"output_tokens"`
	WallTimeMs     int64             `json:"wall_time_ms"`
	ToolRoundTrips int               `json:"tool_round_trips"`
	Error          string            `json:"error,omitempty"`
}

type checkResultJSON struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

type summaryJSON struct {
	TotalCases  int `json:"total_cases"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	TotalChecks int `json:"total_checks"`
	ChecksPassed int `json:"checks_passed"`
}

func main() {
	verbose := flag.Bool("verbose", false, "print full reply and tool log per case")
	casesDir := flag.String("cases", "", "directory containing YAML test cases (default: auto-detect)")
	caseFilter := flag.String("case", "", "run only case(s) matching this substring (comma-separated)")
	flag.Parse()

	dir := *casesDir
	if dir == "" {
		dir = findCasesDir()
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no test cases found in %s\n", dir)
		os.Exit(1)
	}

	var allCases []agent.TestCase
	for _, f := range files {
		cases, err := agent.LoadTestCases(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", f, err)
			os.Exit(1)
		}
		allCases = append(allCases, cases...)
	}

	if *caseFilter != "" {
		filters := strings.Split(*caseFilter, ",")
		var filtered []agent.TestCase
		for _, tc := range allCases {
			for _, f := range filters {
				if strings.Contains(tc.ID, strings.TrimSpace(f)) {
					filtered = append(filtered, tc)
					break
				}
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "no cases matching %q\n", *caseFilter)
			os.Exit(1)
		}
		allCases = filtered
	}

	fmt.Printf("Running %d cases...\n\n", len(allCases))

	ctx := context.Background()
	results := agent.RunEval(ctx, allCases)

	// Print table
	maxID := 0
	for _, r := range results {
		if len(r.ID) > maxID {
			maxID = len(r.ID)
		}
	}

	totalPassed := 0
	totalChecks := 0
	totalChecksPassed := 0

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		fmt.Printf("  [%s] %-*s  %4dms  in:%d out:%d  trips:%d\n",
			status, maxID, r.ID, r.WallTimeMs, r.InputTokens, r.OutputTokens, r.ToolRoundTrips)

		if r.Passed {
			totalPassed++
		}

		for _, c := range r.Checks {
			totalChecks++
			if c.Passed {
				totalChecksPassed++
			}
			if !c.Passed {
				fmt.Printf("    [-] %s: %s\n", c.Name, c.Reason)
			} else if *verbose {
				fmt.Printf("    [+] %s\n", c.Name)
			}
		}

		if *verbose && r.Reply != "" {
			fmt.Printf("    Reply: %s\n", truncateStr(r.Reply, 200))
			if len(r.ToolsCalled) > 0 {
				fmt.Printf("    Tools: %s\n", strings.Join(r.ToolsCalled, ", "))
			}
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Cases: %d/%d passed\n", totalPassed, len(results))
	fmt.Printf("Checks: %d/%d passed\n", totalChecksPassed, totalChecks)

	// Write run JSON
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	run := runJSON{
		Timestamp: ts,
		Summary: summaryJSON{
			TotalCases:   len(results),
			Passed:       totalPassed,
			Failed:       len(results) - totalPassed,
			TotalChecks:  totalChecks,
			ChecksPassed: totalChecksPassed,
		},
	}
	for _, r := range results {
		cr := caseResultJSON{
			ID:             r.ID,
			Passed:         r.Passed,
			InputTokens:    r.InputTokens,
			OutputTokens:   r.OutputTokens,
			WallTimeMs:     r.WallTimeMs,
			ToolRoundTrips: r.ToolRoundTrips,
			Error:          r.Error,
		}
		for _, c := range r.Checks {
			cr.Checks = append(cr.Checks, checkResultJSON{
				Name:   c.Name,
				Passed: c.Passed,
				Reason: c.Reason,
			})
		}
		run.Cases = append(run.Cases, cr)
	}

	runsDir := findRunsDir()
	os.MkdirAll(runsDir, 0o755) //nolint:errcheck
	runPath := filepath.Join(runsDir, "agent-"+ts+".json")
	data, _ := json.MarshalIndent(run, "", "  ")
	if err := os.WriteFile(runPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write run file: %v\n", err)
	} else {
		fmt.Printf("Run saved to %s\n", runPath)
	}

	if totalPassed < len(results) {
		os.Exit(1)
	}
}

func findCasesDir() string {
	return "internal/agent/cases"
}

func findRunsDir() string {
	return "runs"
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
