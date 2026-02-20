client<llm> JudgeClient {
  provider openrouter
  options {
    model "google/gemini-3-flash-preview"
    api_key env.OPENROUTER_API_KEY
    temperature 0.1
    max_tokens 1024
  }
}

client<llm> JudgeClientClaude {
  provider openrouter
  options {
    model "anthropic/claude-haiku-4.5"
    api_key env.OPENROUTER_API_KEY
    temperature 0.1
    max_tokens 1024
  }
}

client<llm> JudgeClientDeepSeek {
  provider openrouter
  options {
    model "deepseek/deepseek-v3.2"
    api_key env.OPENROUTER_API_KEY
    temperature 0.1
    max_tokens 1024
  }
}

client<llm> GeneratorClient {
  provider openrouter
  options {
    model "google/gemini-3-flash-preview"
    api_key env.OPENROUTER_API_KEY
    temperature 0.7
    max_tokens 4096
  }
}
