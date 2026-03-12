class ProfileSignal {
  field string    // "services", "quirks", "target_audience", "brand_vibe"
  value string    // for services: "Name|price_brl" (e.g. "Selagem|150.0"); for others: plain text
}

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
  client ProfileClient
  prompt #"
    Você vai extrair informações de um negócio a partir de uma transcrição de áudio em português falado de forma casual.

    Tipo do negócio: {{ businessType }}

    Transcrição:
    ---
    {{ transcript }}
    ---

    Regras de extração:
    - O áudio é fala informal, com vícios de linguagem, frases incompletas e recomeços. Isso é normal.
    - Extraia serviços e preços literalmente ("selagem por R$150" → name: "Selagem", priceBRL: 150). Para faixas de preço, use o menor valor.
    - Nomes de serviço devem ser curtos e identificáveis, sem fragmentos de fala.
    - Infira targetAudience a partir de pistas de contexto ("mulheres da região", "jovens que querem emagrecer").
    - brandVibe: 1 a 3 adjetivos curtos que descrevem o tom e a atmosfera do lugar ("premium", "acolhedor", "despojado e divertido"). Não inclua adjetivos sobre a personalidade do dono. Não use frases completas.
    - quirks: diferenciais concretos extraídos diretamente do que foi dito — não resumos nem inferências. Cada quirk deve ter 3 a 7 palavras. Não repita o tipo do negócio como quirk. Prefira fatos específicos e incomuns ("atende só por encomenda", "gelato feito na hora") a descrições genéricas ("ambiente agradável", "atendimento de qualidade"). Inclua fatos sobre o dono com o nome se mencionado.
    - Se um campo não for mencionado, retorne null. NUNCA invente. Um resultado parcial com 2 campos é melhor que um resultado completo com valores inventados.

    {{ ctx.output_format }}
  "#
}

function ExtractProfileSignal(message: string, businessType: string) -> ProfileSignal? {
  client JudgeClient
  prompt #"
    Você está analisando uma mensagem de WhatsApp enviada por um cliente de um negócio brasileiro.

    Tipo do negócio: {{ businessType }}

    Mensagem:
    ---
    {{ message }}
    ---

    Verifique se a mensagem menciona um serviço, preço, diferencial ou característica do negócio que ajudaria a melhorar o perfil.

    Regras:
    - Se mencionar um serviço específico com ou sem preço: retorne field="services", value="Nome do Serviço|preco" (ex: "Selagem|150.0" ou "Corte|0")
    - Se mencionar algo que torna o negócio único ou especial: retorne field="quirks", value="o texto relevante"
    - Se descrever o público-alvo: retorne field="target_audience", value="descrição"
    - Se descrever o ambiente ou estilo do negócio: retorne field="brand_vibe", value="descrição"
    - Se a mensagem for apenas saudação, agendamento, reclamação ou não tiver informação útil sobre o perfil: retorne null
    - Retorne apenas o sinal mais relevante. Se não houver nada útil, retorne null.

    {{ ctx.output_format }}
  "#
}
