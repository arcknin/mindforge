# Mindforge — Visão Inicial da Arquitetura

## Objetivo

O Mindforge será um sistema pessoal de organização e conhecimento inspirado no conceito de “segundo cérebro”.

O objetivo não é apenas armazenar arquivos Markdown como o Obsidian, mas permitir que informações sejam recebidas de forma bruta, interpretadas, organizadas e transformadas em dados úteis e persistentes.

Exemplos:

* registrar notas e conhecimentos;
* registrar gastos;
* criar compromissos;
* organizar tarefas;
* armazenar dados estruturados;
* recuperar informações antigas;
* permitir que uma IA utilize as capacidades disponíveis no sistema.

O Obsidian pode continuar sendo uma das interfaces do Mindforge, principalmente para visualização e edição de conhecimento em Markdown, mas o Mindforge não deve depender exclusivamente dele.

A camada agêntica interativa deverá operar no client, próxima do contexto e da interface do usuário. O servidor continuará responsável pela autoridade determinística e manterá uma camada agêntica separada apenas para trabalhos duráveis que não dependam de interação ativa com o usuário.

Essa separação busca reduzir round-trips e carga de orquestração no servidor sem transferir para o client a autoridade sobre regras, permissões ou persistência.

---

# Visão geral

A arquitetura inicial pode ser pensada assim:

```text
Interactive flow

Manual Client ───────────────────────────────────────────────┐
                                                            │
Assisted / Conversational Client                            │
              │                                             │
              ▼                                             │
      Prefrontal Cortex                                     │
      ├── Agent Harness                                     │
      ├── Context                                           │
      ├── Tool Discovery                                    │
      └── Model Gateway ⇄ Remote Provider/Model             │
              │                                             │
              └── structured action ────────────────────────┤
                                                            ▼
                                                   Brain Engine APIs

Background flow

Brain Engine jobs/events
              │
              ▼
       Hidden Cortex
       ├── deterministic background processing when possible
       ├── optional Model Gateway ⇄ Remote Provider/Model
       └── structured actions/results ──────────────────────→ Brain Engine APIs

Common authoritative flow

Brain Engine APIs
       │
       ▼
Auth / Rules / Permissions / Validation
       │
       ▼
Capabilities
├── Notes
├── Calendar
├── Tasks
└── Finance
       │
       ▼
Data
├── Markdown
├── Structured Database
└── Files
```

---

# Cérebro, camadas neurais, subcamadas e Hive Minds

Cada usuário possui exatamente um Cérebro no Mindforge.

O Cérebro é organizado hierarquicamente em camadas neurais e subcamadas. O usuário pode criar livremente camadas dentro de camadas e organizar a árvore conforme entender. Contextos como vida pessoal, trabalho, empresa e projetos pessoais pertencem a essa organização interna; eles não representam Cérebros separados e seus nomes são apenas exemplos, não defaults.

Uma representação inicial desse conceito é:

```text
Usuário
└── Cérebro
    ├── Camada neural: Pessoal
    │   ├── Daily
    │   └── Conhecimento
    │
    ├── Camada neural: Trabalho
    │   ├── Subcamada: Empresa
    │   └── Subcamada: Projetos pessoais
    │
    └── outras camadas — composição livre pelo usuário
```

As conexões neurais relacionam documentos, entidades e conhecimentos dentro de uma camada, entre camadas do mesmo Cérebro ou, através de uma Hive Mind, entre camadas selecionadas de Cérebros diferentes.

Uma Hive Mind não é outro Cérebro, não é um Cérebro coletivo e não funde os Cérebros participantes. Ela é uma conexão/rede neural coletiva que liga recortes explicitamente selecionados em cada Cérebro participante. Cada camada ou subcamada selecionada funciona como raiz do recorte compartilhado e inclui a subárvore contida nela. Por exemplo, duas pessoas podem conectar suas respectivas camadas `Relacionamento`, ou membros de uma empresa podem conectar suas respectivas subcamadas `Trabalho > Empresa`.

Ancestrais, irmãs e outros ramos de cada Cérebro permanecem fora da Hive Mind e do compartilhamento correspondente. Uma conexão coletiva não concede acesso implícito além da subárvore selecionada. Ainda serão definidos o comportamento de descendentes criados depois da conexão, consentimento, visibilidade, escrita, revogação e permissões.

