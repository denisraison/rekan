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

    Já foi verificado que o texto menciona o nome do negócio e a localização. Sua tarefa é diferente: avaliar se o conteúdo vai ALÉM dos dados do perfil, ou se é apenas os dados reformatados em tom de Instagram.

    Perfil do negócio (estes são os dados que a IA recebeu para gerar o conteúdo):
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

    Compare o conteúdo com os dados do perfil acima. Cada fato no conteúdo veio de um campo do perfil, ou existe pelo menos um detalhe inventado criativamente?

    Reprove se:
    - Toda informação no conteúdo é rastreável a um campo do perfil acima
    - O conteúdo é o perfil reescrito em tom casual, sem adicionar nada
    - Não há nenhuma cena, momento, anedota ou detalhe além dos dados
    - O conteúdo inventa cenas ou personagens, mas toda cena existe para montar um pitch: apresentar um problema e explicar como o produto/serviço resolve. Teste: se tirar a menção ao produto/serviço, a cena perde o propósito? Se sim, é pitch com moldura narrativa, não especificidade genuína

    Aprove se:
    - Pelo menos um detalhe concreto que não está no perfil (uma cena do dia a dia, uma observação sobre o bairro, um hábito dos clientes, um momento específico)
    - O conteúdo conta uma micro-história ou pinta um cenário, não apenas lista atributos
    - Os detalhes inventados enriquecem a experiência do leitor por si sós, independente de vender algo

    Exemplo de reprovação (dados do perfil reformatados):
    "Aqui no Setor Bueno a gente faz smash burger com nosso blend secreto 🍔 O molho da casa é preparado todo dia! Simples por R$28, duplo por R$38, combo completo por R$52. Bora provar?"
    Motivo: Setor Bueno = campo bairro, blend secreto = campo diferenciais, molho da casa = campo diferenciais, preços = campo serviços. Cada informação veio do perfil. Zero textura, zero cenas.

    Exemplo de reprovação (pitch disfarçado de história):
    "Era uma terça à noite e a Maria, dona de uma loja de roupas, tava exausta tentando escrever uma legenda pro Instagram. Ela não sabia o que postar. Foi aí que ela descobriu o AppX. O AppX olha pro conteúdo dela e escreve a legenda perfeita. Maria nunca mais travou."
    Motivo: a cena da Maria existe apenas para apresentar o problema que o AppX resolve. Tire o AppX e a história não tem razão de existir. É um pitch embrulhado em narrativa.

    Exemplo de aprovação:
    "Sexta 18h e o cheiro da chapa já tá chamando a galera aqui no Bueno 🔥 Tem fila? Tem. Mas quem já mordeu o duplo sabe que vale cada minuto. Hoje o Rafa tá no comando da chapa, capricho dobrado 😂"
    Motivo: "sexta 18h" (cena temporal), "cheiro da chapa" (sensorial), "tem fila" (observação real), "Rafa no comando" (personagem). Detalhes que dão vida e não estão nos dados do perfil. A cena tem valor mesmo sem vender nada.

    No seu raciocínio, responda obrigatoriamente:
    1. Que detalhes no conteúdo NÃO estão no perfil?
    2. Para cada detalhe inventado: se tirar a menção ao produto/serviço, o detalhe ainda tem valor para o leitor?
    Depois dê o veredito.
    Veredito: true se vai além dos dados COM detalhes que valem por si sós, false se é perfil reformatado OU pitch disfarçado de história.

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

function JudgeVariedade(profile: BusinessProfile, content: string) -> JudgeResult {
  client JudgeClient
  prompt #"
    Você é um avaliador rigoroso de conteúdo para Instagram brasileiro.

    Analise se os posts são genuinamente diferentes ou se são o mesmo template com temas trocados.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}

    Conteúdo a avaliar:
    ---
    {{ content }}
    ---

    Não basta os posts terem temas diferentes. Eles precisam ter:
    - Estruturas diferentes (um narrativo, outro lista, outro pergunta/resposta)
    - Aberturas com ritmos diferentes (não todos com emoji + frase de efeito)
    - Tons diferentes (um animado, outro informativo, outro íntimo)
    - CTAs diferentes (não todos terminam com "chama no WhatsApp")

    Teste do template: pegue a estrutura de um post e troque o tema. Se funcionar sem alterar nada, os posts são o mesmo template.

    Reprove se:
    - Todos seguem o padrão: [abertura animada] → [info sobre serviço] → [pergunta ao público] → [CTA] → [hashtags]
    - As aberturas têm o mesmo ritmo/energia mesmo com palavras diferentes
    - Troque os temas entre posts e a estrutura ainda funciona
    - Os posts têm estruturas diferentes (narrativo, lista, pergunta) mas o leitor leva a MESMA mensagem de todos: "use/experimente este produto/serviço." Se cada post é apenas uma razão diferente para usar a mesma coisa, é pitch repetido com estrutura variada, não variedade real

    Aprove se:
    - Pelo menos 2 posts têm estruturas claramente diferentes (ex: um é lista numerada, outro é narrativa, outro começa com pergunta provocativa)
    - A energia varia entre posts (não são todos no mesmo nível de entusiasmo)
    - Os posts dão ao leitor algo DIFERENTE para pensar ou fazer (não apenas razões diferentes para usar o mesmo produto/serviço)

    Exemplo de reprovação (template com temas trocados):
    Post 1: "Gente, vocês sabiam que [corte X] é sucesso? 😍 Aqui no salão a gente faz com carinho! Qual é o seu favorito? Comenta! 💇‍♀️ Chama no WhatsApp!"
    Post 2: "Gente, olha que incrível essa [coloração Y]! 🔥 Aqui no salão a gente arrasa! Já experimentou? Conta pra gente! 💕 Chama no WhatsApp!"
    Post 3: "Gente, prepara o coração pra [tratamento Z]! ✨ Aqui no salão a gente cuida de tudo! O que vocês acham? Comenta aqui! 💆‍♀️ Chama no WhatsApp!"
    Motivo: todos abrem com "Gente" + emoji, mesma estrutura (fato → serviço → pergunta → CTA), mesmo tom, mesmo CTA. Temas mudaram, template não.

    Exemplo de reprovação (estruturas diferentes, monotonia temática):
    Post 1 (narrativo): "A Carla estava exausta tentando postar todo dia. Então descobriu o AppX e nunca mais travou..."
    Post 2 (dados): "2.000 pessoas já usam o AppX. Ninguém aguenta criar conteúdo sozinho..."
    Post 3 (depoimento): "O João me mandou mensagem: 'consigo postar sem sofrer, graças ao AppX'..."
    Motivo: narrativo, dados, depoimento — três estruturas. Mas o tema é um só: "criar conteúdo é difícil, AppX resolve." Troque o produto e os posts funcionam igual.

    Se houver apenas um post, avalie se demonstra criatividade e não segue um template óbvio.

    No seu raciocínio, responda obrigatoriamente estas 3 perguntas:
    1. Qual é a estrutura de cada post? (narrativa, lista, pergunta, dados, etc.)
    2. Qual é a CONCLUSÃO que o leitor leva de cada post? (O que ele deve fazer ou pensar depois de ler? Resuma em uma frase curta.)
    3. As conclusões são genuinamente diferentes ou todas dizem "use/experimente este produto/serviço"?
    Se a conclusão de todos os posts é a mesma ação ("use X", "teste X", "conheça X"), reprove por monotonia temática mesmo que as estruturas variem.
    Veredito: true se há variedade real (estrutural E temática), false se falta uma das duas.

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
