---
schema_version: 1
flow: orchestration-flow
feature: brain
task: brain-core
revision: 2
status: approved
owner_approved_at: 2026-09-17T03:17:44Z
tdd: yes
implementation_authorized_at: 2026-09-17T03:00:04Z
repositories:
  - mindforge
---

# Task Contract: Brain Core

## Outcome

Estabelecer o primeiro limite comportamental do Brain privado: cada usuário
possui um único Brain, criado com um root estrutural interno e sem qualquer
camada visível; o usuário cria a primeira camada. A hierarquia observável é
livremente aninhável, governada pelo Brain Engine. A verificação independente
de conformidade do ATDD retornou PASS e o contrato foi aprovado conforme a
autorização prévia do owner.

## Scope

### In

- Um Brain privado por usuário; o Brain não é compartilhado por padrão.
- Bootstrap do Brain no primeiro acesso de forma idempotente e convergente:
  acessos repetidos do mesmo usuário resolvem o mesmo Brain e não criam
  duplicatas.
- Criação do Brain com um root estrutural interno e zero camadas visíveis;
  não existe starter tree e o usuário cria a primeira camada visível.
- Observação da hierarquia pertencente ao próprio usuário.
- Criação de uma camada diretamente sob o root ou sob qualquer camada existente
  do mesmo Brain.
- Nesting arbitrário, sem profundidade fixa, conjunto de nomes obrigatório ou
  taxonomia imposta.
- Validação de nome: nome vazio ou composto apenas por whitespace é rejeitado;
  espaços externos são removidos, enquanto espaços internos, grafia, case e
  acentos são preservados.
- Camadas irmãs podem repetir o mesmo nome; a identidade é determinada pelo ID,
  não pelo nome.
- Renomeação de camada sem alterar sua identidade estável nem as referências
  que a utilizam; somente o nome visível é alterado.
- Concorrência determinística: os dois primeiros acessos concorrentes do mesmo
  usuário convergem para um único Brain; criações concorrentes sob o mesmo
  parent criam camadas distintas, inclusive com o mesmo nome; renames
  concorrentes da mesma camada baseados em estado antigo não perdem atualização
  silenciosamente, e o segundo recebe conflito para refazer a operação.
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
- Operação offline, sincronização offline, réplicas locais e conflitos fora das
  criações e renames concorrentes explicitamente definidos neste contrato.
- Retrieval, documentos, grafo, embeddings e indexação.
- Schema, SQL, migração, API, método/path HTTP, transporte, OpenAPI,
  implementação, código de teste, runtime, deploy ou publicação.

## Observable behavior

- O usuário autenticado encontra um único Brain privado associado à sua
  identidade.
- O primeiro acesso inicializa ou recupera esse Brain de modo convergente; o
  mesmo usuário nunca observa um segundo Brain criado por repetição do
  bootstrap.
- A inicialização cria um root estrutural interno e nenhuma camada visível.
  Inbox não é camada, não aparece como default e não existe starter tree; o
  usuário cria a primeira camada visível.
- A observação da hierarquia expõe somente o root e as camadas pertencentes ao
  Brain privado do usuário autorizado.
- O owner pode criar uma camada sob o root ou sob qualquer camada do mesmo
  Brain, inclusive em níveis arbitrariamente profundos.
- A criação não rejeita uma camada por não pertencer a uma taxonomia ou lista
  fixa de nomes.
- A criação rejeita nome vazio ou somente whitespace, remove espaços externos e
  preserva espaços internos, grafia, case e acentos. Camadas irmãs podem ter o
  mesmo nome.
- Os dois primeiros acessos concorrentes do mesmo usuário resolvem o mesmo
  Brain. Criações concorrentes sob o mesmo parent produzem camadas distintas,
  inclusive quando o nome é igual.
- Ao renomear uma camada, sua identidade, parent, filhos e referências
  permanecem os mesmos; somente o nome visível muda. Se dois renames da mesma
  camada usarem estado antigo, o segundo recebe conflito, sem sobrescrever
  silenciosamente a atualização anterior, e deve refazer a operação.
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
  estrutural interno que não é camada visível e camadas com identidade estável,
  nome mutável e relação hierárquica com um parent do mesmo Brain.
- Nome é um valor visível normalizado apenas nos espaços externos; não há
  unicidade por nome entre irmãs, e espaços internos, grafia, case e acentos
  fazem parte do valor preservado.
