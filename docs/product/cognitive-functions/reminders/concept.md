# Lembrete

> **Aviso de pré-contrato:** este documento registra apenas um conceito de produto. Não cria endpoint, schema, capability, persistência, interface, algoritmo ou autorização de implementação. O que não estiver explicitamente definido aqui permanece aberto e exige decisão do owner antes de virar contrato.

## Essência

Um Lembrete é uma intenção futura que nasce de um pedido explícito do usuário para que algo indicado por ele seja trazido de volta à atenção.

## Propósito

O propósito do Lembrete é preservar a intenção expressa pelo usuário sem exigir que ele mantenha essa intenção na memória. O conceito deve poder acompanhar aquilo que o usuário indicar, mesmo quando o alvo não se encaixar em uma categoria previamente escolhida.

## Definição atual

- A origem do Lembrete é um pedido explícito do usuário. A forma concreta desse pedido ainda não está definida.
- O Lembrete pode apontar para qualquer coisa que o usuário indique: um conteúdo, projeto, meta, camada, entidade, acontecimento ou outro contexto. Essa enumeração é ilustrativa, não uma taxonomia técnica.
- “Apontar” permanece uma relação semântica. Este documento não define identidade, representação, armazenamento, resolução do alvo ou qualquer modelo técnico.
- O conceito não decide quando, como ou por qual meio o Lembrete será apresentado. Também não pressupõe recorrência, expiração ou ação automática.

## Relações

- **Com Reativação:** são conceitos distintos. O Lembrete tem origem em pedido explícito; a Reativação é proativa. O fato de ambos poderem tratar do mesmo contexto ou aparecerem próximos não os funde nem muda silenciosamente a origem de nenhum deles.
- **Com o Cérebro:** quando o alvo estiver associado ao Cérebro de um usuário, a futura concretização deverá respeitar o modelo de um único Cérebro, sua hierarquia de camadas e as fronteiras de uma eventual Hive Mind. Isso não decide herança, visibilidade, compartilhamento ou permissões.
- **Com os modos de interação:** a arquitetura admite interação manual, assistida e conversacional. Ela não decide como um pedido explícito de Lembrete será reconhecido em cada modo.
- **Com a arquitetura de autoridade:** qualquer operação futura deverá permanecer sujeita ao Brain Engine como autoridade sobre autenticação, autorização, regras, validação, persistência, efeitos e estado durável. Esta definição não escolhe produtor, consumidor ou transporte.

## Fronteiras

- Sem pedido explícito, não há base para chamar uma intenção de Lembrete nesta definição; uma iniciativa proativa não deve ser reclassificada como Lembrete e pode pertencer à Reativação conforme sua definição própria, ainda que trate do mesmo alvo.
- A possibilidade de apontar para qualquer coisa não autoriza a criação de tipos, campos, IDs, links, schemas ou endpoints.
- Lembrete não define por si só uma tela, notificação, canal, job, agenda, ferramenta de IA, capability ou comportamento de background.
- O conceito não transforma o alvo em tarefa, evento, projeto, meta, camada ou outra entidade técnica.
- Não há decisão sobre regras de segurança, consentimento, visibilidade, auditoria, explicação ao usuário, exclusão, restauração ou retenção.

## Maturidade

Conceito de produto inicial, com uma fronteira semântica preservada (pedido explícito e alvo aberto), mas sem comportamento operacional fechado. Ainda não é especificação, contrato, plano, implementação ou evidência de runtime.

## Questões abertas

- O que caracteriza um pedido explícito em linguagem natural, interação manual ou autorização anterior?
- Como o alvo é identificado, desambiguado e mantido quando seu conteúdo, nome ou posição mudar?
- Quais momento, recorrência, fuso, adiamento, cancelamento, conclusão e expiração podem existir?
- Como o usuário recebe, consulta, confirma ou ignora um Lembrete?
- Que consentimento, visibilidade e permissão se aplicam quando o alvo estiver em camada compartilhada ou em uma Hive Mind?
- Como ficam as relações do Lembrete quando o alvo for movido para a lixeira temporária ou restaurado?
- Como evitar sobreposição ou composição com uma Reativação sem alterar a origem do conceito?
- Quais regras determinísticas e quais evidências serão necessárias antes de qualquer contrato ou implementação?

## Fontes

- [Visão inicial da arquitetura](../../../../docs/architecture-overview.md), especialmente o modelo de Cérebro e camadas, a autoridade do Brain Engine e os modos de interação.
- [Decisões de produto](../../../../docs/product-decisions.md), especialmente PD-001 sobre Cérebro, camadas e Hive Mind e o limite pré-contrato do documento.
- [Decisões em aberto](../../../../docs/open-decisions.md), especialmente OD-002, OD-008 e OD-009, que mantêm pendentes hierarquia, efeitos de lixeira e enumeração de operações.
- [Diretrizes do checkout](../../../../AGENTS.md), sobre autoridade do owner, separação de estados e proibição de transformar conceitos em endpoints.
