---
schema_version: 1
managed_by: spec-drive
feature: brain
task: api-auth
revision: 1
status: implemented
---

# Login e sessão somente pelo Brain Engine

## Contexto e fontes

Owner determinou que o client use somente nossa API, inclusive para login e sessão. Após a spec com TDD, aprovou explicitamente o contrato e a implementação backend/client, incluindo aplicação posterior na VM develop. [Discovery](discovery.md) confronta código, contratos e entregas anteriores. Esta é uma tarefa nova que altera a fronteira Auth de manual-online/Web/Android, sem reescrever seus resultados históricos.

Na discovery, native chamava GoTrue diretamente e Web usava proxy Nginx direto para ele. Camadas/documentos já passam pelo Brain Engine. A intermediação foi aprovada e implementada; [evidência local](implementation-evidence.md) e [corte develop](vm-runtime-evidence.md) registram resultados e limites.

Repositório: `/Users/arcknin/Desktop/mindforge`, Git único, branch `feat/brain-core-foundation`; preservar o checkout existente. Referências: `AGENTS.md`, arquitetura/stack, `contracts/openapi/README.md`, registry, [spec Android](../android-workspace/spec-01-main.md), [manual-online](../manual-online/spec-01-main.md), [Web](../web-local/web-local-checkpoint.yaml).

## Objetivos

- Um único endereço de serviço Mindforge no client; toda operação de login, troca de código, validação/restauração de sessão, refresh, logout, Core e documentos passa pelo Brain Engine.
- GoTrue develop permanece responsável internamente por Google OAuth, emissão/rotação/revogação das credenciais; Brain Engine controla a interface pública, elegibilidade e autorização.
- Preservar contas/IDs, dados, rascunhos, CAS, sync30min/manual, layouts e isolamento entre gerações de sessão.
- No laboratório Android, reduzir para um encaminhamento11001 da API; o navegador continua participando do consentimento Google.

## Fora do escopo

Senha própria, SMTP, Firebase, novos providers, MFA, novo framework, JWT próprio, BFF com cookies HttpOnly, novo banco/tabela/ledger de sessões, migrations, grafo, funções cognitivas, IndexedDB/documentos Web, redesign de UI, implantação main/NSR, domínio público/HTTPS de produção e instalação no aparelho.

Nenhuma operação externa é autorizada por escrever a spec. Túnel/ADB continuam necessários neste laboratório; a fachada não transforma loopback em endereço público.

## Desenho escolhido

### Fronteira e fluxo

```text
Client ── login/sessão/Core/documentos ──> Brain Engine ── Auth interno ──> GoTrue
Navegador ──> API /authorize ──302──> Google ──302──> API /callback
API /callback ── GoTrue interno ──302 filtrado──> retorno do app com código PKCE
Client ── código + verifier ──> API /token ──> GoTrue interno
```

O navegador falar com Google é parte do OAuth. Ele não navegará para um endpoint externo do GoTrue. Client não conhece host Docker, porta Auth, chave administrativa ou segredo Google.

Adotar fachada **compatível apenas com o subconjunto necessário do SDK gotrue2.27.2**, mantendo-o encapsulado em `lib/auth/`. Base do SDK derivada exclusivamente de `apiUrl + /v1/auth`; não há URL Auth independente. Não implementar um proxy catch-all e não expor todas as operações que o SDK conhece. A API valida/seleciona requests, chama upstream fixo, valida respostas, aplica elegibilidade e produz DTO/erros próprios compatíveis com esse subconjunto.

### Contrato aprovado: cinco rotas

Owner aprovou explicitamente as cinco rotas em2026-10-13. Contrato completo em `contracts/openapi/auth.openapi.yaml`; registry marca approved/pending/pending no início da execução, sem gate de decisão. Implementação/runtime serão atualizados somente com evidência.

