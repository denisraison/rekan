client<llm> AgentClient {
  provider anthropic
  options {
    model "claude-sonnet-4-6"
    api_key env.CLAUDE_API_KEY
    temperature 0.3
    max_tokens 1024
  }
}

enum AgentActionType {
  STATUS_OVERVIEW
  CUSTOMER_LIST
  CUSTOMER_CREATE
  CUSTOMER_UPDATE
  CUSTOMER_PAUSE
  CUSTOMER_INFO
  POST_GENERATE
  POST_LIST_PENDING
  POST_APPROVE
  POST_REJECT
}

enum AgentActionStatus {
  EXECUTE
  NEEDS_CONFIRMATION
}

class CustomerCreateParams {
  name string
  type string
  city string
  phone string?
  targetAudience string? @alias("target_audience")
  brandVibe string? @alias("brand_vibe")
  quirks string?
}

class CustomerUpdateParams {
  name string
  type string?
  city string?
  phone string?
  targetAudience string? @alias("target_audience")
  brandVibe string? @alias("brand_vibe")
  quirks string?
}

class CustomerPauseParams {
  name string
  reason string?
}

class CustomerInfoParams {
  name string
}

class PostGenerateParams {
  name string
}

class PostApproveParams {
  postId string @alias("post_id")
  name string
}

class PostRejectParams {
  postId string @alias("post_id")
  name string
  feedback string
}

class AgentAction {
  actionType AgentActionType
  actionStatus AgentActionStatus
  customerCreate CustomerCreateParams? @alias("customer_create")
  customerUpdate CustomerUpdateParams? @alias("customer_update")
  customerPause CustomerPauseParams? @alias("customer_pause")
  customerInfo CustomerInfoParams? @alias("customer_info")
  postGenerate PostGenerateParams? @alias("post_generate")
  postApprove PostApproveParams? @alias("post_approve")
  postReject PostRejectParams? @alias("post_reject")
}

class AgentResponse {
  reply string?
  action AgentAction?
}

function AgentProcess(operatorName: string, message: string, systemContext: string, conversationHistory: string) -> AgentResponse {
  client AgentClient
  prompt #"
    Você é o assistente do grupo de operações da Rekan no WhatsApp. Toda mensagem neste grupo é para você. Descubra o que a operadora precisa e responda.

    Operadora atual: {{ operatorName }}

    Contexto do sistema:
    {{ systemContext }}

    Histórico recente do grupo:
    {{ conversationHistory }}

    Mensagem:
    {{ message }}

    Regras de resposta:
    - Sempre chame a operadora pelo nome: "Elenice, ..." ou "Bruna, ..."
    - Máximo 300 caracteres na resposta
    - Português brasileiro informal, direto e caloroso
    - NUNCA invente dados. Use apenas o que está no contexto.
    - NUNCA use travessão (—). Use vírgula ou ponto.
    - Se a mensagem não for clara, peça esclarecimento de forma simpática.

    Ações disponíveis (cada ação preenche o campo tipado correspondente em AgentAction):
    - STATUS_OVERVIEW: visão geral do sistema. Sem campo tipado.
    - CUSTOMER_LIST: lista de clientes ativas. Sem campo tipado.
    - CUSTOMER_CREATE: cadastrar nova cliente. Preencha customer_create com: name, type, city (obrigatórios), phone, target_audience, brand_vibe, quirks (opcionais). Se faltar campo obrigatório, peça na resposta sem emitir action.
    - CUSTOMER_UPDATE: alterar dados de uma cliente. Preencha customer_update com: name (obrigatório, identifica a cliente), e os campos a alterar (type, city, phone, target_audience, brand_vibe, quirks).
    - CUSTOMER_PAUSE: pausar uma cliente. Preencha customer_pause com: name (obrigatório), reason (opcional).
    - CUSTOMER_INFO: detalhes de uma cliente. Preencha customer_info com: name.
    - POST_GENERATE: gerar posts. Preencha post_generate com: name (obrigatório).
    - POST_LIST_PENDING: posts pendentes. Sem campo tipado.
    - POST_APPROVE: aprovar post. Preencha post_approve com: post_id, name.
    - POST_REJECT: rejeitar post. Preencha post_reject com: post_id, name, feedback.

    Regras de ações:
    - Ações que modificam dados (CREATE, UPDATE, PAUSE, POST_GENERATE, POST_APPROVE, POST_REJECT) SEMPRE usam actionStatus "NEEDS_CONFIRMATION". Liste os campos extraídos no reply e peça "Confirma?"
    - Ações de leitura (STATUS_OVERVIEW, CUSTOMER_LIST, CUSTOMER_INFO, POST_LIST_PENDING) usam actionStatus "EXECUTE".
    - Se a mensagem pedir informação que exige uma ação, inclua a action correspondente.
    - Se for uma pergunta simples que o contexto já responde, responda direto no reply sem action.

    Regras de clientes:
    - Se uma cliente com mesmo nome já existe no contexto, avise e não crie duplicata.
    - Se houver ambiguidade de nome (duas Marias), peça para especificar.
    - Interprete abreviações comuns: "BH" = "Belo Horizonte", "SP" = "São Paulo", "RJ" = "Rio de Janeiro".
    - Use o histórico da conversa para inferir contexto. Se a operadora disse "muda pra 5x" depois de falar da Patricia, a mudança é na Patricia.

    Regras de posts:
    - POST_LIST_PENDING mostra posts que ainda não foram revisados. Se a operadora não especificar cliente, mostre todos.
    - POST_APPROVE e POST_REJECT usam o post_id do contexto. Se a operadora diz "aprova" sem especificar, use o post em foco no histórico.
    - POST_REJECT deve incluir o feedback da operadora no campo "feedback".

    Regras de mídia:
    - Mensagens com "[Imagem: ...]" contêm descrição de uma imagem enviada. Use a descrição para entender o contexto.
    - Se a imagem é um cartão de visita, extraia nome, tipo de negócio, cidade e telefone.
    - Se a imagem está desfocada ou ilegível, diga honestamente que não conseguiu ler. NUNCA invente conteúdo de imagens.
    - Mensagens com "[Mensagem encaminhada de +NÚMERO]" são mensagens reencaminhadas. Tente identificar o cliente pelo número de telefone no contexto.

    Regras de confirmação no histórico:
    - Se a operadora diz "sim", "confirma", "isso", "pode fazer", é confirmação de ação pendente.
    - Se a operadora diz "não", "deixa", "cancela", "esquece", é cancelamento.
    - Se não há ação pendente no histórico e a operadora diz "sim", responda algo como "Sim pra quê, [nome]? Não tem nada pendente."

    {{ ctx.output_format }}
  "#
}

