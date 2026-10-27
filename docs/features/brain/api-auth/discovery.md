# Discovery — autenticação intermediada pela API

Investigação local em 2026-10-13. Raiz: `/Users/arcknin/Desktop/mindforge`.

## Pedido e limite

Owner pediu uma spec com TDD para que o client fale somente com o Brain Engine, inclusive login e sessão; GoTrue permanece provedor interno. Autoriza documentação/propostas, não implementação ou operação externa nesta unidade.

## Comportamento constatado

| Fonte atual | Evidência | Consequência |
|---|---|---|
| `src/mindforge-client/lib/main.dart` | Compõe `GoTrueSdkPort(authUrl: config.authUrl)` separadamente dos repositories Brain/documentos. | Login/sessão ainda não passam pelo Brain Engine. |
| `lib/auth/gotrue_sdk_port.dart` | SDK `gotrue 2.27.2`, PKCE, exchange, recover, refresh e logout local. | É possível preservar o SDK como detalhe interno apontando para uma fachada restrita da API. Isso ainda não está implementado. |
| `lib/app/app_config.dart` | Native exige Auth11000 e API11001; Web usa `/auth` e `/api`. | Precisam mudar configuração, composição e contratos de transporte. |
| `lib/auth/gotrue_auth_gateway.dart` | Gera tentativa10min, filtra callbacks, serializa refresh, separa gerações e conserva storage em falhas transitórias. | Preservar garantias e testar contra o SDK real, não só port fake. |
| `lib/auth/namespaced_auth_secret_store.dart`, `lib/platform/client_platform_stub.dart` | Namespace documental deriva de `storageNamespace`, que no native deriva da origem Auth11000. | Trocar essa origem sem separar namespaces pode tornar rascunhos/lixeira inacessíveis. |
| `lib/platform/client_platform_web.dart` | Sessão em sessionStorage por aba; documentos Web desabilitados sem adapter durável. | Não ampliar escopo para IndexedDB, cookies de sessão ou documentos Web. |
| `src/brain-engine/auth/gotrue.go` | `GoTrueVerifier.Verify` consulta `/user` interno fixo; exige usuário confirmado, Google, não anônimo, aud/role authenticated; falha fechada. | Reutilizar elegibilidade e principal autoritativo; não emitir JWT próprio ou confiar em claims do client. |
| `src/brain-engine/api/handler.go`, `cmd/brain-engine/{main,runtime}.go` | Só Core/documentos são roteados; Auth interno fixo develop. | Não existe fachada de login/sessão no produtor. |
| `contracts/openapi/registry.yaml`, documentos OpenAPI | Dez operações Core/documentos; nenhuma operação de login/sessão. | Novas rotas precisam ficar proposed até decisão explícita do owner. |
| `features/infra/web-local/nginx/default.conf` | `/auth/{authorize,callback,token,user,logout,settings,health}` vai diretamente ao Auth. | Mesma origem no navegador não significa intermediação pelo Brain Engine. |
| `features/infra/auth-google-develop/compose.auth-google-develop.yaml` e helper de túneis | Callback Google configurável explicitamente; helper encaminha11000→Auth e11001→API. | Planejar corte coordenado de callback/consumidor/proxy/túnel, com autoridade externa separada. |

Paths `lib/` acima são relativos a `src/mindforge-client/`; paths `features/infra/` são relativos a `src/brain-engine/`.

## Fontes primárias e conclusão técnica

- [GoTrue v2.196.0 — external.go](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/api/external.go): authorize cria FlowState; callback PKCE devolve código, erros podem aparecer simultaneamente em query e fragment. A fachada deve filtrar parâmetros e normalizar redirects; copiar Location cegamente não basta.
- [token.go](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/api/token.go): grants separados; troca PKCE verifica verifier, destrói FlowState e pode retornar tokens Google. A API deve remover tokens do provider antes da resposta.
- [logout.go](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/api/logout.go): scope local revoga a sessão, mas ausência de sessão no contexto pode cair em logout global. [auth.go](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/api/auth.go), `maybeLoadUserOrSession`, confirma a diferença entre session_id ausente/UUID zero e sessão não existente; exigir UUID não nulo e verificação confiável.
- [Google provider](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/api/provider/google.go): usa `ext.RedirectURI` explicitamente e scopes email/profile; API_EXTERNAL_URL não substitui esse campo. [Discovery OIDC Google](https://accounts.google.com/.well-known/openid-configuration) confirma o endpoint de autorização esperado. [Token response](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/tokens/service.go) também admite id_token; o DTO allowlisted da fachada o exclui.
- SDK instalado em `/Users/arcknin/.pub-cache/hosted/pub.dev/gotrue-2.27.2/lib/src/`: `gotrue_client.dart`, `fetch.dart`, `types/session.dart`, `types/user.dart`, `constants.dart`. Principal leu construção de authorize (`flow_type=pkce`, `code_challenge_method=s256`), DTOs, signOut local e refresh single-flight com retry transitório limitado pelo tick10s. A fachada não será um passthrough de todas as operações do SDK.
- Leitura obrigatória de arquitetura/stack/contratos concluída. A direção Brain Engine como porta de entrada é compatível com o pedido; a entrega histórica usou outra fronteira de transporte.

Worker `auth_facade_research`, modelo solicitado `gpt-5.6-luna`, esforço `xhigh`, folha read-only, concluiu pesquisa de GoTrue/PKCE/SDK. Principal conferiu diretamente provider Google, external/token/logout, auth.go, DTO de tokens, callback PKCE e SDK instalado. Retorno confirmou fachada restrita, redirect URI explícito e a guarda de sessão não nula; sem alterações/VM/requests autenticados.

## Evidência e incertezas

- Git: branch `feat/brain-core-foundation`, checkout amplamente modificado antes desta unidade; nenhum reset/commit/push.
- Checkpoint Android registra código/APK e SQL/API develop aplicados, com runtime Android autenticado pendente. Web tem aceite histórico de login/camadas/rename pelo owner. Não houve nova prova de serviço nesta investigação.
- Nenhuma conexão à VM, chamada autenticada, SQL ou UI QA executada aqui. Callback/API/proxy novos não existem em runtime.
- A mudança operacional de callback no Google e Auth develop será necessária para o desenho proposto; depende de autorização posterior. Não inferir acesso externo a partir de credenciais antigas.

## Continuação

Spec: [spec-01-main.md](spec-01-main.md). Checkpoint: [api-auth-checkpoint.yaml](api-auth-checkpoint.yaml). Próxima decisão: aprovar ou ajustar as cinco rotas propostas e o corte coordenado, antes de OpenAPI/código.
