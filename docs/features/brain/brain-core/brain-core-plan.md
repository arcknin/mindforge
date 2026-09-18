---
schema_version: 1
flow: orchestration-flow
feature: brain
task: brain-core
revision: 1
contract_revision: 2
status: ready
tdd: yes
implementation_authorized_at: 2026-09-17T03:00:04Z
repositories:
  - mindforge
---

# Code Plan: Brain Core

## Execution summary

Implementar o primeiro núcleo executável do Brain Engine em Go como uma
superfície de domínio/aplicação sem transporte HTTP. O núcleo recebe um
principal autenticado opaco, aplica autorização owner-only, mantém um Brain por
principal, cria um root estrutural persistido que não aparece como camada,
permite nesting livre e usa identidade estável com controle otimista de
conflito em rename. A persistência será PostgreSQL por um adapter explícito e
uma migration de produto separada da superfície de infraestrutura; a
inicialização do processo não executará migrations automaticamente.

São dois lotes sequenciais porque o segundo depende dos ports e invariantes do
primeiro. Cada lote mantém expected-red -> implementação -> relevant-green no
mesmo escopo. Nenhum endpoint, documento OpenAPI, client, Hidden Cortex,
capability futura ou alteração em `src/brain-engine/features/infra/` faz parte
deste plano.

## Tasks

