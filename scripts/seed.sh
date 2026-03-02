#!/usr/bin/env bash
# Reset dev environment and seed with test data.
# Usage: make seed

set -euo pipefail

PB_URL="http://localhost:8090"
PB_DIR="api/pb_data"

ADMIN_EMAIL="${SEED_ADMIN_EMAIL:-admin@rekan.local}"
ADMIN_PASSWORD="${SEED_ADMIN_PASSWORD:-admin1234567}"
USER_EMAIL="${SEED_USER_EMAIL:-operador@rekan.local}"
USER_PASSWORD="${SEED_USER_PASSWORD:-senha1234567}"

command -v jq >/dev/null 2>&1 || { echo "jq is required"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl is required"; exit 1; }

echo "=== Resetting dev environment ==="

# Kill any process using port 8090
ss -lptn 'sport = :8090' | grep -oP '(?<=pid=)\d+' | sort -u | xargs kill -9 2>/dev/null || true
sleep 0.3

# Remove existing data
echo "Removing $PB_DIR..."
rm -rf "$PB_DIR"

# Bootstrap DB + superadmin (runs migrations, no server needed)
echo "Creating superadmin $ADMIN_EMAIL..."
(cd api && go run . superuser upsert "$ADMIN_EMAIL" "$ADMIN_PASSWORD") 2>&1 | tail -1

# Start PocketBase in background
echo "Starting PocketBase..."
(cd api && go run . serve --http=0.0.0.0:8090 >/tmp/pocketbase-seed.log 2>&1) &
trap "ss -lptn 'sport = :8090' | grep -oP '(?<=pid=)\d+' | sort -u | xargs kill -9 2>/dev/null || true" EXIT

# Wait for PocketBase
echo "Waiting for PocketBase on :8090..."
for _ in $(seq 1 60); do
  nc -z localhost 8090 2>/dev/null && break
  sleep 0.5
done
nc -z localhost 8090 2>/dev/null || { echo "PocketBase failed to start"; exit 1; }

# Authenticate as superadmin
TOKEN=$(curl -sf -X POST "$PB_URL/api/collections/_superusers/auth-with-password" \
  -H "Content-Type: application/json" \
  -d "{\"identity\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" \
  | jq -r '.token')

auth=(-H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN")

# Create operator user
echo "Creating user $USER_EMAIL..."
curl -sf -X POST "$PB_URL/api/collections/users/records" \
  "${auth[@]}" \
  -d "{
    \"email\": \"$USER_EMAIL\",
    \"password\": \"$USER_PASSWORD\",
    \"passwordConfirm\": \"$USER_PASSWORD\",
    \"emailVisibility\": true
  }" | jq -r '"  created user: " + .email'

# Helper: create business, print name to stderr, return ID
create_business() {
  local data="$1"
  local name id
  name=$(echo "$data" | jq -r '.name')
  id=$(curl -sf -X POST "$PB_URL/api/collections/businesses/records" \
    "${auth[@]}" \
    -d "$data" | jq -r '.id')
  echo "  created business: $name ($id)" >&2
  echo "$id"
}

# Helper: create a text message
create_message() {
  local business_id="$1" phone="$2" direction="$3" content="$4" wa_timestamp="$5"
  curl -sf -X POST "$PB_URL/api/collections/messages/records" \
    "${auth[@]}" \
    -d "{
      \"business\": \"$business_id\",
      \"phone\": \"$phone\",
      \"type\": \"text\",
      \"content\": $(echo "$content" | jq -R .),
      \"direction\": \"$direction\",
      \"wa_timestamp\": \"$wa_timestamp\"
    }" >/dev/null
}

# Helper: create a post
create_post() {
  local business_id="$1" caption="$2" source="${3:-operator}"
  curl -sf -X POST "$PB_URL/api/collections/posts/records" \
    "${auth[@]}" \
    -d "{
      \"business\": \"$business_id\",
      \"caption\": $(echo "$caption" | jq -R .),
      \"hashtags\": [],
      \"source\": \"$source\"
    }" >/dev/null
}

# Helper: create a scheduled message
create_scheduled() {
  local business_id="$1" text="$2" scheduled_for="$3" approved="${4:-false}" dismissed="${5:-false}"
  curl -sf -X POST "$PB_URL/api/collections/scheduled_messages/records" \
    "${auth[@]}" \
    -d "{
      \"business\": \"$business_id\",
      \"text\": $(echo "$text" | jq -R .),
      \"scheduled_for\": \"$scheduled_for\",
      \"approved\": $approved,
      \"dismissed\": $dismissed
    }" >/dev/null
}

