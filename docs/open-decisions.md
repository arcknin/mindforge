# Decisões em aberto

Backlog de questões para debate do owner. Nenhum item abaixo contém recomendação vinculante ou autorização de implementação.

Este documento é pré-contrato. Ele não cria schema, endpoint, capability, plano de entrega ou regra operacional. Decisões futuras devem ser registradas antes de serem promovidas a contrato ou implementação.

## Organização do Cérebro

### OD-001 — Papel e lifecycle do Inbox

**Decisão já fechada:** PD-006 define Inbox como porta de entrada/funcionalidade de captura rápida, não como camada neural. Ela pode futuramente receber sugestões assistidas de destino quando o acesso da IA for autorizado.

**Por que ainda importa:** falta definir como entradas brutas se relacionam com o Cérebro, as camadas neurais e o processamento posterior.

**Perguntas para o owner:**

- Qual será o nome de produto e o nome técnico da entrada bruta?
- Uma entrada permanece fora das camadas até ser classificada, ou pertence a alguma camada em algum momento? Quando?
- Quando a IA pode acessar a entrada e qual contexto do usuário pode usar para sugerir o destino?
- A sugestão exige confirmação do usuário e quem executa a classificação autorizada?
- Qual é o lifecycle de processamento, incluindo entradas que permanecem brutas?

### OD-002 — Camadas iniciais, lifecycle e limites da hierarquia

**Por que importa:** determina a organização inicial que o usuário encontra e os limites de contexto das operações.

**Perguntas para o owner:**

- O Cérebro começa somente com seu root estrutural ou existe alguma camada inicial explicitamente escolhida pelo owner?
- Quais validações se aplicam ao novo nome de uma camada, inclusive vazio, duplicidade entre irmãs, normalização e concorrência?
- Em quais condições o usuário pode remover, arquivar, mover ou reorganizar camadas livremente criadas?
- Quais guardrails técnicos de profundidade ou volume são necessários para proteção de recursos sem impor níveis fixos de produto?
- Qual é o lifecycle de camadas e subcamadas?

### OD-003 — Nome técnico da entrada bruta

**Por que importa:** o nome influencia semântica de domínio, contratos e comunicação entre client e Brain Engine.

**Perguntas para o owner:**

- Qual termo deve representar a entrada bruta quando a semântica estiver definida: `capture`, `note`, `document` ou outro?
- O termo deve distinguir conteúdo bruto, documento persistido e resultado processado?

## Persistência e dados

### OD-004 — Persistência e formato inicial

**Por que importa:** define identidade, durabilidade, interoperabilidade e a autoridade dos dados.

**Perguntas para o owner:**

- Para documentos, PD-012 já definiu corpo Markdown como texto no PostgreSQL. Como será a representação técnica e a persistência dos outros tipos de dados?
- Qual é a identidade canônica de cada item?
- Qual é a fonte da verdade?
- Como ocorre eventual conversão entre formatos, se houver?

## Hive Mind, empresa, compartilhamento e permissões

### OD-005 — Operação e governança de Hive Minds

**Decisão já fechada:** PD-008 define que uma camada selecionada para Hive Mind é a raiz de um recorte que inclui sua subárvore, enquanto ancestrais, irmãs e outros ramos continuam privados.

**Por que ainda importa:** falta definir como essa conexão funciona sem ampliar implicitamente o acesso além da subárvore selecionada.

**Perguntas para o owner:**

- Como uma Hive Mind é criada, descoberta, aceita, recusada, encerrada ou abandonada?
- Quem pode conectar ou desconectar uma camada e qual consentimento dos participantes é necessário?
- Uma camada pode participar de várias Hive Minds? Conexões podem ser transitivas?
- Descendentes criados depois da conexão entram automaticamente no recorte compartilhado ou exigem nova decisão?
- Quais regras de leitura, escrita, visibilidade e administração se aplicam às camadas, itens, relações e operações conectados?
- O que acontece com dados, relações, histórico e acesso após revogação ou desconexão?
- Como empresas, residências, famílias e relacionamentos especializam a mesma definição de Hive Mind sem criar modelos incompatíveis?
- Onde ficam o ownership técnico, a fonte da verdade e a trilha de auditoria dos dados compartilhados?

## Client, offline e sincronização

### OD-006 — Offline e sincronização no primeiro client

**Por que importa:** afeta disponibilidade, cópias locais, conflitos e a fonte de verdade operacional.

**Perguntas para o owner:**

- O primeiro client precisa operar offline? Com qual escopo?
- Como alterações offline são sincronizadas?
- Como conflitos são detectados e resolvidos?
- Qual é a fonte da verdade durante e depois da convergência?

### OD-007 — Forma da primeira entrega

**Por que importa:** determina a ordem de validação do produto e os limites do primeiro recorte implementável.

