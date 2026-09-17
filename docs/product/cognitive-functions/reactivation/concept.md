# Reativação

> **Aviso de pré-contrato:** este documento registra apenas um conceito de produto. Não cria endpoint, schema, capability, persistência, interface, algoritmo ou autorização de implementação. O que não estiver explicitamente definido aqui permanece aberto e exige decisão do owner antes de virar contrato.

## Essência

Uma Reativação é uma iniciativa proativa para fazer ressurgir algo que ficou esquecido ou perdeu presença na atenção do usuário.

## Propósito

O propósito da Reativação é reabrir a possibilidade de contato do usuário com um contexto relevante que deixou de ocupar sua atenção. Ela pode devolver visibilidade a um projeto, uma meta, uma camada ou outro contexto esquecido, sem presumir qual ação o usuário deverá tomar.

## Definição atual

- A Reativação é proativa: sua origem não é um pedido explícito feito no momento em que o contexto volta a ser apresentado.
- O que pode ressurgir inclui um projeto, uma meta, uma camada ou outro contexto esquecido. Esses exemplos não fecham tipos, escopo ou representação.
- “Esquecido” descreve a intenção do conceito, não um critério já definido. Este documento não determina como reconhecer, medir ou provar esquecimento.
- A Reativação não fecha algoritmo, uso de IA, fonte de sinais, critérios de seleção, canal, frequência, consentimento, explicabilidade ou lifecycle.

## Relações

- **Com Lembrete:** são conceitos distintos. Um Lembrete nasce de pedido explícito; uma Reativação é proativa. Um contexto pode, no futuro, estar relacionado aos dois, mas nenhum deve ser convertido silenciosamente no outro ou receber a mesma semântica apenas por aparecer na mesma experiência.
- **Com o Cérebro:** uma Reativação pode ressurgir um contexto pertencente à organização hierárquica de um único Cérebro. Qualquer futura seleção deverá respeitar camadas, subcamadas e a fronteira de compartilhamento de uma Hive Mind; essa relação não define alcance ou permissão.
- **Com memória e conexões:** o conceito pode se relacionar a informações, projetos, metas, camadas e conexões já existentes, mas não escolhe recuperação exata, textual, vetorial, híbrida ou outra estratégia.
- **Com os runtimes:** a separação entre Prefrontal Cortex e Hidden Cortex permanece a referência arquitetural. Ainda não está decidido se, quando ou em qual deles a Reativação seria preparada, apresentada ou operada.
- **Com a autoridade do sistema:** se um comportamento futuro produzir efeitos duráveis, ele deverá passar pelo Brain Engine e por suas regras. Esta definição não cria capability, produtor, consumidor ou transporte.

## Fronteiras

- Reativação não é um Lembrete solicitado pelo usuário, nem uma autorização implícita para agir sobre o contexto ressurgido.
- Ser proativa não autoriza o produto a presumir relevância, urgência, consentimento ou ação desejada.
- O conceito não fixa algoritmo, modelo ou uso de IA, sinais, critérios, limiares, ranking, periodicidade, canal, interface, notificação ou job.
- Não define o que conta como contexto esquecido, nem transforma projeto, meta, camada ou outro contexto em schema, entidade ou endpoint.
- Não define explicabilidade, controle do usuário, opt-out, permissões, visibilidade, auditoria, retenção, exclusão, restauração ou qualquer lifecycle.
- Não une Reativação e Lembrete em uma função única; eventual composição exige decisão explícita do owner.

## Maturidade

Conceito de produto inicial, limitado à natureza proativa e à possibilidade de ressurgir contextos esquecidos. Os mecanismos e garantias necessários para torná-lo observável e governável continuam indefinidos; não há especificação, contrato, plano, implementação ou evidência de runtime.

## Questões abertas

- O que significa “esquecido” para o produto e quais sinais, critérios ou decisões do usuário podem sustentar essa interpretação?
- Qual é a responsabilidade do sistema ao escolher o contexto a ressurgir e como o usuário pode contestar essa escolha?
- A Reativação pode considerar um Cérebro inteiro, uma camada, uma subcamada ou somente escopos explicitamente selecionados?
- Como consentimento, visibilidade e permissões funcionam em camadas compartilhadas e Hive Minds?
- Em quais condições a Reativação é preparada ou entregue pelo client, pelo trabalho durável ou por ambos?
- Haverá uso de IA? Se houver, que limites, supervisão e fallback determinístico serão necessários?
- Quais canal, frequência, horário, agrupamento, adiamento e encerramento serão possíveis?
- Como o usuário entende por que algo ressurgiu e como controla novas ocorrências?
- Qual lifecycle abrange criação, apresentação, interação, recusa, repetição, encerramento, exclusão e restauração?
- Como preservar a distinção de origem quando o mesmo contexto também possuir um Lembrete?

## Fontes

- [Visão inicial da arquitetura](../../../../docs/architecture-overview.md), especialmente Cérebro, recuperação de memória, Prefrontal Cortex, Hidden Cortex e autoridade do Brain Engine.
- [Decisões de produto](../../../../docs/product-decisions.md), especialmente PD-001 sobre Cérebro, camadas, Hive Mind e fronteira de compartilhamento.
- [Decisões em aberto](../../../../docs/open-decisions.md), especialmente OD-002, OD-005, OD-006, OD-008 e OD-009, que mantêm pendentes escopo, permissões, sincronização, lixeira e operações.
- [Diretrizes do checkout](../../../../AGENTS.md), sobre distinção entre conceito, contrato, implementação e runtime e sobre a autoridade do owner.
