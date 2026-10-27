# Histórico de planejamento — revisões1..6

Registro histórico, não normativo e sem autoridade de execução. Decisões atuais: [spec01](spec-01-main.md) e [backend spec02](spec-02-backend.md). Propostas antigas e pendências aqui não reabrem decisões posteriores.

---
schema_version: 1
managed_by: spec-drive
feature: brain
task: layer-content
revision: 6
status: draft_planning_only
---

# Planejamento integrado — conteúdos das camadas

## Contexto e autoridade

Esta unidade registra conceitos confirmados e propõe uma sequência para seu futuro detalhamento. O owner autorizou **somente planejamento**, depois solicitou registrar o conceito de funções e deixar a escolha/detalhamento da primeira para outra conversa. As decisões PD-009..019 estão aprovadas como direção de produto; este documento não é um contrato técnico congelado nem autorização de execução.

Fontes: [discovery](discovery.md), [decisões de produto](../../../product-decisions.md), [questões abertas](../../../open-decisions.md), [arquitetura](../../../architecture-overview.md), [Core atual](../brain-core/brain-core-contract.md), [aceitação web](../web-local/web-local-checkpoint.yaml) e [registry](../../../../contracts/openapi/registry.yaml). Estado atual: Core organiza camadas; conteúdo e usos de funções ainda não possuem implementação ou contratos HTTP.

## Objetivos

1. Planejar a camada como contexto organizador de documentos Markdown e de dados de usos de funções.
2. Preservar o texto Markdown como conteúdo; o client o interpreta/apresenta como `.md`.
3. Separar definição reutilizável de função, uso associado a uma camada e registros/estado desse uso.
4. Permitir o uso das funções em camadas escolhidas pelo usuário, sem limitar pela semântica dos nomes.
5. Preservar autoria humana/agêntica autorizada, operação manual e autoridade do Brain Engine.
6. Planejar primeiro o editor documental local no client web; contratos/armazenamento remoto e sincronização entram em etapa posterior.

## Fora desta unidade

Código, testes de código, SQL/migrations, schema, VM, instalação, builds/deploy, edição de dados reais, novos endpoints ou alterações nos quatro existentes, commit/push. Também não se escolhem a primeira função, seus campos/regras, schema físico, workflow engine, execução de agentes, LLM/retrieval/embeddings, sincronização com servidor, Inbox, Hive Mind ou purge automático sem definição do lifecycle da lixeira.

## Modelo conceitual confirmado

| Conceito | Responsabilidade | Separação necessária |
| --- | --- | --- |
| Camada/subcamada | Organizar documentos e usos funcionais que pertencem àquele contexto | Nome não determina funções permitidas; árvore não é diretório físico obrigatório |
| Documento Markdown | Preservar conteúdo textual produzido pelo usuário ou agente autorizado | Corpo Markdown como texto no PostgreSQL; ID estável e título separado; schema ainda aberto |
| Definição de função | Capacidade reutilizável desenvolvida pela equipe | Não contém um único estado compartilhado por todos os usos |
| Uso de função | Aplicação da definição em uma camada, com configuração e estado próprios | Separado dos usos dessa definição em outras camadas |
| Registros do uso | Dados produzidos/governados pela função naquele uso | Campos, operações e regras serão definidos por domínio |
| Contexto consultado | Recorte autorizado de documentos e dados pertinentes | Não equivale automaticamente à totalidade dos dados nem funde estados |

“Uso” é a expressão de planejamento; “instância” não é um nome de UI/schema decidido. Uma camada pode ter documentos e funções diferentes. A quantidade de usos da mesma função na mesma camada permanece aberta (OD-011).

### Exemplo ilustrativo

```text
Estudos
└── Curso de investimento
    ├── Documentos em Markdown
    ├── Uso de uma função de tarefas → dados de tarefas desse uso
    ├── Uso de uma função de foco → sessões/tempo desse uso
    ├── Uso de uma função de pontos → dados de pontos desse uso
    └── Uso de uma função de agendamento → configurações/registros desse uso
```

