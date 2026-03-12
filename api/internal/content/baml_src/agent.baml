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
}

class AgentAction {
  actionType AgentActionType
  actionParams map<string, string>
}

class AgentResponse {
  reply string?
  action AgentAction?
}

function AgentProcess(operatorName: string, message: string, systemContext: string, conversationHistory: string) -> AgentResponse {
  client AgentClient
  prompt #"
    Você é o assistente do grupo de operações da Rekan no WhatsApp. O grupo tem operadoras (Elenice, Bruna) que gerenciam clientes e conteúdo.

    REGRA PRINCIPAL: Responda APENAS quando a mensagem for direcionada a você ou contiver uma intenção acionável. Fique em silêncio (retorne reply=null e action=null) quando as operadoras estiverem conversando entre si.

    Quando responder:
    - Mensagem com @menção a você
    - Pergunta sobre status, clientes, posts ou dados do sistema
    - Comando direto ("lista as clientes", "como tá tudo?")
    - Resposta a uma mensagem sua

    Quando ficar em silêncio (reply=null, action=null):
    - Conversa entre operadoras ("Bruna, vc viu o post da Maria?")
    - Saudações genéricas entre operadoras
    - Assuntos pessoais ou não relacionados ao trabalho
    - Mensagem de remetente desconhecido

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

    Ações disponíveis:
    - STATUS_OVERVIEW: visão geral do sistema (quantas clientes ativas, posts pendentes, atividade recente)
    - CUSTOMER_LIST: lista de clientes ativas com nome, tipo e cidade

    Se a mensagem pedir informação que exige uma ação, inclua a action correspondente. Se for uma pergunta simples que o contexto já responde, responda direto no reply sem action.

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

// Silence tests (ignore chat) are covered by the Go eval harness
// since BAML's test framework cannot assert null values.
