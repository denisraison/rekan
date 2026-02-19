class Service {
  name string
  priceBRL float
}

class BusinessProfile {
  businessName string
  businessType string
  city string
  neighbourhood string
  services Service[]
  targetAudience string
  brandVibe string
  quirks string[]
}

class JudgeResult {
  reasoning string
  verdict bool
}

function JudgeNaturalidade(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador de conteúdo para Instagram brasileiro.

    Analise se o texto abaixo soa como se uma pessoa real brasileira tivesse escrito para o Instagram.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade: {{ profile.city }}
    - Bairro: {{ profile.neighbourhood }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Critérios de reprovação:
    - Linguagem corporativa ou formal demais
    - Português de Portugal (ex: "consigo", "telemóvel", "autocarro")
    - Padrões robóticos ou repetitivos
    - Falta de naturalidade típica do Instagram brasileiro

    Critérios de aprovação:
    - Tom conversacional e informal
    - Uso natural de emojis
    - Expressões brasileiras (bora, gente, né, tá, pra)
    - Parece algo que um microempreendedor postaria

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se soa natural, false se não.

    {{ ctx.output_format }}
  "#
}

function JudgeEspecificidade(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador de conteúdo para Instagram brasileiro.

    Analise se o conteúdo abaixo é específico para este negócio, ou se é genérico o suficiente para funcionar com qualquer outro negócio.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade: {{ profile.city }}
    - Bairro: {{ profile.neighbourhood }}
    - Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}
    - Público: {{ profile.targetAudience }}
    - Vibe: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Critério de reprovação:
    - Se trocar o nome do negócio por outro qualquer, a legenda ainda funciona
    - Não menciona serviços, localização ou diferenciais específicos
    - Conteúdo genérico que serve para qualquer tipo de negócio

    Critério de aprovação:
    - Referencia o nome, cidade/bairro, serviços ou diferenciais específicos
    - O conteúdo só faz sentido para este negócio em particular

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se é específico, false se é genérico.

    {{ ctx.output_format }}
  "#
}

function JudgeAcionavel(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador de conteúdo para Instagram brasileiro.

    Analise se o dono do negócio poderia copiar e colar este conteúdo e postar hoje no Instagram.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    O conteúdo precisa ter TODOS estes elementos para ser aprovado:
    - Legenda completa (não apenas um esboço ou sugestão)
    - Hashtags relevantes
    - Call-to-action (ex: "chama no WhatsApp", "link na bio", "agende")
    - Nota de produção (sugestão de foto/vídeo para acompanhar)

    Critério de reprovação:
    - Falta qualquer um dos elementos acima
    - É apenas uma ideia ou rascunho, não um post pronto

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se está pronto para postar, false se não.

    {{ ctx.output_format }}
  "#
}

function JudgeVariedade(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador de conteúdo para Instagram brasileiro.

    Analise se os posts abaixo são genuinamente diferentes entre si. Considere que o conteúdo pode conter múltiplos posts separados.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Critérios de aprovação:
    - Mix de formatos (educativo, bastidores, antes/depois, prova social, dicas, promoção)
    - Aberturas diferentes (não começam todos iguais)
    - Estruturas variadas (não seguem o mesmo template)

    Critérios de reprovação:
    - Todos os posts seguem a mesma estrutura/template
    - Mesma abertura repetida
    - Mesmo tipo de conteúdo em todos

    Se houver apenas um post, avalie se ele demonstra criatividade e não segue um template óbvio e repetitivo.

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se há variedade, false se é repetitivo.

    {{ ctx.output_format }}
  "#
}

function JudgeEngajamento(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador de conteúdo para Instagram brasileiro.

    Analise se este conteúdo faria alguém parar de rolar o feed e interagir.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Público: {{ profile.targetAudience }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Critérios de aprovação:
    - Hook forte na primeira linha (pergunta, dado surpreendente, provocação)
    - Motivo para o leitor comentar, salvar ou compartilhar
    - Conexão emocional ou valor prático claro
    - Linguagem que convida interação

    Critérios de reprovação:
    - Primeira linha genérica ou sem gancho
    - Apenas informativo sem motivo para engajar
    - Tom passivo, sem energia
    - Nada que incentive ação do leitor

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se gera engajamento, false se não.

    {{ ctx.output_format }}
  "#
}