| ID | Método/path público da API | Entrada e autenticação | Sucesso | Efeitos/repetição |
|---|---|---|---|---|
| auth_authorize | `GET /v1/auth/authorize` | Sem bearer; query exata `provider=google`, `flow_type=pkce`, `redirect_to`, `code_challenge`, `code_challenge_method=s256`. |302 Location HTTPS Google validado. | GoTrue cria tentativa; não renovar automaticamente por falha/reload. Nova tentativa só por ação de login. |
| auth_callback | `GET /v1/auth/callback` | Sem bearer; `state` obrigatório e exatamente um de `code`/`error`; extras Google permitidos abaixo. |302 para callback final allowlisted com código PKCE ou erro sanitizado. | GoTrue valida state e conclui a etapa Google; não reenviar callback após resultado incerto. |
| auth_token | `POST /v1/auth/token` | Sem bearer obrigatório; query única `grant_type=pkce` ou `refresh_token`; JSON estrito segundo grant. |200 DTO de sessão abaixo. | Troca/rotação no GoTrue; não idempotente, sem retry no servidor. |
| auth_user | `GET /v1/auth/user` | Bearer Mindforge/GoTrue develop, sem body/query. |200 DTO mínimo de usuário elegível. | Read-only, repetível; não cria Brain. |
| auth_logout | `POST /v1/auth/logout` | Bearer; query única `scope=local`; body ausente ou objeto vazio para compatibilidade SDK. |204 sem body, revogação da sessão confirmada pelo GoTrue. | Nunca logout global/others; repetição pode retornar401 se sessão já encerrada. |

Não há `/settings`, `/health`, `/admin`, signup, password, id_token, identidades ou wildcard público de Auth. As operações Core/documentos mantêm contratos e bearer existentes.

#### Validação de requests

- Paths exatos, sem barras/segmentos codificados alternativos; método conhecido incorreto405+Allow; rota desconhecida404. Duplicatas de query/chaves JSON, trailing JSON, campos desconhecidos e mistura de grants400. JSON válido UTF-8; conteúdo/form diferente415, body acima16KiB413. Query total até8KiB; campo `state`/`auth_code`/`code` até4096bytes, refresh até8192bytes. Nunca refletir entrada nos erros.
- `authorize`: PKCE obrigatório, challenge base64url sem padding de43 caracteres/32bytes; apenas s256, nunca plain/implicit. `redirect_to` deve ser exatamente uma das duas URIs finais configuradas abaixo, sem prefix matching/wildcard/fragment/query/userInfo. API envia upstream somente provider, redirect_to e challenge/método; `flow_type` é validado e não precisa ser propagado. Scopes extras, audience, invite/linking e params OAuth livres são rejeitados.
- `callback`: query permitida `state`, `code`, `error`, `error_description`, `error_uri`, `scope`, `authuser`, `prompt`, `hd`, `iss`; valores únicos e limitados. Validar state UUID canônico; não interpretar como principal. `iss`, quando presente, deve ser exatamente `https://accounts.google.com`. Enviar ao GoTrue somente state/code ou state/error + descrição limitada. Extras Google são descartados. Não aceitar token em query ou fragment; erro URI não vira redirect. State/fluxo pertencem ao GoTrue, sem state/assinatura própria na API.
- PKCE JSON exato `{auth_code,code_verifier}`; verifier43..128 caracteres RFC7636, sem espaços; SDK gera criptograficamente e guarda no storage seguro existente. Refresh JSON exato `{refresh_token}`. Não aceitar email, senha, owner, aud ou role definidos pelo client.

#### DTOs e elegibilidade

Sessão200 contém **somente** `access_token`, `refresh_token`, `token_type:"bearer"`, `expires_in` positivo e `expires_at` Unixseconds coerente, `user`. API não reemite/reassina JWT nem altera claims. Não incluir `provider_token`, `provider_refresh_token`, metadados arbitrários ou credenciais Google em qualquer nível.

User200/`session.user` usa DTO compatível com `User.fromJson`: `id` UUID, `aud:"authenticated"`, `role:"authenticated"`, `is_anonymous:false`, `created_at` RFC3339, `email_confirmed_at` RFC3339 normalizado da confirmação válida no upstream, `app_metadata:{provider:"google",providers:["google"]}` como projeção de elegibilidade verificada, `user_metadata` limitado a `name`/`full_name` strings até255 caracteres e `email` opcional até320 caracteres. Campos de nome/email são apresentação, nunca autorização; não repassar identities/metadata brutos. Datas/strings inválidas na resposta obrigatória causam503, não defaults permissivos.

