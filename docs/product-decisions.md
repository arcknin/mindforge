# Decisões de produto

Ledger conciso de decisões confirmadas pelo owner.

Este documento é pré-contrato. Ele registra decisões de produto, mas não autoriza implementação, criação de schema, persistência, endpoint, plano de entrega ou mudança de arquitetura. Detalhes não registrados como fechados permanecem pendentes em [`open-decisions.md`](open-decisions.md).

## Decisões fechadas

### PD-001 — Um Cérebro por usuário, camadas neurais e Hive Minds

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** cada usuário possui exatamente um Cérebro, criado automaticamente no primeiro login. Dentro dele, a organização é hierárquica, em camadas neurais e subcamadas. Contextos distintos pertencem ao mesmo Cérebro e não criam Cérebros separados.
- **Hive Mind:** uma Hive Mind não é outro Cérebro nem um Cérebro coletivo. É uma conexão/rede neural coletiva entre Cérebros de usuários diferentes, ligando somente camadas neurais ou subcamadas explicitamente selecionadas em cada Cérebro participante. Exemplos como `Relacionamento` ou `Trabalho > Empresa` ilustram a conexão, mas não definem camadas obrigatórias ou default.
- **Fronteira de compartilhamento:** o restante de cada Cérebro permanece fora da Hive Mind e do compartilhamento correspondente. Conectar uma camada não autoriza acesso implícito a outras camadas do mesmo Cérebro.
- **Limites:** esta decisão não define detalhes de transação ou bootstrap do primeiro login, schema, storage, APIs, papéis e permissões detalhados, ownership técnico dos dados compartilhados, sincronização ou lifecycle da Hive Mind. A flexibilidade da hierarquia e a direção de seleção para Funções Cognitivas são complementadas por PD-005.

### PD-002 — Superfície inicial de dados orientada pela finalidade

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** a superfície de dados da finalidade inicial deve contemplar todas as operações adequadas à finalidade.
- **Limites:** toda operação continua sujeita a autorização e regras próprias. A decisão não inventa uma lista final de operações, semântica, falhas, efeitos ou interface.
- **Próximo registro necessário:** um contrato futuro deve enumerar explicitamente as operações e suas regras; essa enumeração permanece aberta em `open-decisions.md`.

### PD-003 — Exclusão para lixeira temporária

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** exclusão envia o item para uma lixeira temporária por 10 dias.
- **Limites:** não estão decididos o início exato da contagem, restauração, purge, retenção legal, clock, visibilidade, relações, anexos ou jobs. Esses detalhes permanecem abertos em `open-decisions.md`.

### PD-004 — Nome técnico da entrada bruta permanece em aberto

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** `capture`, `note`, `document` ou outro nome técnico de entrada bruta não é uma decisão de produto por si só.
- **Limites:** o nome deve ser escolhido posteriormente para refletir corretamente a semântica aprovada. Nenhum desses termos é promovido aqui a entidade, capability, endpoint ou contrato.

### PD-005 — Hierarquia livre e seleção flexível para Funções Cognitivas

- **Status:** owner-approved; fechado em 2026-09-12.
- **Origem:** decisões explícitas do owner nesta conversa.
- **Hierarquia livre:** o usuário pode criar livremente camadas e camadas dentro de camadas no próprio Cérebro, organizando a hierarquia conforme entender. O aninhamento pode continuar além dos níveis usados nos exemplos, sem estabelecer uma taxonomia obrigatória ou um conjunto fixo de níveis.
- **Seleção flexível:** uma Função Cognitiva poderá considerar ou atuar sobre uma camada, subcamadas específicas ou um grupo explicitamente escolhido de camadas. O grupo pode combinar camadas de ramos distintos, inclusive com pais diferentes.
- **Descendentes:** selecionar uma camada-pai não estabelece uma regra universal de inclusão automática obrigatória de todos os descendentes. O usuário poderá escolher subcamadas específicas; os modos exatos de inclusão e exclusão ainda dependem da futura definição de cada relação.
- **Exemplos:** nomes e árvores usados na documentação são apenas ilustrações e não criam defaults, categorias obrigatórias ou uma estrutura inicial fechada.
- **Limites:** esta decisão não define a natureza exata da relação entre Função Cognitiva e camada — contexto, vínculo, alcance, participação, execução ou outra modelagem permanecem em aberto. Também não decide se grupos são temporários, salvos, reutilizáveis ou nomeados; os efeitos de mover, renomear, arquivar ou excluir camadas; a estratégia técnica de identidade e referências; nem a integração desse modelo com Hive Mind.