Uma Função Cognitiva poderá futuramente considerar ou atuar sobre uma camada, subcamadas específicas ou um grupo explicitamente escolhido de camadas, inclusive de ramos com pais diferentes. Selecionar uma camada-pai não estabelece inclusão automática obrigatória de todos os descendentes, mas os modos exatos de seleção e a natureza dessa relação permanecem abertos.

Os nomes internos do modelo de dados podem permanecer neutros, por exemplo `brain`, `neural_layers`, `segments`, `neural_connections` e `documents`. A representação técnica da hierarquia e da Hive Mind, os comportamentos de lifecycle das camadas, a identidade e continuidade das referências e a combinação das conexões com a recuperação vetorial ainda serão definidos posteriormente.

---

# Brain Engine

O Brain Engine é o núcleo/orquestrador do Mindforge.

Ele é a porta de entrada do sistema e é responsável principalmente pelas partes determinísticas e pelas regras de negócio.

Responsabilidades possíveis:

* expor APIs e operações determinísticas aos clients e componentes agênticos;
* autenticar e autorizar cada solicitante;
* sincronização;
* versionamento;
* controlar o estado durável de filas, eventos e jobs;
* aplicar regras de negócio;
* controlar permissões;
* validar ações solicitadas pelo Prefrontal Cortex e pelo Hidden Cortex;
* executar capabilities autorizadas;
* controlar a persistência e os efeitos reais das operações.

Exemplo:

```text
Client envia uma ação estruturada:
create_expense(amount=80, category="mercado")

Brain Engine
→ valida o usuário
→ valida os argumentos
→ aplica as regras financeiras
→ executa a Finance Capability
→ persiste o resultado
```

O Brain Engine não precisa interpretar a frase livre do usuário no fluxo interativo.

Essa interpretação pertence ao Prefrontal Cortex no client. Depois que a intenção for transformada em uma ação estruturada, o Brain Engine deve conhecer e aplicar integralmente o contrato da funcionalidade solicitada.

---

# Prefrontal Cortex

O Prefrontal Cortex representa a camada agêntica interativa do Mindforge.

Ele executa no client e funciona como o Agent Runtime das interações ativas com o usuário.

O Agent Harness, o gerenciamento de contexto, a descoberta e seleção de tools e a comunicação com modelos ficam próximos da interface. O modelo pode continuar sendo fornecido remotamente; isso não significa que a inferência ocorra no dispositivo.

Responsabilidades possíveis:

* interpretar mensagens;
* identificar intenção;
* utilizar o contexto da interface e da sessão;
* decidir quais capabilities utilizar;
* organizar informações;
* resumir conteúdo;
* relacionar conhecimentos;
* pesquisar informações;
* descobrir e selecionar tools;
* conversar com o Model Gateway e o Remote Provider/Model;
* produzir rascunhos e ações estruturadas;
* solicitar capabilities através das APIs do Brain Engine.

Exemplo:

```text
Entrada:
"Gastei R$ 80 no mercado"

Prefrontal Cortex interpreta:

→ isso parece ser uma despesa
→ preciso encontrar uma capability financeira
```

O Prefrontal Cortex não deve possuir acesso irrestrito ao sistema nem gravar diretamente em Data.

Ele solicita ações e o Brain Engine continua sendo a autoridade responsável por permitir ou bloquear a execução.

Quando o client é encerrado, o Prefrontal Cortex deixa de executar. Uma possível transferência de trabalho interativo para background deverá ser explícita, auditável e definida posteriormente; não deve existir continuação implícita da sessão no servidor.

---

# Hidden Cortex

O Hidden Cortex é a camada agêntica server-side destinada a trabalhos duráveis que não dependem de uma interação ativa e em tempo real com o usuário.

Responsabilidades possíveis:

* consumir jobs e eventos disponibilizados pelo Brain Engine;
* conduzir tarefas agendadas;
* executar automações em background;
* coordenar processamento assíncrono;
* atualizar índices;
* aplicar retries operacionais;
* preparar notificações;
* utilizar interpretação agêntica quando uma tarefa de background realmente exigir raciocínio.

O Brain Engine continua sendo o proprietário do estado durável dos jobs, filas e eventos. O Hidden Cortex pode executar e orquestrar o trabalho, mas não se torna uma autoridade paralela sobre transições, permissões ou efeitos.

Tarefas determinísticas de background devem ser executadas sem LLM sempre que interpretação ou raciocínio não forem necessários.

Quando precisar realizar uma ação, o Hidden Cortex também utiliza as capabilities expostas pelo Brain Engine e passa pelas mesmas validações e políticas aplicáveis aos demais solicitantes.