**Perguntas para o owner:**

- A primeira entrega deve começar por backend/API?
- Ou deve ser um slice vertical envolvendo Flutter/client e backend?
- Quais superfícies ficam explicitamente fora da primeira entrega?

## Exclusão e lixeira

### OD-008 — Detalhes da lixeira temporária de 10 dias

**Por que importa:** transforma a decisão de retenção em comportamento determinístico, recuperável e operável.

**Refinamento local:** PD-017 confirma lixeira local por 10 dias; o planejamento adota 240h desde o commit local, clock UTC do dispositivo, restore antes do vencimento e purge na próxima execução. A etapa remota poderá definir clock/retensão autoritativos próprios.

**Perguntas para a etapa remota e casos restantes:**

- Como reconciliar exclusão/restore locais e a contagem autoritativa no servidor?
- Quem pode ver e restaurar um item, e até quando?
- Quando e como ocorre o purge?
- Há retenção legal ou exceções ao prazo?
- Qual clock/timezone é usado?
- Como ficam visibilidade, relações e anexos durante a lixeira?
- Quais jobs ou efeitos operacionais são necessários?

## Operações, autorização e falhas

### OD-009 — Enumeração exata da superfície inicial

**Por que importa:** torna verificável a decisão de cobrir todas as operações adequadas à finalidade sem deixar semântica implícita.

**Perguntas para o owner:**

- Quais são exatamente as operações da finalidade inicial?
- Qual autorização e regra de negócio se aplica a cada operação?
- Quais falhas e efeitos devem ser definidos por operação?
- Quais operações são manuais, assistidas ou conversacionais, sem mudar a autoridade do Brain Engine?

## Funções Cognitivas e seleção de camadas

### OD-010 — Natureza da relação, grupos e continuidade dos vínculos

**Parte fechada em 2026-10-11:** PD-009..011 definem conteúdo Markdown e dados funcionais nas camadas, funções desenvolvidas pela equipe e estado/configuração próprios por uso. A associação de uso com camada está fechada conceitualmente; seleção de contexto ou de múltiplas camadas não equivale a compartilhar/fundir estados.

**Por que ainda importa:** falta definir cardinalidade de usos na mesma camada, continuidade dos vínculos, seleção, agregação e composição operacional.

**Perguntas futuras:**

- A relação representa contexto, vínculo, alcance, participação, execução, uma combinação desses conceitos ou outra modelagem?
- Em quais situações selecionar uma camada inclui algum descendente, e como o usuário inclui ou exclui subcamadas específicas sem uma regra universal de herança automática?
- Um grupo de camadas é temporário, salvo, reutilizável, nomeado ou representado de outra forma?
- O que acontece com relações e grupos quando uma camada é movida, renomeada, arquivada ou excluída?
- Qual estratégia técnica preserva identidade e referências sem depender de nomes ou caminhos mutáveis?
- Como essa seleção se integrará futuramente com Hive Mind? Essa discussão fica adiada até o amadurecimento posterior de Hive Mind.

## Conteúdos das camadas e usos de funções

### OD-011 — Usos: cardinalidade, configuração e lifecycle

**Fechado:** PD-010/011; equipe desenvolve as funções, usuário escolhe/configura/combina as disponíveis e cada uso mantém seu próprio estado.

**Ainda aberto:** múltiplos usos da mesma função na mesma camada; parâmetros configuráveis por função; estados e registros de cada domínio; desativação sem perda de dados; cópia/movimentação; exclusão/lixeira/restauração; evolução de definições sem alterar silenciosamente usos existentes. A lixeira de PD-003 continua dependendo dos detalhes de OD-008.

### OD-012 — Documentos Markdown e persistência do conteúdo

**Fechado:** PD-009/012/014..016; corpo Markdown em texto PostgreSQL na fase do servidor; primeira entrega apenas com documentos locais e salvamento automático a cada segundo; ID estável e título separado; documento pertence a uma camada, consulta do pai reúne a subárvore; operações incluem mover/excluir/restaurar; editor texto + prévia. PD-013/015 fecham referências por ID e adiam a visualização do grafo.

**Ainda aberto:** schema/adaptador local e posterior remoto (IndexedDB definido no planejamento da revisão5); regras do título; conteúdo vazio/tamanho; variantes e detalhes de editor; detalhamento dos testes de concorrência já planejada; histórico navegável não incluído; autoria/proveniência; operações de mover e restaurar; referências e seu lifecycle. Upload de arquivos/anexos, importação/exportação e filesystem do computador continuam não escolhidos. Sincronização automática em 30 minutos é direção posterior, não parte da primeira entrega. OD-008/016 concentram lixeira e salvamento local; OD-015 concentra links.

### OD-013 — Primeiro domínio funcional e forma de composição

