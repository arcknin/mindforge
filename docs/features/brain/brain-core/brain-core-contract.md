---
schema_version: 1
flow: orchestration-flow
feature: brain
task: brain-core
revision: 1
status: draft
owner_approved_at: null
tdd: null
implementation_authorized_at: null
repositories:
  - mindforge
---

# Task Contract: Brain Core

## Outcome

Estabelecer o primeiro limite comportamental do Brain privado: cada usuário
possui um único Brain, com uma hierarquia observável e livremente aninhável de
camadas, governada pelo Brain Engine. O contrato permanece em rascunho até a
aprovação explícita do owner.

## Scope

### In

- Um Brain privado por usuário; o Brain não é compartilhado por padrão.
- Bootstrap do Brain no primeiro acesso de forma idempotente e convergente:
  acessos repetidos do mesmo usuário resolvem o mesmo Brain e não criam
  duplicatas.
- Um ponto superior da hierarquia capaz de receber camadas criadas pelo owner;
  a política de camadas iniciais/default permanece pendente neste rascunho.
- Observação da hierarquia pertencente ao próprio usuário.
- Criação de uma camada diretamente sob o root ou sob qualquer camada existente
  do mesmo Brain.
- Nesting arbitrário, sem profundidade fixa, conjunto de nomes obrigatório ou
  taxonomia imposta.
- Renomeação de camada sem alterar sua identidade estável nem as referências
  que a utilizam.
- Identidade de Brain e camada independente de nome de exibição e posição na
  hierarquia; nome e posição são atributos mutáveis, não identidade.
- Brain Engine como autoridade única para autenticação, autorização, regras,
  validação, persistência, efeitos e estado durável do Brain.

### Out

- Inbox.
- Cognitive Functions, incluindo seleção, grupos e semântica de descendentes.
- Execução ou compartilhamento de Hive Mind e qualquer operação entre Brains.
- Move, archive, delete, restore ou purge de camadas, incluindo retenção,
  recuperação e tratamento de descendentes.
- Operação offline, sincronização, conflitos e réplicas locais.
- Retrieval, documentos, grafo, embeddings e indexação.
- Schema, SQL, migração, API, método/path HTTP, transporte, OpenAPI,
  implementação, código de teste, runtime, deploy ou publicação.

## Observable behavior

- O usuário autenticado encontra um único Brain privado associado à sua
  identidade.
- O primeiro acesso inicializa ou recupera esse Brain de modo convergente; o
  mesmo usuário nunca observa um segundo Brain criado por repetição do
  bootstrap.
- O comportamento inicial não trata Inbox como camada nem transforma exemplos
  conceituais em defaults. A existência de outras camadas iniciais ainda
  depende de decisão explícita do owner.
- A observação da hierarquia expõe somente o root e as camadas pertencentes ao
  Brain privado do usuário autorizado.
- O owner pode criar uma camada sob o root ou sob qualquer camada do mesmo
  Brain, inclusive em níveis arbitrariamente profundos.
- A criação não rejeita uma camada por não pertencer a uma taxonomia ou lista
  fixa de nomes.
- Ao renomear uma camada, sua identidade estável e suas referências permanecem
  as mesmas; somente o nome de exibição muda.
- Um solicitante que não seja o owner não pode observar nem alterar o Brain
  privado, e não recebe acesso implícito a outro Brain. A representação exata
  do erro permanece para contrato posterior.
- Nenhum comportamento atual cria, executa ou compartilha uma Hive Mind. A
  hierarquia e sua identidade permanecem compatíveis com uma futura seleção de
  um subtree-root para compartilhamento, sem definir consentimento, ACL,
  visibilidade, escrita, revogação, propagação ou qualquer outra semântica de
  Hive Mind.

## Interfaces and data

- Este rascunho define comportamento e invariantes, não uma interface técnica.
  Método, path, transporte, request, response, códigos de erro e idempotency
  key permanecem não definidos.
- O modelo conceitual contém uma associação de um usuário a um Brain, um root
  estrutural e camadas com identidade estável, nome mutável e relação
  hierárquica com um parent do mesmo Brain.
- Nenhum formato de identificador, esquema de persistência, banco, migração ou
  contrato OpenAPI é escolhido por este documento.
- Client, Prefrontal Cortex e Hidden Cortex solicitam ações ao Brain Engine;
  não possuem autoridade direta sobre dados persistentes.

## Constraints

- O acesso do primeiro recorte é owner-only e privado; não há delegação,
  compartilhamento entre usuários ou Hive Mind neste contrato.
- Inbox não pode ser camada default. Qualquer outra camada inicial ou starter
  tree depende de decisão explícita ainda pendente.
- A hierarquia deve aceitar nesting livre sem transformar exemplos conceituais
  em regras obrigatórias.
- Nome e posição não podem ser usados como identidade estável de Brain ou
  camada, inclusive para preservar a compatibilidade futura com subtree-root
  Hive Mind sharing.
- A compatibilidade futura não autoriza implementar ou antecipar semântica de
  Hive Mind.
- O Brain Engine é a única autoridade; nenhuma decisão de regra, autorização,
  persistência ou efeito pode ser transferida ao client ou a um runtime de
  agente.

## Acceptance criteria

ATDD draft — cenários abaixo são rascunho e não constituem testes executados.