O Prefrontal Cortex e o Hidden Cortex poderão compartilhar contratos de Agent Harness, descoberta de capabilities, tool calls, resultados e erros. As ferramentas e permissões efetivamente disponíveis serão diferentes em cada ambiente.

O Hidden Cortex será implementado inicialmente em Go, assim como o Brain Engine, mas permanecerá um processo independente. Essa escolha compartilha o toolchain sem unir autoridade, ciclo de vida ou falhas dos dois componentes.

Processamento server-side de modelos não é foco inicial. Se futuramente trabalhos como indexação, cálculo de grafos, sincronização intensiva ou inferência local exigirem processamento pesado, executores ou bibliotecas especializados poderão ser implementados em Rust sem transferir a autoridade do Brain Engine.

---

# Modos de interação e independência da IA

O backend do Mindforge não será apenas um intermediário que entrega todas as requisições a uma IA para que ela decida como o sistema deve funcionar.

Cada funcionalidade deverá existir de forma determinística no backend, com suas próprias operações, contratos, validações e regras de negócio.

Os clients poderão utilizar essas funcionalidades de três maneiras:

```text
Manual
→ o usuário informa os dados e utiliza a funcionalidade diretamente

Assistida
→ a IA ajuda a preencher, classificar ou organizar os dados
→ o Prefrontal Cortex opera no client
→ o usuário pode revisar antes da execução

Conversacional
→ o usuário descreve o que deseja em linguagem natural
→ o Prefrontal Cortex interpreta a intenção e solicita uma ação estruturada
```

Todos os modos devem convergir para as mesmas capabilities e regras do backend.

```text
Manual Client ───────────────────────────────┐
                                             │
Assisted Client ────────┐                    │
                        ▼                    │
Conversational Client → Prefrontal Cortex ──┼──→ Brain Engine ──→ Capability ──→ Data
                                             │
Hidden Cortex ───────────────────────────────┘
```

O Prefrontal Cortex e o Hidden Cortex podem produzir ou completar os argumentos necessários para uma operação, mas não devem implementar regras de negócio, gravar diretamente nos dados nem ignorar as validações do Brain Engine.

Assim, a IA é uma forma opcional de interação com o Mindforge, e não uma dependência para o funcionamento das suas capacidades.

Mesmo que o Prefrontal Cortex, o Hidden Cortex, o Model Gateway ou as LLMs estejam indisponíveis, as funcionalidades manuais do sistema devem continuar operando normalmente.

A granularidade e o formato técnico dessas interfaces serão definidos posteriormente.

---

# Model Gateway

O Model Gateway será a camada lógica responsável pelo acesso aos modelos.

O objetivo é evitar que os Agent Runtimes fiquem diretamente acoplados a um único fornecedor.

No fluxo interativo, o Prefrontal Cortex deverá conversar com o provedor a partir do client quando isso for compatível com as políticas de segurança e autenticação. Assim, a interpretação e a resposta estruturada não precisam fazer um round-trip intermediário pelo Brain Engine.

No fluxo de background, o Hidden Cortex poderá utilizar uma implementação server-side do mesmo contrato quando uma tarefa realmente precisar de modelo.

Exemplo:

```text
Model Gateway

├── OpenAI
├── Anthropic
├── Gemini
├── Local Model
└── Outros provedores
```

Isso permitirá futuramente utilizar diferentes modelos dependendo da tarefa.

Exemplo:

```text
classificação simples
→ modelo barato

raciocínio complexo
→ modelo mais forte

tarefas privadas/offline
→ modelo local
```

Também poderá existir fallback entre provedores.

A localização operacional exata do Model Gateway e a estratégia de credenciais ainda serão definidas. Segredos permanentes ou privilegiados não devem ser incorporados ao client.

---

# Capability Registry

O Capability Registry representa o catálogo de funcionalidades que o Brain Engine pode disponibilizar aos clients e componentes agênticos.

Uma capability representa uma área funcional do Mindforge.

Exemplo:

```text
Capabilities

├── Notes
├── Calendar
├── Tasks
├── Finance
├── Projects
└── Files
```

Cada capability pode possuir várias tools.

Exemplo:

```text
Finance

├── create_expense
├── update_expense
├── search_expenses
├── create_income
└── get_month_summary
```

O Prefrontal Cortex e o Hidden Cortex não precisam receber todas as tools existentes em todas as execuções.

A ideia é utilizar descoberta progressiva.

