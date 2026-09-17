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

- A persistência inicial usa PostgreSQL, Markdown, uma combinação ou outra forma?
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

**Perguntas para o owner:**

- Quando começa a contagem dos 10 dias?
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

**Por que importa:** PD-005 permite que uma Função Cognitiva considere ou atue sobre uma camada, subcamadas específicas ou grupos de ramos diferentes. Ainda falta definir a semântica e o comportamento dessa relação sem reduzir prematuramente as possibilidades do produto.

**Perguntas futuras:**

- A relação representa contexto, vínculo, alcance, participação, execução, uma combinação desses conceitos ou outra modelagem?
- Em quais situações selecionar uma camada inclui algum descendente, e como o usuário inclui ou exclui subcamadas específicas sem uma regra universal de herança automática?
- Um grupo de camadas é temporário, salvo, reutilizável, nomeado ou representado de outra forma?
- O que acontece com relações e grupos quando uma camada é movida, renomeada, arquivada ou excluída?
- Qual estratégia técnica preserva identidade e referências sem depender de nomes ou caminhos mutáveis?
- Como essa seleção se integrará futuramente com Hive Mind? Essa discussão fica adiada até o amadurecimento posterior de Hive Mind.

## Nota de coerência arquitetural

A definição de Cérebro, camadas e subcamadas está fechada em PD-001; a fronteira de Hive Mind foi refinada em PD-008. A criação livre da hierarquia e a seleção granular/transversal por Funções Cognitivas estão fechadas em PD-005, sem fechar sua modelagem. Inbox está fora da hierarquia conforme PD-006, e as operações iniciais do Brain Core estão delimitadas em PD-007. Os detalhes operacionais ainda listados, além de nomes concretos de camadas, Markdown, banco estruturado e capabilities em [`architecture-overview.md`](architecture-overview.md), continuam referências conceituais enquanto não forem decididos. O Brain Engine permanece a autoridade arquitetural sobre autenticação, autorização, regras, validação, persistência, effects e estado durável; esta lista não altera esse limite.