Finanças pode ser usada em `Finanças` e em `Estudos`: definição reutilizada, configurações e estados separados. O exemplo não aprova um catálogo nem obriga todo usuário a ativar essas funções. Tarefas não é automaticamente um novo nome para o conceito exploratório de Atividades.

## Frentes do planejamento futuro

### A. Documentos Markdown

**Confirmado:** corpo textual Markdown armazenado como texto no PostgreSQL, preservado e interpretado no client, com ID estável separado do título. Nesta primeira entrega, o owner escolheu documentos locais com autosave de um segundo; o corpo PostgreSQL é o destino da etapa remota posterior. Conteúdo de agentes dependerá de autorização e integração futuras.

**Operações escolhidas:** listar, criar, abrir, editar, renomear, mover entre camadas, excluir para lixeira e restaurar documentos. São operações da primeira experiência local; não são novas operações de camadas do Brain Core. Prazo de lixeira de 10 dias está decidido em PD-003, mas clock, início, restauração e purge precisam ser fechados antes da implementação.

**Decisões técnicas futuras:** representação do ID e vínculo com camada; schema do corpo no PostgreSQL; variante Markdown; regras de título/conteúdo vazio; tamanho; engine local e durabilidade; concorrência/histórico; autoria/proveniência; detalhes de links/anexos; refinamentos do editor texto + preview já escolhido. Arquivo no Mac, import/export e acesso do client ao filesystem do servidor não foram aprovados.

**Riscos a tratar no contrato:** edição concorrente não deve perder alterações silenciosamente; conteúdo inválido/limites devem produzir falha compreensível e preservar rascunho. Renderização precisa definir tratamento de HTML, links e conteúdo ativo. Texto recebido por agente é conteúdo, não concessão de ferramentas ou permissão para executar instruções.

Uma checkbox escrita no Markdown não é, por si só, uma tarefa autoritativa; uma eventual conversão/vinculação deve ser explícita. O mesmo vale para valores financeiros, pontos e horários mencionados no texto.

### A.1 Links internos e grafo — direção confirmada

O documento possui ID estável, título e corpo Markdown. A intenção `[[Título]]` deve ser resolvida para um destino cuja identidade seja conservada; títulos duplicados exigem seleção explícita. O nome é um rótulo, não a chave do vínculo. A sintaxe exata de referência por ID/rótulo e o momento de canonicalização continuam abertos.

Na etapa remota, o Brain Engine valida referências e acesso. Nesta primeira etapa local, o client resolve/renderiza/navega referências aos documentos disponíveis localmente, sem afirmar validação remota. A visualização do grafo foi adiada; sua direção futura continua documentos e relações derivadas de referências, inclusive entre camadas quando houver acesso.

O detalhamento precisa preservar os IDs no conteúdo/referências canônicos para que rename não quebre vínculos nem uma reconstrução do grafo passe a resolver apenas títulos antigos. Escolher a representação dessa relação, consistência da gravação texto/referências e estratégia de índice/projeção é trabalho posterior, sem criar duas fontes independentes da verdade.

OD-015 concentra links pendentes/quebrados, rótulos após rename, exclusão/restauração, seleção do grafo e navegação. Não se escolhe renderer, biblioteca de grafos ou banco de grafos; não se inclui automaticamente todo registro funcional como nó.

### B. Definições, usos e estados

**Confirmado:** definições desenvolvidas pela equipe; usuário escolhe/configura/combina as existentes; cada uso possui estado próprio e pertence ao contexto de sua camada.

**Intenções a discutir:** descobrir funções disponíveis, iniciar um uso na camada, observar sua configuração/estado e executar as operações autorizadas da função. Alterar configuração, desativar, copiar, mover ou remover usos requer lifecycle próprio, sem pressupor descarte de dados.

**Propostas técnicas para discutir, não decisões fechadas:** referências estáveis de definição, uso e camada; registros vinculados ao uso; separação entre configuração e dados; versão da definição/compatibilidade; validação tipada pelo domínio. Evitar um armazenamento genérico de payload arbitrário como substituto das regras de cada função. Nenhuma tabela, representação JSON ou framework de plugins está escolhido. Esta frente funcional permanece adiada pelo owner.

