# Entrega local sem credenciais Google

Owner autorizou executar o plano até onde fosse possível sem credenciais, deixando cadastro Google para depois. Contratos aprovados; implementação local concluída; integração real permanece pendente.

## O que foi implementado

- Quatro contratos em `contracts/openapi/brain-core.openapi.yaml`; registry separa aprovação, implementação e runtime.
- `src/brain-engine/auth/`: verificação remota pelo GoTrue `/user`, identidade elegível Google, deadline3s, sem redirects/cache/proxy ambiental; rejeição e indisponibilidade distintas e opacas.
- `src/brain-engine/api/`: bootstrap, observar, criar e renomear; JSON estrito, limite de body, owner somente do verificador, isolamento404 e CAS409.401 representa rejeição anterior ao Engine; falha de autenticação inesperada depois do Engine vira500, sem promessa de replay seguro.
- `src/brain-engine/cmd/brain-engine/`: composição real de verifier/Engine/PostgresStore. Config trava develop/role/endpoints; probe read-only confirma DB, owner, privilégios limitados e relações Brain antes do Store. Sem migrations no startup ou fallback para Store em memória.
- `src/mindforge-client/`: projeto Flutter somente macOS, Google via GoTrue/PKCE, callback nativo, sessão/verifier no Keychain, refresh/logout e isolamento de gerações; árvore/criação/rename e estados de conflito/resultado incerto. Não há login fictício para contornar credenciais ausentes.
- `features/infra/auth-google-develop/`: overlay seletivo, exemplos/validador de config privada, Compose API separado, builder do pacote e helper de túnel; sem execução externa.

## Testes realmente executados

- Go `test -race -count=1 ./...`: PASS em todos os pacotes após permitir HTTP de teste loopback fora do sandbox. `go vet ./...`: PASS.
- Integração HTTP local: servidor Auth simulado → verifier real → servidor API → Engine real → Store fake; dois usuários sintéticos; bootstrap/árvore vazia/criação aninhada/rename/conflito/isolamento e rejeição sem efeitos: PASS.
- Composição: TDD de target/main/role/owner inválidos, pré-requisitos e abertura do Store somente após o probe correto: PASS. Binário nativo sem configuração foi executado e recusou antes de DB/listener.
- Infra existente static all e unit all: PASS. Guarda obsoleta de registry vazio corrigido mantendo APIs de produto fora do ownership infra.
- Helpers novos:22 testes PASS, com Compose render-only, Go fake e SSH fake. Nenhum daemon/túnel/serviço iniciado.
- Flutter:41 testes unit/widget PASS e `flutter analyze` limpo, conferidos pelo principal. Plists/entitlements válidos. TDD incluiu falha transitória de restore sem apagar sessão, evento SDK durante PKCE, resposta/ação antiga em troca de usuário, replay401 limitado, criação incerta e conflito preservando edição.
- `git diff --check`: PASS. SQL canônico e checks Brain mantêm os hashes da entrega anterior; nenhum novo SQL de domínio.

## Build e pacote

- API compila para Linux/amd64, ELF estático, e Darwin/arm64. O build macOS mencionado aqui é do **binário Go**, não do Flutter.
- Pacote Linux final: `/private/tmp/mindforge-manual-online-package-20261008-r2/`.
- Binário SHA-256: `29e771185d9d19a67c846e5d7f95e4d4eb2d18f716e9a06a8598952b73c62f65`.
- Manifesto conferido contra19 fontes atuais, binário e arquivos de pacote; sem env privado, segredo ou symlink. Builder usa cache fora do pacote e módulos/toolchain offline.
- Flutter instalado3.38.7 / Dart3.10.7. Lockfile adotado: gotrue2.27.2, http1.6.0, flutter_secure_storage11.2.0, app_links7.0.0, url_launcher6.3.2. Duas versões mais recentes exigiam Dart superior; mantido o SDK existente.
- Keychain macOS single-app usa `usesDataProtectionKeychain:false`, alternativa documentada do plugin sem compartilhamento entre apps/perfil de provisioning. Isso permanece Keychain, não armazenamento em prefs. Canal nativo ainda não exercitado. [Documentação do plugin](https://pub.dev/packages/flutter_secure_storage)

## Limites reais

- `flutter build macos --debug` foi tentado e parou porque `xcrun` não encontra `xcodebuild`. Apenas CommandLineTools foi identificado no caminho ativo. Não há `.app` compilado nem prova de callback/Keychain nativo neste host.
- Sem Google client/credenciais/consentimento configurados: login/callback OAuth reais não executados.
- Nenhum acesso à VM, SQL real, deploy, criação Google Cloud, mudança main/NSR/Studios, commit ou push nesta unidade. Os endpoints têm `implementation_status: implemented` e `runtime_status: pending`.
- Widget/HTTP fake/build não são prova fim a fim. Não houve visual/browser QA nem aplicativo aberto para aceitar consentimento.

## Retomada exata

1. Ler a [spec](spec-01-main.md), seção B01 de Google, e o [runbook infra](../../../../src/brain-engine/features/infra/auth-google-develop/README.md).
2. Com autorização específica, configurar Google Cloud/client/callbacks e guardar credenciais privadamente. Não colar Client Secret no chat/Git/Flutter.
3. Conferir/instalar/configurar Xcode completo e ferramentas necessárias com autorização correspondente; repetir build Flutter para comprovar o app nativo.
4. Preparar operação B04 concreta para Auth/API develop e túnel. Aplicar somente com autoridade VM/dados de teste; validar dois usuários Google e persistência real.

A etapa local solicitada está concluída. Estes passos pendentes não foram escondidos por login fake, outro provedor ou deploy sem credenciais.


## Continuação externa posterior

Owner autorizou e a unidade aplicou Auth Google/API/túnel no develop. [Evidência de serviços](runtime-service-evidence.md) supersede a ausência de deployment registrada nesta entrega local; OAuth autenticado/app nativo ainda pendentes.