**Fechado:** PD-010; combinar funções existentes é permitido como direção de produto, sem criação de funções executáveis pelo usuário.

**Ainda aberto:** qual função detalhar primeiro, suas operações e dados tipados; relação entre Tarefas e o conceito exploratório de Atividades; composição por ação manual, vínculo, evento ou automação; escopo entre camadas e autorização por efeito. Foco, tarefas, pontos, Finanças e agendamentos permanecem exemplos, não um catálogo executável.

**Disposição do owner:** “daqui a pouco debatemos melhor, só registra o conceito”. Não escolher a primeira função nem um mecanismo de composição nesta unidade.

### OD-014 — Conteúdo pertencente à camada e contexto consultado

**Fechado:** PD-014 define consulta do pai sobre documentos próprios e de todos os descendentes, sem duplicar dados nem fundir estados. Camada também poderá reunir dados funcionais; PD-005 continua orientando seleção/atuação de funções, que está adiada.

**Ainda aberto:** UX para separar documentos diretos e herdados; paginação/filtros; projeções/resumos versus registros completos; orçamento e seleção para prompts; acesso por agentes e atualização dos resumos. Disponibilidade da subárvore está decidida para documentos; processamento integral por agentes e agregação funcional não estão definidos.

### OD-015 — Sintaxe, lifecycle dos links e projeção do grafo

**Fechado:** PD-013/015; intenção por wikilink/título, ID de destino estável e desambiguação, rename sem quebrar vínculo e links sem destino sinalizados, sem criar documento automaticamente. Validação autoritativa futura no Brain Engine e grafo derivado permanecem a direção; visualização do grafo adiada. Na fase local-only, resolução/navegação alcançam documentos disponíveis localmente e não provam validação remota.

**Ainda aberto:** sintaxe exata para ID/rótulo; se/quando o texto é canonicalizado; representação das referências e reconstrução do grafo sem perder identidade; nome padrão atualizado por rename fechado em PD-019; eventual rótulo personalizado fora do recorte atual; tratamento de links inexistentes, pendentes, removidos ou sem acesso; múltiplas ocorrências/direcionalidade; reconciliação atômica entre texto e projeção; exclusão/lixeira/restauração; grafo por camada versus seleção explícita; layout/navegação e inclusão futura de registros funcionais ou URLs externos. Não escolher biblioteca, banco de grafos, job ou API nesta discussão.

### OD-016 — Persistência local e evolução para sincronização

**Fechado:** PD-016; gravar alterações localmente a cada segundo; entrega atual sem upload de documentos, sem botão de sync e sem scheduler de 30 minutos. PostgreSQL e envio periódico continuam como etapa futura.

**Ainda aberto:** schema/adaptador local e seus testes (IndexedDB e namespace por usuário/ambiente/origem definidos na revisão5); ID estável antes do primeiro upload e sua validação futura; detalhamento/testes de confirmação de commit e recuperação após reload/fechamento; limites/quota/falhas; detalhamento/testes dos conflitos/revisões e isolamento por conta já definidos na revisão5/PD-018; fila/ordem de create/move/delete/restore; cópias antigas e conflitos no envio futuro. Lixeira local detalhada em PD-017; clock remoto, reconciliação de exclusão/restore e outros casos continuam em OD-008.

**Fronteira:** cache/réplica e documentos locais pendentes não concedem autoridade de estado remoto. O primeiro editor pode exigir login/carregamento online de camadas existentes; suporte completo offline da aplicação ainda não foi decidido. Evitar depender de API/schema de documentos inexistentes para a fase local.

## Nota de coerência arquitetural

A definição de Cérebro, camadas e subcamadas está fechada em PD-001; a fronteira de Hive Mind foi refinada em PD-008. A criação livre da hierarquia e a seleção granular/transversal por Funções Cognitivas estão fechadas em PD-005, sem fechar sua modelagem. Inbox está fora da hierarquia conforme PD-006, e as operações iniciais do Brain Core estão delimitadas em PD-007. Os detalhes operacionais ainda listados, além de nomes concretos de camadas, Markdown, banco estruturado e capabilities em [`architecture-overview.md`](architecture-overview.md), continuam referências conceituais enquanto não forem decididos. O Brain Engine permanece a autoridade arquitetural sobre autenticação, autorização, regras, validação, persistência, effects e estado durável; esta lista não altera esse limite.

PD-009..011 complementam essa direção com conteúdo Markdown/dados funcionais, funções da equipe e estado por uso. OD-011..014 concentram o detalhamento restante; a escolha da primeira função foi adiada pelo owner. O planejamento inicial está em [layer-content](features/brain/layer-content/spec-01-main.md).

PD-012/013 fecham corpo Markdown em texto PostgreSQL e referências internas por identidade/grafo derivado. A escolha de storage do corpo não permanece pendente; OD-012/015 preservam apenas o detalhamento restante.

