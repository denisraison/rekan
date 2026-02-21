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

	"github.com/denisraison/rekan/eval"
)

func main() {
	judges := flag.Bool("judges", false, "enable LLM judges (default: heuristics only)")
	verbose := flag.Bool("verbose", false, "print generated content and judge reasoning")
	profile := flag.String("profile", "", "run a single profile instead of all")
	fromRun := flag.String("from-run", "", "re-judge content from a previous run file (skips generation)")
	diff := flag.Bool("diff", false, "compare two run files: --diff <before.json> <after.json>")
	fast := flag.Bool("fast", false, "optimization mode: single judge (Gemini Flash) + 4 profiles")
	roles := flag.String("roles", "", "comma-separated role names (e.g. \"bastidor,opinião,marco\")")
	chain := flag.Int("chain", 0, "generate N consecutive batches for one profile, passing hooks forward")
	rekan := flag.Bool("rekan", false, "use Rekan-specific generation prompt")
	message := flag.String("message", "", "generate a single post from a WhatsApp message (requires --profile)")
	flag.Parse()

	if *fast {
		eval.JudgeClients = eval.JudgeClients[:1]
		*judges = true
	}

	if *message != "" {
		// Skip variedade judge (compares across posts, meaningless for single post)
		filtered := make([]string, 0, len(eval.JudgeNames))
		for _, n := range eval.JudgeNames {
			if n != "variedade" {
				filtered = append(filtered, n)
			}
		}
		eval.JudgeNames = filtered
	}

	gen := eval.GenerateFunc(eval.Generate)
	if *rekan {
		gen = eval.GenerateRekan
		if *profile == "" {
			*profile = "Rekan"
		}
	}

	if *diff {
		args := flag.Args()
		if len(args) != 2 {
			fmt.Fprintf(os.Stderr, "usage: eval --diff <before.json> <after.json>\n")
			os.Exit(1)
		}
		if err := printDiff(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var results []result
	var err error

	if *message != "" {
		if *profile == "" {
			fmt.Fprintf(os.Stderr, "error: --message requires --profile\n")
			os.Exit(1)
		}
		results, err = messageGenerate(context.Background(), *profile, *message, *judges, *verbose)
	} else if *chain > 0 {
		if *profile == "" {
			fmt.Fprintf(os.Stderr, "error: --chain requires --profile\n")
			os.Exit(1)
		}
		results, err = chainGenerate(context.Background(), *chain, *profile, *judges, *verbose, *roles, gen)
	} else if *fromRun != "" {
		results, err = loadRun(*fromRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading run: %v\n", err)
			os.Exit(1)
		}
		if *profile != "" {
			filtered := make([]result, 0, 1)
			for _, r := range results {
				if strings.EqualFold(r.name, *profile) {
					filtered = append(filtered, r)
				}
			}
			if len(filtered) == 0 {
				fmt.Fprintf(os.Stderr, "profile %q not found in run\n", *profile)
				os.Exit(1)
			}
			results = filtered
		}
		results, err = evaluateContent(results, *judges, *verbose)
	} else {
		sample := 0
		if *fast && *profile == "" {
			sample = 4
		}
		results, err = generateAndEvaluate(*judges, *verbose, *profile, sample, *roles, gen)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	printTable(results, *judges)

	if err := saveRun(results, *judges); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save run: %v\n", err)
	}

	anyFailed := false
	for _, r := range results {
		passed := 0
		for _, c := range r.checks {
			if c.Pass {
				passed++
			}
		}
		if passed == 0 {
			anyFailed = true
		}
	}
	if anyFailed {
		os.Exit(1)
	}
}

func parseRoles(names string) ([]eval.Role, error) {
	pool := make(map[string]eval.Role, len(eval.RolePool))
	for _, r := range eval.RolePool {
		pool[strings.ToLower(r.Name)] = r
	}
	var roles []eval.Role
	for _, name := range strings.Split(names, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		r, ok := pool[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("unknown role %q", name)
		}
		roles = append(roles, r)
	}
	return roles, nil
}

func generateAndEvaluate(withJudges, verbose bool, profileFilter string, sample int, rolesFlag string, gen eval.GenerateFunc) ([]result, error) {
	profiles, err := loadProfiles("testdata")
	if err != nil {
		return nil, err
	}

	if profileFilter != "" {
		filtered := make([]eval.BusinessProfile, 0, 1)
		for _, p := range profiles {
			if strings.EqualFold(p.BusinessName, profileFilter) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("profile %q not found", profileFilter)
		}
		profiles = filtered
	} else if sample > 0 && sample < len(profiles) {
		step := len(profiles) / sample
		sampled := make([]eval.BusinessProfile, 0, sample)
		for i := 0; i < len(profiles) && len(sampled) < sample; i += step {
			sampled = append(sampled, profiles[i])
		}
		profiles = sampled
	}

	var fixedRoles []eval.Role
	if rolesFlag != "" {
		fixedRoles, err = parseRoles(rolesFlag)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()

	// Generate content for all profiles in parallel.
	type genOut struct {
		idx     int
		profile eval.BusinessProfile
		posts   []eval.Post
		err     error
	}

	genCh := make(chan genOut, len(profiles))
	for i, p := range profiles {
		r := fixedRoles
		if r == nil {
			r = eval.PickRoles(3, nil)
		}
		go func(i int, p eval.BusinessProfile, roles []eval.Role) {
			fmt.Fprintf(os.Stderr, "Generating: %s...\n", p.BusinessName)
			posts, err := gen(ctx, p, roles, nil)
			genCh <- genOut{idx: i, profile: p, posts: posts, err: err}
		}(i, p, r)
	}

	generated := make([]genOut, len(profiles))
	for range profiles {
		out := <-genCh
		if out.err != nil {
			return nil, fmt.Errorf("generating %s: %w", out.profile.BusinessName, out.err)
		}
		generated[out.idx] = out
	}

	// Build results with posts, then evaluate.
	results := make([]result, len(generated))
	for _, g := range generated {
		results[g.idx] = result{name: g.profile.BusinessName, posts: g.posts, profile: g.profile}
	}

	return evaluateResults(ctx, results, withJudges, verbose)
}

func chainGenerate(ctx context.Context, n int, profileName string, withJudges, verbose bool, rolesFlag string, gen eval.GenerateFunc) ([]result, error) {
	profiles, err := loadProfiles("testdata")
	if err != nil {
		return nil, err
	}
	var prof eval.BusinessProfile
	found := false
	for _, p := range profiles {
		if strings.EqualFold(p.BusinessName, profileName) {
			prof = p
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("profile %q not found", profileName)
	}

	var fixedRoles []eval.Role
	if rolesFlag != "" {
		fixedRoles, err = parseRoles(rolesFlag)
		if err != nil {
			return nil, err
		}
	}

	var allHooks []string
	var results []result

	for i := 1; i <= n; i++ {
		roles := fixedRoles
		if roles == nil {
			roles = eval.PickRoles(3, nil)
		}

		roleNames := make([]string, len(roles))
		for j, r := range roles {
			roleNames[j] = r.Name
		}
		fmt.Fprintf(os.Stderr, "Batch %d/%d [%s] (%d hooks excluded)...\n", i, n, strings.Join(roleNames, ", "), len(allHooks))

		posts, err := gen(ctx, prof, roles, allHooks)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i, err)
		}

		hooks := eval.ExtractHooks(posts)
		allHooks = append(allHooks, hooks...)

		results = append(results, result{name: fmt.Sprintf("%s [batch %d]", prof.BusinessName, i), posts: posts, profile: prof})
	}

	fmt.Fprintf(os.Stderr, "\n--- Hook summary (%d total) ---\n", len(allHooks))
	for i, h := range allHooks {
		fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, h)
	}
	fmt.Fprintf(os.Stderr, "---\n\n")

	return evaluateResults(ctx, results, withJudges, verbose)
}

func messageGenerate(ctx context.Context, profileName, message string, withJudges, verbose bool) ([]result, error) {
	profiles, err := loadProfiles("testdata")
	if err != nil {
		return nil, err
	}
	var prof eval.BusinessProfile
	found := false
	for _, p := range profiles {
		if strings.EqualFold(p.BusinessName, profileName) {
			prof = p
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("profile %q not found", profileName)
	}

	fmt.Fprintf(os.Stderr, "Generating from message: %s...\n", prof.BusinessName)
	post, err := eval.GenerateFromMessage(ctx, prof, message, nil)
	if err != nil {
		return nil, fmt.Errorf("generating from message for %s: %w", prof.BusinessName, err)
	}

	results := []result{{
		name:    prof.BusinessName,
		posts:   []eval.Post{post},
		profile: prof,
	}}

	return evaluateResults(ctx, results, withJudges, verbose)
}

func evaluateContent(results []result, withJudges, verbose bool) ([]result, error) {
	// Load profiles so we can match them for judges.
	profiles, err := loadProfiles("testdata")
	if err != nil {
		return nil, err
	}
	profileMap := make(map[string]eval.BusinessProfile, len(profiles))
	for _, p := range profiles {
		profileMap[strings.ToLower(p.BusinessName)] = p
	}

	for i := range results {
		p, ok := profileMap[strings.ToLower(results[i].name)]
		if !ok {
			return nil, fmt.Errorf("profile %q not found in testdata", results[i].name)
		}
		results[i].profile = p
	}

	return evaluateResults(context.Background(), results, withJudges, verbose)
}

func evaluateResults(ctx context.Context, results []result, withJudges, verbose bool) ([]result, error) {
	if verbose {
		for _, r := range results {
			fmt.Printf("\n=== %s ===\n%s\n", r.name, eval.RenderPosts(r.posts))
		}
	}

	// Run heuristics (instant, no need for goroutines).
	for i := range results {
		results[i].checks = eval.RunChecks(results[i].posts, results[i].profile)
	}

	if !withJudges {
		return results, nil
	}

	// Run judges for all profiles in parallel.
	type judgeOut struct {
		idx     int
		judges  []eval.JudgeResult
		err     error
	}

	judgeCh := make(chan judgeOut, len(results))
	for i, r := range results {
		go func(i int, r result) {
			fmt.Fprintf(os.Stderr, "Judging: %s...\n", r.name)
			rendered := eval.RenderPosts(r.posts)
			j, err := eval.RunAllJudges(ctx, r.profile, rendered)
			judgeCh <- judgeOut{idx: i, judges: j, err: err}
		}(i, r)
	}

	for range results {
		out := <-judgeCh
		if out.err != nil {
			return nil, out.err
		}
		results[out.idx].judges = out.judges
		if verbose {
			for _, j := range out.judges {
				fmt.Printf("  [%s] %v", j.Name, j.Verdict)
				if len(j.Votes) > 0 {
					fmt.Print(" (")
					for i, v := range j.Votes {
						if i > 0 {
							fmt.Print(", ")
						}
						if v.Error != "" {
							fmt.Printf("%s:ERR", v.Client)
						} else {
							mark := "+"
							if !v.Verdict {
								mark = "-"
							}
							fmt.Printf("%s:%s", v.Client, mark)
						}
					}
					fmt.Print(")")
				}
				fmt.Printf(": %s\n", j.Reasoning)
			}
		}
	}

	return results, nil
}

// runRecord is the JSON structure saved to disk for each eval run.
type runRecord struct {
	Timestamp string           `json:"timestamp"`
	Judges    bool             `json:"judges"`
	Results   []businessRecord `json:"results"`
	Summary   summaryRecord    `json:"summary"`
}

type businessRecord struct {
	Business string        `json:"business"`
	Content  string        `json:"content"`
	Posts    []eval.Post   `json:"posts,omitempty"`
	Checks   []checkRecord `json:"checks"`
	Judges   []judgeRecord `json:"judges,omitempty"`
}

type checkRecord struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"`
}

type judgeRecord struct {
	Name      string       `json:"name"`
	Verdict   bool         `json:"verdict"`
	Reasoning string       `json:"reasoning"`
	Votes     []voteRecord `json:"votes,omitempty"`
}

type voteRecord struct {
	Client    string `json:"client"`
	Verdict   bool   `json:"verdict"`
	Reasoning string `json:"reasoning"`
	Error     string `json:"error,omitempty"`
}

type summaryRecord struct {
	TotalChecks  int            `json:"totalChecks"`
	PassedChecks int            `json:"passedChecks"`
	JudgeTotals  map[string]int `json:"judgeTotals,omitempty"`
}

func saveRun(results []result, withJudges bool) error {
	if err := os.MkdirAll("runs", 0o755); err != nil {
		return err
	}

	now := time.Now()
	totalChecks, totalPassed := 0, 0
	judgeTotals := map[string]int{}

	records := make([]businessRecord, 0, len(results))
	for _, r := range results {
		checks := make([]checkRecord, len(r.checks))
		for i, c := range r.checks {
			checks[i] = checkRecord{Name: c.Name, Pass: c.Pass, Reason: c.Reason}
			totalChecks++
			if c.Pass {
				totalPassed++
			}
		}

		var judges []judgeRecord
		for _, j := range r.judges {
			var votes []voteRecord
			for _, v := range j.Votes {
				votes = append(votes, voteRecord{Client: v.Client, Verdict: v.Verdict, Reasoning: v.Reasoning, Error: v.Error})
			}
			judges = append(judges, judgeRecord{Name: j.Name, Verdict: j.Verdict, Reasoning: j.Reasoning, Votes: votes})
			if j.Verdict {
				judgeTotals[j.Name]++
			}
		}

		records = append(records, businessRecord{
			Business: r.name,
			Content:  eval.RenderPosts(r.posts),
			Posts:    r.posts,
			Checks:   checks,
			Judges:   judges,
		})
	}

	run := runRecord{
		Timestamp: now.UTC().Format(time.RFC3339),
		Judges:    withJudges,
		Results:   records,
		Summary: summaryRecord{
			TotalChecks:  totalChecks,
			PassedChecks: totalPassed,
		},
	}
	if withJudges {
		run.Summary.JudgeTotals = judgeTotals
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}

	filename := filepath.Join("runs", now.UTC().Format("2006-01-02T15-04-05Z")+".json")
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Run saved to %s\n", filename)
	return nil
}

func loadRun(path string) ([]result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var run runRecord
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	results := make([]result, len(run.Results))
	for i, r := range run.Results {
		posts := r.Posts
		if len(posts) == 0 && r.Content != "" {
			// Legacy run file without structured posts.
			posts = []eval.Post{{Caption: r.Content}}
		}
		results[i] = result{name: r.Business, posts: posts}
	}
	return results, nil
}

func loadRunRecord(path string) (runRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runRecord{}, err
	}
	var run runRecord
	if err := json.Unmarshal(data, &run); err != nil {
		return runRecord{}, err
	}
	return run, nil
}

