# Evidência de implementação — API Auth

Owner aprovou spec/cinco rotas e execução TDD backend/client, com aplicação develop em2026-10-13. Esta evidência separa fonte/testes, artefatos e runtime. Nenhum commit/push.

## Backend B02

Folha `api_auth_backend`, modelo solicitado `gpt-5.6-luna`, esforço `max`. Paths: `src/brain-engine/auth/{facade.go,facade_test.go,gotrue.go}`, `api/{auth.go,auth_test.go,handler.go}`, `cmd/brain-engine/{main.go,runtime.go,runtime_test.go}`.

RED: símbolos da fachada ausentes; fixtures fiéis do GoTrue sem PKCE no redirect Google e com `redirect_to` final; normalize404/422 de PKCE; JWT sub/exp incompatíveis; Origin hostil/nulo/duplicado; orçamento global ausente. GREEN: `go test ./...`, `go test -race ./api ./auth ./cmd/brain-engine`, `go vet ./...`, build nativo CGO=0. Principal conferiu código/DTOs/composição, identificou e acompanhou correções de compatibilidade com fonte oficial do GoTrue.

Fachada restrita Google PKCE, DTO mínimo e verificação interna; grants não autorizados negados. Upstream fixo develop, sem cookies/proxy ambiente/redirecionamento aberto. Callback Google da API distinto do retorno final do app; erros legados fragment/query sanitizados. Logout session_id não nulo evita fallback global. Core/documentos preservados.

## Infra B04 local

Folha `api_auth_infra`, `gpt-5.6-luna`, esforço `max`; ownership `auth-google-develop/` e `web-local/`. RED: Auth23 testes com3 falhas e Web9 com7 falhas; follow-up RED de dois novos campos runtime ausentes. GREEN final: Auth23 e Web9 testes.

Nginx possui somente cinco rotas Auth API explícitas; `/auth/*`404 e nenhum upstream direto GoTrue. Core allowlist mantida, documentos Web fora. Helper encaminha somente11001→API18087. Validação/exemplos/Compose exigem callback/allowlist do backend; package manifest inclui domínio documental. Nenhum segredo, SQL ou operação VM por essas folhas.

## Artefato API

Pacote credential-free Linux/amd64 preparado pelo helper versionado em `/private/tmp/mindforge-api-auth-package-r3`; manifesto conferido pelo principal,34 fontes e7 arquivos de pacote. ELF estático x86-64,10.911.928bytes, SHA256 `b6a563f1160667b96359ea770b8062431c688fb392bcd17d7a85b3fe12122001`. Sem credenciais Google/DSN privado no pacote.

## Client / integração / runtime

Folha `api_auth_client`, `gpt-5.6-luna`, esforço `max`, implementou SDK real apontando exclusivamente à API, HTTP restrito, validação remota e namespaces separados. RED inicial por adapter/configuração/storage ausentes; RED de resposta tardia e limpeza do verifier de tentativa B por tentativa A. Correções: instância SDK por tentativa, guarda de geração e `OwnedPkceStorage` com compare-and-delete serializado por store. Teste de interleaving garante que write novo não fique entre comparação e remoção. Refresh com identidade diferente rejeitado sem assumir troca de conta. Fixture de storage conserva rascunho/lixeira/intents.

GREEN final: `flutter test`117 testes, `flutter analyze` sem issues. O último build da folha esbarrou em locks/socket Gradle e telemetry fora do sandbox; principal reconstruiu ambos com permissão do ambiente. APK final:176.747.747bytes, SHA256 `808fef63509667a56fcb82adf63d2d56d12a1019443887327cf7768e20dadc42`, assinatura v2 verificada, appID `app.mindforge.client`, minSDK24/target36. Build Web release sem CDN/service worker passou, main.dart.js SHA256 `fc0b5f3e6ca5fed8dd985ebb2b5a30e1f4f5e2c370ddf3592074a96071ca5329`; warning de fonte CupertinoIcons existente e sugestão Wasm informativa. Nenhuma QA visual/device.

Builds: `flutter build apk --debug --no-pub --dart-define=MINDFORGE_ENV=develop --dart-define=MINDFORGE_API_URL=http://127.0.0.1:11001`; Web `flutter build web --release --no-pub --no-web-resources-cdn --pwa-strategy=none --dart-define=MINDFORGE_ENV=develop --dart-define=MINDFORGE_WEB_ORIGIN=http://web-mind.home.arpa`. Pacote Web com38 arquivos/manifesto, incluindo proxy Auth raw-path guard; fonte API congelada corresponde às34 entradas do manifesto.

Pré-corte: configurações privadas comparadas; API imagem nova construída, nenhum serviço alterado até iniciar a operação registrada em `vm-runtime-evidence.md`. Owner adicionará URI Google ao final. Não atribuir prova OAuth completa a mocks/build/302/401.

Última regressão: cleanup de tentativa usa identidade, porque aceitar usuário incrementa geração. Teste de login seguido de tokenRefreshed garante atualização da credencial/storage. API também recebeu RED/GREEN para marcador `sb` vazio observado no GoTrue real; testes Auth/API/race/vet e pacote final recompilados. [Corte e22 checks reais](vm-runtime-evidence.md) concluídos; sessão Google autenticada permanece dependente do owner.
