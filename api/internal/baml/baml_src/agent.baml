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

class AgentAction {
  actionType AgentActionType
  actionStatus AgentActionStatus
  actionParams map<string, string>
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

    Ações disponíveis:
    - STATUS_OVERVIEW: visão geral do sistema (quantas clientes ativas, posts pendentes, atividade recente)
    - CUSTOMER_LIST: lista de clientes ativas com nome, tipo e cidade
    - CUSTOMER_CREATE: cadastrar nova cliente. Campos: name, type, city, instagram, frequency. Se faltar algum campo obrigatório (name, type, city), peça na resposta.
    - CUSTOMER_UPDATE: alterar dados de uma cliente existente. Campos: name (identificação), e os campos a alterar.
    - CUSTOMER_PAUSE: pausar uma cliente. Campos: name, reason (opcional).
    - CUSTOMER_INFO: detalhes de uma cliente específica. Campos: name.
    - POST_GENERATE: gerar novos posts para uma cliente. Campos: name (obrigatório). Leva alguns segundos.
    - POST_LIST_PENDING: lista posts pendentes de revisão. Campos: name (opcional, filtra por cliente).
    - POST_APPROVE: aprovar um post. Campos: post_id, name (cliente).
    - POST_REJECT: rejeitar um post com feedback. Campos: post_id, name (cliente), feedback.

    Regras de ações:
    - Ações que modificam dados (CREATE, UPDATE, PAUSE, POST_GENERATE, POST_APPROVE, POST_REJECT) SEMPRE usam actionStatus "NEEDS_CONFIRMATION". Liste os campos extraídos no reply e peça "Confirma?"
    - Ações de leitura (STATUS_OVERVIEW, CUSTOMER_LIST, CUSTOMER_INFO, POST_LIST_PENDING) usam actionStatus "EXECUTE".
    - Se a mensagem pedir informação que exige uma ação, inclua a action correspondente.
    - Se for uma pergunta simples que o contexto já responde, responda direto no reply sem action.

    Regras de clientes:
    - Se uma cliente com mesmo nome já existe no contexto, avise e não crie duplicata.
    - Se houver ambiguidade de nome (duas Marias), peça para especificar.
    - Interprete abreviações comuns: "BH" = "Belo Horizonte", "SP" = "São Paulo", "RJ" = "Rio de Janeiro", "4x" = frequência 4 por mês.
    - Use o histórico da conversa para inferir contexto. Se a operadora disse "muda pra 5x" depois de falar da Patricia, a mudança é na Patricia.

    Regras de posts:
    - POST_LIST_PENDING mostra posts que ainda não foram revisados. Se a operadora não especificar cliente, mostre todos.
    - POST_APPROVE e POST_REJECT usam o post_id do contexto. Se a operadora diz "aprova" sem especificar, use o post em foco no histórico.
    - POST_REJECT deve incluir o feedback da operadora no campo "feedback".

    Regras de mídia:
    - Mensagens com "[Imagem: ...]" contêm descrição de uma imagem enviada. Use a descrição para entender o contexto.
    - Se a imagem é um cartão de visita, extraia nome, tipo de negócio, cidade, instagram e telefone.
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
    message "cria a Ana, manicure em Goiânia, instagram @anamanicure, 4x por mês"
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