echo ""
echo "=== Creating businesses ==="

# Today: 2026-03-02
# Health indicators: green < 5 days, yellow 5-9 days, red >= 10 days

# 1. Elenice — Confeitaria — last msg 2 days ago (green)
ELENICE=$(create_business '{
  "name": "Confeitaria da Elenice",
  "type": "Confeitaria",
  "city": "São Paulo",
  "state": "SP",
  "phone": "5511988887777",
  "services": [{"name":"Bolos personalizados","price_brl":0},{"name":"Docinhos","price_brl":0},{"name":"Cupcakes","price_brl":0},{"name":"Tortas","price_brl":0}],
  "target_audience": "Famílias que buscam bolos artesanais para festas",
  "brand_vibe": "Acolhedor, artesanal, delicioso",
  "quirks": "Usa somente ingredientes naturais sem corantes artificiais",
  "client_name": "Elenice Silva",
  "client_email": "elenice@example.com",
  "invite_status": "active",
  "tier": "parceiro",
  "commitment": "mensal",
  "onboarding_step": 4
}')

# 2. João — Personal Trainer — last msg 3 days ago (green)
JOAO=$(create_business '{
  "name": "Academia do João",
  "type": "Personal Trainer",
  "city": "Campinas",
  "state": "SP",
  "phone": "5519977776666",
  "services": [{"name":"Treino funcional","price_brl":0},{"name":"Musculação","price_brl":0},{"name":"HIIT","price_brl":0},{"name":"Avaliação física","price_brl":0}],
  "target_audience": "Adultos que querem emagrecer e ganhar massa muscular",
  "brand_vibe": "Motivador, resultado, energia",
  "quirks": "Atende em domicílio e online",
  "client_name": "João Santos",
  "client_email": "joao@example.com",
  "invite_status": "active",
  "tier": "parceiro",
  "commitment": "mensal",
  "onboarding_step": 4
}')

# 3. Marina — Salão de Beleza — last msg 9 days ago (yellow)
MARINA=$(create_business '{
  "name": "Salão da Marina",
  "type": "Salão de Beleza",
  "city": "Rio de Janeiro",
  "state": "RJ",
  "phone": "5521966665555",
  "services": [{"name":"Corte feminino","price_brl":0},{"name":"Coloração","price_brl":0},{"name":"Escova progressiva","price_brl":0},{"name":"Manicure","price_brl":0}],
  "target_audience": "Mulheres de 25 a 50 anos que valorizam cuidado pessoal",
  "brand_vibe": "Elegante, feminino, acolhedor",
  "quirks": "Especialista em coloração platinada sem dano",
  "client_name": "Marina Costa",
  "client_email": "marina@example.com",
  "invite_status": "active",
  "tier": "profissional",
  "commitment": "mensal",
  "onboarding_step": 4
}')

# 4. Carlos — Barbearia — last msg 14 days ago (red), charge pending
CARLOS=$(create_business '{
  "name": "Barbearia do Carlos",
  "type": "Barbearia",
  "city": "São Paulo",
  "state": "SP",
  "phone": "5511955554444",
  "services": [{"name":"Corte masculino","price_brl":0},{"name":"Barba","price_brl":0},{"name":"Sobrancelha","price_brl":0},{"name":"Hidratação capilar","price_brl":0}],
  "target_audience": "Homens de 18 a 45 anos que se preocupam com a aparência",
  "brand_vibe": "Estilo, tradição, modernidade",
  "quirks": "Ambiente descontraído com cerveja artesanal inclusa no atendimento",
  "client_name": "Carlos Oliveira",
  "client_email": "carlos@example.com",
  "invite_status": "active",
  "tier": "basico",
  "commitment": "mensal",
  "onboarding_step": 4,
  "charge_pending": true
}')

# 5. Fernanda — Nail Designer — last msg 22 days ago (very red)
FERNANDA=$(create_business '{
  "name": "Nail Art da Fernanda",
  "type": "Nail Designer",
  "city": "Belo Horizonte",
  "state": "MG",
  "phone": "5531944443333",
  "services": [{"name":"Gel","price_brl":0},{"name":"Fibra de vidro","price_brl":0},{"name":"Nail art","price_brl":0},{"name":"Encapsulado","price_brl":0}],
  "target_audience": "Mulheres que gostam de unhas elaboradas e exclusivas",
  "brand_vibe": "Criativo, delicado, exclusivo",
  "quirks": "Cada design é único e feito à mão, nunca se repete",
  "client_name": "Fernanda Lima",
  "client_email": "fernanda@example.com",
  "invite_status": "active",
  "tier": "parceiro",
  "commitment": "trimestral",
  "onboarding_step": 4
}')

