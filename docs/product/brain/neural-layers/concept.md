# Camadas neurais e subcamadas

> **Aviso de escopo:** este é um documento conceitual de produto. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Detalhes operacionais continuam sujeitos a decisões e contratos próprios.

## Essência

Camadas neurais são os contextos hierárquicos que organizam o único Cérebro de um usuário. Subcamadas são níveis internos dessa mesma organização. A hierarquia separa contextos sem separar a pessoa em múltiplos Cérebros.

## Propósito

Oferecer uma organização flexível na qual o usuário possa estruturar os próprios contextos. A hierarquia permite camadas dentro de camadas e também que recortes de ramos distintos sejam considerados em conjunto no futuro, sem transformar os exemplos em uma taxonomia obrigatória.

Ela também fornece unidades que podem ser selecionadas explicitamente para uma conexão por Hive Mind, mantendo fora dela o restante do Cérebro de cada participante.

## Definição atual

- Camadas e subcamadas pertencem ao mesmo Cérebro do usuário.
- O usuário pode criar livremente camadas dentro de camadas e organizar a árvore conforme entender.
- O aninhamento pode continuar além dos níveis usados nos exemplos, sem limitar a árvore a duas camadas nem definir aqui classes técnicas para cada nível.
- O produto não impõe níveis fixos à hierarquia. Nomes, camadas iniciais, defaults e eventuais guardrails técnicos de recursos ainda exigem contratos próprios.
- Uma camada é um contexto relativamente persistente; não é automaticamente uma atividade, tarefa, projeto, sessão ou capability.
- Uma Função Cognitiva poderá considerar ou atuar sobre uma camada, subcamadas específicas ou um grupo explicitamente escolhido de camadas, inclusive de ramos com pais diferentes.
- Uma Hive Mind só pode conectar subárvores cujas raízes sejam camadas ou subcamadas explicitamente selecionadas de Cérebros diferentes.
- Para Hive Mind, selecionar a raiz inclui o recorte já contido em sua subárvore; o comportamento de descendentes criados depois continua aberto. Para Funções Cognitivas, PD-005 preserva seleções granulares e não cria herança universal.

Para a unidade que contém a hierarquia, consulte [Cérebro](../concept.md). Para o compartilhamento entre Cérebros, consulte [Hive Mind](../../hive-mind/concept.md) e [Conexões neurais coletivas](../../hive-mind/neural-connections/concept.md).

## Relações

- **Cérebro:** é o limite de pertencimento; todas as camadas e subcamadas descritas aqui são internas ao único Cérebro do usuário.
- **Outras camadas do mesmo Cérebro:** podem ter relações entre si, sem que isso crie um novo Cérebro ou uma autorização de compartilhamento externo.
- **Funções Cognitivas:** poderão considerar ou atuar sobre seleções granulares ou transversais da árvore. Ainda não está decidido se essa relação será contexto, vínculo, alcance, participação, execução ou outra modelagem.
- **Hive Mind:** pode relacionar recortes de participantes diferentes somente quando suas raízes forem explicitamente selecionadas.
- **Brain Engine:** deve continuar sendo a autoridade sobre regras, permissões, validação, persistência, efeitos e estado durável quando esses conceitos forem operacionalizados.

## Fronteiras

- `Pessoal`, `Trabalho`, `Empresa` e `Projetos pessoais` são exemplos ilustrativos, não camadas obrigatórias.
- Uma subcamada não é um Cérebro independente.
- A existência de uma relação entre camadas não implica leitura, escrita, administração, consentimento ou visibilidade automáticos.
- A semântica de descendentes depende da relação: Hive Mind usa a raiz de uma subárvore conforme PD-008; Funções Cognitivas preservam seleção granular conforme PD-005.
- A hierarquia não define por si só ACL, lifecycle, ownership, storage, sincronização, retrieval ou regras de propagação.
- Este documento não introduz nomes técnicos de entidades, rotas HTTP, tabelas, eventos ou ferramentas.

## Estado de maturidade

O princípio de um Cérebro por usuário organizado em camadas e subcamadas é uma decisão de produto fechada em PD-001. A criação livre da árvore e a direção de seleção granular e transversal por Funções Cognitivas estão fechadas em PD-005. A forma concreta escolhida por cada usuário e o comportamento operacional dos níveis continuam em maturidade conceitual, com questões pendentes em OD-002 e OD-010.

## Questões abertas

- Quais camadas iniciais existem e quais são apenas exemplos?
- Em quais condições o usuário pode renomear, remover, arquivar, mover ou reorganizar camadas?
- Quais guardrails técnicos de profundidade ou volume protegem recursos sem impor níveis fixos de produto?
- Como descendentes criados depois de uma conexão Hive Mind entram ou não no recorte compartilhado?
- Grupos de camadas serão temporários, salvos, reutilizáveis, nomeados ou representados de outra forma?
- O que acontece com vínculos quando uma camada é movida, renomeada, arquivada ou excluída, e como identidades e referências serão preservadas tecnicamente?
- Uma camada pode participar de várias Hive Minds? Uma conexão pode ser transitiva entre níveis ou redes?
- Como atividades, tarefas, projetos, sessões e documentos pertencem a uma camada sem redefinir a hierarquia como execução?
- Como consentimento, ACL, visibilidade, revogação, sincronização, ownership e retrieval respeitarão cada fronteira?

## Fontes de autoridade

- [Decisões de produto](../../../product-decisions.md), especialmente PD-001, PD-005, PD-007 e PD-008.
- [Decisões em aberto](../../../open-decisions.md), especialmente OD-001, OD-002, OD-005 e OD-010.
- [Visão inicial da arquitetura](../../../architecture-overview.md), como referência conceitual e não como contrato.
- [Orientação do checkout](../../../../AGENTS.md) e [registro de contratos HTTP](../../../../contracts/openapi/README.md), que impedem promover conceitos a schema, endpoint ou implementação.