Inicialmente cada runtime pode possuir apenas ferramentas como:

```text
capabilities.search()
capabilities.describe()
capabilities.list()
```

Exemplo:

```text
Usuário:
"Gastei R$ 80 no mercado"

Prefrontal Cortex:

capabilities.search(
    "registrar despesa financeira"
)

↓

Finance encontrada

↓

capabilities.describe("finance")

↓

Prefrontal Cortex descobre:
finance.create_expense
```

Depois pode solicitar:

```text
finance.create_expense(
    amount=80,
    category="mercado"
)
```

O Capability Registry deve retornar apenas capabilities autorizadas para a identidade, a sessão e o contexto atuais.

O Brain Engine valida a solicitação antes da execução. Nenhum dos Agent Runtimes utiliza o registry como atalho para acessar Data diretamente.

---

# Permissions / Policies

Uma regra importante da arquitetura:

```text
Prefrontal Cortex ou Hidden Cortex decide o que gostaria de fazer.

Brain Engine decide se pode fazer.
```

Exemplo:

```text
Prefrontal Cortex:
finance.create_expense(...)

Brain Engine:
✓ usuário autorizado
✓ capability habilitada
✓ argumentos válidos

→ executar
```

Outro exemplo:

```text
Hidden Cortex:
calendar.delete_all_events()

Brain Engine:
✗ operação não permitida
```

Isso permite que a IA tenha autonomia controlada sem se tornar a autoridade do sistema.

Clients manuais também passam pelas validações do Brain Engine. Uma origem não se torna confiável apenas por não utilizar IA.

Ações de background que exigem nova confirmação do usuário devem aguardar aprovação ou depender de uma autorização previamente delimitada. Os detalhes desse fluxo ainda serão definidos.

---

# Data

A camada de Data representa a memória persistente do Mindforge.

O acesso persistente ocorre por meio das capabilities governadas pelo Brain Engine. Clients, Prefrontal Cortex e Hidden Cortex não devem gravar diretamente nessa camada.

Nem toda informação deve ser armazenada da mesma maneira.

Possíveis tipos:

```text
Data

├── Markdown
├── Structured Database
├── Files
└── Metadata
```

## Markdown

Ideal para conhecimento humano e conteúdo que queremos visualizar através do Obsidian.

Exemplo:

```text
Conhecimento/
    Arquitetura modular.md
```

## Structured Database

Ideal para informações estruturadas.

Exemplos:

* gastos;
* tarefas;
* compromissos;
* hábitos;
* projetos;
* registros recorrentes.

Exemplo:

```text
expenses

date        category    amount
2026-09-02  Mercado     80
2026-09-02  Gasolina    120
```

Assim, quando o usuário pedir:

```text
"Mostre meus gastos de setembro."
```

o sistema consulta os dados reais em vez de pedir para a LLM reconstruir uma tabela a partir de textos antigos.

---

# Desenho conceitual da memória e recuperação

Esta seção registra um desenho inicial para orientar as discussões. Ela não representa um contrato fechado, um esquema SQL definitivo nem uma decisão irreversível de implementação. A modelagem poderá evoluir durante o desenvolvimento.

## Memória documental

As camadas neurais podem armazenar Markdown, PDFs, imagens, pesquisas, anotações, anexos e outros documentos genéricos.

Os arquivos originais permanecem armazenados integralmente. O banco estruturado mantém o catálogo necessário para encontrá-los e governá-los, incluindo uma identidade estável, a camada neural, o caminho físico, o tipo do arquivo, o hash do conteúdo, a origem, as permissões e o estado de extração e indexação.

O caminho físico não deve ser a identidade permanente de um documento. Renomear ou mover um arquivo não deve quebrar suas conexões, seu histórico ou seus índices.

## Grafo universal e conexões neurais

Todo documento ou entidade pode participar das conexões neurais, mesmo quando o formato original não possui links próprios, desde que permaneça dentro do escopo autorizado do Cérebro ou das camadas explicitamente conectadas por uma Hive Mind.

* links em Markdown podem ser importados como relações explícitas;
* PDFs, imagens e anexos podem receber IDs estáveis e relações registradas no banco estruturado;
* projetos, documentos, pessoas, conceitos e dados estruturados podem ser conectados pelo mesmo modelo de relações, sem atravessar implicitamente a fronteira de uma Hive Mind;
* a similaridade vetorial pode sugerir uma conexão, mas não deve transformá-la automaticamente em uma relação permanente sem confirmação do usuário ou uma regra previamente autorizada.