- Um rename deve ser condicionado ao estado que o solicitante observou; se esse
  estado já não for atual, o Brain Engine retorna conflito e não aplica a
  alteração. O formato técnico do conflito permanece não definido.
- Nenhum formato de identificador, esquema de persistência, banco, migração ou
  contrato OpenAPI é escolhido por este documento.
- Client, Prefrontal Cortex e Hidden Cortex solicitam ações ao Brain Engine;
  não possuem autoridade direta sobre dados persistentes.

## Constraints

- O acesso do primeiro recorte é owner-only e privado; não há delegação,
  compartilhamento entre usuários ou Hive Mind neste contrato.
- O Brain nasce com root estrutural interno e nenhuma camada visível. Não há
  starter tree; Inbox nunca é camada; a primeira camada é criada pelo usuário.
- A hierarquia deve aceitar nesting livre sem transformar exemplos conceituais
  em regras obrigatórias.
- Nome vazio ou somente whitespace é inválido; espaços externos são removidos,
  e espaços internos, grafia, case e acentos são preservados. Nomes repetidos
  entre camadas irmãs são permitidos.
- Nome e posição não podem ser usados como identidade estável de Brain ou
  camada, inclusive para preservar a compatibilidade futura com subtree-root
  Hive Mind sharing.
- Criações concorrentes sob o mesmo parent não são deduplicadas por nome. Um
  rename concorrente baseado em estado antigo deve falhar com conflito, sem
  lost update silencioso.
- A compatibilidade futura não autoriza implementar ou antecipar semântica de
  Hive Mind.
- O Brain Engine é a única autoridade; nenhuma decisão de regra, autorização,
  persistência ou efeito pode ser transferida ao client ou a um runtime de
  agente.

## Acceptance criteria

ATDD formalizado — os cenários abaixo foram verificados independentemente com
resultado `PASS` e ainda não constituem testes executados. Eles descrevem
apenas comportamento observável; método, path, schema, persistência e
representação técnica de erros permanecem fora deste contrato.

- [ ] **ATDD-01 — Um Brain por owner:** dado um owner autenticado sem Brain
  inicializado, quando o primeiro acesso dispara o bootstrap, então existe um
  único Brain privado associado a esse owner.
- [ ] **ATDD-02 — Bootstrap repetido convergente:** dado um owner cujo Brain já
  foi resolvido, quando o bootstrap é solicitado repetidamente, então todas as
  solicitações resolvem o mesmo Brain e nenhuma duplicata é criada ou exposta.
- [ ] **ATDD-03 — Bootstrap simultâneo convergente:** dado um owner sem Brain,
  quando os dois primeiros bootstraps são solicitados simultaneamente, então
  ambos resolvem o mesmo Brain, existe uma única associação para o owner e
  nenhum segundo Brain é criado ou exposto.
- [ ] **ATDD-04 — Início sem camadas visíveis:** dado um Brain recém-
  inicializado, quando o owner observa sua hierarquia, então existe um root
  estrutural interno como âncora, nenhuma camada visível aparece, Inbox não é
  camada e não existe starter tree; quando o owner cria uma camada, ela é a
  primeira camada visível.
- [ ] **ATDD-05 — Observação e alteração owner-only sem vazamento:** dado o
  Brain privado de um owner, quando outro solicitante tenta observá-lo ou
  alterá-lo, então o Brain Engine nega a operação sem revelar a existência,
  identidade, nomes ou relações do Brain e sem produzir mutação; o owner
  continua podendo observar sua própria hierarquia.
- [ ] **ATDD-06 — Criação diretamente sob o root:** dado o owner e o root
  estrutural do seu Brain, quando cria uma camada, então a nova camada pertence
  ao mesmo Brain e fica diretamente sob o root.
- [ ] **ATDD-07 — Criação sob qualquer camada e nesting arbitrário:** dado o
  owner e uma camada existente do seu Brain, quando cria descendentes sob essa
  camada e sob descendentes sucessivos, então cada relação parent-filho é
  preservada no mesmo Brain, sem profundidade máxima imposta.
- [ ] **ATDD-08 — Parent de outro Brain rejeitado:** dado o owner de um Brain e
  um parent pertencente a outro Brain, quando tenta criar uma camada usando
  esse parent, então a operação é rejeitada, nenhum vínculo cross-Brain é
  criado e nenhum dado do outro Brain é revelado ou alterado.