```yaml
- id: B01
  goal: "Entregar o núcleo determinístico do Brain Core e seus ports, cobrindo os 16 cenários ATDD sem escolher transporte ou provedor de autenticação."
  repository: mindforge
  depends_on: []
  allowed_scope:
    - src/brain-engine/go.mod
    - src/brain-engine/features/brain/model.go
    - src/brain-engine/features/brain/errors.go
    - src/brain-engine/features/brain/ports.go
    - src/brain-engine/features/brain/ids.go
    - src/brain-engine/features/brain/engine.go
    - src/brain-engine/features/brain/engine_test.go
    - src/brain-engine/features/brain/memory_store_test.go
  changes:
    - "Expected-red: adicionar testes table-driven e concorrentes para ATDD-01, ATDD-02, ATDD-03, ATDD-04, ATDD-05, ATDD-06, ATDD-07, ATDD-08, ATDD-09, ATDD-10, ATDD-11, ATDD-12, ATDD-13, ATDD-14, ATDD-15 e ATDD-16; eles devem falhar porque o pacote, os tipos, os ports e o Engine ainda não existem."
    - "Expected-red: incluir asserções de que o primeiro bootstrap expõe apenas root e zero camadas, não cria Inbox/starter tree, que IDs não dependem de nome/posição, que nomes irmãos iguais são aceitos, que nome vazio/whitespace falha e que trim externo preserva espaços internos, grafia, case e acentos."
    - "Expected-red: incluir owner-only sem vazamento, parent estrangeiro sem vínculo, nesting profundo sem limite de produto, duas criações simultâneas distintas e rename CAS no qual exatamente uma operação vence, a segunda recebe conflito e só um retry com versão relida vence."
    - "Implementação: criar o módulo `github.com/arcknin/mindforge/src/brain-engine` na raiz `src/brain-engine`, declarando `go 1.25`, sem dependências externas neste lote; modelar `Principal`, `Brain`, `Root`, `Layer`, `Hierarchy`, `LayerVersion` e IDs UUID gerados pelo Engine."
    - "Implementação: expor apenas a superfície de aplicação `Bootstrap`, `Observe`, `CreateLayer` e `RenameLayer`, com Store/IDSource injetáveis; validar principal não vazio e nomes com `strings.TrimSpace`, preservando o valor interno e rejeitando somente o resultado vazio."
    - "Implementação: definir erros opacos e tipados para autenticação, acesso/não encontrado, nome inválido, conflito e falha interna; nunca interpolar owner, Brain ID, layer ID ou nome em erro retornado ao chamador."
    - "Implementação: criar fake Store somente nos testes, protegido por mutex, para provar as invariantes sem apresentá-lo como persistência de produção; o Engine deve consultar sempre o principal e nunca aceitar autorização derivada de nome ou posição."
    - "Relevant-green: formatar e executar testes unitários e de race do pacote, confirmando que os cenários passam e que não existe importação de HTTP, endpoint ou acesso direto a PostgreSQL nesta camada."
  validation:
    - "cd src/brain-engine && go test ./features/brain/..."
    - "cd src/brain-engine && go test -race ./features/brain/..."
    - "cd src/brain-engine && go vet ./features/brain/..."
    - "gofmt -d src/brain-engine/features/brain/*.go"
    - "rg -n 'net/http|ListenAndServe|database/sql|pgx' src/brain-engine/features/brain/{model.go,errors.go,ports.go,ids.go,engine.go} deve retornar vazio"
  done_when:
    - "O módulo e o pacote compilam, e todos os ATDD-01..ATDD-16 têm uma prova unitária que passa sem Banco, HTTP ou client."
    - "Bootstrap concorrente converge por owner; criação não deduplica nome; rename usa versão observada e não admite lost update."
    - "Nenhum default, Inbox, taxonomia, profundidade fixa, operação destrutiva, move, Hive Mind ou Cognitive Function aparece no código ou nos testes."
    - "Os ports deixam explícita a futura injeção de persistência e principal autenticado sem fingir que autenticação/transportes técnicos já foram contratados."
  parallelizable: false

- id: B02
  goal: "Entregar persistência PostgreSQL do Brain Core, migration de produto isolada da infra e prova integrada das invariantes sob concorrência."
  repository: mindforge
  depends_on:
    - B01
  allowed_scope:
    - src/brain-engine/go.mod
    - src/brain-engine/go.sum
    - src/brain-engine/features/brain/migrations/0001_brain_core.sql
    - src/brain-engine/features/brain/postgres_store.go
    - src/brain-engine/features/brain/migration_test.go
    - src/brain-engine/features/brain/postgres_store_test.go
  changes:
    - "Expected-red: criar testes estáticos para a migration ausente e testes de integração que falhem sem o PostgresStore, sem as tabelas `brain_core` e sem as constraints de owner, root, parent e versão."
    - "Expected-red: provar em banco que duas transações de bootstrap do mesmo principal não podem produzir dois brains/roots, que criações simultâneas com mesmo parent e nome não são deduplicadas e que rename concorrente com a mesma versão produz um sucesso e um conflito sem lost update."
    - "Expected-red: provar que parent/layer de outro owner não entra na consulta escopada, não cria FK cross-Brain e não revela IDs/nomes; provar também que uma leitura após rename mantém ID, parent, root, filhos e nomes irmãos intactos."
    - "Implementação: adicionar `github.com/jackc/pgx/v5 v5.7.4` pinado no `go.mod`/`go.sum`, sem expor tipos pgx no modelo; construir adapter parametrizado com pool/transações e mapear ausência, conflito e falha de storage para erros opacos do Engine."
    - "Implementação: criar `0001_brain_core.sql` fora de `src/brain-engine/features/infra/postgres/migrations/`, com schema `brain_core`, associação única `owner_subject -> brain`, nodes com `kind=root|layer`, UUID estável, `parent_id` e FK composta `(brain_id,parent_id)` para impedir parent de outro Brain, índice de um root por Brain e `version` para CAS; não criar users, documents, Inbox, referências, endpoints ou starter rows."
    - "Implementação: fazer bootstrap em transação com unique owner e root idempotente; observar root separadamente das camadas visíveis; criar layer com insert owner-scoped que aceita root/layer válido e não aplica unique por nome; renomear com `UPDATE ... WHERE version = expected_version`, distinguir conflito de não encontrado sem devolver detalhes."
    - "Implementação: manter a migration como arquivo versionado consumido por procedimento de migration autorizado; os testes podem aplicar o arquivo somente em um banco/volume descartável explicitamente identificado, sem auto-migrate no construtor ou startup do Engine."
    - "Relevant-green: executar testes estáticos/unitários acumulados e, quando `BRAIN_CORE_DATABASE_URL` apontar para um PostgreSQL descartável exclusivo, aplicar a migration de teste e repetir toda a matriz concorrente com `go test -race`; validar também que a infra existente e o registry HTTP não foram alterados."
  validation:
    - "cd src/brain-engine && go test ./features/brain/..."
    - "cd src/brain-engine && go test -race ./features/brain/..."
    - "cd src/brain-engine && go vet ./..."
    - "cd src/brain-engine && BRAIN_CORE_DATABASE_URL=\"$DISPOSABLE_DSN\" go test -count=1 ./features/brain/..."
    - "bash src/brain-engine/features/infra/tests/run.sh static all"
    - "bash src/brain-engine/features/infra/tests/run.sh unit all"
    - "rg -n -F 'endpoints: []' contracts/openapi/registry.yaml"
    - "git diff --check"
  done_when:
    - "A migration de produto isolada cria somente o estado necessário ao Brain Core e não é adicionada ao ledger/allowlist de migrations da infra reduzida."
    - "O adapter PostgreSQL preserva um Brain por principal, root estrutural sem camada inicial, isolamento owner-only, nesting arbitrário, nomes duplicados, IDs estáveis e conflito otimista de rename sob concorrência."
    - "Testes sem DSN provam static/unit; o comando com DSN prova apenas integração local em banco descartável e não autoriza nem declara execução em VM, produção, dados reais ou deploy."
    - "Nenhum arquivo de `contracts/openapi/`, client, Hidden Cortex, endpoint HTTP ou migration real é criado/modificado por este lote."
  parallelizable: false
```