Em token200, confirmar o novo access token pelo `/user` interno e comparar ID/sub com o DTO projetado antes de liberar credenciais. Reutilizar a política do verificador existente: confirmado, Google, aud/role authenticated, não anônimo. `/user` público aplica a mesma política. Claims apenas decodificadas nunca autorizam. Se essa verificação falhar após emissão/rotação, não devolver credenciais e não afirmar rollback do GoTrue; tratar a resposta como falha/resultado incerto segundo a seção de sessão.

Logout exige bearer validado e `session_id` UUID não nulo presente, coerente com o token aceito pelo GoTrue; rejeitar antes do upstream caso ausente/inválido/zero. A consulta `/user` não transforma leitura local de claim em verificação de assinatura independente. A fonte `auth.go` confirma que uma sessão não existente com ID não nulo é rejeitada pelo próprio GoTrue; a API conserva essa rejeição. Esse bloqueio impede o fallback global de `logout.go`. Não incluir uma opção para encerrar todas as sessões neste lote.

#### Erros e upstream

Erros Auth usam envelope plano compatível com o SDK: `{code,error_code,message,request_id}`; `code` e `error_code` contêm o mesmo código estável, message genérica. Isso não muda o envelope já aprovado do Core/documentos.

-400 `invalid_request`: shape/params inválidos ou PKCE rejeitado, incluindo tentativa expirada/reutilizada/verifier incorreto. Client encerra essa tentativa, sem retry de exchange.
-401 `unauthenticated`: bearer/identidade inelegível ou refresh definitivamente inválido/revogado/reutilizado. Limpa sessão local; não apagar documentos.
-404 `not_found`,405 `method_not_allowed`,413 `payload_too_large`,415 `unsupported_media_type` e500 `internal`: protocolo/falha interna sanitizados, sem body upstream.
-503 `auth_unavailable`: timeout, transporte, redirect inesperado, body inválido/grande, falha de verificação ou5xx/429 upstream. É falha transitória para o SDK; não mapear indisponibilidade para401. A fachada não repete upstream automaticamente nem garante que exchange/refresh não tiveram efeito.

Upstream privado fixo develop, sem destino/path/host vindo do usuário. Reusar transporte sem proxy ambiente, sem cookie jar e sem seguir redirects. Prazo por operação upstream5s e orçamento total10s, bodies até1MiB; cancelamento propagado. Encaminhar somente Content-Type/Accept e Authorization onde exigido; não encaminhar Cookie, apikey, x-use-cookie, headers de audience/admin, Origin/Referer ou X-Forwarded arbitrários. Referrer do GoTrue vem do `redirect_to` explícito. Não compartilhar cookies entre usuários; este fluxo PKCE não usa cookies de sessão do serviço. Respostas têm no-store/no-cache, no-referrer e nosniff; não registrar query/body/Authorization/Location nem DTO de sessão.

#### Redirects

No authorize, aceitar somente302 com URL HTTPS no endpoint Google de autorização configurado/confirmado na versão atual (`accounts.google.com/o/oauth2/v2/auth`), sem userInfo/fragment. Conferir `redirect_uri` igual ao callback da API, response_type=code, state presente e scopes contendo email/profile, com openid opcional e nenhum outro escopo. Nada de host suffix aberto ou reescrita baseada em Host do request.

No callback, aceitar upstream302 somente com base exatamente igual a um callback final allowlisted; preservar código opaco e parâmetros de sucesso necessários, nunca tokens. Para erro, reconstruir query com `error=access_denied|server_error` e mensagem genérica; remover fragment legado e detalhes upstream. Location fora do allowlist, fragment com tokens, código+erro ou campos inesperados falham503. Callback malformado400, sem fallback para uma URL fornecida pelo client. Não emitir redirect ao host interno GoTrue.

### Client e estado de sessão