# 6. Leo — Hamburgueria — invited (not yet active, no messages)
LEO=$(create_business '{
  "name": "Hamburgueria do Léo",
  "type": "Hamburgueria",
  "city": "São Paulo",
  "state": "SP",
  "phone": "5511933332222",
  "services": [{"name":"Hambúrgueres artesanais","price_brl":0},{"name":"Batata frita","price_brl":0},{"name":"Milk shake","price_brl":0}],
  "target_audience": "Jovens e famílias que buscam hambúrgueres gourmet",
  "brand_vibe": "Jovem, descontraído, saboroso",
  "quirks": "Carne brangus 100% nacional, sem conservantes",
  "client_name": "Leonardo Ferreira",
  "client_email": "leo@example.com",
  "invite_status": "invited",
  "tier": "basico",
  "commitment": "mensal",
  "onboarding_step": 1
}')

echo ""
echo "=== Creating messages ==="

# --- Elenice (last incoming: 2026-02-28, 2 days ago) ---
create_message "$ELENICE" "5511988887777" "incoming" \
  "Oi! Posso fazer um pedido de bolo de aniversário para o final de semana?" \
  "2026-02-20T10:15:00Z"
create_message "$ELENICE" "5511988887777" "outgoing" \
  "Claro, Elenice! Me conta: pra quantas pessoas e qual o tema?" \
  "2026-02-20T10:18:00Z"
create_message "$ELENICE" "5511988887777" "incoming" \
  "Pra 20 pessoas, tema unicórnio. Preciso pra sábado" \
  "2026-02-20T10:22:00Z"
create_message "$ELENICE" "5511988887777" "outgoing" \
  "Maravilha! Vou te mandar o orçamento ainda hoje" \
  "2026-02-20T10:25:00Z"
create_message "$ELENICE" "5511988887777" "incoming" \
  "Perfeito! Aguardo" \
  "2026-02-21T09:00:00Z"
create_message "$ELENICE" "5511988887777" "outgoing" \
  "Elenice, o bolo ficou pronto ontem. Você tem foto do resultado? Quero postar no Instagram!" \
  "2026-02-25T14:00:00Z"
create_message "$ELENICE" "5511988887777" "incoming" \
  "Oi! Vou tirar foto ainda hoje e te mando" \
  "2026-02-26T11:30:00Z"
create_message "$ELENICE" "5511988887777" "outgoing" \
  "Otimo! Enquanto isso já vou preparando a legenda" \
  "2026-02-26T11:35:00Z"
create_message "$ELENICE" "5511988887777" "incoming" \
  "Aqui a foto do bolo que entreguei ontem! Ficou lindo demais" \
  "2026-02-28T08:45:00Z"
create_message "$ELENICE" "5511988887777" "outgoing" \
  "Ficou INCRÍVEL! Criando o post agora, te aviso quando subir" \
  "2026-02-28T09:10:00Z"

# --- João (last incoming: 2026-02-27, 3 days ago) ---
create_message "$JOAO" "5519977776666" "incoming" \
  "Oi! Quero montar um post sobre os resultados dos meus alunos essa semana" \
  "2026-02-22T08:00:00Z"
create_message "$JOAO" "5519977776666" "outgoing" \
  "Boa ideia João! Tem fotos de antes e depois pra usar?" \
  "2026-02-22T08:10:00Z"
create_message "$JOAO" "5519977776666" "incoming" \
  "Tenho sim, 3 alunos que me liberaram pra divulgar" \
  "2026-02-22T08:15:00Z"
create_message "$JOAO" "5519977776666" "outgoing" \
  "Perfeito. Manda as fotos que montamos algo impactante" \
  "2026-02-22T08:20:00Z"
create_message "$JOAO" "5519977776666" "incoming" \
  "Mandei as fotos no drive, o link é drive.google.com/xyz" \
  "2026-02-25T16:00:00Z"
create_message "$JOAO" "5519977776666" "outgoing" \
  "Recebi! Vou preparar 3 opções de legenda pra você escolher" \
  "2026-02-25T16:30:00Z"
create_message "$JOAO" "5519977776666" "incoming" \
  "Aquela segunda opção ficou ótima. Posso postar hoje?" \
  "2026-02-27T07:30:00Z"

