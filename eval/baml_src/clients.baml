client<llm> JudgeClient {
  provider openrouter
  options {
    model "google/gemini-3-flash-preview"
    api_key env.OPENROUTER_API_KEY
    temperature 0.1
    max_tokens 1024
  }
}