**Regra de separação confirmada:** uma mudança em um uso não altera implicitamente o estado de outro. Consultas ou combinações entre usos precisam de escopo e autorização explícitos. A associação com uma camada não concede acesso às camadas de outro usuário.

### C. Dados possíveis por domínio

| Família ilustrativa | Informação esperada | Decisões específicas adiadas |
| --- | --- | --- |
| Tarefas/atividades | Ações e andamento/conclusão | Relação Tarefas/Atividades, estados, prazos, recorrência, operações |
| Foco/estudo | Sessões e tempo dedicado | Início/pausa/fim, registros manuais, cálculo de duração e totais |
| Pontos | Pontuação relacionada às atividades | Origem, regras, concessão/estorno, duplicação, ledger ou outra representação |
| Finanças | Registros e organização financeira de cada uso | Tipos, categorias, moeda, precisão, saldos e validação |
| Agendamentos/notificações | Quando uma ação permitida deve ocorrer | Clock/fuso, recorrência, cancelamento, destinatário, retries e deduplicação |

Essa matriz explica classes de dados, sem campos ou catálogo normativos. Agendamento não significa liberar shell/cron arbitrário. Totais, resumos e projeções precisam ser distinguidos dos registros de origem no domínio correspondente. Nenhuma dessas funções foi escolhida como a primeira.

### D. Contexto e composição

Conteúdo de documentos e dados funcionais pode participar do contexto da camada. O planejamento posterior deve definir seleção direta/subárvore/grupo, permissões, projeções, limites e referência às fontes. PD-014 define consulta documental recursiva: selecionar Trabalho permite consultar seus documentos e os dos descendentes. Isso amplia a disponibilidade de contexto documental sem copiar registros nem tornar obrigatório enviar toda a subárvore ao agente. A seleção de atuação de funções de PD-005 permanece distinta e adiada.

Combinar funções existentes é a direção aprovada; sua mecânica permanece aberta. “Concluir tarefa e conceder pontos” é apenas um exemplo para debate, sem trigger, fórmula ou automação definidos. Qualquer futura composição deve especificar autorização por efeito, falha parcial, repetição/duplicação e rastreabilidade. Não construir uma engine universal de automação como pré-requisito implícito para conteúdo manual.

## Proposta de sequência, sem execução

| Passo | Trabalho de planejamento | Saída necessária para avançar |
| --- | --- | --- |
| P0 — Conceito | Registrar PD-009..011, limites e questões abertas | Concluído nesta unidade; owner pode retomar a discussão por qualquer frente |
| P1 — Conteúdo textual | Detalhar identidade/vínculo, operações/UX, schema textual e referências internas do Markdown | Comportamento de documentos e aceite definidos; storage textual já escolhido; schema e consistência de referências definidos |
| P2 — Usos de funções | Fechar associação/cardinalidade, configuração, separação de estado e lifecycle mínimo | Modelo de uso compreensível, sem antecipar campos de funções ainda não escolhidas |
| P3 — Um domínio concreto | Owner escolher uma função; depois detalhar registros e regras dessa função | Um exemplo real valida o modelo de uso; escolha adiada pelo owner |
| P4 — Contratos e entrega | Planejar APIs, persistência e client para o recorte escolhido | Propostas de endpoints no registry, decisão do owner, OpenAPI somente depois da aprovação; lotes limitados e autoridade explícita |
| P5 — Contexto/composição | Definir uma consulta ou combinação concreta quando necessária | Escopo/dados/efeitos/aceite explícitos, sem tornar todos os mecanismos futuros pré-requisitos |

Sequência ajustada pelo owner, ainda **sem autorização de implementação**: primeiro editor/documentos locais no client web aproveitando login e árvore existentes; depois contrato documental remoto e sincronização via Brain Engine/PostgreSQL. Funções Cognitivas permanecem para outra entrega. A infraestrutura comum de usos deve nascer das necessidades desse domínio, sem construir todos os módulos previstos de antemão. A ordem P1/P2 pode ser conciliada na discussão; não se congela uma sequência técnica antes das decisões.