# --- Marina (last incoming: 2026-02-21, 9 days ago) ---
create_message "$MARINA" "5521966665555" "incoming" \
  "Oi! Quero fazer um post sobre coloração platinada, fiz um trabalho lindo essa semana" \
  "2026-02-10T10:00:00Z"
create_message "$MARINA" "5521966665555" "outgoing" \
  "Oi Marina! Que legal. Tem foto do resultado com autorização da cliente?" \
  "2026-02-10T10:15:00Z"
create_message "$MARINA" "5521966665555" "incoming" \
  "Tenho sim! Ela adorou e pediu pra aparecer" \
  "2026-02-10T10:20:00Z"
create_message "$MARINA" "5521966665555" "outgoing" \
  "Manda! Vou criar algo que destaque a técnica e o resultado" \
  "2026-02-10T10:25:00Z"
create_message "$MARINA" "5521966665555" "incoming" \
  "Mandei 5 fotos. Qual você acha melhor pra usar?" \
  "2026-02-12T15:00:00Z"
create_message "$MARINA" "5521966665555" "outgoing" \
  "A terceira ficou incrível! Post criado, vai sair amanhã de manhã" \
  "2026-02-12T15:30:00Z"
create_message "$MARINA" "5521966665555" "incoming" \
  "Adorei o post! Ganhei 3 clientes novas por causa disso essa semana" \
  "2026-02-21T09:00:00Z"

# --- Carlos (last incoming: 2026-02-16, 14 days ago) ---
create_message "$CARLOS" "5511955554444" "incoming" \
  "Oi, quero começar a postar mais regularmente. Por onde começo?" \
  "2026-01-28T11:00:00Z"
create_message "$CARLOS" "5511955554444" "outgoing" \
  "Oi Carlos! Vamos começar com uma apresentação da barbearia. Tem fotos do espaço e dos serviços?" \
  "2026-01-28T11:20:00Z"
create_message "$CARLOS" "5511955554444" "incoming" \
  "Tenho sim, vou te mandar agora" \
  "2026-01-28T11:25:00Z"
create_message "$CARLOS" "5511955554444" "outgoing" \
  "Post pronto! Ficou muito bom. Agendado pra amanhã de manhã 9h" \
  "2026-02-05T14:00:00Z"
create_message "$CARLOS" "5511955554444" "incoming" \
  "Post saiu bem demais! Tive bastante engajamento e dois agendamentos novos. Valeu!" \
  "2026-02-10T10:00:00Z"
create_message "$CARLOS" "5511955554444" "outgoing" \
  "Que ótimo Carlos! Que tal um post de antes e depois de barba agora pra manter o ritmo?" \
  "2026-02-12T09:00:00Z"
create_message "$CARLOS" "5511955554444" "incoming" \
  "Boa ideia. Semana que vem te mando o material" \
  "2026-02-16T16:00:00Z"

# --- Fernanda (last incoming: 2026-02-08, 22 days ago) ---
create_message "$FERNANDA" "5531944443333" "incoming" \
  "Oi! Comecei a oferecer encapsulado e tá tendo muita procura. Quero divulgar" \
  "2026-01-05T09:00:00Z"
create_message "$FERNANDA" "5531944443333" "outgoing" \
  "Que ótimo Fernanda! Manda foto do trabalho que crio o post" \
  "2026-01-05T09:30:00Z"
create_message "$FERNANDA" "5531944443333" "incoming" \
  "Aqui estão as fotos, ficaram lindas" \
  "2026-01-07T14:00:00Z"
create_message "$FERNANDA" "5531944443333" "outgoing" \
  "Post criado! Ficou maravilhoso. Quanto você cobra pelo encapsulado?" \
  "2026-01-07T16:00:00Z"
create_message "$FERNANDA" "5531944443333" "incoming" \
  "R$ 180 a mão completa, R$ 100 a metade" \
  "2026-01-08T10:00:00Z"
create_message "$FERNANDA" "5531944443333" "outgoing" \
  "Perfeito, incluí no post. Alguma novidade essa semana pra postar?" \
  "2026-02-06T09:00:00Z"
create_message "$FERNANDA" "5531944443333" "incoming" \
  "Oi! Tive alguns perrengues com fornecedor mas to resolvendo. Semana que vem volto com força" \
  "2026-02-08T10:30:00Z"

echo ""
echo "=== Creating posts ==="

# Elenice — 3 posts this month (March 2026)
create_post "$ELENICE" \
  "Bolo de unicórnio pra iluminar a festa da pequena Ana! Cada detalhe feito com amor e ingredientes 100% naturais. Encomende o seu pelo WhatsApp!" \
  "operator"