func printDiff(beforePath, afterPath string) error {
	before, err := loadRunRecord(beforePath)
	if err != nil {
		return fmt.Errorf("loading before: %w", err)
	}
	after, err := loadRunRecord(afterPath)
	if err != nil {
		return fmt.Errorf("loading after: %w", err)
	}

	type row struct {
		name                     string
		beforeChecks, afterChecks string
		beforeJudges, afterJudges map[string]bool
	}

	// Index before results by business name.
	beforeMap := make(map[string]businessRecord, len(before.Results))
	for _, r := range before.Results {
		beforeMap[r.Business] = r
	}

	// Collect all business names in after order.
	var rows []row
	for _, ar := range after.Results {
		br := beforeMap[ar.Business]
		r := row{name: ar.Business}

		bPassed, bTotal := countChecks(br.Checks)
		aPassed, aTotal := countChecks(ar.Checks)
		r.beforeChecks = fmt.Sprintf("%d/%d", bPassed, bTotal)
		r.afterChecks = fmt.Sprintf("%d/%d", aPassed, aTotal)

		r.beforeJudges = judgeMap(br.Judges)
		r.afterJudges = judgeMap(ar.Judges)

		rows = append(rows, r)
	}

	hasJudges := before.Judges || after.Judges

	nameWidth := len("Business")
	for _, r := range rows {
		if len(r.name) > nameWidth {
			nameWidth = len(r.name)
		}
	}

	judges := []struct{ short, key string }{
		{"NAT", "naturalidade"},
		{"ESP", "especificidade"},
		{"ACI", "acionavel"},
		{"VAR", "variedade"},
		{"ENG", "engajamento"},
	}

	// Header.
	fmt.Printf("%-*s  BEFORE  AFTER", nameWidth, "Business")
	if hasJudges {
		for _, j := range judges {
			fmt.Printf("  %s", j.short)
		}
	}
	fmt.Println()
	sep := strings.Repeat("-", nameWidth)
	fmt.Printf("%s  ------  -----", sep)
	if hasJudges {
		for range judges {
			fmt.Print("  ---")
		}
	}
	fmt.Println()

	// Rows.
	beforeTotalP, beforeTotalT := 0, 0
	afterTotalP, afterTotalT := 0, 0
	beforeJudgeTotals := map[string]int{}
	afterJudgeTotals := map[string]int{}

	for _, r := range rows {
		bp, bt := parseChecks(r.beforeChecks)
		ap, at := parseChecks(r.afterChecks)
		beforeTotalP += bp
		beforeTotalT += bt
		afterTotalP += ap
		afterTotalT += at

		fmt.Printf("%-*s  %-6s  %-5s", nameWidth, r.name, r.beforeChecks, r.afterChecks)
		if hasJudges {
			for _, j := range judges {
				bv, bOK := r.beforeJudges[j.key]
				av, aOK := r.afterJudges[j.key]
				if bOK && bv {
					beforeJudgeTotals[j.key]++
				}
				if aOK && av {
					afterJudgeTotals[j.key]++
				}
				fmt.Printf("  %s", diffVerdict(bv, bOK, av, aOK))
			}
		}
		fmt.Println()
	}

	// Totals.
	fmt.Printf("%s  ------  -----", sep)
	if hasJudges {
		for range judges {
			fmt.Print("  ---")
		}
	}
	fmt.Println()

	fmt.Printf("%-*s  %-6s  %-5s",
		nameWidth, "TOTAL",
		fmt.Sprintf("%d/%d", beforeTotalP, beforeTotalT),
		fmt.Sprintf("%d/%d", afterTotalP, afterTotalT),
	)
	if hasJudges {
		for _, j := range judges {
			b := beforeJudgeTotals[j.key]
			a := afterJudgeTotals[j.key]
			delta := a - b
			sign := " "
			if delta > 0 {
				sign = "+"
			}
			if delta == 0 {
				fmt.Printf("  %2d ", a)
			} else {
				fmt.Printf(" %s%d", sign, delta)
			}
		}
	}
	fmt.Println()

	return nil
}