## Critérios e validação planejados

### Aceite documental desta unidade

- Decisões do owner registradas separadamente de propostas técnicas e exemplos.
- Markdown e dados funcionais contemplados; definição/uso/estado/contexto distintos.
- Mesmo tipo de função em camadas diferentes com estados independentes; combinação sem autoria de funções executáveis pelo usuário.
- Primeira função não escolhida; storage do corpo em texto PostgreSQL registrado; schema e demais dependências localizáveis.
- Fontes e links válidos; checkpoint consistente; nenhum artefato de execução modificado.

### Cenários a detalhar em contratos posteriores

1. Criar/salvar/abrir um documento local conserva o texto Markdown, ID estável e vínculo com uma única camada; consultar o pai reúne documentos diretos e dos descendentes sem duplicar.
2. Renomear camada ou documento conserva a associação por identidade; mecanismo técnico a decidir.
3. Usos da mesma definição em duas camadas mantêm configuração e registros separados.
4. Dois usuários não acessam conteúdo/usos privados um do outro; seleção não amplia ownership.
5. Tentativas concorrentes de edição/configuração não sobrescrevem mudanças silenciosamente; estratégia a decidir.
6. Falha de rede preserva rascunho e distingue resultado incerto de rejeição antes de reenviar uma mutação.
7. A função concreta valida os próprios registros; texto Markdown não aplica efeitos de domínio automaticamente.
8. Composição e consulta agregada usam somente os usos/camadas selecionados e autorizados.
9. Documento de agente tem os mesmos limites de validação/autoridade que uma solicitação humana; conteúdo não executa código.

TDD: manter a preferência já confirmada pelo owner (`yes`) no futuro recorte implementável; nenhum teste de código é escrito/executado neste planejamento. Prova local, integração VM e aceite real continuarão separados. SQL somente na VM indicada, após autorização específica, sem banco descartável/local.

## Arquivos, automação e limites

Alterações desta unidade: ledger PD, backlog OD, suplementos nos conceitos de camadas/funções e este diretório de planejamento (discovery/spec/checkpoint). Ownership: principal; não houve delegação, novos serviços, scripts de produto, dependências ou ferramentas de automação. Validação documental por YAML, links, consistência e comparação de hashes do código/contratos; não substitui testes de implementação.

Novos endpoints não foram propostos nesta unidade: intenções de produto não são rotas HTTP. Registry e OpenAPI permanecem com os quatro contratos atuais e estados inalterados. Nenhum SQL, novo catálogo técnico, campo durável, API genérica de estado ou automação foi criado.

## Aprovações e retomada

- Direção de produto: PD-009..019 aprovadas pelo owner.
- Autoridade atual: registrar e planejar apenas; código, DB/VM, build/deploy, commit/push não autorizados nesta unidade.
- Planejamento técnico: rascunho para discussão, sem congelamento ou aprovação executável.
- Escolha da primeira função: expressamente adiada; não solicitar novamente nesta entrega.
- Próximo passo: retomar uma frente com o owner, usando OD-008/012/015/016; não executar a proposta automaticamente.
- Checkpoint: [layer-content-checkpoint.yaml](layer-content-checkpoint.yaml).

### Validação documental realizada

YAML/autoridade do checkpoint, links locais, IDs únicos PD/OD e `git diff --check` passaram. Comparação de hashes e conjunto de 209 paths em `src/` e `contracts/` confirmou preservação integral. Nenhum teste de código, SQL, VM ou runtime executado.

### Refinamento confirmado em 2026-10-11 — revisão 2

Owner aprovou registro de corpo Markdown em texto PostgreSQL, identidade estável/título, wikilinks resolvidos para IDs, validação pelo Brain Engine e grafo derivado. PD-012/013 e OD-012/015 sincronizados. A unidade continua somente documental; schema, sintaxe exata de IDs e operações não foram congelados. Primeira função continua adiada.

## Escopo ativo após direcionamento do owner — documentos Markdown

