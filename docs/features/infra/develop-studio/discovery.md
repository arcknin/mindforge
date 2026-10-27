# Discovery: develop persistente e portal de Studios

## Pedido e limite

Em 08/10/2026 o owner escolheu Studios separados com entrada comum em
`db.home.arpa` e pediu planejamento e execução com agentes. Confirmou TDD `yes`
e acesso administrativo para o Studio de main, como no NSR. Preparação e
implementação local estão autorizadas por esse pedido. A aplicação concreta
em VM, bancos, serviços e resolução de nomes deve ser apresentada antes de
executar mudanças externas; a senha SSH está autorizada para a investigação.

## Fontes atuais

- [Retomada anterior](../../project/continuation/discovery.md).
- [Brain Core r4](../../brain/brain-core/brain-core-contract.md) e
  [infra r6](../../../stack/back-end/features/infra/contract.md).
- `src/brain-engine/features/infra/`: operador, configuração, migrations,
  adapters e testes. `src/brain-engine/features/brain/`: Engine/Store e SQL.
- Git em `feat/brain-core-foundation`, HEAD `0204ba4`; dirty work anterior
  preservado. Nenhum commit/push autorizado por esta unidade.
- Seis fontes obrigatórias de AGENTS relidas pelo principal, em ordem.

## Evidência verificada

- O Studio/Meta do NSR usa o PostgreSQL separado do NSR, database `postgres`.
  Mindforge usa `infra-db-1`; Auth main é `infra-auth-1`.
- Studio instalado `2026.09.07-sha-7996410`, Meta `v0.99.0`. Cada Studio padrão
  possui um projeto e um destino Meta; não há seletor multi-project nativo.
- Mac do owner possui entrada `/etc/hosts` para `db.home.arpa` apontando à VM.
  O resolver da VM não resolve esse nome. Os três novos subhosts não possuem
  entradas observadas no Mac. A etapa de roteamento depende de confirmar o
  serviço HTTP atual e planejar explicitamente a resolução desses nomes.
- `provision-develop` necessita login de administração de cluster com
  superuser; nenhum privilégio deve ser criado automaticamente para contornar
  credencial ou configuração insuficiente.
- O operador invoca `psql` no host, mas usa hoje `POSTGRES_HOST=db`, alias
  interno Compose. É necessário distinguir destino cliente do host e URL
  interna do Auth antes da entrega.
- `up.sh`, quando os dois bancos existem, migra infrastructure de develop e
  main e executa lifecycle de ambos Auths. Não é um comando develop-only.
  A execução desta unidade deve selecionar develop e serviços específicos,
  sem usar esse caminho global para mudar main implicitamente.
- SQL 0006 usa payload Brain Core e verificação em uma transação, gravando
  o ledger único da infra. Sucesso comprova schema/migration, não o Engine.
- `PostgresStore` confirma suas transações; um harness real do Engine cria
  dados sintéticos duráveis em develop, a menos que haja cleanup explicitamente
  autorizado. Não fingir que um BEGIN/ROLLBACK externo contém suas operações.

## Delegação

- `infra`: configuração/DNS/proxy/roles/pré-requisitos VM, somente leitura.
- `brain_core`: sequência operator/SQL/prova de persistência, somente leitura
  nesta descoberta.
- `produto_client`: requisitos Studio/Meta/portal e TDD, somente leitura.
- Configuração solicitada para todos: `gpt-6-luna`, `xhigh`, agentes folha.
  Principal lê as fontes decisivas antes de integrar retornos.

## Decisões e limites

- Acesso administrativo main aprovado; não criar roles read-only/grants/0007
  por iniciativa própria. Reusar as credenciais server-private existentes do
  alvo escolhido, sem misturá-las com NSR ou develop.
- Studio e Meta serão uma extensão aprovada desta unidade além do out-scope
  histórico infra r6; não implica stack Supabase completo, novo banco ou Auth
  compartilhado. HTTP Brain Engine continua sem novos endpoints.
- Usar subhosts para evitar reescrever basePath da aplicação Studio. Portal
  deve identificar NSR, Mindforge main/produção e Mindforge develop.
- Validação base atual: `go test ./...` com GOPROXY off/GOTOOLCHAIN local
  passou; infra static all e unit all passaram. Nenhum banco local/descartável.

## Retorno final VM e implementação local

- Evidência sanitizada final em
  `/private/tmp/mindforge-develop-studio-vm-preflight.txt`, lida pelo principal.
  Nginx host ativo porta80; vhost `/etc/nginx/sites-enabled/db-home-arpa` aponta
  ao NSR gateway 18084, sem htpasswd próprio. Gateway Kong tem configuração
  privada declarativa, sem hash file separado nos mounts. Autenticação dos
  novos hosts/portal deve ser preparada sem expor credenciais.
- Login Mindforge READ ONLY tem superuser/createdb; develop continua ausente.
  Auth tem 70 migrations; ledger main 0001–0004 coincide com o local. Sem
  Brain Core/infra evidence 0005. Host psql padrão18.4, Compose2.40.3-3,
  arquitetura x86_64 e Go ausente. Confirmar cliente16 antes da operação.
- A query preflight corrigida pelo principal e a variante com relações
  ausentes passaram em READ ONLY `mindforge`: nenhuma tabela/base criada,
  sem user rows ou conexão ao NSR.
- Lotes locais dos workers: client-host separado, reuse admin main existente
  sem novo privilégio, verifier develop-only com dados sintéticos duráveis e
  Studio/Meta/portal. Bootstrap de extensões pelo operador está em preparação
  para manter o owner develop sem superuser.
- TDD Studio: 12 testes verdes e Compose render-only. Helper de pacote:
  quatro testes verdes de fontes/hashes, exclusão privada, symlinks e output
  fora do checkout. Resultados não provam runtime.

## Continuação

- [Spec integrada](spec-01-main.md).
- [Checkpoint](develop-studio-checkpoint.yaml).
- Confirmar roteamento/credencial/privilégio reais, congelar a execução local
  e preparar pacote + operação externa concreta, sem tocar NSR/produção por
  inferência.
