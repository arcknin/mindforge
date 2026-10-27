# Runtime develop — fachada API Auth

Executado pelo principal em2026-10-14 na VM indicada pelo owner; conferências finais registradas até03:47UTC (00:47 de Brasília). Autoridade: aprovação explícita da spec/cinco rotas, implementação TDD e mudanças necessárias na VM em2026-10-13. Owner informou que adicionará a nova URI no Google Console depois da entrega.

## Corte aplicado

Recriados somente API Brain Engine develop, Auth develop e Web develop. Nenhuma operação SQL do operador, migration nova, conta criada/excluída, restore, promoção main ou mudança NSR/Studios/volume. Queries normais de startup/Auth continuam pertencendo aos serviços; não são uma migration operacional desta tarefa.

- Release: `/opt/mindforge/releases/api-auth-20261013-r1`.
- API final: `api-r3/`, imagem `mindforge/brain-engine:api-auth-20261013-r2`, image ID `sha256:0cc93b6395b9161caba0447e82d195cdd7f4ec2faf53c7e3558cd1b4d347354c`; running. Binário SHA256 `b6a563f1160667b96359ea770b8062431c688fb392bcd17d7a85b3fe12122001`,10.911.928bytes. Fonte/manifesto e arquivos de pacote conferidos.
- Web final: `web-r2/`, imagem `mindforge/web-develop:api-auth-20261013-r2`, image ID `sha256:0d8eaf4d0126d0273b500843eeb411fd4001177811b0255b73d0c62ae898a475`; running/healthy. `nginx -t` passou. JavaScript servido corresponde ao build final: SHA256 `fc0b5f3e6ca5fed8dd985ebb2b5a30e1f4f5e2c370ddf3592074a96071ca5329`;38 arquivos do pacote conferidos.
- Auth: mesma imagem `supabase/gotrue:v2.196.0`, running/healthy. Ambiente efetivo comparado integralmente: somente `API_EXTERNAL_URL` e `GOTRUE_EXTERNAL_GOOGLE_REDIRECT_URI` mudaram. JWT secret/aud/default role/issuer, DB e Google client/secret preservados; issuer explícito continua ausente.
- API: DSN/verificador/listener e PATH anteriores preservados; somente dois campos Auth acrescentados: callback API e allowlist exata native/Web. Binds API18087, Web18088, Auth10000 mantidos no loopback; rede existente preservada.

As imagens precedentes e os manifests/configurações efetivos anteriores continuam disponíveis para rollback. Evidências/configurações privadas mode0600 em `/home/kali/mindforge-api-auth-20261013` (diretório0700); nenhum segredo foi copiado ao Git ou aos artefatos client.

## Prova real e correção durante integração

O primeiro teste de cancelamento revelou503: GoTrue legítimo retorna marcador vazio `sb` no fragmento. Foi reproduzido em teste RED, corrigido para permitir somente o marcador vazio conhecido e removê-lo do retorno, seguido de GREEN Auth/API/race/vet e rebuild da API. Prova real posterior passou para native e Web. Não se alterou contrato nem abriu passthrough genérico.

Conferência final: **22 checks de API/proxy passaram**, mais3 checks do vhost externo:

| Caminho | Resultado observado | Limite |
|---|---|---|
| API authorize, final native e Web |302 Google, redirect_uri exatamente callback API11001, state presente e redirect_to final correto | Início OAuth real; não consentimento Google/login concluído. |
| API callback com FlowState válido + access_denied |302 final correto, erro sanitizado query-only, fragmento removido | Denial injetado pelo operador para testar cancelamento; sem identidade Google ou código de sucesso. |
| Callback sem state |400 invalid_request | Validação antes do upstream. |
| Token PKCE inválido |400 invalid_request | Rejeição real GoTrue normalizada; não emissão de sessão. |
| User com bearer inválido |401 unauthenticated | Não prova user200 elegível. |
| Logout sem bearer / scope global |401 /400 | Não prova revogação de sessão real. |
| Auth admin não contratado |404 | Sem exposição administrativa. |
| Core/documentos sem token |401 | Não prova CRUD autenticado/CAS/sync. |
| Web `/auth/authorize`, `/auth/token`, `/auth/health` |404 | Bypass legado bloqueado. |
| Web aliases de path codificado Auth |404 | Query percent-encoded do authorize continua válida. |
| Web proxy Auth user / Core |401 | Encaminhamento à API, sem proxy direto GoTrue. |
| Web authorize |302 com callback API correto | Mesma fronteira pelo proxy Web. |
| Web callback SPA / JavaScript |200 | Assets servidos, sem QA visual/browser. |

Respostas Auth observadas têm no-store; URL/Authorization/body/state/código bruto não estão nesta evidência. Provas privadas: `runtime-proof.json`, `final-metadata.json`, snapshots before/after e logs específicos de build/render/up/rollback. Não houve rollback de banco ou serviço no corte bem-sucedido.

O vhost do host `web-mind.home.arpa` também foi chamado via loopback da VM com Host explícito: SPA200, `/api/v1/auth/user`401 e `/auth/user`404. Evidência privada `host-vhost-proof.json`; não é prova de DNS/browser no Mac.

## Preservação e transporte

Baseline50 containers. Os **47 protegidos** mantiveram exatamente ID, estado e StartedAt na comparação final; inclui PostgreSQL, main, NSR e Studios. Os três alvos permitidos mantiveram ambiente esperado, binds e rede; somente artefatos/URLs previstos mudaram.

Túnel Mac aberto pelo helper versionado, sessão92545, SSH PID18043; `lsof` constatou somente `127.0.0.1:11001` em LISTEN, sem listener11000. Mapeamento11001→API18087. Esse é um snapshot, não garantia de disponibilidade futura; [guia](test-guide.md) mostra como reabrir. Nenhum ADB/install/emulador/QA visual executado.

## Estados e próximo aceite

Cinco contratos approved e implementações implemented. `auth_authorize` runtime verified para iniciação302 native/Web. Callback tem apenas ramo de cancelamento comprovado; token/user/logout autenticados e callback de sucesso permanecem pending. Core/documentos mantêm estados anteriores; mocks/build/401 não os promovem.

Pendente com owner: adicionar `http://127.0.0.1:11001/v1/auth/callback` ao cliente OAuth Google existente; depois testar login/código/sessão/refresh/logout e dados no aparelho/Web. A mudança de namespace Auth requer novo login, preservando conta e documentos. Nenhuma conta foi apagada para fazer essa prova.
