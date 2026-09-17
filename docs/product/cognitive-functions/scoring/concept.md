# Pontos — conceito de produto

> Natureza: documentação conceitual, pré-contrato.

## Essência

Pontos são um conceito de produto voltado a sinalizar progresso, incentivo e equilíbrio. O objetivo é oferecer uma linguagem para feedback e acompanhamento, sem presumir que qualquer número seja uma medida objetiva da pessoa, do seu valor ou da qualidade do que ela faz.

O conceito permanece deliberadamente incompleto: fórmula, lifecycle e regras ainda estão em aberto.

## Propósito

Pontos devem ser investigados como uma possível forma de:

- tornar conversas sobre progresso mais compreensíveis;
- apoiar incentivo sem substituir a autonomia e a intenção da pessoa;
- observar equilíbrio entre contextos e formas de dedicação;
- relacionar feedback a Atividades e Metas quando isso fizer sentido;
- apoiar Funções Cognitivas sem tornar a IA ou a pontuação autoridade do sistema.

Esses propósitos não definem que Pontos serão usados em toda situação, nem que haverá uma única interpretação.

## Definição atual

Pontos visam progresso, incentivo e equilíbrio. Essa é a definição atual de finalidade; ela não determina como um Ponto surge, é associado, é interpretado, é corrigido ou deixa de valer.

Pontos podem vir a se relacionar a Atividades, Metas, camadas ou subcamadas, mas não há regra de vínculo, escopo, comparação, acumulação ou apresentação decidida. Fórmula, lifecycle e regras de negócio permanecem perguntas abertas do produto.

Pontos são linguagem de produto, não uma capability técnica do Brain Engine. Se uma futura decisão exigir comportamento durável, a autoridade sobre validação, autorização, persistência e efeitos continuará sendo do Brain Engine.

## Relações

- **Atividades:** podem fornecer contexto para refletir sobre o que está sendo feito e sobre dedicação, tempo, atenção ou equilíbrio; nenhuma atividade é automaticamente pontuada.
- **Metas:** preservam intenções que podem receber uma leitura de progresso ou equilíbrio; Pontos não substituem a intenção nem definem por si só seu atendimento.
- **Funções Cognitivas:** podem usar Pontos como sinal de acompanhamento ou incentivo, sem mapeamento para capability, operação ou interface.
- **Camadas neurais e subcamadas:** podem delimitar um contexto de análise futura, mas não há regra de alcance, herança ou comparação entre camadas.
- **Hive Mind:** qualquer uso envolvendo camadas conectadas precisa respeitar seleção, consentimento, visibilidade e revogação explícitos; Pontos não ampliam o compartilhamento.
- **Brain Engine e agentes:** um runtime pode futuramente solicitar ações autorizadas, mas não calcula, ajusta ou interpreta Pontos fora das regras aprovadas pelo Brain Engine.

## Fronteiras

Pontos:

- não são automaticamente dinheiro, recompensa, ranking, nota, punição ou gamificação;
- não são uma métrica aprovada nem uma prova objetiva de progresso, incentivo ou equilíbrio;
- não definem fórmula, unidade, escala, normalização, fonte, agregação, correção ou lifecycle;
- não estabelecem tipos, estados, tabelas, operações, telas, notificações, limites ou valores default;
- não obrigam quantificar Atividades ou Metas, nem tornam comparáveis pessoas ou Cérebros;
- não substituem Metas, Atividades, julgamento da pessoa ou regras de negócio;
- não concedem acesso implícito a camadas, subcamadas ou participantes de uma Hive Mind;
- não permitem que client, modelo ou runtime agêntico escreva diretamente em dados autoritativos.

## Maturidade

O conceito está em maturidade exploratória. A finalidade geral — progresso, incentivo e equilíbrio — está registrada nesta unidade; a fórmula, o lifecycle, as regras e as relações executáveis continuam sem decisão e não formam contrato.

## Perguntas abertas

- O que “progresso”, “incentivo” e “equilíbrio” significam em cada contexto do Cérebro?
- Quais relações com Atividades e Metas são úteis sem transformar Pontos em obrigação ou substituto da intenção?
- Que fórmula, fonte e interpretação seriam aceitáveis, e quais riscos de distorção ou pressão precisam ser evitados?
- Como Pontos são criados, revisados, corrigidos, explicados e deixam de se aplicar, caso esse comportamento seja aprovado?
- O escopo é pessoal, de uma camada, de uma subcamada ou de uma Hive Mind? Como consentimento e revogação afetam a leitura?
- É possível acompanhar progresso e equilíbrio sem comparação entre pessoas ou contextos incomparáveis?
- Quais regras devem ser determinísticas no Brain Engine e quais informações podem ser apenas orientação de produto?
- Como manter o acompanhamento manual possível e tornar qualquer assistência agêntica revisável?

## Fontes

- Direção do owner registrada no pedido desta unidade, em 12/09/2026.
- [AGENTS.md](../../../../AGENTS.md), sobre autoridade do Brain Engine, camadas, Hive Mind e independência da IA.
- [Visão inicial da arquitetura](../../../architecture-overview.md), nas seções sobre Cérebro, Hive Mind, modos de interação, capabilities e Data.
- [Decisões de produto](../../../product-decisions.md), especialmente PD-001 e seus limites.
- [Decisões em aberto](../../../open-decisions.md), especialmente OD-001, OD-002, OD-005, OD-006, OD-007 e OD-009.
- [Contratos HTTP do Brain Engine](../../../../contracts/openapi/README.md) e [registro de endpoints](../../../../contracts/openapi/registry.yaml).

## Aviso de escopo

Este documento é pré-contrato. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Também não cria tipos, estados, métricas, tabelas, operações, UI, valores default ou contrato. Qualquer definição executável depende de decisão explícita do owner e dos gates correspondentes.
