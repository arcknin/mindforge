# Atividades — conceito de produto

> Natureza: documentação conceitual, pré-contrato.

## Essência

Atividade é o conceito de produto para aquilo que está sendo feito em um contexto do Cérebro. Ela se associa a uma camada neural ou subcamada para acompanhar tempo, atenção, dedicação e equilíbrio relacionados ao fazer cotidiano.

O conceito não se limita a tarefas de produtividade. Pode abranger o que a pessoa realiza em diferentes contextos, sem transformar esses contextos em uma taxonomia ou em tipos previamente definidos.

## Propósito

Atividades devem ajudar o Mindforge a:

- manter visível o que está sendo realizado e em qual camada ou subcamada isso se insere;
- observar a distribuição de tempo, atenção e dedicação entre contextos;
- apoiar a reflexão sobre equilíbrio sem reduzir a vida da pessoa a uma única medida;
- conectar execução e intenção quando houver relação com Metas;
- oferecer uma base conceitual para Funções Cognitivas sem depender de uma experiência agêntica.

## Definição atual

Uma Atividade representa algo que está sendo feito e se associa a uma camada neural ou subcamada. A camada responde a “onde isso pertence”; a Atividade responde a “o que está sendo feito”. Essa distinção é conceitual e não define como a associação será escolhida, alterada ou validada.

Tempo, atenção, dedicação e equilíbrio são dimensões que o produto deseja acompanhar por meio do conceito. Ainda não há definição de medida, fonte, cálculo, interpretação, periodicidade ou unidade para nenhuma delas.

Atividade não é, por definição, tarefa, projeto, rotina, sessão, evento ou outro tipo técnico. Essas aproximações podem aparecer em discussões futuras, mas não são promovidas por este documento.

## Relações

- **Camada neural e subcamada:** fornecem o contexto da Atividade. As regras de pertencimento, alcance, movimentação e relação com a hierarquia permanecem abertas.
- **Funções Cognitivas:** podem usar o conceito para organizar, acompanhar ou refletir sobre o fazer em uma camada, sem criar um mapeamento técnico obrigatório.
- **Metas:** uma Atividade pode estar relacionada a uma intenção preservada por uma Meta, mas não há cardinalidade, obrigatoriedade ou regra de contribuição decidida.
- **Pontos:** Pontos podem vir a sinalizar progresso, incentivo ou equilíbrio em torno de Atividades; nenhuma atribuição, cálculo ou ajuste está definido.
- **Hive Mind:** se a camada associada participar de uma conexão coletiva, qualquer compartilhamento deve permanecer limitado à seleção explícita e às permissões que forem aprovadas.
- **Brain Engine:** uma futura capability técnica pode apoiar a finalidade, mas Atividade continua sendo um conceito de produto e não uma operação ou endpoint.

## Fronteiras

Atividades:

- não são camadas neurais, subcamadas, Cérebros ou Hive Minds;
- não obrigam que toda ação seja registrada nem definem a unidade mínima do fazer;
- não definem tipos, estados, lifecycle, sessões temporizadas, recorrência, estimativas, métricas ou fórmulas;
- não definem uma lista de operações, telas, notificações, defaults ou regras de classificação;
- não implicam que tempo, atenção, dedicação ou equilíbrio sejam automaticamente inferidos;
- não dão ao client, a um modelo ou a um runtime agêntico acesso direto aos dados autoritativos;
- não alteram a autoridade do Brain Engine sobre autorização, validação, persistência e efeitos.

## Maturidade

O conceito está em maturidade exploratória. A associação com camada ou subcamada e o interesse em acompanhar tempo, atenção, dedicação e equilíbrio são a direção atual desta unidade; a semântica operacional, as relações e os limites de compartilhamento ainda precisam de decisão do owner e de contrato próprio, se forem necessários.

## Perguntas abertas

- O que diferencia uma Atividade de uma entrada bruta, tarefa, projeto, rotina, sessão ou evento quando esses conceitos forem discutidos?
- A associação com uma camada ou subcamada é necessária em todos os casos? Uma Atividade pode se relacionar a mais de um contexto?
- Como a pessoa expressa ou revisa tempo, atenção e dedicação, e o que “equilíbrio” significa em cada contexto?
- Quais relações entre Atividades e Metas são permitidas, e como uma intenção continua preservada quando a atividade muda?
- Pontos devem observar Atividades diretamente, por meio de Metas ou por outra relação ainda não definida?
- Como ficam seleção, visibilidade, consentimento, revogação e histórico quando a camada está conectada a uma Hive Mind?
- Que comportamentos devem permanecer possíveis sem Prefrontal Cortex, Hidden Cortex ou LLM?
- Quais aspectos precisam ser comuns aos modos manual, assistido e conversacional sem transferir autoridade ao agente?

## Fontes

- Direção do owner registrada no pedido desta unidade, em 12/09/2026.
- [AGENTS.md](../../../../AGENTS.md), sobre a autoridade do Brain Engine e a organização em um Cérebro, camadas e subcamadas.
- [Visão inicial da arquitetura](../../../architecture-overview.md), nas seções sobre Cérebro, Hive Mind, modos de interação, capabilities e Data.
- [Decisões de produto](../../../product-decisions.md), especialmente PD-001.
- [Decisões em aberto](../../../open-decisions.md), especialmente OD-001, OD-002, OD-006, OD-007 e OD-009.
- [Contratos HTTP do Brain Engine](../../../../contracts/openapi/README.md) e [registro de endpoints](../../../../contracts/openapi/registry.yaml).

## Aviso de escopo

Este documento é pré-contrato. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Também não cria tipos, estados, métricas, tabelas, operações, UI, valores default ou contrato. Qualquer definição executável depende de decisão explícita do owner e dos gates correspondentes.
