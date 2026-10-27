# Discovery — conteúdos das camadas e usos de Funções Cognitivas

## Pedido e limite

Owner solicitou registrar decisões e montar o planejamento de conteúdo Markdown e dados de Funções Cognitivas, cada uso com estado próprio. Em seguida esclareceu que o texto será guardado como Markdown e interpretado pelo client como `.md`; adiou a escolha/detalhamento da primeira função: “daqui a pouco debatemos melhor, só registra o conceito”. Unidade exclusivamente documental, em 2026-10-11.

## Estado verificado

- Monorepo em `/Users/arcknin/Desktop/mindforge`, branch `feat/brain-core-foundation`, com alterações anteriores extensas preservadas.
- Leitura obrigatória realizada na ordem do AGENTS: arquitetura, stack geral/front/back e OpenAPI README/registry.
- `src/brain-engine/features/brain/engine.go`: bootstrap, observar hierarquia, criar e renomear camada. `model.go`, `ports.go` e migration canônica delimitam Brain/root/layers; não há modelo contratado de documentos ou usos de funções nesse Core.
- `src/mindforge-client/lib/ui/workspace_screen.dart` e `brain/`: árvore e operações manuais de camadas, sem editor de conteúdo ou catálogo/estado de funções implementado.
- `contracts/openapi/registry.yaml`: quatro endpoints aprovados/implementados de Brain Core; `runtime_status: pending`. Não há endpoints de conteúdo ou funções aprovados.
- `../web-local/web-local-checkpoint.yaml`: owner confirmou login, criação/leitura da hierarquia e rename em três níveis, incluindo nomes repetidos. Conflito concorrente, sessão e prova HTTP detalhada continuam separados dessa aceitação.
- `docs/product-decisions.md` PD-001..008 e conceitos de produto preservam um Cérebro, hierarquia livre, autoridade do Brain Engine, seleção granular e distinção função/capability. Atividades, Metas e Pontos continuam conceitos exploratórios em seus domínios.

## Decisões novas e evidência

PD-009 registra conteúdo Markdown + dados funcionais como contexto da camada, com formato inicialmente fechado; PD-012 posteriormente definiu texto PostgreSQL. PD-010 registra funções desenvolvidas pela equipe, uso em qualquer camada e combinação das existentes pelo usuário. PD-011 registra definição reutilizável separada de cada uso com configuração/estado próprios. Fonte de autoridade: mensagens explícitas do owner nesta conversa; registradas no ledger, não inferidas de memória ou de exemplos arquiteturais.

O exemplo Estudos > Curso de investimento inclui anotações, tarefas, sessões/horas de foco, pontos e agendamentos. Finanças em duas camadas exemplifica estados separados. Isso não seleciona a primeira função nem define o schema desses dados.

## Questões deliberadamente abertas

- Markdown: formato confirmado; variante, schema, regras de título, editor, operações, concorrência, histórico, limites, anexos e importação/exportação ainda abertos.
- Usos: associação a camada e separação de estado confirmadas; múltiplos usos da mesma função na mesma camada e lifecycle ainda abertos.
- Funções: primeira função, campos/regras e mapeamento para capabilities adiados pelo owner.
- Composição: mecanismo manual/eventos/automação e efeitos ainda abertos; nenhuma engine genérica de workflows decidida.
- Contexto: pertencimento de dados não estabelece herança de subcamadas nem envio automático de todo o histórico ao agente.

## Continuação

Spec de planejamento: [spec-01-main.md](spec-01-main.md). Checkpoint: [layer-content-checkpoint.yaml](layer-content-checkpoint.yaml). Próxima conversa deve detalhar os pontos que o owner escolher, sem novas perguntas obrigatórias nesta entrega. Nenhum worker necessário para este registro documental; nenhuma operação externa realizada.

## Refinamento de documentos confirmado pelo owner

Em 2026-10-11, o owner confirmou registro da proposta discutida: corpo Markdown como texto no PostgreSQL; client interpreta; ID estável/título; intenção por wikilink resolvida para referência por ID com desambiguação; Brain Engine valida e oferece relações; grafo derivado de documentos/links. PD-012/013 registram essas decisões. Storage do corpo e estabilidade de identidade estão fechados; schema, sintaxe exata de links e lifecycle detalhado permanecem em OD-012/015. Não houve pesquisa externa ou alteração de código/contratos/VM nesta atualização.

## Refinamento do escopo documental e salvamento local

Owner confirmou consulta de todos os documentos descendentes a partir do pai (PD-014), operações documentais incluindo move/delete/restore, editor texto+preview e links sinalizados sem auto-create; grafo visual adiado (PD-015). Solicitou autosave local1s e direção futura de upload30min, depois escolheu expressamente só local nesta entrega (PD-016). Isso altera o antigo pressuposto online-only para documentos locais pendentes, sem alterar a autoridade remota do Brain Engine. PostgreSQL permanece destino da fase posterior.

Primeira etapa precisa planejar persistência no navegador, IDs antes de sync, usuário/origem, conflitos entre abas e lixeira local. IndexedDB é uma proposta técnica, não decisão do owner. Fontes primárias consultadas: MDN IndexedDB, setTimeout em abas inativas e quotas/eviction, links no spec. Nenhuma mudança no código, contrato HTTP, banco ou VM.

## Refinamento em 2026-10-12

Owner aceitou lixeira local10dias (PD-017), delegou ao principal a solução técnica de conflitos entre abas e sugeriu sync no logout. Após esclarecer o impacto, escolheu manter fase inicial local e adiar o gatilho de envio (PD-018). O principal detalhou controle transacional/revisão em IndexedDB, rascunhos por aba e invalidação opcional BroadcastChannel, sem código ou dependências instalados. Transações/commit e canal entre contextos foram conferidos nas fontes MDN vinculadas na spec. Conteúdo confirmado é mantido por conta após logout; prazo local240h e purge na próxima execução não equivalem a serviço de background.

## Referência Obsidian confirmada em 2026-10-12

Owner confirmou que o nome dos links acompanha renomeações e pediu experiência semelhante ao Obsidian básico (PD-019). Conferidas as fontes oficiais de links, gestão de notas e modos do editor, vinculadas na spec revisão6. A referência orienta o comportamento dentro do recorte: autocomplete de destino, navegação, rename e move preservando vínculo. Não substitui IDs/títulos repetidos/consulta recursiva/autosave local/lixeira e não amplia automaticamente para plugins, Live Preview, anexos ou filesystem. Esta atualização só altera documentação; sem pesquisa na VM, código ou testes de runtime.

## Backend antecipado em 2026-10-12

Owner autorizou backend com TDD, confirmou API/PostgreSQL antes do client/sync e aprovou especificamente seis operações HTTP e lixeira remota240h. Fontes atuais: Brain Core Store/pgx, handler estrito com Auth GoTrue e runtime develop failclosed; não há documentos implementados no início deste lote. Spec02 separa persistência/prova de backend da spec01/client. Não há autorização para SQL/VM/deploy; fonte PostgreSQL será validada estaticamente e com testes de adapter, integração real pendente. Workers folha com ownership exclusivo domínio/parser e adapter/SQL; principal HTTP/runtime/contratos.
