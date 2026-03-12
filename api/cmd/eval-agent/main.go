package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/denisraison/rekan/api/internal/agent"
)

func main() {
	verbose := flag.Bool("verbose", false, "print agent responses")
	casesDir := flag.String("cases", "", "directory containing YAML test cases (default: auto-detect)")
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

	ctx := context.Background()
	totalTests := 0
	totalPassed := 0
	totalChecks := 0
	totalChecksPassed := 0

	for _, f := range files {
		cases, err := agent.LoadTestCases(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", f, err)
			os.Exit(1)
		}

		fmt.Printf("=== %s (%d tests) ===\n", filepath.Base(f), len(cases))

		results, err := agent.RunEval(ctx, cases, *verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running eval: %v\n", err)
			os.Exit(1)
		}

		for _, r := range results {
			totalTests++
			if r.Passed {
				totalPassed++
			}
			for _, c := range r.Checks {
				totalChecks++
				if c.Passed {
					totalChecksPassed++
				}
			}

			status := "PASS"
			if !r.Passed {
				status = "FAIL"
			}
			fmt.Printf("  [%s] %s\n", status, r.ID)
			if !r.Passed || *verbose {
				for _, c := range r.Checks {
					mark := "+"
					if !c.Passed {
						mark = "-"
					}
					desc := fmt.Sprintf("%s.%s=%s", c.Grader.Type, c.Grader.Field, c.Grader.Equals)
					if c.Grader.Type == "llm_judge" {
						desc = fmt.Sprintf("llm_judge.%s", c.Grader.Judge)
					}
					if c.Reason != "" {
						fmt.Printf("    [%s] %s: %s\n", mark, desc, c.Reason)
					} else {
						fmt.Printf("    [%s] %s\n", mark, desc)
					}
				}
			}
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Tests: %d/%d passed\n", totalPassed, totalTests)
	fmt.Printf("Checks: %d/%d passed\n", totalChecksPassed, totalChecks)

	passRate := 0.0
	if totalTests > 0 {
		passRate = float64(totalPassed) / float64(totalTests) * 100
	}
	fmt.Printf("Pass rate: %.0f%%\n", passRate)

	if passRate < 90 {
		fmt.Fprintf(os.Stderr, "FAIL: pass rate %.0f%% < 90%%\n", passRate)
		os.Exit(1)
	}
}

// findCasesDir locates the cases directory relative to the source file or cwd.
func findCasesDir() string {
	// Try relative to the Go source file
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Join(filepath.Dir(filename), "..", "..", "internal", "agent", "cases")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	// Try relative to cwd (when running from api/)
	return "internal/agent/cases"
}