## Indexação vetorial

O índice vetorial é uma projeção reconstruível do conteúdo, não a fonte original dos documentos.

Um fluxo inicial pode ser representado assim:

```text
arquivo original
↓
extração de texto, OCR ou descrição multimodal
↓
divisão em trechos
↓
geração de embeddings
↓
índice vetorial referenciando o documento original
```

Cada embedding deve preservar informações suficientes para uma futura reindexação, como o documento e o trecho de origem, o modelo utilizado, a versão, a dimensão e o hash do conteúdo.

PostgreSQL com `pgvector` é a escolha inicial deste desenho por permitir manter dados estruturados, metadados, relações e vetores na mesma base. Um serviço especializado como Qdrant ou Milvus somente deve ser considerado se volume, desempenho ou isolamento operacional comprovarem essa necessidade.

## Recuperação de memória

Links, IDs e recuperação vetorial cumprem papéis complementares:

* `exact`: resolve IDs, links, entidades e relações conhecidas de forma determinística;
* `semantic`: utiliza busca textual e vetorial para descobrir conteúdo por significado;
* `hybrid`: combina resultados textuais e vetoriais com conexões neurais;
* `auto`: seleciona uma das estratégias a partir da solicitação estruturada.

Uma consulta mais elaborada pode executar recuperação iterativa:

```text
pergunta do usuário
↓
Prefrontal Cortex interpreta a intenção
↓
solicita uma recuperação estruturada
↓
Brain Engine valida identidade, camada neural e permissões
↓
busca por IDs, texto, vetores e conexões conforme a estratégia
↓
deduplica, reordena e devolve evidências com suas fontes
↓
o modelo responde ou solicita uma nova recuperação delimitada
```

Uma capability conceitual para esse fluxo pode assumir a forma:

```text
memory.retrieve(query, scope, anchor_ids, strategy)
```

O Prefrontal Cortex pode propor a estratégia e os termos de pesquisa, mas não acessa diretamente arquivos, PostgreSQL ou o índice vetorial. O Brain Engine permanece responsável por autorização, filtros obrigatórios, isolamento entre camadas neurais e execução das capabilities. Uma recuperação que envolva Hive Mind permanece limitada às camadas explicitamente conectadas e autorizadas; ela não amplia a consulta para o restante dos Cérebros participantes.

## Questões mantidas em aberto

Este desenho ainda não define:

* modelo e provedor de embeddings;
* dimensão dos vetores;
* estratégia de divisão dos documentos em trechos;
* ferramentas de extração, OCR e processamento multimodal;
* algoritmo de fusão, pesos, limites, reranker ou métrica de relevância;
* esquema SQL e contratos finais;
* critérios objetivos para separar o índice vetorial do PostgreSQL;
* regras definitivas para aceitar, rejeitar ou automatizar conexões sugeridas.

---

# Inbox

Inbox é o nome conceitual atual de uma porta de entrada/funcionalidade de captura rápida. Ela não é uma camada neural e o nome final ainda pode mudar.

O usuário não precisa decidir previamente onde cada informação pertence.

Exemplos:

```text
"Dia 15 tenho dentista às 14h."

"Gastei 80 reais no mercado."

"Preciso estudar arquitetura modular."

"Comprar detergente amanhã."
```

Essas informações podem inicialmente ser registradas de forma bruta.

Quando autorizado, o Prefrontal Cortex poderá usar o contexto permitido do usuário para sugerir em qual camada uma entrada se encaixa. Ainda não está decidido se a sugestão exige confirmação, quando a IA pode acessar a entrada ou quem executa a classificação. Qualquer efeito persistente continuará passando pelo Brain Engine.

Entradas destinadas a processamento assíncrono podem gerar jobs para o Hidden Cortex, sempre sob o controle durável do Brain Engine.

Exemplo:

```text
Inbox
↓
"Dentista dia 15 às 14h"
↓
Prefrontal Cortex
↓
Brain Engine API
↓
Calendar Capability
↓
calendar.create_event(...)
```

---

# Sincronização

A sincronização é responsabilidade do Brain Engine e não da LLM.

Ela deve ser determinística.

Exemplo:

```text
Client local
↓
alteração detectada
↓
Brain Engine
↓
versionamento / conflito / persistência
↓
Brain Engine na VPS
```

A VPS poderá funcionar como a principal fonte da verdade do Mindforge.

