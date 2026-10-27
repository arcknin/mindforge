# Discovery: retomada do Mindforge

> Retomada atual: [complemento de 26/10/2026](#retomada-de-26102026--discussão-do-assistente). As seções anteriores ao complemento descrevem evidências históricas e não o inventário atual de código, contratos ou pendências.

## Pedido e limite da investigação

- Pedido do owner em 2026-10-05: usar spec-drive e agentes para entender o estado atual e onde o projeto parou, antes de decidir a continuação.
- Escopo: checkout Mindforge, documentos, contratos, código e estado Git; exploração somente leitura do código, com registro documental desta retomada.
- A recuperação inicial de 05/10 não autorizou implementação, VM, SQL, banco, serviços, backup real, publicação, commit ou push. Em 08/10 o owner autorizou separadamente uma consulta SSH somente leitura para esclarecer os vínculos entre bases e Supabase; essa consulta está registrada abaixo.

## Comportamento atual verificado

- Git: branch `feat/brain-core-foundation`, HEAD `0204ba4`, acompanhando `origin/feat/brain-core-foundation`; alterações locais anteriores extensas em Brain Core/infra e documentos foram preservadas.
- Brain Core: contrato r4 aprovado; `Engine` contém `Bootstrap`, `Observe`, `CreateLayer` e `RenameLayer`. `PostgresStore` implementa persistência com pgx e não aplica migrations ao abrir a conexão. O Engine recebe um principal já autenticado; isso não constitui integração com Auth.
- Infra: contrato r6 aprovado; operador Go contém `preflight`, `provision-develop`, `apply`, `backup`, `restore`, `promote` e auditoria local. Catálogo global `0006` referencia SQL, preflight e verify pertencentes ao Brain Core. Os três hashes foram conferidos e coincidem com o catálogo.
- HTTP: `contracts/openapi/registry.yaml` está em `endpoints: []`; não há documento OpenAPI com rotas aprovadas.
- Não existe client Flutter nem servidor HTTP de produto no inventário atual. O executável Go existente é o operador de infraestrutura.
- O Brain Core r4 já define um Brain por usuário, root interno sem starter tree, isolamento por owner, IDs estáveis, nomes duplicados entre irmãs e conflito de rename por versão. Esses pontos não precisam de nova decisão para o mesmo recorte.
- `postgres_store_test.go` contém atualmente apenas o teste de interface/DSN vazio/ausência de auto-migration. O diff remove 231 linhas, incluindo a antiga matriz de integração PostgreSQL. As provas executáveis atuais do comportamento persistente precisam ser definidas para a VM indicada; o harness descartável antigo não deve ser restaurado por inferência.

## Onde paramos

1. Brain Core r4 / plano r3 B03: histórico de 30/09 registra conclusão local dos checks SQL da feature e retirada da rota `.psql`/ledger local. Payload canônico preservado.
2. Infra r6 / plano r6 B01–B03: histórico de 30/09 registra conclusão local da coexistência passiva de `nsrdb`, integração do catálogo/operador/ledger único e runbook. Root confirmou essas superfícies no código atual.
3. Última inspeção real registrada: 30/09, somente leitura; develop/base/role/Auth ausentes, ledger em `mindforge` com 0001–0004, sem evidência infra 0005 nem schema Brain Core. Isso localiza a parada histórica; não confirma o ambiente em 05/10.
4. Próxima fronteira operacional registrada: entrega do source/operador corrigido e provisionamento aditivo de develop, depois aplicação/verificação autorizada de 0005/0006 em develop. Promoção para main não faz parte desta retomada.

## Fontes e achados delegados

- Principal: seis fontes obrigatórias de `AGENTS.md`, checkpoints/contratos/planos Brain Core e infra, inventário de código, fontes decisivas Engine/PostgresStore/operador/catálogo, Git e hashes.
- `/root/brain_core`, solicitado `gpt-6-luna` / `xhigh`, folha somente leitura: contrato/comportamento, persistência, checks SQL, plano/checkpoint/histórico e lacuna de teste real. Principal confirmou em Engine, PostgresStore, teste atual, contratos, checkpoints e histórico.
- `/root/infra`, solicitado `gpt-6-luna` / `xhigh`, folha somente leitura: operador/catálogo implementados localmente e última fronteira de VM. Principal confirmou em `ProvisionDevelop`, `Apply`, preparação de transação de feature, catálogo global, r6 e histórico de 30/09.
- `/root/produto_client`, solicitado `gpt-6-luna` / `xhigh`, folha somente leitura: separação de conceitos e código; decisões necessárias para Auth/API/client. Principal confirmou contrato Brain Core, `Engine` recebendo principal previamente autenticado, registry e OD-006/OD-007.
- Todos retornaram sem mudanças, testes, banco, VM, rede ou acesso a segredos. Estes foram agentes de pesquisa; não houve revisão automática nem gate de auditoria.
- `skillr resume --docs /Users/arcknin/Desktop/mindforge/docs --limit 5 --json`: sucesso de execução, nenhum candidato, alerta de `remaining` duplicado no checkpoint Brain Core. Infra está sob `docs/stack/back-end/features/`, fora da raiz de features desse comando. Recuperação baseada em leitura direta dos dois checkpoints.

## Incertezas e decisões para discussão

- A última inspeção da VM está registrada em 2026-09-30: `mindforge`, `nsrdb` e `postgres`, sem `mindforge_develop`, Auth de develop, migration infra 0005 ou schema Brain Core. O estado não foi reconsultado.
- Os READMEs gerais da stack ainda apontam infra r3, `mindforge_main`, catálogo vazio e ausência de tabelas de domínio; contratos/código locais posteriores prevalecem para seu escopo.
- Checkpoints antigos contêm duas chaves `remaining` e resultados/limitações históricos acumulados. `status: complete` descreve lotes locais, não produto operante ou atualização da VM.
- Planos e frontmatter dos contratos mantêm autorização de implementação nula, enquanto checkpoint/histórico registram autorização e execução local posteriores. O `resume_from` do Brain Core ainda manda fazer infra B02, já registrado como concluído. Não tratar esses ponteiros isolados como trabalho pendente ou autorização atual.
- A discovery antiga de produto Brain ainda registra falta de Git, contrato em draft e default-layer policy pendente. Hoje há Git, contrato r4 aprovado e root vazio decidido. OD-002/OD-004 têm perguntas parcialmente fechadas para Brain Core; outros domínios seguem abertos.
- Para API/client ainda faltam contrato de autenticação (sessão/token → principal confiável), transporte, método/path, formatos, erros, retry/idempotência e conflito. Flutter/Dart está escolhido; plataforma inicial e inclusão de offline/sync ainda exigem decisão.
- Hive Mind, Inbox, Funções Cognitivas, documentos, retrieval e camadas agênticas continuam fora do Brain Core. Não bloqueiam sua primeira entrega manual.

## Recomendação para a próxima unidade

1. Reconciliar apenas os ponteiros/documentos tocados pela próxima entrega, preservando a história e mudanças locais. Definir uma versão exata do source/operador para distribuição e a prova pretendida; não iniciar reestruturação geral.
2. Preparar uma operação concreta para estabelecer e validar `mindforge_develop` no cluster designado: repetir inventário autorizado, entregar operador atual, provisionar develop de forma aditiva, estabelecer Auth independente e aplicar/verificar 0005/0006 no alvo. Cada operação/target deve ter autoridade explícita. Preservar `mindforge`, `nsrdb` e `infra_mindforge-postgres-data`.
3. Com a base verificável, fechar uma spec integrada para a primeira experiência manual online: autenticação, observar hierarquia, criar e renomear camadas. Escolher entre API primeiro ou uma entrega Flutter+backend, e a plataforma se incluir client. Isso é recomendação, não decisão aprovada.
4. Definir aceitação para bootstrap concorrente, isolamento entre usuários, criação/rename persistentes, conflitos e rollback/evidência de migration. Testes de schema não substituem prova comportamental de `PostgresStore`.

O risco estratégico é expandir o desenho e acumular mais documentação sem reduzir a distância até uma operação real. O próximo recorte deve demonstrar comportamento persistente e utilizável, antes de adicionar agentes, compartilhamento ou recuperação semântica ao produto.

## Validação e limites

- Executados nesta sessão: leitura de fontes/inventário/diffs, `git status`, `git log`, `git diff --stat`, bundled `skillr resume`, SHA-256 dos três SQLs e `git diff --check` (exit 0).
- Fechamento documental: Ruby carregou o YAML e confirmou referência ao discovery e três workers concluídos. `skillr resume` filtrado por `project/continuation` reconheceu o novo checkpoint sem incertezas do candidato; o alerta de duplicidade do checkpoint legado permanece.
- Não executados: testes de código, build, SQL, PostgreSQL, Docker, SSH/VM, Auth, backups, deploy, commit ou push. Resultados de testes anteriores são registros históricos, não uma revalidação de hoje.
- Endpoints: nenhum afetado; registry permanece vazio, sem novo contrato, implementação ou runtime HTTP.
- Únicas alterações desta sessão: este discovery e seu checkpoint. Fontes e documentos antigos preservados.

## Continuação

- Checkpoint: [continuation-checkpoint.yaml](continuation-checkpoint.yaml).
- Próximo passo: owner escolher a próxima unidade e sua fronteira de evidência. Recomendação: preparar a conclusão operacional do develop, seguida da primeira experiência manual. Spec executável e autoridade externa continuam pendentes.

## Complemento de 2026-10-08 — bases PostgreSQL e serviços Supabase

- Pergunta do owner: se `mindforge` e `nsrdb` são apenas bases e se apenas `nsrdb` está ligado ao fluxo Supabase. Owner autorizou um agente a consultar a VM via SSH com a credencial indicada, somente para esclarecer essa topologia.
- Agente `/root/infra` reutilizado como folha somente leitura. SSH com verificação estrita de host; inventário de containers e inspeção de configuração processada em memória. Somente nomes de containers/projetos/serviços/imagens, destinos de banco e resolução de aliases foram emitidos. Nenhum SQL, consulta a dados, conexão a `nsrdb`, alteração remota ou reinício ocorreu.
- Evidência capturada em 08/10 às 19:23 (America/Sao_Paulo): `infra-auth-1` (GoTrue) aponta para `db:5432/mindforge`, schema `auth`; o alias resolve para `infra-db-1`. Portanto Mindforge também usa Supabase Auth na infraestrutura.
- Auth e PostgREST ativos do projeto `nsr-supabase-r5-isolated` apontam para `db:5432/postgres`, resolvido para `nsr-supabase-r5-isolated-db-1`. O worker NSR aponta para `supabase-db:5432/postgres`, resolvido para esse mesmo container. Esse PostgreSQL é distinto de `infra-db-1`.
- O nome `nsrdb`, observado historicamente no cluster infra, não é o destino dessas conexões ativas do NSR. Sua existência atual, conteúdo e eventual uso por outros consumidores não foram rechecados. Não atribuir o tráfego Supabase atual a essa base por semelhança de nome.
- Principal leu o registro sanitizado retornado em `/private/tmp/mindforge-supabase-topology-20261008.txt` e cruzou o vínculo Mindforge com o Compose local. Este complemento preserva as observações decisivas caso o arquivo temporário seja removido.
- Limite: configuração de containers ativos comprova os destinos configurados, não uma chamada funcional de Auth/API, saúde completa, dados ou migrations aplicadas. Nenhum endpoint Mindforge teve estado promovido.

## Complemento de 2026-10-08 — viabilidade de Studio para NSR e Mindforge main/develop

- Novo objetivo do owner: visualizar `mindforge` e `mindforge_develop` no Supabase Studio e compartilhar o acesso com o NSR, combinando essa entrega com a pendência operacional do develop. O pedido nesta etapa é de viabilidade, sem execução de mudanças.
- Agente `/root/infra` reutilizado, folha somente leitura. Evidência capturada na VM às 19:29:45 (America/Sao_Paulo), sanitizada e lida pelo principal em `/private/tmp/mindforge-studio-feasibility-20261008.txt`.
- Studio instalado: `supabase/studio:2026.09.07-sha-7996410`; Meta: `supabase/postgres-meta:v0.99.0`. Studio e Meta apontam para o PostgreSQL do NSR e a base `postgres`; nome do projeto padrão `NSR isolated r5`. Não há binding configurado para Mindforge main/develop.
- Studio/Meta/DB do NSR estão em `nsr-supabase-r5-isolated-internal`; DB/Auth Mindforge estão em `infra_default`, sem rede Docker compartilhada entre esses serviços. Os aliases atuais `db`/`meta` do NSR não alcançam o DB Mindforge. Isso localiza uma necessidade de conectividade administrativa delimitada; não demonstra impossibilidade de acesso por todas as rotas possíveis.
- Verificação direta pelo principal no código do commit correspondente à tag instalada: [projects/index.ts](https://github.com/supabase/supabase/blob/7996410/apps/studio/pages/api/platform/projects/index.ts) devolve `[DEFAULT_PROJECT]`; [constants/api.ts](https://github.com/supabase/supabase/blob/7996410/apps/studio/lib/constants/api.ts) define o projeto `default`; [pg-meta tables.ts](https://github.com/supabase/supabase/blob/7996410/apps/studio/pages/api/platform/pg-meta/%5Bref%5D/tables.ts) usa `PG_META_URL` global. A [documentação oficial self-hosting](https://supabase.com/docs/guides/self-hosting) confirma ausência de suporte nativo a múltiplos projetos/organizações no Studio self-hosted.
- [Postgres Meta](https://github.com/supabase/postgres-meta/blob/master/README.md) admite múltiplos bancos em sua API; isso não acrescenta por si só um seletor de projetos/bancos à interface Studio instalada. Trocar apenas `STUDIO_PG_META_URL`/database no serviço NSR repontaria o target existente, em vez de preservar três projetos selecionáveis.

### Opções e recomendação, ainda não aprovadas

| Opção | Viabilidade | Implicação |
| --- | --- | --- |
| Mesmo Studio instalado, seletor nativo NSR/main/develop | Sem suporte por configuração padrão | Não prometer que adicionar databases as fará aparecer como projetos. |
| Studios distintos com acesso comum | Viável como desenho; runtime pendente | Manter NSR, adicionar uma superfície administrativa por alvo Mindforge, normalmente Studio + Meta, e uma página de entrada com links ou hosts distintos. São três painéis, não um seletor de projetos interno. |
| Um Studio customizado multi-project | Viável em princípio, não validado neste estudo | Exige catálogo/routing por projeto e conexão, autenticação administrativa e isolamento de credenciais; responsabilidade por manutenção e compatibilidade. Não é uma pequena alteração de env. |

- Recomendação: combinar a conclusão operacional do develop com superfícies Studio separadas para main/develop e um ponto de acesso comum ao NSR. O escopo inicial é visualizar schemas/tabelas e consultar dados; outras abas Supabase devem ser declaradas fora do recorte ou validadas explicitamente. Não criar novos clusters PostgreSQL nem unificar Auth como consequência da visualização.
- Unidade futura candidata: inventário/preflight atual, entregar operador corrigido, provisionar develop de modo aditivo se ausente, estabelecer seu Auth e aplicar/verificar 0005/0006 em develop; então conectar a administração de main/develop aos bancos correspondentes, identificar ambientes claramente e provar a visualização. A visualização de main não exige promover o schema Brain Core para main; essa promoção continua separada.
- Main deve ter permissão administrativa delimitada; recomendar visualização/leitura como padrão é uma decisão proposta, não autorização para criar roles/grants. Studio permite SQL e mudanças de schema, portanto a prova de acesso deve incluir isolamento entre os alvos.
- Infra r6 exclui Studio/PostgREST explicitamente. O novo requisito precisa de spec integrada aprovada ou amendment que registre essa extensão e sua fronteira administrativa, antes da implementação. O pedido de viabilidade não aprova instalação, rede, acesso público/LAN, SQL, mudanças no NSR ou produção.
- Limites: somente documentação/código upstream e inspeção de configuração/rede na VM; sem SQL, requisição HTTP, interface aberta, instalação ou alteração remota. Não foi rechecado o catálogo/current readiness de develop. Prova de visualização no Studio e prova de comportamento persistente do Brain Core continuam distintas.


## Continuação de 08/10/2026 — fundação entregue e novo planejamento

A unidade develop-studio foi aplicada e validada, e o owner confirmou acesso aos Studios main/develop. A fronteira histórica de provisionamento desta discovery foi superada; [runtime-evidence](../../infra/develop-studio/runtime-evidence.md) registra o que foi comprovado.

Owner escolheu agora a entrega conjunta Auth/API/Flutter, macOS, Google OAuth como única entrada e TDD, autorizando **planejamento apenas**. O [novo checkpoint](../../brain/manual-online/manual-online-checkpoint.yaml) e a [spec proposta](../../brain/manual-online/spec-01-main.md) são os pontos de retomada. Registry passou de vazio a quatro propostas, todas sem aprovação/implementação/runtime; a fundação não foi reimplementada e não houve novo acesso/mutação de VM.

## Retomada de 26/10/2026 — discussão do assistente

### Pedido e limite da investigação

- Owner solicitou spec-drive para compreender o estado atual e continuar. Após a investigação inicial, confirmou que quer priorizar a discussão em aberto do assistente; considera a correção Android já feita e não lembra se houve validação no aparelho.
- Escopo desta unidade: inspeção local de documentos, contratos, código, Git e artefato existente; atualização desta discovery e do checkpoint. Nenhuma implementação ou spec nova aprovada. Nenhuma autoridade externa atual inferida de execuções históricas.
- Fontes obrigatórias de AGENTS lidas; skill spec-drive, referência de continuação e templates consultados. Sem agentes nesta retomada.

### Estado atual verificado

| Superfície | Evidência encontrada | Limite |
|---|---|---|
| Brain Engine/Core | `cmd/brain-engine/main.go` compõe Core, documentos e fachada Auth. `api/handler.go` verifica principal antes do domínio; Core mantém bootstrap/observe/create/rename. | Leitura atual de fonte; não é nova execução HTTP/PostgreSQL. |
| Auth API-only | `api/auth.go` implementa cinco rotas; `auth/facade.go` e `gotrue_sdk_port.dart`/`api_auth_http_client.dart` mantêm GoTrue interno e client restrito à API. | Corte develop de 14/10 registrado em `api-auth/vm-runtime-evidence.md`; callback de sucesso, token/user/logout autenticados seguem sem prova contratual no registry. |
| Documentos nativos | `documents_controller.dart`: autosave local de 1s, sync de 30min/manual, conflitos/resultados incertos. Store local em arquivos; API/Store PostgreSQL implementados. | Consulta real de 14/10 confirmou dois documentos persistidos, um com corpo não vazio. Não houve comparação byte a byte com o rascunho nem prova completa de HTTP/sync. |
| Web | Auth/Core preservados. `client_platform_web.dart` retorna `null` para store documental. | Documentos Web/IndexedDB continuam sem implementação; build Web anterior não comprova essa feature. |
| Correção Android | `document_panes.dart` usa scroll compartilhado; `document_title_dialog.dart` é dono do controller até desmontagem. Testes widget de teclado/fechamento presentes. | Evidência histórica: 125 testes/analyze/APK PASS; aceite no aparelho não encontrado. |
| Assistente | PD-023..025 e `docs/product/assistant/rag-harness-overview.md`, consolidados em 18/10. Sem componente agêntico/MCP/retrieval encontrado no inventário atual. | Direção de produto e propostas; nenhuma spec executável, contrato novo ou implementação aprovada para essa frente. |

O APK existente foi novamente identificado por SHA-256 `ede5314434a57855e0ef26821a24bc196c71bb724f8c9c263de6a6a81b37f911`, coincidente com a evidência de correção Android. Isso confirma o arquivo local, não sua instalação ou execução no aparelho. A atualização tratava do overflow com teclado e do lifecycle do formulário que gerava tela vermelha; não era uma feature nova do assistente.

Registry atual: **15 operações, todas approved/implemented; 1 runtime verified (`auth_authorize`, iniciação 302), 14 pending**. O aceite Web de 09/10 confirma uso manual histórico; a consulta de 14/10 confirma chegada documental. Esses níveis de evidência coexistem com a falta de provas completas por endpoint. Não declarar o produto inteiramente não funcional nem promover operações a verified com base nesses relatos/SELECTs. A URI Google aparece pendente no checkpoint Auth; não foi consultado o Console e não há base para afirmar se o owner já a configurou.

### Discussão a retomar: decisões e propostas

Fontes decisivas lidas diretamente: `docs/product-decisions.md` PD-023..025, `docs/open-decisions.md` OD-017..020 e a visão consolidada do assistente.

**Fechado como direção de produto:**

- Assistente pessoal contínuo, com continuidade administrada pelo Mindforge e possibilidade de diferentes modelos/provedores (PD-023).
- Trava e aviso no aplicativo para troca de modelo durante operações (PD-024).
- Consumir servidores MCP externos e oferecer capacidades Mindforge a clientes autorizados (PD-025).
- Brain Engine continua autoridade; operação manual permanece independente de LLM; Prefrontal Cortex interativo no Flutter e Hidden Cortex durável futuro mantêm seus limites.

**Ainda aberto:**

- OD-017: o que preservar da conversa, autonomia para transformar falas em memória/documentos, confirmação, correções, retenção e exclusões.
- OD-018: escopo/contexto permitido, cobertura e frescor das fontes, rascunho versus servidor, busca/leitura e critérios de avaliação. Chunking/embeddings/ranking não estão escolhidos.
- OD-019: modelos/provedores, capacidades, credenciais, privacidade, custo e comportamento completo de troca/cancelamento/timeout.
- OD-020: primeiro catálogo e recorte MCP, processo/transporte, identidade, autorização, escopos, consentimento e integrações externas.

Estado de conversa portátil, resumos derivados, fluxo interno harness → MCP → Brain Engine e busca híbrida são propostas discutidas. Não tratá-los como mecanismos aprovados. Exemplos de agenda, tarefas e finanças também não significam que esses domínios já existam.

Recomendação para discussão, ainda não decisão: escolher um cenário observável do assistente usando camadas/documentos existentes, fechar leitura e efeitos permitidos, política de memória e envio de dados, então produzir uma spec integrada executável. A lista inteira de ambições não constitui um lote de implementação. Aceite Android fica como pendência conhecida; não é gate automático para discutir o assistente.

### Riscos de continuidade e drift

- Branch atual `feat/brain-core-foundation`, HEAD `0204ba4`, tracking configurado para origin; sem fetch nesta unidade. **276 arquivos não rastreados**, incluindo client Flutter inteiro e módulos API/Auth, além de 44 paths tracked alterados/removidos. Parte substancial da evolução está apenas no checkout local. Não houve autorização de stage/commit/push.
- Checkpoint geral ainda apontava para develop-studio/manual-online. A direção documental mais recente, de 18/10, não aparece no ranking de checkpoints. Ponteiros gerais foram atualizados nesta unidade, preservando histórico.
- `skillr resume` não incluiu `ui-fixes-checkpoint.yaml` como tarefa distinta: a convenção reconhecida é task-prefixed e esse arquivo compartilha o diretório Android. Leitura direta identificou a correção posterior. O checkpoint Brain Core tem `remaining` duplicado e Web não tem `updated_at` suportado; não se fez migração documental geral.
- A arquitetura conserva frases históricas de ausência de client; stack/checkpoints antigos conservam pendências superadas. Fontes do recorte e evidências posteriores prevalecem; nenhuma divergência de contrato/código/consumidor foi estabelecida por esta retomada para abrir `needs_decision` de endpoint.
- Consulta histórica encontrou um Brain sem usuário Auth correspondente, distinto dos documentos coerentes. Causa não estabelecida; limpeza não autorizada e fora da prioridade escolhida.

### Validação desta retomada

- Executado: inventário/leitura de fonte e artefatos; `git status --short --branch`, `git log`, `git diff --stat`, `git ls-files`; bundled `skillr resume --docs /Users/arcknin/Desktop/mindforge/docs --limit 8 --json`; parse Ruby/YAML e contagem dos 15 endpoints; SHA-256 do APK; `git diff --check` antes das alterações documentais.
- Testes, analyze, builds, HTTP, Google Console, aparelho, SSH/VM, SQL, serviços, instalação, deploy, stage/commit/push **não executados nesta retomada**. Resultados históricos continuam identificados como históricos.
- Somente esta discovery e `continuation-checkpoint.yaml` alterados pela retomada. Contratos/endpoints e seus estados permanecem intactos.
- Fechamento: YAML do checkpoint, `spec: null` e existência dos caminhos referenciados PASS; `git diff --check` PASS. `skillr resume` filtrado por `project/continuation` reconheceu o estado/ponteiro atual sem incertezas do candidato; aviso legado de chave duplicada no Brain Core preservado.

### Continuação atual

- Prioridade escolhida pelo owner: **retomar discussão do assistente**.
- Próxima discussão: recuperar os exemplos de uso da visão consolidada e escolher o primeiro comportamento verificável, preservando decisões já fechadas. Em seguida fechar OD-017..020 apenas no necessário para esse recorte.
- Não há spec executável do assistente criada/aprovada; TDD dessa nova frente continua `undecided`. Nenhuma implementação/externalidade inferida deste diagnóstico.
- Checkpoint: [continuation-checkpoint.yaml](continuation-checkpoint.yaml). Referência de discussão: [visão consolidada](../../../product/assistant/rag-harness-overview.md).

## Publicação Git autorizada em 26/10/2026

- Owner pediu commit e push das alterações acumuladas, classificação por subagente e documentação em um único commit. Isso autoriza publicação Git na branch atual; não autoriza merge, deploy, instalação ou qualquer efeito na VM/banco.
- Subagente folha `/root/classificar_commits`, solicitado `gpt-6-luna`/`xhigh`, classificou o inventário somente por leitura. Principal conferiu as dependências decisivas nos testes Brain/manifesto e catálogo/hash de Documents, fontes HTTP e estrutura do client. Nenhum reviewer ou gate adicional criado.
- Divisão escolhida: (1) persistência/SQL/Core/Documents/operador e `.gitignore`; (2) API/Auth/composição Go e YAMLs funcionais OpenAPI/registry; (3) serviços/configuração/ferramentas Auth/Studio/Web; (4) client Flutter inteiro; (5) documentação inteira, incluindo `docs/**`, AGENTS, READMEs/runbooks e licenças/avisos. SQL de apoio sob `docs/` permanece documental. Total: 320 paths, distribuídos em 71/17/38/123/71.
- A fronteira persistente foi agrupada porque Brain tests leem o manifesto infra, o catálogo global lê SQLs Brain0006/Documents0007 e testes verificam seus hashes. Não se tentou reconstruir a cronologia histórica do client. Checks de higiene leem documentação; validação integral refere-se à árvore final, sem promessa de que cada commit intermediário isolado passe todos os checks.
- Inspeção de candidatos não encontrou nomes de configuração privada, symlinks ou arquivos acima de 1MB. Correspondências de URLs com credenciais foram conferidas como placeholders/fixtures/formatação de validadores. Configurações privadas, builds/APK e caches não entram no inventário. O registro SSH em AGENTS é preexistente neste checkout e traz instrução explícita do owner para sua inclusão; não foi usado para acesso remoto.
- Alteração auxiliar desta unidade: `.gitignore` passou a ignorar `__pycache__/` e `*.py[cod]`, incluindo os dois caches gerados pelas verificações Python. Sem alteração funcional do produto.
- **Verificação atual:** `go test ./...` PASS; infra `static all` e `unit all` PASS; Auth infra 23, Web 9, Studio 12 e package 4 testes PASS; Flutter **125 PASS**. Testes Go HTTP exigiram liberação de portas loopback; não houve banco real. Flutter inicialmente falhou por `package_config.json` apontar para SDK temporário removido; `flutter pub get --offline --enforce-lockfile` regenerou o cache sem atualização de dependências e o rerun passou. Nenhum build/analyze novo nesta unidade.
- `git fetch --no-tags origin feat/brain-core-foundation` confirmou base local/remota coincidente em `0204ba4` antes dos commits. Publicação prevista por push normal à mesma branch, sem force, merge ou rebase. A prova final de commits/push pertence ao histórico Git e à conferência da referência remota; não é evidência de runtime do produto.
- Commits locais de implementação: `782a394` persistência/SQL/operador (71 paths), `ce46b14` API/Auth/contratos (17), `05284ff` serviços Auth/Studio/Web (38), `2c17115` client Flutter (123). Conteúdo de cada path conferido contra snapshot anterior aos commits; exatamente os 71 paths documentais restaram para o commit único. Nenhuma mudança funcional introduzida durante a classificação/publicação.
- Check do índice identificou linha vazia final preexistente em `documents/migrations/0001_documents.sql:101` e, no grupo documental, em `src/mindforge-client/assets/fonts/LICENSE-APACHE-2.0.txt:197`. Bytes preservados, incluindo SHA-256 canônico SQL `37c0378f6a4d18e98505e31baf50ebb8e36da071868beae05e4631a92fed351f` e seu catálogo; check de whitespace sem a regra `blank-at-eof` PASS. Demais grupos de implementação passaram o check padrão do índice.
- Depois da publicação Git, a próxima frente permanece a discussão do assistente, PD-023..025 e OD-017..020. Nenhuma spec do assistente ou novo endpoint foi aprovado por essa publicação.
