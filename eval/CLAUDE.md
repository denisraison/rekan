# Eval Pipeline

Development tool for measuring Instagram content quality. Not part of the product.

## Quick start

```bash
make eval                # heuristics only (~5s)
make eval-judges         # heuristics + LLM judges (~25s)
make test-judges         # integration tests for judge verdicts (needs OPENROUTER_API_KEY)
```

Requires `OPENROUTER_API_KEY` in `.env` at the project root.

## CLI flags

```bash
cd eval
go run ./cmd/eval                                    # heuristics only
go run ./cmd/eval --judges                           # + LLM judges
go run ./cmd/eval --judges --verbose                 # + generated content and judge reasoning
go run ./cmd/eval --profile "Closet da Re"           # single profile
go run ./cmd/eval --judges --from-run runs/FILE.json # re-judge saved content (skips generation)
go run ./cmd/eval --diff runs/BEFORE.json runs/AFTER.json  # compare two runs
```

## How it works

1. Loads 12 business profiles from `testdata/*.json`
2. Generates Instagram content for each profile via `baml_src/content.baml`
3. Runs 6 heuristic checks (business name, location, hashtags, pt-BR markers, caption length, production note)
4. Optionally runs 5 LLM judges (naturalidade, especificidade, acionavel, variedade, engajamento)
5. Prints summary table, saves full results to `runs/`

Generation and judging run in parallel across all profiles.

## Runs

Every eval saves a timestamped JSON file to `runs/` containing: generated content, check results, judge verdicts with reasoning, and summary totals. The `runs/` directory is gitignored.

Use `--diff` to compare two runs side by side. `+!` means improved, `-!` means regressed.

## Prompt optimization loop

1. `make eval-judges` and identify the weakest criterion
2. Pick a failing business: `go run ./cmd/eval --judges --verbose --profile "Name"`
3. Read the judge reasoning, form one hypothesis
4. Edit `baml_src/content.baml` (the generation prompt)
5. `make eval-judges`, then diff: `go run ./cmd/eval --diff runs/BEFORE.json runs/AFTER.json`
6. Keep or revert. Max 5 cycles per session.

For judge prompt changes, use `--from-run` to re-judge existing content without regenerating.

## Structure

```
cmd/eval/main.go     CLI entrypoint
heuristic.go         6 deterministic checks
judge.go             5 LLM judge runner (parallel)
generate.go          Content generation wrapper
baml_src/
  clients.baml       JudgeClient (temp 0.1) + GeneratorClient (temp 0.7)
  judges.baml        5 judge prompt definitions
  content.baml       Generation prompt (the file you optimize)
  generators.baml    BAML codegen config
baml_client/         Auto-generated Go client (do not edit)
testdata/            12 business profile fixtures
runs/                Saved eval results (gitignored)
```