Owner escolheu focar documentos `.md` e adiar Funções Cognitivas. As frentes B/C e a escolha da primeira função ficam como referência futura, sem bloquear o fechamento da entrega de documentos. A próxima discussão trata OD-012 e OD-015: pertencimento/títulos, operações incluídas, editor/dialeto, salvamento/concorrência/histórico, sintaxe/resolução/lifecycle dos links e alcance da primeira visualização do grafo. As opções/recomendações apresentadas na conversa ainda não são decisões aprovadas. Implementação continua não autorizada.

## Decisões mais recentes do owner — revisão 4 (prevalecem sobre propostas anteriores)

### Organização e consulta

Exemplo lógico: Trabalho > Nsr-project > docs.md. Cada documento pertence diretamente a uma camada, possui ID estável e admite título repetido com identidade distinta. A visualização do pai permite consultar todos os documentos próprios e de descendentes, preservando a camada de origem. Primeira etapa consulta os documentos disponíveis no armazenamento local e a árvore conhecida; isso não promete carregar documentos do servidor ou camadas ainda não disponíveis localmente.

UI de documentos separada da área futura de usos funcionais. A consulta herdada não copia dados nem mistura configurações/estados das funções. Escopo de propriedade continua privado; o pai não recebe acesso a conteúdo de outro usuário por existir um link. Para agentes, disponibilidade de contexto da subárvore e seleção do que efetivamente enviar continuam distintas.

### Operações, editor, links e adiamentos

Escolhidos criar/listar/abrir/editar/renomear/mover/excluir/restaurar documentos. No editor, texto Markdown com prévia, incluindo títulos, listas, tabelas, código e links; refinamentos posteriores permitidos. Criar a intenção `[[...]]`, escolher o destino por ID e sinalizar referência sem destino sem criar documentos automaticamente está confirmado. PD-019 fecha que o nome padrão exibido acompanha renomeações; a sintaxe canônica e sua consistência com referências por ID ainda serão detalhadas.

Visualização do grafo adiada; navegação e identidade das referências continuam no recorte. Funções Cognitivas e estado/UI de seus usos adiados. Mover/excluir/restaurar camadas não entra por inferência: o Core continua com suas quatro operações aprovadas.

### Autosave local e servidor posterior

Owner escolheu expressamente nesta entrega apenas autosave local em cadência de um segundo. A direção de sincronização automática a cada 30 minutos pertence à etapa posterior; botão de sync também adiado. Não criar scheduler de upload, schema/API remota de documentos ou cache fingindo sincronização nesta etapa.

Texto/título/identidade/vínculo e metadados necessários às operações locais precisam ser preservados como unidade consistente; sua representação será definida no contrato local. Salvo localmente significa escrita concluída no storage, não apenas alteração em memória. O client deve distinguir gravação pendente, gravação local confirmada e falha de gravação. Não declarar que o servidor contém alterações locais.

**Decisão técnica do planejamento, refinada na revisão 5 por delegação do owner:** usar IndexedDB para documentos e metadados persistentes no client web, separado do sessionStorage de tokens/PKCE. IndexedDB oferece transações assíncronas e índices; a origem do site delimita o armazenamento. Nenhum pacote/adaptador foi escolhido. Fonte: [MDN IndexedDB](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API).