## Assumptions

- O principal recebido pelo Engine é um identificador opaco não vazio já
  produzido por uma fronteira de autenticação confiável. A integração GoTrue,
  tokens, sessão, transporte e mapeamento técnico de identidade continuam
  fora do contrato e deste plano.
- UUID é apenas uma escolha técnica para identidade estável; Brain e layer são
  identificados por ID, nunca por nome, ordem ou caminho. O root é um node
  persistido de tipo estrutural e não entra na coleção de camadas visíveis.
- `pgx/v5` `v5.7.4` é a única dependência externa prevista; seu uso fica
  isolado no adapter e não altera a superfície observável.
- A migration de produto fica em `features/brain/migrations/` e não usa o
  executor `src/brain-engine/features/infra/scripts/migrate.sh`, cujo contrato
  continua limitado a Auth, extensões e bookkeeping de infraestrutura.
- O primeiro slice não possui consumidores de referências/documentos; a prova
  de continuidade cobre IDs, parent e filhos. Qualquer vínculo de conteúdo,
  retrieval, sincronização ou Hive Mind exige contrato próprio posterior.
- A aplicação de qualquer migration é uma ação separada de execução. Esta
  tarefa de planejamento não executa banco, migration, Docker, HTTP, remoto,
  commit, push ou deploy; a integração futura deve usar somente ambiente
  descartável e dados sintéticos dentro da autoridade local registrada.

## TDD execution

- B01 começa pelos reds unitários do comportamento fechado, implementa o
  domínio/Engine e repete exatamente a matriz até green; o fake Store é fixture,
  não evidência de persistência.
- B02 começa pelos reds de schema, constraints e adapter, implementa SQL e
  PostgreSQL owner-scoped, e repete os testes unitários e concorrentes até
  green. A migration só pode ser aplicada no banco descartável explicitamente
  escolhido para a validação; nenhuma aplicação em banco real é presumida.
- Falha de infraestrutura local, ausência de DSN ou indisponibilidade de Docker
  é limitação de evidência, não justificativa para substituir integração por
  mock, VM ou produção. Nenhum batch possui review ou gate intermediário.

## Evidence boundary

- `static`: frontmatter/allowlist, SQL e código compilável, `go vet`, `gofmt`,
  ausência de endpoints no registry e regressão estática/unit da infra.
- `unit/local`: testes do Engine com fake Store e `go test -race`; provam regras
  determinísticas, não PostgreSQL, autenticação real ou runtime.
- `integration`: testes do adapter contra o DSN/volume descartável informado,
  com migration aplicada apenas nesse alvo; provam o banco identificado, não
  produção, VM, rede ou dados reais.
- `runtime`: nenhum endpoint Mindforge é aprovado ou verificado; o registry
  continua `endpoints: []`, e não há `runtime_status: verified` a registrar.

## Residual risks

- A forma técnica de autenticar o principal e qualquer contrato HTTP continuam
  pendentes; o Engine não deve ser apresentado como API publicada após estes
  lotes.
- A migration não define um runner de produção nem auto-migrate. O owner deve
  escolher e autorizar o procedimento operacional antes de aplicar o schema em
  qualquer banco persistente.
- O contrato não inclui move/delete/archive/restore, referências de documentos,
  quotas, auditoria, sync/offline, Hive Mind ou Cognitive Functions; adicionar
  qualquer um deles invalidaria este plano e exigiria nova decisão/contrato.
- A prova PostgreSQL depende da compatibilidade do ambiente descartável com a
  infra atual e não prova deploy, VM, produção, publicação ou segurança de
  rede. Nenhuma mudança existente do checkout sujo pode ser descartada.
