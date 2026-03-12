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