**Limitação real da plataforma:** timers podem ser atrasados/suspensos em abas inativas e não executam com app fechado; 1 segundo é cadência nominal durante execução, não garantia de tempo real ou escrita após crash. Definir flush oportunista ao sair/trocar documento sem depender dele como única proteção; só marcar salvo após commit. Fonte: [MDN timers](https://developer.mozilla.org/en-US/docs/Web/API/Window/setTimeout#timeouts_in_inactive_tabs).

**Durabilidade a planejar:** navegador controla quota e descarte; erro/quota precisa ser visível, mantendo conteúdo ainda em memória e sem falso “salvo”. Limpeza de dados, navegação privada e mudança de origem podem afetar os documentos locais. Não prometer backup remoto antes da futura sincronização. Fonte: [MDN storage](https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria).

### Decisões necessárias antes de congelar o contrato local

1. Engine/adaptador de storage local e schema; namespace por usuário autenticado, ambiente e origem; recuperação após reabrir a aba. Logout e troca de usuário devem ser definidos para não perder pendências nem exibir conteúdo entre contas.
2. Identidade do documento antes do primeiro sync: proposta UUID estável local, cuja aceitação, idempotência e colisões remotas serão contratadas depois. Não decidir um ID descartável que quebre vínculos na sincronização futura.
3. Coordenação/versão entre abas definida na revisão 5: transações, revisão local e rascunhos preservados. Schema/adaptador e limites ainda serão detalhados; histórico navegável não foi escolhido.
4. Lixeira local detalhada na revisão 5/PD-017: 240h, clock localUTC, restore antes do vencimento e purge na próxima execução. Evolução para clock remoto e sincronização da exclusão continuam posteriores.
5. Move/restore: destinos válidos, referências por ID, recuperação quando camada de origem/destino não estiver disponível. Não permitir operação entre usuários/Brains ou criar uma camada implicitamente.
6. Fonte e tamanho do documento, título vazio/repetido, Markdown/links e renderização segura; resolução local de documentos que só existirão em outro dispositivo/servidor na etapa futura.

### Aceitação planejada para a fase local

- Em execução normal, alterações são gravadas localmente na cadência escolhida; confirmação só após commit. Reload/reabertura recupera o último commit e falha de storage é sinalizada.
- A consulta de Trabalho reúne documentos diretos e os de Nsr-project/descendentes, sem cópias. Move muda o pertencimento e as consultas herdadas, preservando ID e links.
- Excluir/restaurar obedece ao lifecycle de lixeira que ainda será fechado, sem modificar estados de funções nem operações do Core.
- Títulos repetidos exigem escolha no link; destino ausente é sinalizado, sem criação automática. Rename não rompe o vínculo por ID.
- Alterações em uma conta não aparecem em outra; as pendências locais e futuras referências remotas são tratadas explicitamente.
- Nenhum upload documental periódico/manual, grafo visual, função cognitiva ou editor de regras é acionado nesta fase.

O recorte continua apenas planejado. Documento local não equivale a suporte offline completo: autenticação e carregamento inicial de camadas existentes podem continuar online. Contrato local primeiro; contrato remoto/sync posterior. Novos endpoints/migrations continuam sem proposta ou autorização nesta unidade.

## Fechamento de lixeira, abas e logout — revisão 5, 2026-10-12

### Lixeira local

Aplicar PD-017: exclusão é uma transação local que mantém corpo/título/ID/camada de origem e marca `deleted_at`/`expires_at` (nomes conceituais, não schema aprovado). Prazo: 240 horas desde o commit, timestamps UTC do dispositivo, sem renovação ao excluir item já excluído. Documento na lixeira sai da listagem ativa e dos resultados/contexto ativo dos pais. Link por ID continua sem auto-criação; destino fica indisponível enquanto excluído.

Restore antes do vencimento preserva identidade, conteúdo e referências, removendo o estado de lixeira. Usar a camada de origem se disponível; caso indisponível, preservar o item e solicitar um destino válido, sem inventar camada ou descartar texto. Restore no instante ou depois do vencimento é rejeitado. Purge físico do corpo, rascunhos/cópias e metadados descartáveis ocorre na próxima execução elegível (startup/acesso à lixeira/ciclo de manutenção enquanto ativo), não em background com app fechado. Excluir após um restore válido inicia um prazo novo.

A remoção precisa ser atômica e condicionada à revisão corrente para não apagar documento que outra aba restaurou. Antes de purgar, reavaliar estado/revisão/prazo na transação. Auto-save de aba desatualizada nunca cria novamente um registro removido; tratar como documento indisponível, preservando a edição ainda não gravada sem ressuscitar o ID. A futura sincronização deve definir tombstones e reconciliação remotos separadamente.

O clock local é referência nesta fase; não transformar isso em alegação de clock confiável no servidor ou de apagamento físico exato em 10 dias. Não apagar bancos/caches da conta inteira para limpar a lixeira.

### Coordenação entre abas — decisão técnica do principal

Para o planejamento web, adotar IndexedDB como armazenamento transacional de documentos/rascunhos e metadados; pacote/adaptador Dart e schema físico serão definidos no contrato técnico, sem instalação nesta unidade. A delegação do owner permite resolver os mecanismos sem nova escolha de tecnologia entre abas. IndexedDB é armazenamento persistente do produto no navegador, não banco descartável para testar migrations SQL.

Namespace sempre inclui ambiente e usuário autenticado; origem do navegador isola o banco. Conteúdo não vai para sessionStorage de Auth. Cada aba possui identificador próprio de edição; revisões locais são independentes de versões futuras do servidor.

Fluxo planejado:

1. Abrir documento com revisão local base e iniciar rascunho da aba.
2. Em cada autosave, persistir o rascunho da aba e tentar atualizar o documento canônico local. A comparação da revisão e a gravação de corpo/título/vínculo/revisão devem ocorrer na mesma transação readwrite.
3. Se a revisão esperada ainda corresponder, gravar e avançar a revisão local; confirmar “salvo localmente” somente quando a transação concluir.
4. Se outra aba tiver mudado a revisão, conservar seu texto como rascunho separado e manter a versão canônica já gravada. O conflito lógico não aborta a gravação do rascunho: a transação conclui apenas esse registro, sem atualizar o documento canônico. Não declarar o rascunho como a versão atual nem substituir a versão canônica silenciosamente.
5. Apresentar conflito com possibilidade de consultar a versão atual e reaplicar conscientemente o rascunho sobre a nova revisão; nova concorrência exige nova verificação. Nenhum merge automático ou CRDT é pré-requisito.
6. Move, rename, delete, restore e purge verificam revisão/estado na mesma fronteira transacional; autosave antigo não desfaz um move, restauração ou exclusão.

O aviso entre abas pode usar BroadcastChannel, com mensagens pequenas de invalidação (IDs/revisão, sem corpo/token), namespace de conta e releitura do banco. Esse aviso não é lock, fonte da verdade ou garantia de entrega; correção depende da transação/revisão mesmo sem aviso. Ao retomar foco, reler revisão/estado; não sobrescrever um rascunho sujo ao receber invalidação.

A proteção aplica-se a todas as conexões que usam o adapter do produto, sem depender de um mutex em memória de uma única aba. Falha/quota/abort deve preservar a edição em memória, sinalizar erro e não prometer durabilidade de rascunho cujo commit falhou. Migração de schema local requer tratar conexões antigas e bloqueio de upgrade, sem apagar o banco como fallback.

Fontes primárias: [IndexedDB e transações](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Using_IndexedDB) e [BroadcastChannel](https://developer.mozilla.org/en-US/docs/Web/API/Broadcast_Channel_API). Não confundir eventos/requests de sucesso intermediário com commit da transação.

### Logout e troca de conta

Owner confirmou adiar o gatilho de sincronização no logout. Primeira entrega não faz upload nem adiciona endpoints documentais.

Antes de sair voluntariamente, tentar concluir a gravação local pendente. Se falhar, informar a falha e permitir cancelar a saída ou sair conscientemente; nunca afirmar que o conteúdo não gravado está salvo. Após saída, limpar a visualização e cancelar/invalidar operações e callbacks da geração anterior. A sessão é encerrada, mas documentos/lixeira/rascunhos confirmados no banco local continuam associados à conta antiga.

Outra conta usa outro namespace; não lê nem aplica esses registros. O mesmo usuário, ao entrar novamente, recupera seu conteúdo local e os rascunhos conservados. Retomar uma transação/callback iniciado por outra sessão não pode salvar dados no namespace da conta nova. A indicação de salvo local não significa upload ou backup remoto.

Na fase posterior de sincronização, o clique de logout deverá primeiro tentar sincronizar enquanto a credencial ainda existe; só então encerrar a sessão. Falha deve conservar pendências. Retry/cancelar/sair sem sincronizar e limitações de fechar aba/expirar sessão serão contratados nessa fase, sem prometer envio garantido pelo evento de fechamento do navegador.

### Cenários de aceitação acrescentados ao planejamento

- Exclusão deixa documento restaurável por 240h; re-delete idempotente não renova prazo; restore válido conserva ID; vencimento rejeita restore; reabertura limpa expirados sem apagar itens restaurados por outra aba.
- Duas abas com a mesma revisão: uma grava; a outra conserva edição e apresenta conflito. Nunca há perda silenciosa de uma das versões por last-write-wins.
- Uma aba move/exclui/restaura enquanto outra edita: o auto-save antigo não reverte metadados nem ressuscita o documento.
- Invalidação perdida ou canal indisponível não compromete a comparação transacional. Rascunho confirmado pode ser recuperado após reload/fechamento.
- Logout/relogin conserva conteúdo confirmado; troca A→B não exibe dados de A nem recebe callbacks antigos. Falha de flush antes de logout não informa sucesso falso.
- Não há envio documental no logout nem periódico nesta primeira entrega; grafo e funções permanecem adiados.

Esses comportamentos são planejamento, não implementação ou prova runtime. A spec permanece rascunho e sem autorização de execução.

## Referência Obsidian básico — revisão 6, 2026-10-12

Aplicar [PD-019](../../../product-decisions.md): o owner quer comportamento documental familiar ao Obsidian básico e confirmou que os links acompanham renomeações. Esta referência orienta as escolhas de UX já no recorte aprovado, sem ampliar a entrega a todo o produto Obsidian.

- Escrever Markdown, consultar a prévia, criar/abrir/renomear/mover documentos e seguir links deve ter comportamento familiar. Mantém-se editor de fonte + prévia; Live Preview inline não é requisito implícito.
- Digitar `[[` apresenta seleção dos documentos locais disponíveis. Mostrar título e camada/caminho para desambiguar nomes repetidos; destino confirmado guarda identidade estável. O usuário não precisa digitar ou administrar IDs.
- Renomear o destino atualiza o nome padrão apresentado em todos os seus links resolvidos; mover conserva o vínculo e atualiza a localização exibida quando aplicável. Não re-resolver links existentes por título, não mudar o destino e não substituir texto livre ou trechos de código por uma busca textual indiscriminada.
- Exemplo de aceite: A contém um link resolvido para B, chamado `Curso`. Renomear B para `Investimentos` mantém o mesmo ID e faz o link exibir `Investimentos`; renomear outro documento também chamado `Curso` não altera esse vínculo. Reabrir A conserva o destino e o nome atual. Um rascunho antigo em outra aba não pode reverter o rename ou redirecionar a referência.
- Links pendentes continuam sinalizados sem auto-criação. Títulos repetidos e consulta recursiva dos pais continuam conforme PD-014/015. Lixeira local de 240h e salvamento local1s continuam conforme PD-016/017; não importar os padrões de lixeira ou storage físico do Obsidian.

Fontes oficiais conferidas: [links internos](https://obsidian.md/help/links) documentam autocomplete `[[`, atualização automática de links e textos personalizados; [gestão de notas](https://obsidian.md/help/manage-notes) documenta operações de notas e lixeira; [modos do editor](https://obsidian.md/help/edit-and-read) distingue Source mode, Reading view e Live Preview. Suporte a texto personalizado, embeds, aliases e links para blocos/headings não entra automaticamente no Mindforge por constar nessas fontes.

O principal consolida schema/adaptador/IDs, limites, representação canônica de referências e UX dentro dessas decisões. Não é necessário devolver escolhas técnicas comuns ao owner. Ainda são propostas a fechar na spec, sem código: defaults de título/conteúdo vazio, limites operacionais e detalhes visuais de documentos próprios/herdados. Nova pergunta só se houver diferença material que a referência e as decisões existentes não resolvam. Esta revisão registra a direção; não congela contrato técnico nem autoriza implementação.

## Ordem de entrega alterada — PD-020

Em 2026-10-12, owner autorizou backend primeiro com TDD e confirmou antecipar API/PostgreSQL; client e sincronização depois. A [spec02 backend](spec-02-backend.md) é o lote ativo, separado por contrato/persistência/prova. Esta spec01 preserva conceitos e planejamento do client; seu status draft não bloqueia o backend autorizado, nem concede execução do client.
