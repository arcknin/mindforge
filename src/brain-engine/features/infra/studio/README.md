# Studios administrativos do Mindforge

Este pacote prepara quatro serviços administrativos separados: Studio e
Postgres Meta para mindforge (produção) e outro par para mindforge_develop. As
imagens são fixadas em supabase/studio:2026.09.07-sha-7996410 e
supabase/postgres-meta:v0.99.0. Os Studios publicam somente em loopback, nas
portas 18085 e 18086. Meta não publica porta. Os pares conectam ao serviço db
da rede externa infra_default; este Compose não define Postgres, Auth,
PostgREST, gateway ou Nginx em container.

O Studio main é administrativo por decisão do owner. Seus rótulos identificam
explicitamente produção e administração. POSTGRES_USER_READ_ONLY e
POSTGRES_USER_READ_WRITE recebem o mesmo role proprietário do alvo; isso não
oferece acesso read-only. SQL administrativo na main pode alterar dados ou
schema fora do catálogo de migrations. A ferramenta não autoriza agentes a
executarem SQL em main.

## Configuração privada

Copie .env.example para .env dentro desta pasta e preencha os campos com o role
e a senha proprietários já existentes em cada banco, além de chaves únicas para
cada ambiente. Não copie o .env da infraestrutura inteira. O arquivo local é
ignorado pelo Git.

    cp .env.example .env
    ./scripts/validate-config.sh .env

O validador lê somente as dez chaves declaradas, sem source ou eval. Ele rejeita
placeholders, alvos trocados, roles repetidos, credenciais ou chaves
reutilizadas, e não exibe valores. Ele não testa senha, conexão, role ou estado
do banco. A senha existente do banco é validada apenas por formato URI-safe,
exigido pela connection string do Studio; isso não é uma política de força e
não exige rotação. Preserve os valores existentes fornecidos pelo owner. Studio
e Meta de cada ambiente recebem a mesma chave criptográfica; main e develop
usam chaves diferentes. AUTH_JWT_SECRET é fornecido de forma explícita por
ambiente; nenhum token de exemplo é incluído.

## Portal e Nginx host

portal/index.html é a página estática para db.home.arpa. Ela liga aos Studios
NSR, Mindforge Main e Mindforge Develop por nomes distintos. Os links são
âncoras HTML nativas e funcionam pelo teclado. Main aparece como produção com
acesso administrativo.

nginx/hosts.conf.template é um fragmento para o Nginx existente no host. Ele
protege todos os quatro hosts com autenticação básica, mantém NSR no gateway
atual 127.0.0.1:18084 e envia main/develop aos ports locais 18085/18086. Meta
permanece privado. O caminho real do portal e o arquivo privado htpasswd são
parâmetros obrigatórios; nenhum caminho de VM é presumido.

Para preservar o roteamento atual do Kong, o upstream NSR recebe Host
db.home.arpa e X-Forwarded-Host com o host público nsr.db.home.arpa. Main e
develop encaminham o próprio hostname como Host.

    ./scripts/render-nginx.sh --portal-root /caminho/absoluto/fornecido/pelo/owner --auth-file /caminho/absoluto/fornecido/pelo/owner/studio.htpasswd --output /diretorio-existente/mindforge-studio.conf

O Nginx valida Basic Auth em cada host. Nas conexões ao Studio main e develop,
ele remove o header Authorization antes de encaminhar a requisição. Para NSR,
preserva o header e o encaminha ao Kong existente, reutilizando o mesmo login
administrativo; isso não é um segundo fator de autenticação.

O renderer só cria um arquivo novo no destino informado, recusa caminhos fora
do conjunto seguro de caracteres e não instala nem recarrega configuração. A
autenticação também continua necessária no gateway NSR já existente.

## Limites e validação

O recorte configura SQL Editor e introspecção de schema via Meta. O Studio
autossediado normalmente espera outros serviços, URLs e chaves do gateway
Supabase; por isso não há promessa de APIs, Auth, logs, funções, snippets ou de
todos os fluxos do Table Editor. A navegação de linhas e as abas necessárias
continuam pendentes de validação runtime na VM designada. O Studio administrativo
na main deve ser usado com essa limitação e com seu risco de SQL fora de
migrations.

Os testes locais usam credenciais sintéticas, verificam a configuração renderizada
e inspecionam os arquivos estáticos. Quando disponível, docker compose config ou
docker-compose config é usado somente para renderização, sem daemon, container
ou conexão ao banco. Se ambos os CLIs estiverem ausentes, o mapeamento YAML
estático continua sendo verificado. Os testes não validam credenciais reais,
permissões, rede, Nginx ativo, SQL, banco ou comportamento do Studio em runtime.

    bash tests/run.sh

Este pacote não instala serviços, não aplica migrations, não executa SQL e não
acessa VM. Provisionamento, deploy, mudanças no Nginx host e qualquer validação
runtime dependem da aprovação operacional apresentada separadamente.
