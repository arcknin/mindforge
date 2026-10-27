# Mindforge client

O client Flutter oferece entrada somente por Google pela fachada Auth do Brain Engine e organiza as
camadas do Brain Engine. Regras, identidade, validação e persistência continuam
no Brain Engine. O Android inclui documentos Markdown, rascunhos locais e
sincronização automática/manual. O app não inclui senha própria, IA ou acesso
direto ao banco.

## Android — entrega atual

Layouts compactos com camadas → documentos → editor, preservando estado ao mudar
largura. Documentos têm autosave local nominal1s, sync30min durante execução,
botão manual, links porID e lixeira. Timers não executam com Android suspenso.

```sh
flutter build apk --debug \
  --dart-define=MINDFORGE_ENV=develop \
  --dart-define=MINDFORGE_API_URL=http://127.0.0.1:11001
```

APK: `build/app/outputs/flutter-apk/app-debug.apk`, AndroidAPI24+.
API/Auth passam pelo túnel no Mac e ADB reverse via USB. Não alterar Google
Cloud: o callback nativo da fachada é configurado no ambiente develop. Instalação e aceite no
aparelho ficam com o owner; nenhum login real é inferido de build ou testes.

[Guia de teste](../../docs/features/brain/android-workspace/android-test-guide.md)
· [Evidência](../../docs/features/brain/android-workspace/implementation-evidence.md).
Web existente mantém Auth/Core/layout; documentos Web/IndexedDB continuam para
outra entrega. Dados nativos por conta ficam no diretório privado do app,
separados de tokens/PKCE. Não desinstalar para atualizar rascunhos pendentes.

## macOS

Use o Flutter já instalado, sem valores de produção ou segredos no Dart:

```sh
cd src/mindforge-client
flutter pub get
flutter run -d macos \
  --dart-define=MINDFORGE_ENV=develop \
  --dart-define=MINDFORGE_API_URL=http://127.0.0.1:11001
```

O Brain Engine e sua fachada Auth precisam estar disponíveis na porta da API.
O Google OAuth é configurado internamente no serviço Auth; o segredo do client Google não pertence ao app. O retorno
ao app usa `mindforge-dev://auth/callback` e PKCE. No macOS, sessão e verifier
ficam no Keychain de uso único deste app, separados por ambiente e pela API.

## Flutter Web no ambiente develop

O Web usa um único origin configurado, com a fachada Auth do Brain Engine em
`/api/v1/auth`, o restante da API em `/api` e o retorno do OAuth em `/login/callback`. A navegação do Google ocorre
na mesma aba. O callback carrega a aplicação novamente, troca o código PKCE uma
vez e remove query e fragment da entrada atual do histórico antes de processar o
retorno.

```sh
cd src/mindforge-client
flutter pub get
flutter build web --release \
  --no-web-resources-cdn \
  --pwa-strategy=none \
  --dart-define=MINDFORGE_ENV=develop \
  --dart-define=MINDFORGE_WEB_ORIGIN=http://web-mind.home.arpa
```

O origin é restrito a `web-mind.home.arpa` e aceita `http` ou `https`. A sessão
Mindforge e o verifier PKCE ficam somente em `sessionStorage` da aba atual; o
client remove tokens do provedor Google antes de guardar a sessão. O provider
Google retorna à fachada Auth em
`http://127.0.0.1:11001/v1/auth/callback`; a fachada retorna à aplicação em
`/login/callback` no mesmo origin. Esse callback privado exige o túnel SSH já
configurado no Mac durante o login.

O origin HTTP escolhido atende somente à rede privada do laboratório. HTTP não
protege tokens em trânsito. Roboto e CanvasKit entram no pacote e o build não
registra service worker. O engine pode buscar fontes Unicode adicionais em
`fonts.gstatic.com` quando um caractere não existir nas fontes locais; a
interface em português usa Roboto local. `sessionStorage` não é criptografia nem
defesa contra scripts executados no mesmo origin; por isso a página e suas
dependências devem permanecer sob controle. HTTPS continua sendo necessário
antes de qualquer uso público.

## Validação local

```sh
cd src/mindforge-client
flutter test
flutter analyze
```

Os testes não exercitam consentimento Google, rede do origin Web ou navegador
real. O build Web prova compilação e empacotamento local, não login autenticado,
operações reais do Brain, deploy ou uso fora da rede do laboratório.