### PD-006 — Inbox é uma porta de entrada, não uma camada neural

- **Status:** owner-approved; fechado em 2026-09-16.
- **Origem:** decisão explícita do owner durante a definição do primeiro Brain Core.
- **Decisão:** Inbox é uma funcionalidade de captura rápida para registrar informação sem exigir que o usuário escolha previamente uma camada. Ela não é uma camada neural nem cria uma camada default.
- **Direção assistida:** uma IA poderá futuramente, quando autorizada a acessar a entrada e o contexto adequado do usuário, sugerir em qual camada o registro se encaixa.
- **Limites:** ainda não estão decididos o nome final da funcionalidade, o nome técnico da entrada bruta, quando a IA pode acessá-la, se a sugestão exige confirmação, quem executa a classificação, o destino quando não houver classificação e o lifecycle da entrada.

### PD-007 — Fronteira inicial do Brain Core e identidade estável

- **Status:** owner-approved; fechado em 2026-09-16.
- **Origem:** decisões explícitas do owner durante a definição do primeiro Brain Core.
- **Operações iniciais:** o owner pode observar a própria hierarquia, criar uma camada na raiz ou sob qualquer camada do mesmo Cérebro e renomear uma camada. Move, archive, delete, restore e purge ficam fora desta primeira feature.
- **Hierarquia:** o produto não impõe uma quantidade fixa de níveis nem uma taxonomia; limites técnicos de proteção de recursos, caso necessários, não podem redefinir a árvore como um conjunto fixo de níveis.
- **Autorização:** somente o usuário autenticado proprietário pode observar ou alterar o núcleo privado do próprio Cérebro nesta primeira feature.
- **Bootstrap e identidade:** acessos repetidos ou concorrentes do mesmo usuário devem convergir para um único Cérebro. Brain e camadas usam identidade estável independente de nome de exibição e posição hierárquica; renomear não altera essa identidade nem referências existentes.
- **Limites:** formato do identificador, persistência, transação, concorrência, erros, API, schema e mecanismo de autorização permanecem para contratos e planejamento posteriores.

### PD-008 — Escopo de Hive Mind por raiz de subárvore

- **Status:** owner-approved; fechado em 2026-09-16.
- **Origem:** refinamento explícito do owner sobre a fronteira definida em PD-001.
- **Decisão:** para uma futura Hive Mind, cada camada ou subcamada escolhida é a raiz de um recorte compartilhado. O recorte inclui essa raiz e tudo que estiver dentro de sua subárvore. Camadas ancestrais, irmãs e demais ramos do Cérebro permanecem privados.
- **Separação conceitual:** esta regra pertence ao compartilhamento por Hive Mind. Ela não altera PD-005: Funções Cognitivas continuam podendo selecionar camadas e subcamadas específicas sem uma regra universal de inclusão automática de descendentes.
- **Limites:** ainda não está decidido se descendentes criados depois da conexão entram automaticamente no recorte, nem consentimento, ACL, leitura, escrita, administração, revogação, desconexão, histórico, ownership, persistência, sincronização ou retrieval. Esta decisão não autoriza implementar Hive Mind no Brain Core.

## Limite de interpretação

As definições de um Cérebro por usuário, organização hierárquica em camadas e subcamadas, Hive Mind e sua fronteira inicial estão fechadas em PD-001 e refinadas em PD-008. A liberdade de criação da árvore e a direção de seleção flexível para Funções Cognitivas estão fechadas em PD-005, sem fechar a modelagem dessa relação. Inbox está separada da hierarquia por PD-006, e o primeiro Brain Core está delimitado em PD-007. Exemplos concretos de nomes de camadas, formatos de dados e capabilities em [`architecture-overview.md`](architecture-overview.md) continuam sendo orientação conceitual quando não aparecem nas decisões acima. Este ledger não promove exemplos arquiteturais a contrato nem autoriza implementação.
