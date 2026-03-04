class PartialService {
  name string
  priceBRL float?
}

class PartialBusinessProfile {
  services PartialService[]?
  targetAudience string?
  brandVibe string?
  quirks string[]?
}

function ExtractBusinessProfile(transcript: string, businessType: string) -> PartialBusinessProfile {
  client JudgeClient
  prompt #"
    Você vai extrair informações de um negócio a partir de uma transcrição de áudio em português falado de forma casual.

    Tipo do negócio: {{ businessType }}

    Transcrição:
    ---
    {{ transcript }}
    ---

    Regras de extração:
    - O áudio é fala informal, com vícios de linguagem, frases incompletas e recomeços. Isso é normal.
    - Extraia serviços e preços literalmente ("selagem por R$150" → name: "Selagem", priceBRL: 150)
    - Infira targetAudience a partir de pistas de contexto ("mulheres da região", "jovens que querem emagrecer")
    - Capture brandVibe a partir de adjetivos e descrições de atmosfera ("acolhedor", "bem-humorado", "família")
    - Para quirks, transcreva literalmente quando possível — devem soar como algo que uma pessoa disse, não um resumo de IA
    - Se um campo não for mencionado, retorne null. NUNCA invente. Um resultado parcial com 2 campos é melhor que um resultado completo com valores inventados.

    {{ ctx.output_format }}
  "#
}
