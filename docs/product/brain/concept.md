# Cérebro

> **Aviso de escopo:** este é um documento conceitual de produto. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Detalhes operacionais continuam sujeitos a decisões e contratos próprios.

## Essência

O Cérebro é a unidade pessoal de organização de informação do Mindforge. Cada usuário possui exatamente um Cérebro. Ele reúne os contextos, conhecimentos, relações e outras informações do usuário em uma organização interna hierárquica.

Um contexto pessoal, profissional, empresarial ou de projeto não cria outro Cérebro. O usuário pode criar livremente esses e outros recortes como camadas dentro de camadas, compondo a própria árvore conforme entender. Os nomes usados aqui são apenas exemplos.

## Propósito

Dar ao usuário uma unidade coerente para organizar e relacionar o que lhe pertence, sem obrigá-lo a separar artificialmente a própria vida em Cérebros independentes. O conceito deve acomodar informação bruta, conhecimento, entidades e atividades futuras sem transformar exemplos de interface ou de armazenamento em modelo fechado.

O Cérebro também estabelece a fronteira de onde uma conexão com outras pessoas pode partir: uma Hive Mind conecta recortes escolhidos de Cérebros distintos, sem tornar o Cérebro de qualquer participante público por inteiro.

## Definição atual

- Cada usuário tem exatamente um Cérebro.
- O Cérebro é organizado em camadas neurais e subcamadas hierárquicas.
- O usuário pode criar livremente camadas dentro de camadas e organizar a hierarquia conforme sua própria necessidade.
- As camadas e subcamadas são contextos internos do Cérebro; não são Cérebros separados.
- Funções Cognitivas poderão considerar ou atuar sobre uma camada, subcamadas específicas ou grupos explicitamente escolhidos de camadas, inclusive de ramos com pais diferentes. A natureza exata dessa relação ainda não foi definida.
- Relações podem existir dentro do mesmo Cérebro e, por meio de uma Hive Mind, entre subárvores cujas raízes foram explicitamente selecionadas em Cérebros de usuários diferentes.
- Exemplos como `Pessoal`, `Trabalho`, `Trabalho > Empresa` ou `Relacionamento` são apenas ilustrações. Não constituem nomes obrigatórios, defaults ou uma lista fechada.

Para a definição conceitual da hierarquia, consulte [Camadas neurais](neural-layers/concept.md). Para o recorte entre usuários, consulte [Hive Mind](../hive-mind/concept.md).

## Relações

- **Camadas neurais:** formam a organização hierárquica interna do único Cérebro do usuário.
- **Hive Mind:** é uma conexão/rede neural coletiva entre Cérebros diferentes, limitada às subárvores cujas raízes forem explicitamente selecionadas em cada participante.
- **Conexões neurais:** podem relacionar conteúdos, entidades ou conhecimentos dentro do Cérebro e, quando estiverem sob uma Hive Mind, operar apenas no escopo coletivo selecionado.
- **Brain Engine:** permanece a autoridade arquitetural sobre autenticação, autorização, regras, validação, persistência, efeitos e estado durável. O conceito de Cérebro não transfere essa autoridade para client, modelo ou agente.

## Fronteiras

- Não existem Cérebros separados implícitos para vida pessoal, trabalho, empresa ou projeto.
- Uma Hive Mind não é outro Cérebro nem um Cérebro coletivo, e não funde os Cérebros dos participantes.
- A conexão coletiva alcança a raiz selecionada e sua subárvore; ancestrais, irmãs e outros ramos permanecem privados e fora do compartilhamento correspondente.
- Seleção de escopo não resolve, por si só, leitura, escrita, visibilidade, administração, consentimento ou revogação.
- Este documento não define identidade técnica, modelo de dados, storage, sincronização, recuperação, transporte, sessão ou lifecycle.

## Estado de maturidade

A identidade de um Cérebro por usuário, sua organização hierárquica e a fronteira conceitual com Hive Minds estão fechadas em PD-001 e refinadas pela subárvore de PD-008. A liberdade de criação da árvore e a direção de seleção flexível para Funções Cognitivas estão fechadas em PD-005. A composição concreta de cada Cérebro é escolha do usuário; os comportamentos necessários para operar, vincular, compartilhar, recuperar ou sincronizar informações permanecem conceituais e não implementáveis a partir deste documento.

## Questões abertas

- Existem camadas iniciais ou defaults, ou o Cérebro começa sem uma taxonomia predefinida?
- Qual é o lifecycle de uma camada e de uma subcamada?
- Como descendentes criados depois de uma conexão Hive Mind passam a integrar ou não a subárvore compartilhada?
- Como Funções Cognitivas se relacionam com uma camada ou grupo de camadas sem fechar prematuramente se isso significa contexto, vínculo, alcance, participação, execução ou outra modelagem?
- Como entradas brutas, atividades, tarefas, sessões, projetos e documentos se relacionam com camadas sem transformar cada execução em uma camada?
- Qual identidade canônica, formato de persistência, fonte de verdade e estratégia de sincronização serão adotados?
- Como retrieval, relações sugeridas e outros índices respeitarão os limites do Cérebro e de uma Hive Mind?
- Como serão governados consentimento, ACL, visibilidade, revogação, lifecycle e ownership técnico quando camadas forem conectadas?

## Fontes de autoridade

- [Decisões de produto](../../product-decisions.md), especialmente PD-001, PD-005, PD-007 e PD-008.
- [Decisões em aberto](../../open-decisions.md), especialmente OD-001, OD-002, OD-004, OD-005, OD-006 e OD-010.
- [Visão inicial da arquitetura](../../architecture-overview.md), como referência conceitual e não como contrato.
- [Orientação do checkout](../../../AGENTS.md) e [registro de contratos HTTP](../../../contracts/openapi/README.md), que preservam a separação entre documentação, contrato, implementação e runtime.


## Continuação conceitual — assistente pessoal

PD-023..025 consolidam a direção de assistente contínuo, trava para troca de modelo e MCP como consumidor/provedor. A [visão de RAG, memória e harness](../assistant/rag-harness-overview.md) distingue essas decisões das propostas e pendências OD-017..020. Não é uma spec executável e não altera os contratos existentes.
