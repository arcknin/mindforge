# Assistente pessoal: RAG, memória, harness e MCP

- **Registro:** 2026-10-18; consolidação da discussão com o owner.
- **Natureza:** visão de produto e propostas técnicas; não é spec executável.
- **Autoridade desta unidade:** somente documentação. Não autoriza código, schema, migrations, serviços, dependências, endpoints, deploy ou operações na VM.
- **Decisões canônicas:** [PD-023..025](../../product-decisions.md#pd-023--assistente-pessoal-contínuo-do-cérebro).
- **Pendências canônicas:** [OD-017..020](../../open-decisions.md#od-017--memória-conversacional-autonomia-retenção-e-esquecimento).

## 1. Visão geral

O Mindforge deve oferecer um assistente pessoal contínuo do Cérebro do usuário: uma extensão de sua organização e memória, descrita pelo owner como uma segunda persona. Essa metáfora não concede consciência, infalibilidade nem autoridade própria ao modelo. A experiência deve permitir conversas naturais sem exigir que o usuário administre chunks, janelas de contexto ou protocolos.

Permanece um Cérebro por usuário, organizado em camadas e subcamadas. A futura Hive Mind conecta apenas subárvores autorizadas de Cérebros diferentes; não constitui um Cérebro coletivo. Funções Cognitivas continuam com definições da equipe e estado próprio por uso. Esta discussão não antecipa sua implementação.

### Exemplos de uso do owner — ilustrativos, não contratos

| Pedido | Necessidade a contemplar |
|---|---|
| “Anote isso: [fala]” | Capturar conteúdo e resolver destino explícito ou futura entrada rápida |
| “Anote na camada Estudos / curso de investimento” | Resolver identidade e destino sem depender apenas de nomes |
| “O que eu fiz ontem?” | Consultar acontecimentos/registros com interpretação temporal adequada |
| “Minha agenda amanhã está como?” | Consultar eventos no intervalo e fuso pertinentes |
| “Iniciei a tarefa X agora, marque o tempo” | Resolver tarefa/camada e solicitar operação determinística |
| “A do curso”, após uma pergunta de desambiguação | Preservar candidatos e pergunta pendente |
| “Onde escrevi sobre diversificação?” | Descobrir evidências documentais relevantes |
| “Resuma todas as anotações deste curso” | Ler o conjunto com cobertura, não apenas os trechos mais semelhantes |

Inbox é o nome conceitual encontrado nos registros, não uma camada neural. O owner mencionou outro nome, mas não foi recuperado um nome substituto confirmado; OD-003 permanece aberto.

## 2. Decisões confirmadas nesta discussão

As decisões são de direção de produto, não aprovação de mecanismos técnicos:

- **PD-023:** assistente pessoal contínuo, com possibilidade de utilizar diferentes modelos, mantendo a experiência de continuidade sob responsabilidade do Mindforge.
- **PD-024:** troca de modelo deve ter trava no aplicativo e aviso ao usuário, evitando troca no meio de uma operação em andamento. Estados exatos e política de espera/cancelamento ainda são proposta.
- **PD-025:** suporte nas duas direções de MCP: consumir servidores externos e oferecer capacidades Mindforge a outros clientes autorizados.

As fronteiras já existentes permanecem: Brain Engine é a autoridade sobre autenticação, autorização, regras, persistência e efeitos; Prefrontal Cortex interativo no Flutter; Hidden Cortex independente para trabalho durável futuro. Operação manual não depende de LLM. O client continua usando Brain Engine para login/sessão; MCP não autoriza acesso direto ao GoTrue ou ao banco.

## 3. Conceitos e responsabilidades discutidos

| Parte | Papel conceitual |
|---|---|
| Harness / Prefrontal Cortex | Conduzir turnos, montar contexto, selecionar ferramentas e acompanhar operações |
| Modelo | Interpretar pedidos, propor ações e produzir respostas |
| RAG / recuperação | Buscar evidências autorizadas para apoiar a resposta |
| Memória conversacional | Preservar continuidade além de uma chamada ou janela de um provedor |
| MCP | Protocolo de integração com ferramentas, recursos e serviços |
| Brain Engine | Validar e executar operações autoritativas |

RAG é uma abordagem; MCP é um protocolo. Um não exige o outro. Embedding da pergunta serve à recuperação; a pergunta original e os textos recuperados continuam relevantes para a geração da resposta. Não se presume que todo dado deva virar vetor nem que todo pedido exija busca semântica.

## 4. Propostas técnicas — ainda não aprovadas como solução

### 4.1 Continuidade independente do provedor

Manter estado da conversa no Mindforge e construir uma projeção limitada para cada chamada/modelo. Histórico, resumos, referências e estado de operações seriam separados. IDs, candidatos de desambiguação, autorizações e resultados incertos não deveriam depender apenas de um resumo narrativo.

Não existe janela infinita compartilhada entre APIs. Dados opacos de continuação de um provedor não são memória universal transferível. Localização, formato, sincronização e retenção desse estado continuam abertos; persistência compartilhada permanece sob Brain Engine.

Proposta para PD-024: enfileirar a troca durante geração/ferramenta, aplicá-la após turno concluído e reconciliar efeitos incertos antes da troca. Aviso sugerido: “A troca será aplicada após concluir a operação atual.” Cancelar geração não comprova cancelamento de um efeito enviado. A política final de falha, timeout e recuperação ainda precisa de contrato.

### 4.2 Adaptadores de modelos

Adaptadores específicos por provedor/protocolo e uma interface genérica para endpoints compatíveis. Seleção por capacidades verificadas, não por fama. Texto, streaming, ferramentas e demais recursos não são presumidos equivalentes; suporte genérico não promete compatibilidade integral. Provedores, SDKs, mecanismos de detecção e fallback não foram escolhidos.

### 4.3 Ferramentas de leitura e recuperação

Oferecer operações adequadas ao objetivo: busca textual/vetorial, leitura por ID, leitura de seção, consulta estruturada por tempo e agregações de domínio. Esses nomes são categorias conceituais, não nomes finais de tools nem rotas aprovadas.

Chunks/embeddings seriam projeções reconstruíveis, vinculadas às fontes e versões. A busca retornaria trechos com identidade, camada e localização; permitiria ampliar a leitura. O agente não precisa navegar pela UI nem consumir exatamente a mesma listagem do usuário.

Paginação/lotes também podem existir internamente para controlar volume. Harness/ferramentas podem coordenar os lotes. Busca top-K não garante cobertura para resumir uma coleção inteira. Manifesto de documentos/versões, fotografia estável ou detecção de mudanças são alternativas a detalhar, não um mecanismo escolhido.

Comparar documento pequeno inteiro, chunks fixos e divisão por títulos/parágrafos Markdown com limites. Comparar busca lexical, semântica e híbrida antes de escolher parâmetros ou acrescentar reranker/contexto gerado. O desenho arquitetural anterior menciona PostgreSQL/pgvector como direção inicial conceitual; este registro não o promove a nova adoção, instalação ou contrato.

### 4.4 Memória e formação de documentos

Separar histórico original, resumo derivado, afirmação extraída e documento editável. O harness poderia detectar candidatos: pedido explícito de anotação, decisão confirmada, procedimento reutilizável ou síntese relevante da conversa. Primeiro sugerir gravações inferidas é uma recomendação, não uma política aprovada de autonomia.

Preservar por afirmação sua origem, natureza, tempo, escopo e correções. Tipos definidos pelo harness e preenchidos/propostos pelo agente são uma hipótese de implementação; validade do formato não comprova validade semântica. Não permitir que uma inferência salva reapareça como confirmação humana.

Exemplo: “estou pensando em estudar à noite” é intenção considerada, não rotina estabelecida. “Fechei terça e quinta” é uma decisão relatada. “Corri ontem” pode ser conservado como relato explícito do usuário, distinto de registro de sensor; não se presume a necessidade de reconfirmar toda fala. Editar uma nota sobre corrida ou agendar um evento não comprova execução/comparecimento.

Timestamps podem distinguir ocorrência, registro, alteração e indexação; campos finais ainda não definidos. Versões/sequência e checagens de fonte complementam datas. Diferenciar rascunho local, estado sincronizado e índice atualizado; um contador de versão não implica histórico de conteúdo conservado.

### 4.5 MCP interno e externo

PD-025 confirma ambas as capacidades. Proposta de fluxo interno: assistente → cliente MCP no harness → servidor MCP Mindforge → operações do Brain Engine. Servidor MCP seria uma entrada para as mesmas regras, não uma autoridade paralela nem acesso direto ao banco. A interface manual pode continuar usando API HTTP.

Localização/processo do servidor MCP, transporte, autenticação, escopos, descoberta seletiva, consentimento e contratos de ferramentas permanecem abertos. Integrações externas têm regras e efeitos próprios; consumir MCP externo não concede permissões sobre o Mindforge nem implica credenciais compartilhadas. MCP não substitui políticas do harness ou autorização do servidor.

### 4.6 Avaliação antes de escolher a solução

Montar episódios em português com perguntas temporais, referências ambíguas, correções, troca de modelo, compactação, rascunhos não sincronizados e timeout após envio de uma ação. Avaliar recuperação e cobertura, fidelidade das respostas, falsas memórias, isolamento, efeitos duplicados, latência e custo. TDD/recorte executável deste trabalho ainda não foi fechado; esta lista é proposta de avaliação, não testes implementados.

## 5. Questões em aberto

- **OD-017:** retenção de conversas/resumos/memórias, esquecimento e propagação de exclusões; autonomia para criar/alterar documentos ou memórias; correções e contradições; significado de confirmação.
- **OD-018:** escopo padrão, orçamento de contexto, fonte/versão, cobertura, frescor e inclusão de rascunhos; chunking, embeddings, indexação e métricas de aceitação.
- **OD-019:** provedores/modelos permitidos, capacidades mínimas, credenciais, privacidade, envio a serviços de embeddings e fallback; comportamento completo da trava e retomada.
- **OD-020:** contratos e governança MCP interno/externo, identidade, autorização, consentimento, ferramentas oferecidas e primeiro recorte.

Decisões de produto/privacy exigem owner. Parâmetros e alternativas técnicas podem ser investigados e medidos sem transformar experimento em escolha de produto ou executar mudanças não autorizadas.

## 6. Referências de pesquisa discutidas

Fontes são evidência técnica externa, não aprovação ou escolha de bibliotecas:

- [MCP: arquitetura](https://modelcontextprotocol.io/docs/learn/architecture) — papéis host/client/server.
- [Context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — composição de contexto e compactação.
- [Contextual Retrieval](https://www.anthropic.com/engineering/contextual-retrieval) — alternativa de contexto nos chunks a avaliar.
- [pgvector](https://github.com/pgvector/pgvector#hybrid-search) — referência de busca híbrida, não instalação autorizada.
- [LongMemEval](https://arxiv.org/abs/2410.10813) — referência de avaliação de memória, sem garantia de resultados no Mindforge.

## 7. Estado e próximo limite

Registro documental concluído; nenhuma spec executável de RAG/harness/MCP aprovada nesta unidade. Nenhum endpoint, tool, schema, pacote, serviço ou modelo novo contratado, implementado ou verificado em runtime por este registro. Contratos existentes em OpenAPI permanecem inalterados.

Próximo trabalho possível: discutir as pendências ou autorizar um recorte de discovery/spec. Não iniciar implementação a partir desta visão geral.