## Fechamento local em 2026-10-12

PD-017 confirma a lixeira local de 10 dias. PD-018 registra a delegação técnica de conflitos entre abas e a escolha explícita de adiar envio no logout. Preservar documentos/rascunhos confirmados e separá-los por conta integra o recorte local; não depende de servidor disponível. Transações/revisões e notificação entre abas estão especificadas no planejamento e não precisam de nova escolha de mecanismo pelo owner. O contrato ainda deve detalhar IDs/schema/limites, referências Markdown e destinos indisponíveis, sem implantar código.

## Referência documental em 2026-10-12 — PD-019

Nome padrão dos links acompanha renomeações: fechado. Experiência básica de documentos do Obsidian é a referência para os padrões de operação/editor/navegação, conciliada com os IDs estáveis e decisões locais do Mindforge. Não reabrir esse comportamento como escolha pendente. Permanecem detalhamento técnico de sintaxe/schema/adaptador, limites e UX; o principal pode propor defaults sem questionário de mecanismos. Funções, grafo e sync continuam adiados. Referência ao Obsidian não inclui automaticamente Live Preview inline, aliases/embeds/blocos/anexos ou vault físico. Ver revisão6 da spec.

## Fechamento documental/backend — PD-020, 2026-10-12

Prevalece sobre listas antigas OD-012/015/016: contrato de produto consolidado na spec01 revisão7; backend antecipado na spec02 e seis endpoints aprovados pelo owner. UUID canônico, título normalizado1..255, conteúdo vazio até1MiB, tokens [[doc:UUID]], título projetado atual, consultas direta/subárvore e lixeira remota240h/CAS estão fechados. Schema/adapter PostgreSQL e tests são implementação, não nova decisão de produto. Client local mantém seu contrato e será implementado depois; envio/reconciliação entre dispositivos, anexos/import/export, aliases/embeds/blocos, grafo e funções continuam adiados. Autoria de agente e políticas de sync não bloqueiam CRUD manual privado. SQL/VM/deploy continuam sem autorização.


## Assistente pessoal, RAG, memória, harness e MCP — 2026-10-18

A [visão consolidada](product/assistant/rag-harness-overview.md) separa PD-023..025 das propostas. O registro não autoriza implementação nem novos contratos.

### OD-017 — Memória conversacional, autonomia, retenção e esquecimento

**Fechado:** experiência contínua em PD-023; autoridade e permissões permanecem no Brain Engine.

**Aberto:** histórico/resumos/memórias/documentos e sua persistência; retenção, apagamento e propagação ao índice; correções/contradições; proveniência e classificação por afirmação; o que constitui confirmação; quando sugerir ou criar/atualizar documentos automaticamente. Tipos definidos pelo harness e timestamps por significado são propostas, não schema aprovado. Nenhuma autorização universal de memória automática foi concedida.

### OD-018 — Recuperação, contexto, cobertura e atualização

**Direção preservada:** índice como projeção reconstruível no desenho conceitual da arquitetura; consultas autoritativas continuam sob Brain Engine. Disponibilidade de documentos da subárvore não significa envio integral ao prompt.

**Aberto:** escopo padrão, orçamento e composição do contexto; ferramentas específicas de busca/leitura/consulta estruturada; cobertura e estabilidade de leituras em lotes; distinção rascunho/servidor/índice; chunking, modelo/dimensão de embeddings, algoritmo lexical/vetorial/híbrido, ranking e critérios de avaliação. PostgreSQL/pgvector é referência inicial do desenho anterior, não adoção operacional nova. Corpus e comparação de alternativas são proposta de investigação.

### OD-019 — Modelos, privacidade e continuidade operacional

**Fechado:** possibilidade de modelos diferentes em PD-023; trava e aviso no aplicativo em PD-024.

**Aberto:** provedores suportados e capacidades mínimas; adaptadores específicos/genérico; estado portátil versus estado opaco por provedor; estados de troca/cancelamento/timeout/reconciliação; credenciais; consentimento e dados enviados a inferência, embeddings ou outros serviços; fallback e limites de custo. Compatibilidade genérica não garante suporte integral a ferramentas.

### OD-020 — Contratos e operação MCP

**Fechado:** consumir MCP externo e oferecer servidor MCP Mindforge em PD-025. Não reabrir a necessidade das duas capacidades como indecisa.

**Aberto:** primeiro recorte e catálogo de ferramentas; fluxo interno client/harness → MCP → Brain Engine ainda proposto; localização/processo, transporte, descoberta, autorização, identidade, escopos, consentimento, credenciais e limites por integração. MCP não substitui regras Brain Engine, não autoriza acesso direto a GoTrue/SQL e não converte exemplos em endpoints aprovados. API manual permanece existente; seu contrato não é alterado nesta unidade.