- Preservar `AuthGateway` e contratos de repositories/UI; evoluir adapter/composição em vez de mover o app inteiro. SDK interno recebe exclusivamente base da API e cliente HTTP injetável restrito à origem/prefixo Auth e aos paths/métodos propostos; redirects HTTP de token/user/logout não são seguidos. Gerador OAuth só pode abrir `/v1/auth/authorize` derivado da API.
- `MINDFORGE_AUTH_URL` deixa de selecionar serviço e sai dos comandos/guias ativos. Native continua API loopback11001; Web continua base `/api` da mesma origem. Origins/paths de API não são lidos de callbacks ou tokens.
- Restauro de sessão só habilita acesso após validação pela API `/user` ou refresh seguido de validação. Não tratar `GoTrueClient.getSession()` local como prova remota. Rede indisponível conserva credenciais/drafts sem apresentar sessão restaurada como validada. JWT local pode ajudar a agendar refresh, não autorizar dados.
- Manter refresh single-flight e gerações; cobertura com SDK real para auto refresh, refresh no401 e restauração simultâneos. Reaproveitar retry transitório limitado do SDK2.27.2 (tick10s), sem adicionar retry no servidor. GoTrue continua autoridade de rotação/reuse/grace; não ampliar esse intervalo por código. Se uma resposta se perder, um retry suportado pode recuperar ou rejeitar a sessão; não prometer exatamente uma vez. Rejeição definitiva encerra login, conservando documentos;503 conserva storage e informa indisponibilidade. Nenhum retry de token implica reenvio cego de mutações documentais/Core.
- Logout preserva a sequência atual: tentar sync com credencial ativa/orçamento existente, permitir cancelar/sair com pendências, flush local e saída. Chamar somente API/logout local; limpar estado local e parar timers mesmo se a revogação remota falhar. Relatar falha remota sem afirmar204. Token emitido pode valer até expirar, conforme GoTrue; não prometer revogação instantânea de todos os access tokens.
- Callback cold/warm, TTL10min da tentativa, PKCE e cancelamento continuam. Resposta tardia de exchange/refresh/validação/logout não pode restaurar sessão anterior nem gravar credenciais sobre outra conta; liberação/descarte do SDK recebe teste real de geração.

### Compatibilidade dos dados locais

Separar `authStorageNamespace` de `documentStorageNamespace`. Conservar **exatamente** a chave/namespace documental já produzido por plataforma/ambiente no APK anterior, mesmo que contenha a origem histórica11000 como identificador inerte; não usá-la como endpoint. Teste com fixture real desse formato deve carregar rascunhos, lixeira, intents e versões antes/depois da mudança.

Usar novo namespace versionado para credenciais Auth da fachada, derivado da API/ambiente/plataforma. No primeiro acesso após atualização haverá **novo login Google**, sem recriar conta/Brain ou dados. Limpar somente chaves legadas de sessão/tentativa/verifier conhecidas; nunca deleteAll, nunca documento. Não adotar tentativa PKCE iniciada pelo fluxo direto antigo. SessionStorage Web continua por aba; não introduzir cookies/tokens em localStorage. APK/Web antigos com Auth direto deixam de ser consumidores suportados após o corte, sem revogar todas as sessões ou apagar usuários por inferência.

### Corte develop autorizado; Google Console pendente

- Google callback candidato do laboratório: `http://127.0.0.1:11001/v1/auth/callback`.
- Callbacks finais: `mindforge-dev://auth/callback` e `http://web-mind.home.arpa/login/callback`; valores exatos em configuração privada de ambiente, sem wildcard.
- Auth develop: alinhar redirect URI Google explícito e API_EXTERNAL_URL à fachada externa da API; preservar issuer/JWT/aud/role, DB, clientID/secret e demais identidades. Não derivar issuer novo do callback, nem trocar segredo/JWT para resolver roteamento.
- Google Console: adicionar callback novo antes do corte, preservando antigo para rollback temporário. Não criar outro provider/credencial por necessidade imaginada.
- Nginx Web: rotas `/api/v1/auth/*` explícitas encaminham somente ao Brain Engine; `/auth/*` legado404 e nenhuma upstream direta de Auth. Preservar allowlist Core existente; não abrir wildcard para documentos Web ou todas as rotas futuras da API. `/login/callback` continua SPA.
- Mac/Android: helper futuro abre só11001→API; ADB reverse só11001. Túnel Auth11000 e regra ADB antiga são removidos apenas depois do corte validado e dentro da autorização operacional. Binding interno da VM para operadores não equivale a endpoint do client e não será removido por inferência.
- Transportes Web same-origin/Android native não precisam de CORS permissivo. Endpoints JSON sem cookies; aceitar Origin Web somente se igual à origem configurada quando presente; callback browser e native sem Origin não são confundidos com identidade autorizada.

