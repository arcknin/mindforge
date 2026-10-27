> Continuação API-only: o transporte deste guia é histórico do APK anterior. Para o novo APK/Auth, use [o guia atual](../api-auth/test-guide.md), com apenas a API11001. A spec aprovada mudou login e sessão para o Brain Engine.

# Teste privado do client Android

APK esperado: `src/mindforge-client/build/app/outputs/flutter-apk/app-debug.apk`. Build debug do laboratório, sem publicação. Requisitos: AndroidAPI24+, USB/debugADB autorizado pelo usuário, Mac com túnel e VM develop preparada conforme [plano VM](vm-develop-plan.md). Não iniciar emulador nem instalar no aparelho por este documento automaticamente.

## Build local

```sh
cd /Users/arcknin/Desktop/mindforge/src/mindforge-client
flutter build apk --debug \
  --dart-define=MINDFORGE_ENV=develop \
  --dart-define=MINDFORGE_AUTH_URL=http://127.0.0.1:11000 \
  --dart-define=MINDFORGE_API_URL=http://127.0.0.1:11001
```

## Transporte e instalação pelo owner

Manter o túnel SSH existente do Mac aberto com o helper versionado e arquivo privado já preenchido (argumento `--file`). Ele encaminha Mac11000 -> Authdevelop10000 e Mac11001 -> APIdevelop18087 na VM. Dados SSH permanecem fora do Git; não usar localhost do Mac diretamente como se fosse acessível ao celular.

Com exatamente um aparelho selecionado em ADB, o owner pode executar:

```sh
adb devices -l
adb reverse tcp:11000 tcp:11000
adb reverse tcp:11001 tcp:11001
adb reverse --list
adb install -r /Users/arcknin/Desktop/mindforge/src/mindforge-client/build/app/outputs/flutter-apk/app-debug.apk
```

Se houver vários aparelhos, acrescentar `-s SERIAL_DO_APARELHO` em todos os comandos de reverse/install. Não desinstalar para atualizar rascunhos pendentes; instalação `-r` preserva dados do app. O callback Google existente http://127.0.0.1:11000/callback passa pelo browser no celular e ADB; o GoTrue retorna ao app pelo scheme nativo. Isso não configura acesso direto pela LAN e depende do USB/túnel ativo.

## Aceite manual

1. Google login, cancelamento e retorno ao app; reabrir e observar restauração da sessão.
2. Criar/renomear camadas e subcamadas; abrir documentos da camada e subárvore com origem identificada.
3. Criar nota, editar Markdown e alternar prévia; aguardar confirmação local. Voltar/girar/reabrir sem perder o último commit.
4. Criar links escolhendo destino, incluindo nomes iguais; renomear/mover alvo e conferir vínculo peloID. Referência indisponível deve ser sinalizada.
5. Sync manual confirma servidor; offline conserva pendências. Auto30min e retomada tentam envio quando há execução/sessão, sem garantia com app suspenso.
6. Conflito de versão conserva rascunho e exige escolha explícita; edição durante sync permanece pendente para outro envio.
7. Excluir/restaurar antes240h, sem renovar prazo ao repetir exclusão; troca de conta não mostra dados da anterior. Logout tenta sync/flush, com escolha consciente se houver pendências.

Server-only funcionalidades dependem da migration0007 e API atualizada. Web existente mantém Auth/Core/layout; documentos Web/IndexedDB não estão incluídos nesta entrega Android. Camadas/login iniciais ainda podem exigir rede: rascunho local não equivale a app integralmente offline.