Os dispositivos possuem cópias locais que eventualmente convergem com o estado central.

Conceitos técnicos relacionados:

* Single Source of Truth;
* local replicas;
* bidirectional synchronization;
* eventual consistency;
* conflict resolution;
* versioning.

Não devemos utilizar uma LLM para decidir coisas simples como se um arquivo precisa ou não ser sincronizado.

---

# Estrutura conceitual de pastas

Uma possível organização inicial do código:

```text
mindforge/

├── src/
│   ├── brain-engine/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── permissions/
│   │   ├── policies/
│   │   ├── validation/
│   │   ├── persistence/
│   │   ├── sync/
│   │   ├── events/
│   │   ├── jobs/
│   │   └── features/
│   │       └── infra/  (infraestrutura operacional autorizada)
│   │
│   ├── hidden-cortex/  (src/hidden-cortex/)
│   │   ├── agent/
│   │   ├── event-handlers/
│   │   ├── schedules/
│   │   ├── automations/
│   │   ├── indexing/
│   │   ├── retries/
│   │   └── background/
│   │
│   ├── capabilities/
│   │   ├── notes/
│   │   ├── calendar/
│   │   ├── tasks/
│   │   └── finance/
│   │
│   ├── data/
│   │   ├── markdown/
│   │   ├── database/
│   │   └── files/
│   │
│   ├── mindforge-client/  (src/mindforge-client/)
│   │   └── prefrontal-cortex/
│   │       ├── agent/
│   │       ├── harness/
│   │       ├── prompts/
│   │       ├── context/
│   │       ├── tool-discovery/
│   │       └── model-gateway/
│   │
│   └── shared/
│       └── agent-contracts/
│
└── docs/
```

Essa estrutura ainda não deve ser considerada definitiva.

Os diretórios futuros sob `src/` nesta árvore são uma organização conceitual; eles não autorizam a criação física de componentes antes de contrato e trabalho aprovados.

Quando forem autorizados, o processo Hidden Cortex será criado em `src/hidden-cortex/` e o client Flutter/Dart em `src/mindforge-client/`; esses diretórios ainda não existem.

Ela serve inicialmente para deixar claras as responsabilidades de cada componente.

---

# Princípio importante

O Mindforge não deve ser pensado como:

```text
Obsidian + IA
```

Mas como:

```text
sistema pessoal de informação
+
armazenamento estruturado
+
conhecimento
+
automação
+
camada agêntica
```

O Obsidian pode ser uma das interfaces.

A IA pode ser uma das ferramentas.

Nenhum dos dois deve ser o próprio Mindforge.

O Prefrontal Cortex pode raciocinar próximo do usuário e o Hidden Cortex pode executar trabalho durável no servidor. Nenhum deles substitui o Brain Engine como autoridade sobre capacidades, regras, permissões e dados.

---

# Decisões técnicas ainda abertas

Esta visão registra responsabilidades e limites, mas ainda não define:

* framework interno das APIs; contratos HTTP aprovados do Brain Engine serão registrados em OpenAPI, enquanto transportes adicionais continuam em aberto; a linguagem do Brain Engine e do Hidden Cortex foi definida como Go;
* transporte utilizado entre client e servidor;
* estratégia de autenticação do client com provedores de modelos;
* custódia e duração das credenciais;
* localização operacional exata do Model Gateway;
* tecnologia de filas, eventos e agendamentos;
* protocolo versionado compartilhado pelos Agent Runtimes;
* persistência de contexto e checkpoints no client;
* mecanismo de transferência explícita de uma tarefa interativa para background.

---

# Estratégia de desenvolvimento

Não tentar implementar tudo inicialmente.

O objetivo é criar uma arquitetura onde funcionalidades futuras possam ser adicionadas sem refazer o núcleo.

Uma primeira versão poderá selecionar um recorte pequeno entre:

```text
Brain Engine

Prefrontal Cortex

Capabilities:
- Notes
- Inbox

Data:
- Markdown
```

Essa lista é histórica e conceitual; ela não promove Inbox, Notes ou Markdown ao Brain Core atualmente contratado.

Depois podem ser adicionados:

```text
Calendar
Tasks
Finance
Projects
Search
Semantic Search
Sync
Mobile Client
Web Client
Automations
Hidden Cortex
Background Processing
```

O desenho deve permitir crescimento incremental.

O princípio central é:

> começar pequeno, mas evitar decisões que impeçam o Mindforge de crescer para o sistema completo imaginado.