Rollback futuro: restaurar em conjunto release API/clientWeb/APK/configuração Auth e callback/túneis antigos; não apagar DB, contas, rascunhos, chaves Google ou aplicar downgrade de migration. O rollback temporariamente retorna à fronteira antiga e deve ser registrado como tal. Não remover namespace documental nem credenciais Auth anteriores por operação em massa.

## Etapas e ownership de execução futura

| Lote | Entrega | Dependência / saída |
|---|---|---|
| B01 — contrato | Aprovação explícita das cinco propostas; congelar `auth.openapi.yaml` e registry. | Sem código antes da decisão; TDD já escolhido=yes. |
| B02 — API/GoTrue | Adapter interno e política/DTOs/redirects, handlers e runtime config com TDD. | B01; upstream fake sem PostgreSQL, testes de deny/default e elegibilidade. |
| B03 — client | Configuração única API, SDK restrito, validação remota, namespaces separados, fluxo completo TDD. | B01; pode avançar com API fake; integra com B02 antes de build final. |
| B04 — preparação | Ajustar helpers/config examples/runbooks/Nginx local, build Go/Linux/APK/Web e manifesto. | B02+B03 verdes; nenhuma VM/Google/instalação automática. |
| B05 — operação/aceite | Plano operacional concreto, autorização separada, corte develop e prova autenticada assistida. | B04; owner participa do Google e aparelho. Sem essa etapa runtime continua pending. |

Principal integra contratos/docs/composição. Delegação futura é seletiva: API/Auth, client e preparação infra em paths não sobrepostos; não impor revisão automática. Mudanças candidatas:

| Raiz relativa | Arquivos/símbolos | Ação prevista |
|---|---|---|
| repo | `contracts/openapi/{registry.yaml,auth.openapi.yaml}`, tarefa api-auth | Contrato/evidência; OpenAPI somente após aprovação. |
| `src/brain-engine/` | `auth/`, `api/`, `cmd/brain-engine/` | Fachada tipada/configuração/composição/testes; sem regras/SQL Brain/documentos. |
| `src/mindforge-client/` | `lib/auth/`, `lib/app/app_config.dart`, `lib/main.dart`, adapters platform e testes | Endereço API único, sessão/storage e transporte limitado. |
| `src/brain-engine/features/infra/` | `auth-google-develop/`, `web-local/`, testes/helpers pertinentes | Preparação de configuração/callback/proxy/túnel; sem executar. |
| repo | Guias ativos Android/stack/AGENTS se necessário | Registrar regra API-only e comandos novos após implementação, preservando evidência histórica. |

## Dependências e autoridade

Google-only, GoTrue v2.196.0, SDK gotrue2.27.2 e Flutter existentes mantidos; nenhuma nova dependência necessária prevista. GoTrue estado de fluxo/sessões existente é reutilizado. Se surgir necessidade de schema, outro provider, cookies/BFF, nova biblioteca ou transporte, retornar ao owner.

Autorização atual: owner aprovou esta spec, cinco rotas, implementação local com TDD e mudanças necessárias na VM develop de API/Auth/Web/túnel. Preservar main/NSR/Studios/DB/volumes. Google Console será configurado pelo owner depois da entrega; instalação no aparelho, QA visual, produção, commit/push e operações destrutivas permanecem fora. Nenhum SQL previsto; não executar SQL ou banco local/descartável.

## Critérios de aceitação

