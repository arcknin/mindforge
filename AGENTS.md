# Mindforge agent guidance

Estas instruções valem para todo o checkout. O owner mantém autoridade sobre decisões de produto, contratos, persistência, publicação, deploy e mudanças destrutivas.

## Leitura obrigatória

Antes de planejar ou alterar o projeto, leia nesta ordem:

1. `docs/architecture-overview.md`
2. `docs/stack/readme.md`
3. `docs/stack/front-end/readme.md`
4. `docs/stack/back-end/readme.md`
5. `contracts/openapi/README.md`
6. `contracts/openapi/registry.yaml`

Use os arquivos atuais como autoridade. Desenhos históricos, exemplos de capabilities e memória de sessões não substituem contratos aprovados.

## Direção atual

- O Mindforge é um sistema pessoal de informação, não apenas um editor Markdown ou uma integração com Obsidian.
- Existe conceitualmente um único Cérebro por usuário, organizado hierarquicamente em camadas neurais e subcamadas.
- Uma Hive Mind não é outro Cérebro nem um Cérebro coletivo. Ela é uma conexão/rede neural coletiva entre Cérebros de usuários diferentes. Cada camada ou subcamada explicitamente selecionada funciona como raiz de uma subárvore compartilhada; ancestrais, irmãs e outros ramos permanecem privados. A inclusão automática de descendentes criados depois da conexão ainda não está decidida.
- Inbox é uma porta de entrada/funcionalidade futura de captura rápida, não uma camada neural. A eventual sugestão de destino por IA não transfere autoridade do Brain Engine nem está contratada no Brain Core.
- O Brain Engine é a única autoridade sobre autenticação, autorização, regras, validação, persistência, effects e estado durável.
- O Prefrontal Cortex executa no client Flutter/Dart e pode usar tools locais registradas ou solicitar capabilities remotas.
- O Hidden Cortex será um processo Go independente para trabalho durável. Ele não acessa dados autoritativos diretamente nem se torna uma autoridade paralela.
- Operação manual deve continuar possível sem LLM. Agentes solicitam ações; não substituem as regras determinísticas.
- PostgreSQL, retrieval, sincronização, filas, Model Gateway, credenciais e protocolo ainda possuem pontos evolutivos. Não transforme desenho conceitual em decisão fechada.

## Estrutura e Git

- Adote a estrutura-alvo registrada em `docs/stack/readme.md`.
- Quando o Git for inicializado, deve existir somente `.git/` na raiz.
- Nunca execute `git init` dentro de `src/brain-engine/`, `src/mindforge-client/`, `contracts/` ou outros componentes.
- Preserve mudanças existentes e limite cada alteração aos arquivos autorizados.
- Backend e client permanecem módulos independentes mesmo dentro do monorepo.

## Fluxo obrigatório para endpoints

1. Procure primeiro o endpoint em `contracts/openapi/`, na implementação real do Brain Engine, nos testes e nos consumidores.
2. Não converta nomes conceituais de capabilities, diagrams ou exemplos em rotas HTTP.
3. Quando uma operação ainda não estiver aprovada, registre-a apenas como `proposed`. Método, path, autenticação, request, response, erros e idempotência devem permanecer explícitos ou marcados como pendentes.
4. Peça decisão do owner antes de promover a proposta para `approved`. Propostas ficam em `registry.yaml`; somente contratos aprovados entram em um documento OpenAPI.
5. Depois da aprovação, registre o contrato completo em um documento OpenAPI e atualize `registry.yaml`.
6. Implemente apenas o produtor e os consumidores autorizados pelo contrato; então marque `implementation_status: implemented` com evidência local.
7. Marque `runtime_status: verified` somente após uma chamada real em ambiente identificado, com resultado sanitizado e compatível com o contrato.
8. Se código, contrato e consumidor divergirem, use `gate: needs_decision`. Esse gate é um bloqueio, não um estado do lifecycle. Não invente fallback nem altere o contrato silenciosamente.

## Estados e evidência

Mantenha separadas estas dimensões:

- `contract_status`: `proposed` ou `approved`.
- `implementation_status`: `pending` ou `implemented`.
- `runtime_status`: `pending` ou `verified`.

Documentação ou aprovação não prova implementação. Código, mock, teste unitário ou build local não prova runtime. Não declare endpoint entregue, publicado ou funcional sem a evidência correspondente.

## Limites de implementação

- O client não acessa PostgreSQL, SQL ou dados autoritativos diretamente.
- Models não recebem shell arbitrário, SQL, paths irrestritos ou acesso direto a índices.
- Tools locais devem ser explicitamente registradas e limitadas ao dispositivo; persistência compartilhada passa pelo Brain Engine.
- Hidden Cortex consome jobs/eventos e solicita efeitos por capabilities do Brain Engine.
- Não adicione framework, transporte, autenticação, fila, embedding, índice ou serviço externo sem decisão registrada.
- Não crie todos os diretórios da estrutura-alvo antecipadamente; crie cada componente quando seu primeiro trabalho for autorizado.

## Migrations e execução de banco

- SQLs e migrations são criados e versionados localmente no repositório.
- Nunca sugerir, criar ou usar Docker, PostgreSQL ou outro banco temporário/descartável para testar, aplicar ou validar migrations.
- A execução de SQL deve ocorrer somente na infraestrutura ou VM explicitamente indicada pelo owner.
- Ausência de acesso, endpoint ou credencial é bloqueio; não autoriza uma alternativa local.
- Endpoint, usuário, porta, credenciais e segredos da VM permanecem fora do Git.

## Handoff do orquestrador

Ao concluir uma unidade, informe de forma compacta:

- escopo alterado;
- contrato ou decisão usada;
- arquivos modificados;
- validação realmente executada;
- limites da evidência;
- estado de endpoints afetados;
- próximo gate ou decisão do owner.
