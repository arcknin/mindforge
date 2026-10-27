# Corte develop — API Auth

Autoridade: owner aprovou spec/cinco rotas, TDD backend/client e aplicação necessária na VM em2026-10-13. Owner acrescentou que adicionará a URI Google **depois da entrega**. Nenhuma operação de Google Console, instalação no aparelho, SQL ou mudança main/NSR/Studios é inferida.

## Alvos e preservação

VM indicada no AGENTS. Somente três serviços existentes podem ser recriados: `infra/auth-develop`, `mindforge-brain-engine-develop/brain-engine-develop` e `mindforge-web-develop/web`. PostgreSQL, main, NSR, Studios, outros serviços e volumes ficam preservados. Sem `down`, `--remove-orphans`, pull global, migration ou limpeza de usuários.

Preflight executado pelo principal:50 containers, três alvos running. Snapshots privados de identidade/estado/StartedAt e inspect dos três alvos guardados em `/home/kali/mindforge-api-auth-20261013`, mode0700/arquivos0600. Compose original Auth renderizado com os env privados existentes reproduz exatamente o ambiente em execução. Ainda não houve mudança de serviços no momento deste registro.

Auth existente: Google callback em11000, aud/default role authenticated, issuer sem variável explícita. A versão GoTrue usa `config.JWT.Issuer` na emissão; conservar ausência/valor e demais credenciais. Não substituir issuer por API_EXTERNAL_URL. Allowlist atual já contempla callbacks finais native/Web.

## Ordem de execução

1. Finalizar TDD/regressões Go, Flutter e helpers, analyze/vet/race, build Linux/amd64, APK e Web. Conferir contratos e DTOs na integração; não publicar mocks como prova OAuth.
2. Empacotar binário/Compose/config Nginx/assets Web com manifesto de hashes, sem segredos. Transferir para release privado próprio, conferir hashes/arquitetura e preparar imagens com tags novas antes de tocar nos serviços ativos.
3. Guardar imagens/configurações anteriores. Preparar configuração privada API mantendo DSN/verificador/listener e acrescentando somente callback URL e allowlist. Renderizar overlay Auth com as duas mudanças planejadas de URL e comparar todo o restante com inspect atual. Renderizar Web/NGINX e executar validação de configuração.
4. Recriar exclusivamente API, Auth develop e Web com `up -d --no-deps --no-build --pull never`, mantendo nomes/rede/binds. Mudanças de Auth: `API_EXTERNAL_URL=http://127.0.0.1:11001/v1/auth` e `GOTRUE_EXTERNAL_GOOGLE_REDIRECT_URI=http://127.0.0.1:11001/v1/auth/callback`. Não modificar JWT secret/aud/role/issuer, DB, Google client/secret ou contas.
5. Verificar estado/startup, resposta302 API authorize com `redirect_uri` novo e retorno final correto, rejeições401/400, rotas legadas Web404, Core sem token401, assets Web servidos e ausência de proxy direto GoTrue. Não seguir consentimento Google ou consumir sessão real. Nenhuma mutação documental de teste inferida.
6. Comparar os47 containers protegidos com baseline (ID/estado/StartedAt), além dos env dos três alvos: API somente dois campos novos; Auth somente duas URLs; Web configuração prevista. Guardar resultado sanitizado e rollback disponível.
7. Abrir somente túnel local11001→API18087 com host key estrita e bind loopback. Conflito de porta falha sem encerrar processo alheio. Não executar ADB/install; entregar comandos ao owner.

## Rollback

Se configuração/startup/redirect/proxy falhar, restaurar os três serviços às imagens e configuração efetiva anteriores, em conjunto, usando snapshots privados e tags anteriores preservadas. Configuração Auth original deve continuar disponível e verificável. Não restaurar banco, apagar dados ou revogar contas; nenhuma migration ocorreu nesta tarefa. Registrar se a fronteira antiga voltou temporariamente.

## Prova e pendência externa

Prontidão302/rejeição401/negação404 provam apenas esses caminhos reais, não exchange/refresh/logout ou CRUD autenticados. A nova URI precisa ser adicionada pelo owner no Google Console antes do login. Depois: instalar APK/configurar ADB ou usar Web; consentir; testar sessão/refresh/logout e dados. Endpoints só recebem runtime verified individualmente após evidência real correspondente. Dados locais serão conservados e haverá novo login por namespace Auth versionado.