- [ ] AC01: todos os requests de Auth/Core/documentos originados pelo client usam apenas API; navegação de OAuth não expõe GoTrue. Web também não tem bypass `/auth/*`.
- [ ] AC02: API permite somente cinco rotas/subconjunto Google PKCE; nega grants/admin/provider/redirect desconhecidos antes de chamar upstream.
- [ ] AC03: iniciar/consentir/cancelar/callback/exchange/restaurar/refresh/logout funcionam por API; callback Google termina na API.
- [ ] AC04: DTOs não vazam tokens Google, metadata arbitrária, cookies ou detalhes/URLs internas; principal/elegibilidade vêm do GoTrue confiável.
- [ ] AC05: refresh é single-flight e suporta falha transitória/rotação/incerteza sem apagar drafts ou retry de mutações; identidade inválida não recebe acesso.
- [ ] AC06: logout local não vira global; resposta tardia não afeta outra geração/conta; tentativa antiga direta não é reutilizada.
- [ ] AC07: atualizar app preserva todos os documentos/rascunhos/lixeira/intents por namespace, com reautenticação explícita sem conta nova.
- [ ] AC08: árvore, criação/rename/CAS, documentos e sync30min/manual mantêm comportamento; layout adaptativo não muda.
- [ ] AC09: artefatos/guias usam uma única porta11001 no laboratório; nenhum segredo Google em client/build/Git.
- [ ] AC10: corte e rollback coordenados documentados; main/NSR/Studios/DB/volumes preservados; runtime só verificado com prova real autorizada.

## Validação

Agora: validar links/YAML/IDs/status e ausência de mudanças executáveis. Não executar testes de produto para fingir implementação inexistente.

Na implementação: Go unit/httptest/race/vet/build; fixtures HTTP de GoTrue e Google redirects, sem DB. Flutter unit/widget com relógio/HTTP/storage injetados e **gotrue2.27.2 real** para contrato/sessão. Rodar regressão existente; APK debug/Web release, inspeção de manifests e testes estáticos do proxy/helper. macOS conserva fonte, build condicionado ao toolchain existente; não instalar Xcode. Testes widget não equivalem a QA visual/device.

Runtime posterior: confirmar em ambiente develop o percurso Google→callback API, troca/refresh/user/logout, acesso autenticado Core/documentos, isolamento, preservação de dados e ausência de requests ao Auth direto; owner testa aparelho. HTTP401/302, mocks ou builds isolados não verificam login ou contratos autenticados. Evidência sanitizada sem conta/email/JWT/code/state/verifier/callback bruto.

## Testes e TDD

**TDD=yes**, explicitamente solicitado em2026-10-13. Por lote: escrever teste relevante e observar RED; implementar mínimo; GREEN; regressão/refactor. Não escrever apenas testes que espelham roteamento.

| Ordem / prova | Cenário antes da implementação | Resultado esperado |
|---|---|---|
| T01 / AC01,02 | Capturar SDK/composição atual; login/exchange/restauro/refresh/logout/Core/docs. | Falhar enquanto Auth11000 ou proxy direto existir; depois somente origem API e paths aprovados. Teste de browser launcher separado do HTTP client. |
| T02 / AC02,04 | Providers/grants não permitidos, implicit/plain, params/JSON duplicados, paths codificados, payload grande, redirect host-prefix/traversal e headers forjados. |400/404/405/413/415 conforme contrato, zero chamadas indevidas ao upstream. |
| T03 / AC03,04 | Redirect Google válido/host malicioso/callback incorreto; retorno PKCE/cancelamento/fragment legado/Location com token. | Somente Google esperado e callback final exato; erro sanitizado query-only; inseguro503. |
| T04 / AC03,04 | Exchange e refresh200; token provider presente em nested metadata; /user não Google/anônimo/aud-role inválidos; novo token e user divergentes. | Sessão mínima validada; nenhum token Google; inelegível401, incompatível503; zero efeito Brain. |
| T05 / AC05 | SDK real:401,503, upstream429, timeout após rotação, refresh simultâneo automático/manual/recover. | Uma operação SDK em voo por token; retry transitório limitado; sem loop/fallback direto ou sessão apagada por503. |
| T06 / AC06 | Logout scope ausente/global/others; bearer sem session_id;204,401 e indisponibilidade; login B antes de resposta de A. | Nunca revogar outras sessões; local sai mesmo com falha, stale não contamina B. |
| T07 / AC07 | Snapshot local do APK anterior e namespace derivado de11000; atualização/reautenticação/conta diferente. | Mesmos drafts/lixeira/intents/versões; novo login usa mesmo userID; conta B não lê A; limpeza só de chaves Auth. |
| T08 / AC08 | Regressão de árvore/CAS/doc CRUD/conflitos/sync/resize/hardware Back e callback cold/warm. | Comportamento preservado; erros Auth não confirmam mutação incerta nem apagam rascunho. |
| T09 / AC01,09 | Nginx renderizado, helper SSH e configuração build. | Nenhum proxy direto GoTrue; apenas11001 para cliente; nenhuma configuraçãoAuth independente. |
| T10 / AC03,10 | OAuth/refresh/logout e CRUD reais após autorização, com sessão/conta de teste escolhida pelo owner. | Evidência autenticada por operação/ambiente; nenhuma limpeza/destruição de usuários para provar login. |

