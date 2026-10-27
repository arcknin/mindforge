# Testar a fachada API Auth

## Google Console

No mesmo cliente OAuth Web já usado pelo Mindforge, adicionar em **URIs de redirecionamento autorizados**:

```text
http://127.0.0.1:11001/v1/auth/callback
```

Owner informou que fará essa alteração depois da entrega. Manter temporariamente a URI antiga para rollback; não criar outro client/secret. Até adicionar a URI nova, Google pode recusar o login com redirect_uri_mismatch. Isso não se resolve trocando role ou apagando conta.

## Android

O APK novo usa só a API11001; Auth é interno. Com aparelho USB conectado, depuração USB autorizada e túnel API ativo no Mac:

```sh
adb reverse tcp:11001 tcp:11001
adb install -r /Users/arcknin/Desktop/mindforge/src/mindforge-client/build/app/outputs/flutter-apk/app-debug.apk
```

Se ainda houver um encaminhamento ADB antigo11000, pode removê-lo:

```sh
adb reverse --remove tcp:11000
```

Se precisar reabrir o túnel neste Mac, usar o helper com o arquivo SSH privado já preparado fora do repositório; nenhuma senha é passada na linha de comando:

```sh
python3 /Users/arcknin/Desktop/mindforge/src/brain-engine/features/infra/auth-google-develop/scripts/open-develop-tunnels.py \
  --file /private/tmp/mindforge-android-ssh.env.private
```

O helper abre somente Mac11001→API develop18087 e deve permanecer em execução. Se esse arquivo privado temporário deixar de existir, usar o exemplo do helper para recriá-lo localmente; não colocar endpoint/credenciais no Git.

Build reproduzível neste projeto:

```sh
cd /Users/arcknin/Desktop/mindforge/src/mindforge-client
flutter build apk --debug --no-pub \
  --dart-define=MINDFORGE_ENV=develop \
  --dart-define=MINDFORGE_API_URL=http://127.0.0.1:11001
```

Haverá novo login após a atualização porque credenciais Auth usam namespace novo. A conta Google/ID remoto e namespace dos documentos são preservados; não apagar dados do aplicativo para testar a migração. Rascunho local não sincronizado continua dependendo do mesmo aparelho/storage.

## Web

Após o corte registrado em `vm-runtime-evidence.md`, acessar `http://web-mind.home.arpa`; reload carrega a versão nova. A Web usa `/api/v1/auth/*`, não `/auth/*`. Continua usando sessionStorage por aba; documentos Web/IndexedDB não entraram nesta entrega. O callback Google privado ainda precisa do túnel11001 no Mac.

## Aceite real ainda necessário

Entrar/cancelar/retornar do Google; restaurar app/sessão; renovar sessão; sair e entrar em outra conta; conferir camadas/documentos/rascunhos; testar sync manual e conflitos. Nenhuma resposta302/401 ou build isolado substitui esse aceite. Falha transitória mantém material local; nunca apagar conta/Brain para contornar autenticação. Não colar tokens, códigos PKCE, secrets ou callback bruto na evidência.