func countChecks(checks []checkRecord) (passed, total int) {
	for _, c := range checks {
		total++
		if c.Pass {
			passed++
		}
	}
	return
}

func judgeMap(judges []judgeRecord) map[string]bool {
	m := make(map[string]bool, len(judges))
	for _, j := range judges {
		m[j.Name] = j.Verdict
	}
	return m
}

func parseChecks(s string) (passed, total int) {
	fmt.Sscanf(s, "%d/%d", &passed, &total)
	return
}

// diffVerdict shows change between two judge verdicts.
func diffVerdict(bv, bOK, av, aOK bool) string {
	if !bOK && !aOK {
		return "  ."
	}
	if !bOK {
		if av {
			return "  +"
		}
		return "  -"
	}
	if !aOK {
		return "  ."
	}
	if bv == av {
		if av {
			return "  +"
		}
		return "  -"
	}
	// Changed
	if av {
		return " +!" // improved
	}
	return " -!" // regressed
}

func printTable(results []result, showJudges bool) {
	nameWidth := len("Business")
	for _, r := range results {
		if len(r.name) > nameWidth {
			nameWidth = len(r.name)
		}
	}

	totalChecks := 0
	totalPassed := 0
	judgeTotals := map[string]int{}

	if showJudges {
		judgeShorts := map[string]string{
			"naturalidade":    "NAT",
			"especificidade":  "ESP",
			"acionavel":       "ACI",
			"variedade":       "VAR",
			"engajamento":     "ENG",
		}

		// Header
		fmt.Printf("%-*s  Checks", nameWidth, "Business")
		for _, jn := range eval.JudgeNames {
			fmt.Printf("  %s", judgeShorts[jn])
		}
		fmt.Println()
		fmt.Printf("%s  ------", strings.Repeat("-", nameWidth))
		for range eval.JudgeNames {
			fmt.Print("  ---")
		}
		fmt.Println()

		for _, r := range results {
			passed := 0
			total := len(r.checks)
			for _, c := range r.checks {
				if c.Pass {
					passed++
				}
			}
			totalChecks += total
			totalPassed += passed

			judgeMap := map[string]bool{}
			for _, j := range r.judges {
				judgeMap[j.Name] = j.Verdict
				if j.Verdict {
					judgeTotals[j.Name]++
				}
			}

			fmt.Printf("%-*s  %d/%d  ", nameWidth, r.name, passed, total)
			for _, jn := range eval.JudgeNames {
				fmt.Printf("   %s ", verdict(judgeMap[jn]))
			}
			fmt.Println()
		}

		fmt.Printf("%s  ------", strings.Repeat("-", nameWidth))
		for range eval.JudgeNames {
			fmt.Print("  ---")
		}
		fmt.Println()
		fmt.Printf("%-*s  %d/%-4d", nameWidth, "TOTAL", totalPassed, totalChecks)
		for _, jn := range eval.JudgeNames {
			fmt.Printf("  %-3d", judgeTotals[jn])
		}
		fmt.Println()
	} else {
		fmt.Printf("%-*s  Checks  Pass\n", nameWidth, "Business")
		fmt.Printf("%s  ------  ----\n", strings.Repeat("-", nameWidth))

		for _, r := range results {
			passed := 0
			total := len(r.checks)
			for _, c := range r.checks {
				if c.Pass {
					passed++
				}
			}
			totalChecks += total
			totalPassed += passed

			status := "OK"
			if passed < total {
				status = "WARN"
			}
			if passed == 0 {
				status = "FAIL"
			}

			fmt.Printf("%-*s  %d/%d     %s\n", nameWidth, r.name, passed, total, status)
		}

		fmt.Printf("%s  ------  ----\n", strings.Repeat("-", nameWidth))
		fmt.Printf("%-*s  %d/%d\n", nameWidth, "TOTAL", totalPassed, totalChecks)
	}
}

func verdict(v bool) string {
	if v {
		return "+"
	}
	return "-"
}

func loadProfiles(dir string) ([]eval.BusinessProfile, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no profiles found in %s", dir)
	}

	profiles := make([]eval.BusinessProfile, 0, len(entries))
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var p eval.BusinessProfile
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

type result struct {
	name    string
	checks  []eval.CheckResult
	judges  []eval.JudgeResult
	posts   []eval.Post
	profile eval.BusinessProfile
}
