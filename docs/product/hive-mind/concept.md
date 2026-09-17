# Hive Mind

> **Aviso de escopo:** este é um documento conceitual de produto. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Detalhes operacionais continuam sujeitos a decisões e contratos próprios.

## Essência

Uma Hive Mind é uma conexão/rede neural coletiva entre Cérebros de usuários diferentes. Ela cria um espaço conceitual de relação entre recortes escolhidos, sem criar outro Cérebro, sem fundir os participantes e sem converter seus Cérebros em um Cérebro coletivo.

## Propósito

Permitir que pessoas compartilhem contexto, conhecimento ou relações em uma fronteira deliberadamente estreita — por exemplo, um relacionamento ou uma subcamada de trabalho — enquanto mantêm o restante de seus Cérebros fora desse compartilhamento.

O propósito é conectar redes neurais selecionadas, não substituir a organização individual de cada participante nem definir antecipadamente os modelos de família, empresa, residência ou relacionamento.

## Definição atual

- Uma Hive Mind conecta Cérebros de usuários diferentes por meio de subárvores cujas raízes são camadas neurais ou subcamadas explicitamente selecionadas.
- Cada participante continua possuindo exatamente um Cérebro independente.
- O escopo coletivo contém cada raiz selecionada e sua subárvore. Ancestrais, irmãs e demais ramos permanecem privados e fora do compartilhamento correspondente.
- Exemplos como conectar `Relacionamento` ou `Trabalho > Empresa` ilustram possibilidades, mas não estabelecem defaults, papéis, categorias ou camadas obrigatórias.
- A Hive Mind pode ser entendida conceitualmente como uma rede de conexões; sua forma técnica, conteúdo e operações ainda não foram definidos.

Veja [Cérebro](../brain/concept.md), [Camadas neurais](../brain/neural-layers/concept.md) e [Conexões neurais coletivas](neural-connections/concept.md) para os conceitos relacionados.

## Relações

- **Cérebros participantes:** fornecem os recortes conectados, mas não deixam de existir nem se tornam um único objeto coletivo.
- **Camadas e subcamadas:** são raízes possíveis do escopo da conexão; a escolha precisa ser explícita.
- **Conexões neurais coletivas:** expressam a relação entre recortes conectados sem determinar schema, direção, cardinalidade ou semântica de dados.
- **Brain Engine:** continua sendo a autoridade sobre autenticação, autorização, regras, validação, persistência, efeitos e estado durável. A Hive Mind não autoriza acesso direto por client, modelo ou agente.

## Fronteiras

- Hive Mind não é outro Cérebro e não é um Cérebro coletivo.
- Ela não funde, replica ou torna público o Cérebro completo de qualquer participante.
- Conectar uma camada ou subcamada não concede acesso implícito a ancestrais, irmãs ou outros ramos do mesmo Cérebro.
- A conexão ainda não define leitura, escrita, visibilidade, administração, consentimento, revogação, transitividade, sincronização nem o tratamento de descendentes criados depois.
- Uma Hive Mind não cria por si só uma rota HTTP, uma capability, uma tabela, um índice, uma política ACL ou um lifecycle.

## Estado de maturidade

A definição de Hive Mind como conexão/rede neural coletiva entre Cérebros diferentes, com o restante privado, está fechada em PD-001. PD-008 refina cada seleção como raiz de uma subárvore compartilhada. Sua governança e operação permanecem em maturidade conceitual e dependem de decisões do owner registradas em OD-005.

## Questões abertas

- Como uma Hive Mind é criada, descoberta, aceita, recusada, encerrada ou abandonada?
- Quem pode selecionar, conectar e desconectar camadas, e qual consentimento é necessário?
- Uma camada pode participar de várias Hive Minds? Conexões podem ser transitivas?
- Descendentes criados depois da conexão entram automaticamente na subárvore compartilhada?
- Quais regras de leitura, escrita, visibilidade e administração valem para camadas, itens, relações e operações conectados?
- O que acontece com dados, relações, histórico e acesso após revogação ou desconexão?
- Como empresas, residências, famílias e relacionamentos especializam a mesma definição sem criar modelos incompatíveis?
- Onde ficam ownership técnico, fonte da verdade e auditoria dos dados compartilhados?
- Como lifecycle, persistência, sincronização e retrieval respeitarão o escopo explícito sem ampliar a consulta para o restante dos Cérebros?

## Fontes de autoridade

- [Decisões de produto](../../product-decisions.md), especialmente PD-001 e PD-008.
- [Decisões em aberto](../../open-decisions.md), especialmente OD-002 e OD-005.
- [Visão inicial da arquitetura](../../architecture-overview.md), como referência conceitual e não como contrato.
- [Orientação do checkout](../../../AGENTS.md) e [registro de contratos HTTP](../../../contracts/openapi/README.md), que preservam os gates de contrato, implementação e runtime.