- [ ] **ATDD-09 — Ausência de taxonomia obrigatória:** dado um nome escolhido
  pelo owner que não pertence a qualquer lista fixa de categorias, quando a
  camada é criada sob um parent válido, então a criação é aceita e não é
  rejeitada por ausência de taxonomia.
- [ ] **ATDD-10 — Nome vazio ou só whitespace rejeitado:** dado o owner e um
  parent válido, quando tenta criar uma camada com nome vazio ou composto apenas
  por whitespace, então a operação é rejeitada e nenhuma camada é criada.
- [ ] **ATDD-11 — Trim externo e preservação do valor interno:** dado um nome
  com whitespace externo e com espaços internos, grafia, case e acentos
  significativos, quando o owner cria a camada, então o nome visível remove
  apenas o whitespace externo e preserva exatamente os espaços internos, a
  grafia, o case e os acentos.
- [ ] **ATDD-12 — Nomes irmãos duplicados permitidos:** dado um parent válido,
  quando o owner cria duas camadas irmãs com o mesmo nome, então as duas são
  mantidas como camadas distintas, cada uma com identidade estável própria.
- [ ] **ATDD-13 — Criações concorrentes não são deduplicadas:** dado um parent
  válido, quando duas criações concorrentes do owner usam esse mesmo parent e o
  mesmo nome, então resultam em duas camadas distintas, sem deduplicação ou
  sobrescrita por nome.
- [ ] **ATDD-14 — Rename altera somente o nome visível:** dado uma camada com
  identidade estável, parent, filhos e referências existentes, quando o owner a
  renomeia, então somente o nome visível muda e a identidade, o Brain, o parent,
  os filhos e as referências permanecem inalterados.
- [ ] **ATDD-15 — Rename baseado em estado antigo conflita e exige releitura:**
  dado que duas operações do owner tenham observado o mesmo estado de uma
  camada, quando a primeira renomeia a camada e a segunda tenta aplicar seu
  rename usando esse estado antigo, então a segunda recebe conflito, não
  sobrescreve a atualização já aplicada e deve reler o estado antes de refazer a
  operação; após a releitura, o retry explícito pode aplicar o novo nome.
- [ ] **ATDD-16 — Identidade independente de nome e posição:** dado que camadas
  distintas possam ter o mesmo nome e ocupar posições hierárquicas diferentes,
  quando o owner as observa ou usa suas referências e uma delas é renomeada,
  então cada camada continua sendo distinguida por sua identidade estável, sem
  nome ou posição serem usados como identidade ou como motivo para mesclar as
  camadas.

## TDD selection

- TDD: yes.
- O owner fechou explicitamente TDD como `yes`; a implementação futura deve
  seguir ciclos test-first quando a unidade for autorizada.

## Validation boundary

- A validação desta unidade é documental e estática: leitura das fontes
  normativas e inspeção do contrato.
- Nenhum teste ATDD/TDD, build, servidor, chamada HTTP, banco, runtime,
  deployment ou publicação foi executado; nada disso é provado por este
  contrato.
- O registro OpenAPI continua vazio e este documento não autoriza endpoint,
  schema, SQL, código ou plano. A formalização do ATDD está concluída nesta
  revisão, a verificação independente de conformidade retornou `PASS` e o
  contrato foi promovido a aprovado conforme a autorização prévia do owner.

## Authority required

- A autorização explícita e condicionada do owner para promover este contrato
  foi satisfeita; a aprovação está registrada em `owner_approved_at`.
- A verificação independente de conformidade do ATDD retornou `PASS`; a
  condição para aprovação está satisfeita.
- O owner autorizou, após esses gates, a aprovação do contrato e a criação do
  plano; depois do plano, autorizou execução local por nova tarefa/orquestrador
  sem worktree.
- Essa autorização é condicionada e local: não autoriza commit, push, deploy,
  produção, migrations executadas ou ações remotas.

## Failure and recovery

- Bootstrap repetido deve convergir para o Brain já associado e nunca criar ou
  expor um segundo Brain para o mesmo usuário.
- Os dois primeiros bootstraps concorrentes do mesmo usuário devem convergir ao
  mesmo Brain. Se a convergência falhar, a operação deve parar para decisão do
  owner; não é permitido criar um Brain adicional como fallback.
