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

class ContentRole {
  name string
  description string
}

class Post {
  caption string
  hashtags string[]
  productionNote string
}

class JudgeResult {
  reasoning string
  verdict bool
}

class JudgeVariedadeResult {
  postMessages string[]
  reasoning string
  verdict bool
}

function JudgeNaturalidade(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador rigoroso de conteúdo para Instagram brasileiro.

    Já foi verificado que o texto usa português brasileiro informal. Sua tarefa é diferente: avaliar se o texto parece escrito por uma PESSOA REAL ou por uma IA imitando o estilo do Instagram.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade: {{ profile.city }}
    - Bairro: {{ profile.neighbourhood }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Sinais de conteúdo gerado por IA (reprove se encontrar 2 ou mais):
    - Emoji em quase toda frase, como decoração automática
    - Mesma estrutura nos posts: abertura animada → informação → pergunta → CTA
    - Informalidade forçada: acumula gente, bora, né, tá no mesmo parágrafo como checklist
    - Frases genéricas de preenchimento ("feito com muito carinho", "você merece o melhor", "a gente ama o que faz")
    - Tom uniformemente entusiasmado do início ao fim, sem variação de energia

    Sinais de conteúdo autêntico (aprove se predominarem):
    - Voz com personalidade própria, não "brasileiro genérico de Instagram"
    - Ritmo variado: mistura frases curtas e longas naturalmente
    - Emojis com intenção, não em toda frase
    - Pelo menos um momento que soa como opinião pessoal, não fórmula

    Exemplo de reprovação (deve receber verdict: false):
    "Gente, vocês não tão prontos! 😍🔥 Nosso smash é feito com muito amor e dedicação pra vocês! A gente ama o que faz e isso faz toda a diferença, né? 💕 Cada detalhe é pensado com carinho pra vocês! Bora experimentar? Chama no WhatsApp! 😘"
    Motivo: emoji em toda frase, "feito com amor e dedicação" + "pensado com carinho" (filler genérico), gente + né + bora empilhados no mesmo parágrafo, tom 100% entusiasmado sem pausa. Parece IA performando informalidade.

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se soa autêntico, false se parece gerado por IA.

    {{ ctx.output_format }}
  "#
}

function JudgeEspecificidade(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador rigoroso de conteúdo para Instagram brasileiro.

    Sua tarefa: o conteúdo tem detalhes que existem POR SI SÓS, ou todo detalhe inventado serve apenas para vender o produto/serviço?

    Perfil do negócio (dados que a IA recebeu):
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

    Teste decisivo: para cada detalhe inventado, tire a menção ao produto/serviço. O detalhe ainda tem valor para o leitor? Se não, é decoração de pitch.

    EXEMPLO 1 — verdict: false (dados do perfil reformatados)
    "Aqui no Setor Bueno a gente faz smash burger com nosso blend secreto 🍔 O molho da casa é preparado todo dia! Simples por R$28, duplo por R$38, combo completo por R$52. Bora provar?"
    Motivo: Setor Bueno = campo bairro, blend secreto = campo diferenciais, preços = campo serviços. Cada informação veio do perfil. Zero textura.

    EXEMPLO 2 — verdict: false (pitch embrulhado em história)
    "Era uma terça à noite e a Maria, dona de uma loja de roupas, tava exausta tentando escrever uma legenda pro Instagram. Ela não sabia o que postar. Foi aí que ela descobriu o AppX. O AppX olha pro conteúdo dela e escreve a legenda perfeita. Maria nunca mais travou."
    Motivo: tire o AppX e a história da Maria não tem razão de existir. A cena foi inventada apenas para montar o pitch. Isso não é especificidade, é narrativa instrumental.

    EXEMPLO 3 — verdict: true (detalhes com vida própria)
    "Sexta 18h e o cheiro da chapa já tá chamando a galera aqui no Bueno 🔥 Tem fila? Tem. Mas quem já mordeu o duplo sabe que vale cada minuto. Hoje o Rafa tá no comando da chapa, capricho dobrado 😂"
    Motivo: "sexta 18h" (cena temporal), "cheiro da chapa" (sensorial), "tem fila" (observação), "Rafa no comando" (personagem). Tire o produto e a cena ainda pinta um momento real. Os detalhes enriquecem por si sós.

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se os detalhes inventados valem por si sós, false se servem apenas ao pitch.

    {{ ctx.output_format }}
  "#
}

function JudgeAcionavel(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador rigoroso de conteúdo para Instagram brasileiro.

    Já foi verificado que o texto tem legenda, hashtags, CTA e nota de produção. Sua tarefa é diferente: avaliar a QUALIDADE desses elementos.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Avalie estes 4 critérios de qualidade:

    HASHTAGS: reprove se a maioria for genérica (#sucesso #empreendedorismo #foco #motivação #trabalho #pequenonegocio #apoielocal). Aprove se forem específicas do nicho, do bairro ou do negócio.

    NOTA DE PRODUÇÃO: reprove se for vaga ("tire uma foto do produto", "grave um vídeo mostrando o serviço"). Aprove se disser o que filmar, de que ângulo, em que momento.

    CTA: reprove se for o mesmo CTA copiado em todos os posts. Aprove se variar entre posts.

    FLUIDEZ: reprove se a legenda parecer seções coladas (texto → bloco de hashtags → CTA solto → nota solta). Aprove se a transição entre elementos for natural.

    Reprove se 2 ou mais critérios falharem.

    Exemplo de reprovação (elementos existem mas sem qualidade):
    Post 1: "... Chama no WhatsApp! #empreendedorismo #sucesso #foco #negociolocal #motivação. Nota de produção: tire uma foto bonita do produto."
    Post 2: "... Chama no WhatsApp! #empreendedorismo #sucesso #negociolocal #trabalho #foco. Nota de produção: grave um vídeo do serviço."
    Motivo: hashtags genéricas repetidas, mesmo CTA em ambos, notas de produção vagas. Tudo existe estruturalmente mas nada tem qualidade.

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se os elementos têm qualidade, false se são genéricos/repetitivos.

    {{ ctx.output_format }}
  "#
}

function JudgeVariedade(profile: BusinessProfile, content: string) -> JudgeVariedadeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador rigoroso de conteúdo para Instagram brasileiro.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    TAREFA em 2 passos:

    PASSO 1: Para cada post, escreva em UMA frase curta o que o leitor leva depois de ler. Coloque cada frase no campo postMessages. ATENÇÃO: se todas as frases mencionam o mesmo produto/serviço como solução, elas são a mesma mensagem. Escreva sem mencionar o nome do produto.

    PASSO 2: Compare as frases. Se são essencialmente a mesma ("use X", "experimente X", "X resolve"), reprove.

    Exemplo que REPROVA (verdict: false):
    Post 1 (história): "Era terça à noite e eu vi minha amiga Ana travada tentando escrever uma legenda. O AppX nasceu ali. Testa, o link tá na bio."
    Post 2 (números): "1.500 pessoas já baixaram o AppX. O pequeno negócio quer mostrar o trabalho sem gastar horas num post."
    Post 3 (citação): "Um dono de oficina me disse que Instagram virou trabalho não remunerado. O AppX resolve isso."
    postMessages: ["Existe solução pra quem trava na hora de postar", "Existe solução pra quem trava na hora de postar", "Existe solução pra quem trava na hora de postar"]
    Motivo: sem o nome do produto, as três mensagens são idênticas. Três estruturas, um só pitch.

    Exemplo que APROVA (verdict: true):
    Post 1: "Sexta 18h e o cheiro da chapa já tá chamando a galera 🔥 Tem fila? Tem. Mas quem já mordeu o duplo sabe que vale cada minuto."
    Post 2: "3 erros que todo mundo comete na hora de montar o hambúrguer em casa: carne fria na chapa, pão sem tostar, queijo errado."
    Post 3: "Pergunta honesta: alguém consegue comer smash sem fazer sujeira? Porque aqui a gente já desistiu 😂"
    postMessages: ["Vale esperar na fila", "Como fazer melhor em casa", "Hambúrguer é pra curtir sem frescura"]
    Motivo: cada post dá ao leitor algo diferente para pensar.

    Se houver apenas um post, coloque sua mensagem em postMessages e avalie se demonstra criatividade.

    {{ ctx.output_format }}
  "#
}

function JudgeEngajamento(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador rigoroso de conteúdo para Instagram brasileiro.

    Já foi verificado que o texto tem CTA e convite à interação. Sua tarefa é diferente: avaliar se o gancho é GENUINAMENTE interessante ou apenas uma fórmula de engajamento.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Público: {{ profile.targetAudience }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Reprove se:
    - O gancho usa fórmulas batidas: "Você sabia que...?", "Gente, prepara o coração!", "Vocês não estão prontos!", "[Número] coisas que..."
    - O engajamento depende de pedir ação genérica ("comenta aqui 👇", "marca um amigo") sem dar motivo real para fazê-lo
    - O conteúdo descreve o serviço de forma positiva mas sem ângulo surpreendente
    - Qualquer negócio do mesmo tipo poderia usar o mesmo gancho

    Aprove se:
    - A primeira linha cria curiosidade real (um dado específico, uma cena, uma contradição, uma história que começa no meio)
    - Há motivo real pra salvar, compartilhar ou comentar (aprendi algo novo, me identifiquei com a situação, quero mandar pra alguém específico)
    - O gancho só funciona para este negócio, não é genérico do nicho

    Exemplo de reprovação (fórmula de engajamento):
    "Você sabia que um bom corte pode mudar completamente seu visual? 😱 Pois é! Aqui no nosso espaço a gente transforma! Antes e depois que vai te deixar de queixo caído! Comenta aqui se você também ama! 👇 Marca aquele amigo que tá precisando! 😂"
    Motivo: "Você sabia" (gancho genérico), "mudar completamente seu visual" (óbvio, qualquer salão diria isso), "comenta + marca" sem dar motivo real. Fórmula, não engajamento.

    Primeiro explique seu raciocínio em 2-3 frases, depois dê o veredito.
    Veredito: true se o engajamento é genuíno, false se é fórmula.

    {{ ctx.output_format }}
  "#
}