test AgentStatusOverview {
  functions [AgentProcess]
  args {
    operatorName "Elenice"
    message "como tá tudo?"
    systemContext #"
      Clientes ativas: 8
      Posts pendentes de revisão: 3
      Última ação: Bruna aprovou post da Patricia há 2h
    "#
    conversationHistory ""
  }
  @@assert( {{ this.action != _ }} )
  @@assert( {{ this.reply != _ }} )
}

test AgentCustomerList {
  functions [AgentProcess]
  args {
    operatorName "Bruna"
    message "quais são as clientes?"
    systemContext #"
      Clientes ativas: 8
      Posts pendentes de revisão: 3
    "#
    conversationHistory ""
  }
  @@assert( {{ this.action != _ }} )
}

test AgentCustomerCreate {
  functions [AgentProcess]
  args {
    operatorName "Elenice"
    message "cria a Ana, manicure em Goiânia, atende mulheres do bairro, ambiente bem descontraído"
    systemContext #"
      Clientes ativas: 2
        - Patricia (Salão de Beleza, Belo Horizonte)
        - Maria (Confeitaria, São Paulo)
    "#
    conversationHistory ""
  }
  @@assert( {{ this.action != _ }} )
  @@assert( {{ this.action.actionType == "CUSTOMER_CREATE" }} )
  @@assert( {{ this.action.actionStatus == "NEEDS_CONFIRMATION" }} )
  @@assert( {{ this.action.customerCreate != _ }} )
  @@assert( {{ this.reply != _ }} )
}

test AgentCustomerPause {
  functions [AgentProcess]
  args {
    operatorName "Bruna"
    message "pausa a Patricia, ela vai viajar"
    systemContext #"
      Clientes ativas: 2
        - Patricia (Salão de Beleza, Belo Horizonte)
        - Maria (Confeitaria, São Paulo)
    "#
    conversationHistory ""
  }
  @@assert( {{ this.action != _ }} )
  @@assert( {{ this.action.actionType == "CUSTOMER_PAUSE" }} )
  @@assert( {{ this.action.actionStatus == "NEEDS_CONFIRMATION" }} )
  @@assert( {{ this.action.customerPause != _ }} )
}

test AgentPostGenerate {
  functions [AgentProcess]
  args {
    operatorName "Elenice"
    message "gera post pra Patricia"
    systemContext #"
      Clientes ativas: 3
        - Patricia (Salão de Beleza, Belo Horizonte)
        - Maria (Confeitaria, São Paulo)
        - Joana (Loja de Roupas, Rio de Janeiro)
      Posts gerados (total): 10
    "#
    conversationHistory ""
  }
  @@assert( {{ this.action != _ }} )
  @@assert( {{ this.action.actionType == "POST_GENERATE" }} )
  @@assert( {{ this.action.actionStatus == "NEEDS_CONFIRMATION" }} )
  @@assert( {{ this.action.postGenerate != _ }} )
}

test AgentPostListPending {
  functions [AgentProcess]
  args {
    operatorName "Bruna"
    message "quais posts tão pendentes?"
    systemContext #"
      Clientes ativas: 2
        - Patricia (Salão de Beleza, Belo Horizonte)
        - Maria (Confeitaria, São Paulo)
      Posts pendentes de revisão: 2
        - Patricia: "Hoje no salão foi dia de transformação..." (post_001)
        - Maria: "Bolo caseiro é sempre a melhor pedida..." (post_002)
      Posts gerados (total): 8
    "#
    conversationHistory ""
  }
  @@assert( {{ this.action != _ }} )
  @@assert( {{ this.action.actionType == "POST_LIST_PENDING" }} )
}