- Criações concorrentes sob o mesmo parent devem preservar ambas as intenções
  como camadas distintas, ainda que os nomes sejam iguais.
- Um rename baseado em estado antigo deve retornar conflito e preservar a
  atualização já aplicada; o solicitante deve reler o estado e refazer a
  operação. Não há fallback de sobrescrita silenciosa.
- Solicitação de outro usuário, parent de outro Brain ou operação fora do
  recorte deve ser negada sem mutação nem vazamento de dados; o formato exato
  do erro fica pendente de contrato técnico posterior.
- Se uma implementação futura não conseguir preservar a unicidade, isolamento
  ou identidade estável exigidos aqui, deve parar para decisão do owner; não há
  fallback autorizado neste rascunho.

## Assumptions

- “Usuário autenticado” identifica o owner lógico do Brain; o mecanismo de
  autenticação e o vínculo técnico ainda não são definidos.
- “Root estrutural interno” descreve o ponto superior usado para a hierarquia;
  seu formato e persistência técnica permanecem fora do escopo, mas ele não é
  uma camada visível.
- “Identidade estável” é uma invariante de produto; formato, geração e
  armazenamento do identificador permanecem fora do escopo.
- A posição é tratada como atributo organizacional, embora operações de move
  estejam explicitamente fora deste recorte.
- O Brain inicia sem camadas visíveis; a primeira camada é criada pelo usuário.
- O mecanismo técnico do conflito de rename, seu código e sua representação
  permanecem pendentes, mas a ausência de lost update e a necessidade de refazer
  a operação são normativas.

## Owner decisions

- Um usuário possui exatamente um Brain privado.
- O bootstrap do primeiro acesso é idempotente e convergente.
- O Brain é criado com root estrutural interno e nenhuma camada visível; não há
  starter tree, Inbox nunca é camada e o usuário cria a primeira camada.
- O owner pode observar sua hierarquia e criar camadas sob o root ou sob
  qualquer camada do mesmo Brain.
- A hierarquia aceita nesting arbitrário e não impõe taxonomia.
- Nome vazio ou somente whitespace é rejeitado; espaços externos são removidos,
  e espaços internos, grafia, case e acentos são preservados.
- Camadas irmãs podem repetir nome, pois identidade é por ID.
- Renomear altera somente o nome visível, preservando ID, parent, filhos e
  referências.
- Dois primeiros acessos concorrentes convergem para um único Brain; criações
  concorrentes no mesmo parent criam camadas distintas, inclusive com o mesmo
  nome; renames concorrentes baseados em estado antigo resultam em conflito
  para o segundo, que deve refazer a operação.
- O acesso e as alterações são owner-only.
- O Brain Engine é a autoridade única.
- Em 2026-09-17T03:00:04Z, o owner autorizou que, após o ATDD ser
  formalizado e considerado conforme, este contrato seja aprovado e o plano
  seja criado; após o plano, autorizou execução local por nova
  tarefa/orquestrador sem worktree. Essa autorização não cobre commit, push,
  deploy, produção, migrations executadas ou ações remotas.
- Inbox, Cognitive Functions, Hive Mind execution/sharing, lifecycle
  destrutivo/reorganização, offline/sync, retrieval/docs e todas as superfícies
  técnicas permanecem fora deste contrato.
- A hierarquia e a identidade devem permanecer compatíveis com um futuro
  subtree-root Hive Mind sharing, sem semântica inventada ou implementação
  presente.

## Amendments

```yaml
amendment:
  id: brain-core-owner-closure-2026-09-17
  resulting_revision: 2
  reason: "Registrar as decisões fechadas do owner sobre inicialização, nomes, identidade, concorrência, TDD e autorização condicionada."
  evidence:
    - "Decisões do owner registradas no handoff desta tarefa, capturadas em 2026-09-17T03:00:04Z."
  changed_clauses:
    - outcome
    - scope
    - observable_behavior
    - interfaces_and_data
    - constraints
    - acceptance_criteria
    - implementation_authorized_at
    - tdd_selection
    - validation_boundary
    - authority_required
    - failure_and_recovery
    - assumptions
    - owner_decisions
  preserved_clauses:
    - owner-only privacy and Brain Engine authority
    - arbitrary nesting and future Hive Mind compatibility boundary
    - technical interfaces, schema, API, migration and runtime remaining out of scope
  owner_approved_at: 2026-09-17T03:17:44Z
```
