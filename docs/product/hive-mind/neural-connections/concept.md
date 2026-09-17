# Conexões neurais coletivas

> **Aviso de escopo:** este é um documento conceitual de produto. Ele não autoriza schema, API, endpoint, persistência, implementação ou plano de entrega. Detalhes operacionais continuam sujeitos a decisões e contratos próprios.

## Essência

Uma conexão neural coletiva é a relação conceitual que liga subárvores cujas raízes foram explicitamente selecionadas em Cérebros de usuários diferentes dentro de uma Hive Mind. Ela descreve uma rede compartilhada delimitada; não define uma estrutura de dados nem transforma os Cérebros participantes em uma entidade única.

## Propósito

Dar nome ao vínculo entre camadas, subcamadas, conhecimentos, documentos ou entidades que possam ser relacionados no escopo coletivo autorizado. A conexão deve tornar visível que o vínculo atravessa Cérebros diferentes somente por uma fronteira escolhida, sem sugerir que o restante de qualquer Cérebro também está conectado.

## Definição atual

- A conexão pertence conceitualmente a uma Hive Mind e liga recortes de Cérebros de usuários diferentes.
- Cada recorte tem como raiz uma camada neural ou subcamada explicitamente selecionada e inclui o que estiver dentro dessa subárvore.
- Conexões internas a um mesmo Cérebro continuam sendo possíveis, mas não são automaticamente conexões de Hive Mind.
- A conexão é uma relação de escopo, não uma concessão implícita de acesso.
- Direção, cardinalidade, tipos de conteúdo, permanência, transição e representação técnica ainda não foram decididos.

Veja [Hive Mind](../concept.md), [Cérebro](../../brain/concept.md) e [Camadas neurais](../../brain/neural-layers/concept.md) para a cadeia de pertencimento e escopo.

## Relações

- **Hive Mind:** fornece o contexto coletivo no qual a conexão existe.
- **Camadas e subcamadas:** definem as raízes dos recortes de origem e destino explicitamente selecionados.
- **Cérebro:** continua sendo a unidade pessoal de cada participante; nenhuma conexão o substitui ou o funde com outro.
- **Documentos, entidades e conhecimentos:** podem ser relacionados conceitualmente quando estiverem dentro do escopo autorizado, sem que este documento escolha formato, identidade ou persistência.
- **Brain Engine:** permanece responsável por autorização, regras, validação, persistência, efeitos e estado durável caso uma conexão venha a ser operacionalizada.

## Fronteiras

- Uma conexão não é um Cérebro coletivo e não cria um Cérebro compartilhado.
- Uma conexão alcança a raiz selecionada e sua subárvore; ancestrais, irmãs e outros ramos permanecem privados.
- Ela não define descendentes criados depois, acesso transitivo entre ramos, leitura, escrita, visibilidade, administração, consentimento ou revogação.
- Ela não define ACL, lifecycle, ownership, storage, sincronização ou retrieval.
- Não há autorização para inferir schema, API, endpoint, capability, índice ou evento a partir do termo “conexão neural”.

## Estado de maturidade

O limite conceitual — conexão coletiva somente entre recortes selecionados de Cérebros diferentes, mantendo o restante privado — decorre de PD-001. PD-008 refina cada recorte como uma subárvore com raiz explicitamente selecionada. A semântica operacional da conexão permanece sem contrato aprovado.

## Questões abertas

- Uma conexão é direcional, bidirecional ou depende do tipo de relação?
- Quais cardinalidades e tipos de relação podem existir entre camadas, subcamadas, documentos e entidades?
- A mesma camada pode participar de várias conexões ou Hive Minds?
- Uma conexão pode ser transitiva? Descendentes criados depois passam a integrar automaticamente o recorte compartilhado?
- Quem cria, aceita, altera, revoga ou encerra uma conexão, e como consentimento e ACL funcionam?
- Como ficam dados, relações, histórico, sincronização e retrieval depois de desconexão ou revogação?
- Qual ownership técnico e fonte da verdade serão aplicáveis aos elementos compartilhados?

## Fontes de autoridade

- [Hive Mind](../concept.md), definição conceitual deste produto.
- [Decisões de produto](../../../product-decisions.md), especialmente PD-001 e PD-008.
- [Decisões em aberto](../../../open-decisions.md), especialmente OD-002, OD-004, OD-005 e OD-006.
- [Visão inicial da arquitetura](../../../architecture-overview.md), como referência conceitual e não como contrato.
- [Orientação do checkout](../../../../AGENTS.md) e [registro de contratos HTTP](../../../../contracts/openapi/README.md), que separam conceito de contrato, implementação e runtime.