- [ ] **ATDD-01 — Um Brain por usuário:** dado um usuário autenticado sem Brain
  inicializado, quando o primeiro acesso dispara o bootstrap, então existe um
  único Brain privado associado a esse usuário.
- [ ] **ATDD-02 — Bootstrap convergente:** dado o mesmo usuário, quando o
  bootstrap é solicitado repetidamente, então todas as solicitações resolvem o
  mesmo Brain e nenhuma duplicata é criada ou exposta.
- [ ] **ATDD-03 — Inicialização sem Inbox como camada:** dado um Brain
  recém-inicializado, quando sua hierarquia é observada, então Inbox não aparece
  como camada. A expectativa sobre outras camadas iniciais será completada após
  a decisão pendente do owner.
- [ ] **ATDD-04 — Observação privada:** dado o owner autenticado, quando observa
  seu Brain, então vê sua hierarquia; dado outro solicitante, quando tenta
  observá-la, então o Brain Engine nega a operação sem revelar ou alterar dados.
- [ ] **ATDD-05 — Criação sob root:** dado o owner e o root do seu Brain, quando
  cria uma camada, então a nova camada pertence ao mesmo Brain e fica sob o
  root.
- [ ] **ATDD-06 — Criação sob qualquer camada:** dado o owner e qualquer camada
  existente do seu Brain, quando cria uma nova camada sob ela, então a relação
  hierárquica é preservada, inclusive em níveis arbitrariamente profundos.
- [ ] **ATDD-07 — Sem taxonomia:** dado um nome escolhido pelo owner, quando a
  camada é criada, então o nome não é rejeitado por ausência de uma taxonomia ou
  lista fixa de categorias.
- [ ] **ATDD-08 — Rename preserva identidade:** dado uma camada existente com
  referências, quando o owner altera seu nome, então a identidade estável e as
  referências permanecem inalteradas.
- [ ] **ATDD-09 — Identidade não é nome/posição:** dado que o nome ou a posição
  de uma camada possa mudar em evolução futura, então nenhuma regra deste
  contrato exige esses atributos como identidade estável.

## TDD selection

- TDD: pending until explicit closing decision.
- O owner ainda deve responder explicitamente, sem limite de tempo, se a
  implementação futura usará TDD (`yes` ou `no`). Nenhum valor é inferido do
  rascunho.

## Validation boundary

- A validação desta unidade é documental e estática: leitura das fontes
  normativas e inspeção do rascunho.
- Nenhum teste ATDD/TDD, build, servidor, chamada HTTP, banco, runtime,
  deployment ou publicação foi executado; nada disso é provado por este
  contrato.
- O registro OpenAPI continua vazio e este documento não autoriza endpoint,
  schema, SQL, código ou plano.

## Authority required

- Aprovação explícita do owner para promover este rascunho a contrato aprovado.
- Decisão explícita do owner sobre TDD (`yes`/`no`) no fechamento do contrato.
- Autorização separada do owner antes de qualquer especificação derivada,
  planejamento ou implementação.

## Failure and recovery

- Bootstrap repetido deve convergir para o Brain já associado e nunca criar ou
  expor um segundo Brain para o mesmo usuário.
- Solicitação de outro usuário, parent de outro Brain ou operação fora do
  recorte deve ser negada sem mutação nem vazamento de dados; o formato exato
  do erro fica pendente de contrato técnico posterior.
- Se uma implementação futura não conseguir preservar a unicidade, isolamento
  ou identidade estável exigidos aqui, deve parar para decisão do owner; não há
  fallback autorizado neste rascunho.

## Assumptions

- “Usuário autenticado” identifica o owner lógico do Brain; o mecanismo de
  autenticação e o vínculo técnico ainda não são definidos.
- “Ponto superior da hierarquia” descreve a capacidade de criar camadas de
  primeiro nível; não exige que o root seja uma entidade persistida nem decide
  se outras camadas iniciais existirão.
- “Identidade estável” é uma invariante de produto; formato, geração e
  armazenamento do identificador permanecem fora do escopo.
- A posição é tratada como atributo organizacional, embora operações de move
  estejam explicitamente fora deste recorte.
- A política de camadas iniciais além da exclusão de Inbox permanece uma decisão
  material em aberto; este rascunho não assume Brain vazio nem starter tree.

## Owner decisions

- Um usuário possui exatamente um Brain privado.
- O bootstrap do primeiro acesso é idempotente e convergente.
- Inbox não é uma camada e não pode ser criada implicitamente como default; a
  política das demais camadas iniciais permanece pendente.
- O owner pode observar sua hierarquia e criar camadas sob o root ou sob
  qualquer camada do mesmo Brain.
- A hierarquia aceita nesting arbitrário e não impõe taxonomia.
- Renomear não muda a identidade da camada; nome e posição não são identidade.
- O acesso e as alterações são owner-only.
- O Brain Engine é a autoridade única.
- Inbox, Cognitive Functions, Hive Mind execution/sharing, lifecycle
  destrutivo/reorganização, offline/sync, retrieval/docs e todas as superfícies
  técnicas permanecem fora deste contrato.
- A hierarquia e a identidade devem permanecer compatíveis com um futuro
  subtree-root Hive Mind sharing, sem semântica inventada ou implementação
  presente.

## Amendments

No amendments.