## Automação

Reusar helper de túnel/pacote e testes existentes; alterar no lote autorizado para uma porta, sem criar daemon. Helpers recebem arquivo privado fora do Git, não senha em args; falham em conflito de bind/hash/config/target. Reexecução não encerra processos alheios nem reconfigura Auth/Google silenciosamente. Pacote/manifesto reproduzível prepara B05, não o executa. Mudança no Google pode ser instrução manual do owner; não automatizar console sem autorização.

## Riscos e decisões

- Decisão arquitetural, TDD, métodos/paths/DTOs e execução estão aprovados pelo owner. Preservar a separação entre código green, deploy e prova OAuth real.
- Fachada reduz exposição/acoplamento do client, mas Brain Engine passa a estar no caminho crítico de login/refresh. Não alegar disponibilidade/performance maior sem medida.
- Loopback callback depende do Mac e ADB/túnel; público/HTTPS/distribuição independente são outra entrega.
- GoTrue já pode ter emitido/rotacionado credenciais quando a resposta/verificação falha. Não há transação distribuída/exatamente-uma-vez; testes devem demonstrar recuperação/relogin honesto.
- SDK compatível evita reimplementar PKCE/scheduler, mas sua forma de erro/refresh precisa dos testes reais; não trocar biblioteca ou abrir proxy genérico se um teste falhar. Ajuste interno seguro pode evoluir adapter; mudança de contrato volta ao owner.
- Armazenamento de sessão Web continua com os limites do sessionStorage e scripts da mesma origem. Cookies/BFF/HTTPS de produção ficam fora, sem afirmar segurança equivalente.
- Retornar ao owner se for necessário mudar identidade/claims/JWT/SQL, adotar sessão persistida pela API, ampliar provider, contratos ou autoridade externa.

## Aprovações e retomada

- Owner direção: client→API para login/sessão, API→GoTrue interno, confirmada.
- TDD: **yes**.
- Spec/contrato HTTP: **owner-approved em2026-10-13**.
- Implementação backend/client: **concluída com TDD**.
- VM/deploy develop: **autorizados após validação local**. Google Console ficará com o owner; SQL/instalação/QA visual/commit/push fora desta execução.

Checkpoint: [api-auth-checkpoint.yaml](api-auth-checkpoint.yaml). Próxima ação: owner configurar a URI Google e testar o fluxo autenticado conforme test-guide.md. Não reaplicar0007 nem refazer as entregas documentais.

## Autorização de execução — 2026-10-13

Owner aprovou explicitamente esta spec e as cinco rotas, autorizou implementação backend/client com TDD e uso posterior da VM para aplicar as mudanças necessárias. Isso libera B01–B04 e o corte develop B05 da fachada/API/Auth/Web/túnel após validação local. Preservar main/NSR/Studios/DB/volumes; nenhum SQL necessário ou autorizado por inferência. Configuração do Google Console e consentimento/instalação no aparelho continuam dependentes do owner. Pendências de aprovação acima são histórico do momento da proposta e ficam substituídas por esta autorização; código/runtime ainda precisam de evidência.

Owner informou durante a execução que adicionará a URI Google ao final. O corte develop pode avançar após validação local; login real ficará indisponível pelo callback novo até essa configuração manual, e nenhuma prova OAuth será declarada antes disso.

## Resultado de execução — 2026-10-14

B01–B04 concluídos; B05 código/serviços/corte develop aplicados,22 checks operacionais passaram e47 containers protegidos preservados.117 testes Flutter/analyze, Go test/race/vet, builds Linux/APK/Web e nginx-t passaram. Google Console e aceite autenticado permanecem com o owner; não marcar callback de sucesso/token/user/logout/CRUD como runtime verificado sem essa prova. APK/guia/evidências nos links desta tarefa.
