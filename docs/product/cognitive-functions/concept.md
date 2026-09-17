# Funções Cognitivas — conceito de produto

> Natureza: documentação conceitual, pré-contrato.

## Essência

Funções Cognitivas é uma linguagem de produto para nomear funcionalidades que poderão considerar ou atuar sobre partes escolhidas da organização do Cérebro. O termo ajuda a descrever, do ponto de vista da pessoa, formas de perceber, organizar, orientar e acompanhar o que acontece em seus contextos.

O termo é uma metáfora de produto. Ele não afirma que o Mindforge possui cognição própria, não exige uma LLM e não transforma uma interação agêntica em autoridade sobre os dados.

## Propósito

Esta linguagem serve para:

- dar um vocabulário comum às funcionalidades que trabalham com o contexto de uma camada;
- manter o foco do produto no Cérebro e em sua organização, em vez de tratar cada recurso como uma ferramenta isolada;
- relacionar Atividades, Metas e Pontos sem decidir antecipadamente sua forma técnica;
- permitir que decisões de produto sejam discutidas antes de contratos e implementação.

## Definição atual

Cada usuário possui exatamente um Cérebro, organizado hierarquicamente em camadas neurais e subcamadas criadas livremente pelo usuário. Funções Cognitivas descreve funcionalidades que poderão considerar ou atuar sobre seleções dessa organização; não descreve novos Cérebros nem uma mente coletiva.

Uma seleção poderá abranger uma camada, subcamadas específicas ou um grupo explícito de camadas, inclusive de ramos com pais diferentes. Selecionar uma camada-pai não cria uma regra universal de inclusão automática obrigatória de todos os descendentes. A natureza dessa relação — contexto, vínculo, alcance, participação, execução ou outra modelagem — permanece deliberadamente aberta.

Função Cognitiva e capability do Brain Engine são conceitos distintos:

- Função Cognitiva é vocabulário de produto para a finalidade e o comportamento percebidos pela pessoa;
- capability é a linguagem técnica usada pelo Brain Engine para disponibilizar áreas funcionais autorizadas e aplicar regras, validações, permissões, persistência e efeitos;
- uma Função Cognitiva pode futuramente depender de uma ou mais capabilities, mas não há mapeamento, catálogo ou contrato definido por este documento.

As funcionalidades do Mindforge devem continuar utilizáveis sem LLM. O Prefrontal Cortex e o Hidden Cortex podem participar de interações assistidas ou de trabalhos duráveis quando autorizados, mas não substituem a autoridade determinística do Brain Engine.

## Relações

- **Cérebro, camadas e subcamadas:** fornecem a organização sobre a qual uma função poderá considerar ou atuar por seleção do usuário. Seleções granulares e grupos de ramos diferentes são possíveis, mas sua semântica exata ainda não está definida.
- **Atividades:** são o conceito de produto para acompanhar o que está sendo feito em uma camada ou subcamada, incluindo tempo, atenção, dedicação e equilíbrio.
- **Metas:** preservam intenções que podem dar sentido às atividades e a outras funções. A natureza dessa relação ainda está em aberto.
- **Pontos:** visam sinalizar progresso, incentivo e equilíbrio. Seu uso por uma Função Cognitiva não implica fórmula ou regra aprovada.
- **Hive Mind:** a integração com Funções Cognitivas será abordada posteriormente. A fronteira já decidida de uma Hive Mind não autoriza antecipar aqui sua futura semântica de seleção, governança ou execução.
- **Brain Engine e runtimes agênticos:** a função é uma descrição de produto; execução autorizada, validação e efeitos continuam sob o Brain Engine, enquanto os runtimes apenas solicitam ações dentro dos limites que forem definidos.

## Fronteiras

Funções Cognitivas:

- não são um segundo Cérebro, um Cérebro coletivo ou uma autorização implícita entre camadas;
- não são sinônimo de capability técnica, ferramenta local, endpoint ou operação;
- não obrigam uma forma de interação manual, assistida ou conversacional específica;
- não definem taxonomia, nomes, camadas default, tipos, estados, métricas, tabelas, regras de acesso, lifecycle ou efeitos;
- não autorizam o client, o Prefrontal Cortex ou o Hidden Cortex a acessar diretamente dados autoritativos;
- não substituem decisões do owner sobre produto, contrato, persistência, segurança ou operação.

## Maturidade

O conceito está em maturidade exploratória e serve como linguagem de alinhamento de produto. A organização de um Cérebro por camadas e subcamadas e a fronteira da Hive Mind são referências fechadas em PD-001. A possibilidade de seleções granulares e transversais está fechada em PD-005; a natureza da relação, a definição operacional de Funções Cognitivas e suas relações com Atividades, Metas e Pontos ainda não são um contrato aprovado.

## Perguntas abertas

- Quais finalidades de produto devem ser reconhecidas como Funções Cognitivas?
- A relação com a hierarquia significa contexto, vínculo, alcance, participação, execução, uma combinação ou outra modelagem?
- Como o usuário inclui ou exclui descendentes específicos em cada tipo de relação, sem uma regra universal de herança automática?
- Grupos de camadas são temporários, salvos, reutilizáveis, nomeados ou representados de outra forma?
- Como movimentação, renomeação, arquivamento e exclusão afetam relações existentes, e como identidade e referências serão preservadas?
- Quais relações entre Funções Cognitivas, Atividades, Metas e Pontos são essenciais, e quais são apenas possibilidades?
- Como a mesma finalidade deve permanecer disponível nos modos manual, assistido e conversacional?
- Como Funções Cognitivas se integrarão futuramente com Hive Mind? Esta discussão fica adiada até o amadurecimento posterior de Hive Mind.
- Que parte da linguagem de produto deve ser traduzida para capabilities técnicas, e em que momento essa tradução será aprovada?

## Fontes

- Direção do owner registrada no pedido desta unidade, em 12/09/2026.
- [AGENTS.md](../../../AGENTS.md), especialmente as fronteiras entre Cérebro, Brain Engine, Prefrontal Cortex, Hidden Cortex e capabilities.
- [Visão inicial da arquitetura](../../architecture-overview.md), nas seções sobre Cérebro, Hive Mind, Brain Engine, runtimes agênticos e Capability Registry.
- [Decisões de produto](../../product-decisions.md), especialmente PD-001 e PD-005.
- [Decisões em aberto](../../open-decisions.md), especialmente OD-001, OD-002, OD-005, OD-007, OD-009 e OD-010.
- [Contratos HTTP do Brain Engine](../../../contracts/openapi/README.md) e [registro de endpoints](../../../contracts/openapi/registry.yaml), que mantêm a fronteira pré-contrato.

## Aviso de escopo

Este documento é pré-contrato. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Também não cria tipos, estados, métricas, tabelas, operações, UI, valores default ou contrato. Qualquer promoção para uma decisão executável depende de decisão explícita do owner e dos gates correspondentes.
