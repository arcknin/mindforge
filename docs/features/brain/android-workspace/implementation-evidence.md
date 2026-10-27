# Evidência — client Android, layouts e documentos

Entrega em 2026-10-13. Owner autorizou código/build com TDD, autosave local1s, sync30min/manual e, depois do pacote preparado, backup/migration0007/API develop na VM. [Spec](spec-01-main.md), [checkpoint](android-workspace-checkpoint.yaml), [operação VM](vm-runtime-evidence.md) e [guia de teste](android-test-guide.md).

## Resultado implementado

- Mesmo Flutter com composições compacta/ampla por largura, árvore/panes compartilhados e controllers acima das variantes. Camadas/Google existentes preservados; Android Back e toolbar navegam editor → documentos → camadas, com foco/teclado tratados.
- Documentos: criar/listar/abrir/editar/renomear/mover/excluir/restaurar, origem/subárvore, Markdown fonte/prévia, links porID, destino remoto fora do cache e estados indisponíveis.
- Snapshot nativo privado por conta/origem, commit atômico e proteção contra corrupção; autosave nominal1s. Sync manual/30min com intents persistidos, CAS, reconciliação GET, preservação de edição em voo e isolamento de geração/conta. Logout tenta sync/flush com prazo finito e saída consciente em caso de pendência.
- Android minSDK24/target36, applicationId app.mindforge.client, callback nativo cold/warm, namespace próprio e cleartext somente loopback. Token/PKCE em secure storage separado dos documentos; sem segredo Google no APK.
- Web existente continua compilando e preserva Auth/Core/layout; não foi publicado novamente nem recebeu adapter documental IndexedDB nesta entrega.

## TDD e validação local

Baseline:54 testes Flutter passavam antes das mudanças. REDs registrados: modelos/portas documentais ausentes; configuração/helper nativo ausentes; integração MindforgeApp sem parâmetro/controller documental; teste Android Back esperava document-list-pane e encontrava0 antes do PopScope. Correções de sync foram protegidas por testes de regressão, incluindo perda por placeholder, conflitos/retries, callbacks antigos, limites da lixeira e links externos ao cache. Nenhuma prova de UI real é inferida desses testes.

| Validação | Resultado |
|---|---|
| `flutter test --no-pub` |99 testes PASS após integração final, incluindo os54 anteriores|
| `flutter analyze --no-pub` |PASS, sem issues|
| `flutter build apk --debug ...` |PASS; reconstruído após correção de Back/foco|
| `apksigner verify --verbose` |PASS, assinatura v2; build privado/debug|
| `aapt dump badging` |app.mindforge.client; version0.1.0/code1; SDK24/target36; arm64-v8a, armeabi-v7a e x86_64|
| `flutter build web --release --no-web-resources-cdn --pwa-strategy=none ...` |PASS após última mudança compartilhada|
| Links/YAML/registry e `git diff --check` |PASS no fechamento|

APK: `src/mindforge-client/build/app/outputs/flutter-apk/app-debug.apk`;176.720.655 bytes. SHA-256: `fc7b5cbcfc011941642b3b3b18545f7fe4f878c3813dbf9767ce917beeb0e33e`. Artefato local gerado/ignorado pelo Git, não publicação. Flutter3.38.7/Dart3.10.7 e Android SDK/NDK/Java existentes foram usados; não houve upgrade Flutter nem instalação de emulador/Xcode.

O build Web emitiu aviso de fonte CupertinoIcons ausente e dry-run Wasm informativo, mas compilou. Não houve renderização/browser QA para promover isso a prova visual. Nenhum teste golden, screenshot, emulador ou aparelho foi executado.

Arquivos principais: `lib/documents/`, `lib/ui/workspace/`, `ui/workspace_screen.dart`, `ui/login_screen.dart`, composição `main.dart`/`app/mindforge_app.dart`, AppConfig/platform callbacks, `android/`, testes, pubspec/lock. Shared state/frontend existente preservado; nenhuma mudança de endpoint/schema por causa do client.

## VM e transporte

Conforme evidência operacional vinculada: backup develop validado por checksum/listing;0007 aplicada pelo operador no mesmo ledger/transação, checks e preflight posteriores passaram; somente API develop recriada.49 containers protegidos preservados, Auth/callbacks intactos. Seis métodos/paths documentais responderam401 sem token. Backup permanece na VM; restore não executado.

Túnel Mac aberto com helper versionado e endpoint file privado fora do Git; PID34017 escutava127.0.0.1:11000 e11001 no fechamento (unified exec session5180). Trata-se de prova de binding local, não de chamada HTTP ponta a ponta nem garantia de permanecer ativo após encerramento/suspensão do Mac.

A revisão automática rejeitou a checagem HTTP adicional pelo Mac com motivo “Sem chamadas externas”. Nenhuma chamada rejeitada foi executada nem contornada por outro transporte. As verificações HTTP ocorreram pelo operador autorizado na VM; localmente verificaram-se apenas as portas via lsof. Não houve instalação/ADB reverse no aparelho.

## Limites e próximo aceite

Endpoint states: approved/implemented/runtime pending. DDL/serviço/rejeição401 são provas operacionais; Google/PKCE no aparelho, sessão segura nativa, CRUD/CAS autenticado, sync real e fidelidade visual aguardam instalação/teste do owner. Timers não garantem execução com Android suspenso; rascunho local não significa app integralmente offline. Main/NSR/Studios/volumes não foram alterados. Sem commit/push.
