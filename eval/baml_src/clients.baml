client<llm> JudgeClient {
  provider google-ai
  options {
    model "gemini-3-flash-preview"
    api_key env.GEMINI_API_KEY
    temperature 0.1
    max_tokens 1024
  }
}

client<llm> JudgeClientClaude {
  provider anthropic
  options {
    model "claude-haiku-4-5-20251001"
    api_key env.CLAUDE_API_KEY
    temperature 0.1
    max_tokens 1024
  }
}

client<llm> GeneratorClient {
  provider anthropic
  options {
    model "claude-opus-4-6"
    api_key env.CLAUDE_API_KEY
    temperature 0.7
    max_tokens 4096
  }
}
