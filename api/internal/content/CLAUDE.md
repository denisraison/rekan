# Eval Pipeline

Requires `CLAUDE_API_KEY` and `GEMINI_API_KEY` in `.env` at the project root.

## CLI flags

```bash
cd api
go run ./cmd/eval                                    # heuristics only
go run ./cmd/eval --judges                           # + LLM judges
go run ./cmd/eval --judges --verbose                 # + generated content and judge reasoning
go run ./cmd/eval --profile "Closet da Re"           # single profile
go run ./cmd/eval --judges --from-run runs/FILE.json # re-judge saved content (skips generation)
go run ./cmd/eval --diff runs/BEFORE.json runs/AFTER.json  # compare two runs
go run ./cmd/eval --rekan --judges --verbose               # Rekan-specific prompt (founder-voice)
go run ./cmd/eval --cheap --fast                           # Gemini Flash generation + single judge + 4 profiles
```

Every eval saves a timestamped JSON to `runs/` (gitignored). Use `--diff` to compare two runs (`+!` improved, `-!` regressed).

## Prompt optimization loop

1. `make eval-cheap` to get a fast baseline (Gemini Flash generation, single judge, 4 profiles)
2. Pick a failing business: `go run ./cmd/eval --cheap --judges --verbose --profile "Name"`
3. Read the judge reasoning, form one hypothesis
4. Edit `baml_src/content.baml` (the generation prompt)
5. `make eval-cheap`, then diff: `go run ./cmd/eval --diff runs/BEFORE.json runs/AFTER.json`
6. Keep or revert. Max 5 cycles per session.
7. Final validation: `make eval-judges` (Opus generation, both judges, all profiles)

For judge prompt changes, use `--from-run` to re-judge existing content without regenerating.

The `--cheap` flag uses `CheapGeneratorClient` (Gemini Flash) instead of Opus for generation. ~10x cheaper per run. Use `/optimize` skill for automated multi-candidate optimization.