create_post "$ELENICE" \
  "Nossos docinhos artesanais são o toque especial que faltava na sua festa. Brigadeiros, beijinhos e muito mais — sem corantes artificiais!" \
  "proactive"
create_post "$ELENICE" \
  "Cupcakes personalizados para qualquer ocasião! Escolha o tema, escolha o sabor, e a gente faz acontecer com muito carinho." \
  "operator"

# João — 2 posts this month
create_post "$JOAO" \
  "Resultado real de 3 meses de treino com acompanhamento personalizado! Meu aluno perdeu 12kg e ganhou disposição pra vida toda. Quer chegar lá? Me chama no WhatsApp!" \
  "operator"
create_post "$JOAO" \
  "Avaliação física GRATUITA em março! Venha descobrir seu potencial e montar um plano de treino personalizado pra você. Vagas limitadas!" \
  "proactive"

# Marina — 4 posts this month (most active)
create_post "$MARINA" \
  "Transformação total! Coloração platinada executada com técnica exclusiva e cuidado máximo com a fibra capilar. Agende sua avaliação gratuita!" \
  "operator"
create_post "$MARINA" \
  "Antes e depois que fala por si. Mechas feitas com carinho e profissionalismo há mais de 10 anos. Agende pelo WhatsApp!" \
  "operator"
create_post "$MARINA" \
  "Escova progressiva + corte + hidratação por apenas R$ 280 neste mês! Vagas limitadas, agende já pelo WhatsApp." \
  "proactive"
create_post "$MARINA" \
  "Dia das Mulheres chegando! Presente especial: 20% de desconto em todos os serviços no dia 8 de março. Compartilhe com uma amiga!" \
  "proactive"

# Carlos — 1 post this month (less active)
create_post "$CARLOS" \
  "Antes e depois: barba degradê estilo lenhador. Que tal renovar o visual essa semana? Agende pelo WhatsApp!" \
  "operator"

# Fernanda — 2 posts (also this month, created by seed today)
create_post "$FERNANDA" \
  "Encapsulado artesanal exclusivo! Cada design é único e feito à mão com muito amor e atenção aos detalhes. R$ 180 a mão completa." \
  "operator"
create_post "$FERNANDA" \
  "Nail art especial! Venha renovar suas unhas com um design único, feito especialmente pra você." \
  "operator"

echo ""
echo "=== Creating scheduled messages ==="

# Pending — awaiting operator approval
create_scheduled "$CARLOS" \
  "Carlos, faz 14 dias que a gente não posta. Que tal um antes e depois de barba essa semana? Seus seguidores adoram esse tipo de conteúdo!" \
  "2026-03-03T10:00:00Z" "false" "false"

create_scheduled "$FERNANDA" \
  "Fernanda, vi que faz um tempo! Quer retomar? Posso te mandar ideias de conteúdo pra essa semana." \
  "2026-03-03T11:00:00Z" "false" "false"

create_scheduled "$MARINA" \
  "Marina, Dia da Mulher vem aí! Que tal um post com promo especial pro dia 8 de março?" \
  "2026-03-04T09:00:00Z" "false" "false"

# Approved — operator already confirmed
create_scheduled "$JOAO" \
  "Joao, verao chegando! Momento perfeito pra postar sobre preparacao fisica e resultados dos alunos." \
  "2026-03-05T08:00:00Z" "true" "false"

# Dismissed — operator skipped this one
create_scheduled "$ELENICE" \
  "Elenice, Pascoa chegando! Vamos montar os posts das encomendas de ovo de Pascoa?" \
  "2026-03-10T09:00:00Z" "false" "true"

echo ""
echo "=== Seed complete ==="
echo "  Admin:    $ADMIN_EMAIL / $ADMIN_PASSWORD"
echo "  Operador: $USER_EMAIL / $USER_PASSWORD"
echo ""
echo "Clients seeded (sorted by urgency):"
echo "  Leo      (Hamburgueria)    invited       — no messages"
echo "  Fernanda (Nail Designer)   active        — last msg 22 days ago (red)"
echo "  Carlos   (Barbearia)       active        — last msg 14 days ago (red, charge pending)"
echo "  Marina   (Salão de Beleza) active        — last msg  9 days ago (yellow)"
echo "  João     (Personal Trainer) active       — last msg  3 days ago (green)"
echo "  Elenice  (Confeitaria)     active        — last msg  2 days ago (green)"
echo ""
echo "Run: make dev"
