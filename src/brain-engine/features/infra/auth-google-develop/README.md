# Manual online: Auth Google e pacote develop

Este diretório prepara a configuração local do primeiro fluxo manual macOS. Os arquivos de exemplo são deliberadamente incompletos e os validadores os rejeitam até que o owner forneça valores privados. Nenhuma credencial Google foi criada nem incluída aqui.

## Limites da unidade

- `compose.auth-google-develop.yaml` é um overlay do Compose de infraestrutura existente e declara somente `auth-develop`. O mapa de ambiente do Compose é mesclado com o serviço base, preservando banco, JWT, claims e issuer do Auth develop. O overlay não declara banco, migration, Auth main, NSR, volumes nem rede.
- O Google volta à fachada do Brain Engine em `http://127.0.0.1:11001/v1/auth/callback`. O site público do Auth é `http://web-mind.home.arpa`; o allowlist contém exatamente `mindforge-dev://auth/callback` e `http://web-mind.home.arpa/login/callback`. O túnel encaminha somente a porta local `11001` à API develop publicada no loopback remoto `18087`.
- Email, telefone e login anônimo ficam desligados; o cadastro geral não é desativado, para permitir a primeira conta via Google. As chaves foram conferidas no [`example.env` oficial do GoTrue tag `v2.196.0`](https://raw.githubusercontent.com/supabase/auth/v2.196.0/example.env); o código dessa versão mostra o default de telefone como `false` em [`configuration.go`](https://raw.githubusercontent.com/supabase/auth/v2.196.0/internal/conf/configuration.go).
- `compose.brain-engine-develop.yaml` contém somente o serviço Brain Engine. Publica `127.0.0.1:18087` para `8080`, usa a rede externa `infra_default` e recebe os nomes de ambiente públicos do runtime Auth/API. Não cria DB, Auth, volume ou rota de health.
- A imagem usa o binário Linux estático do pacote, `scratch` e UID/GID `65532`. A compilação impede CGO, acesso à rede de módulos e download de toolchain.
- As cinco operações Auth estão `approved` no registry e no OpenAPI. A implementação da fachada fica no worker de API; esta unidade prepara configuração local e não altera contratos nem prova runtime.

## Preparar Google OAuth quando houver credencial

O setup Google Cloud é uma operação futura separada. A sequência aprovada na spec é:

1. O owner escolhe a conta responsável e um projeto de desenvolvimento. Guardar projeto e responsável no inventário privado; não reutilizar credencial de produção.
2. Em Google Auth Platform, configurar Branding como `Mindforge develop`, com contato e email de suporte do responsável.
3. Configurar público External para login Google normal. Com somente nome/email/perfil, Google documenta que a lista manual de testadores não é necessária, inclusive em Testing. Separar duas contas reais para a prova de isolamento não exige cadastrar emails em allowlist. Esse modo não substitui autorização do Brain Engine.
4. Pedir somente `openid`, `email` e `profile`; não pedir APIs Google de email, arquivos ou calendário.
5. Confirmar a configuração de callback do Brain Engine e os redirects finais das aplicações. Os valores preparados são `http://127.0.0.1:11001/v1/auth/callback`, `mindforge-dev://auth/callback` e `http://web-mind.home.arpa/login/callback`.
6. Criar um OAuth client `Web application` para develop. Não usar API key, service account nem credencial de produção.
7. Guardar Client ID e Client Secret em local privado. Não colar segredos em chat, comandos, logs, Flutter, pacote ou Git.
8. Criar `google-auth.env.private` copiando `google-auth.env.example`, restringir permissões e preencher os dois valores sem adicionar outras chaves. A alteração correspondente no Google Console continua pendente da ação do owner; este pacote apenas prepara e valida a configuração:

   ```sh
   cp google-auth.env.example google-auth.env.private
   chmod 600 google-auth.env.private
   $EDITOR google-auth.env.private
   python3 scripts/validate-config.py google --file google-auth.env.private
   ```

   O parser lê linhas `KEY=VALUE` sem `source` ou `eval`, não exibe valores, rejeita placeholders e exige arquivo regular sem acesso para grupo/outros. A configuração privada deve ficar fora de commits. Não acrescente as chaves Google ao `infra.env` do operador.

Essa validação apenas confirma o formato local. Aplicar o overlay e reiniciar Auth develop exigem a etapa externa B04 com referência de autorização e validação do arquivo privado de infraestrutura. Não use o `up.sh` global para essa alteração.

## Construir o pacote da API develop

Depois da integração do código Go, escolha um destino novo fora do checkout. O builder compila sem rede para Linux amd64 por padrão; `arm64` também é aceito:

```sh
python3 scripts/prepare-manual-online-package.py \
  --output /private/tmp/mindforge-manual-online-package \
  --goarch amd64
```

O destino precisa estar ausente ou vazio. O pacote contém somente o binário, Dockerfile, regras do contexto Docker, Compose API, runbook, exemplo do ambiente runtime, validador e manifesto SHA-256. O manifesto lista os arquivos Go autorizados e os módulos do build. Arquivos privados, symlinks e nomes de segredo/temporário nos diretórios de fonte abortam a preparação. Nenhum `.env` privado é copiado e `.dockerignore` impede que ele entre no contexto do build.

Dependências Go precisam estar disponíveis localmente; o builder não instala ferramentas nem busca módulos. Sem cache suficiente, o build falha sem mostrar saída que possa conter valores do ambiente. A imagem roda `/brain-engine` como UID/GID `65532`, na rede existente `infra_default`, sem filesystem persistente.

O ambiente privado da API é independente do OAuth. No diretório do pacote, a configuração futura usa:

```sh
cp .env.runtime.example .env.runtime
chmod 600 .env.runtime
$EDITOR .env.runtime
python3 scripts/validate-config.py runtime --file .env.runtime
```

`MINDFORGE_DATABASE_URL` deve usar o usuário provisionado `mindforge_develop`, sua senha privada, host Compose `db`, porta `5432`, database `mindforge_develop` e `sslmode=disable`. O validador exige ainda `MINDFORGE_ENVIRONMENT=develop`, `MINDFORGE_AUTH_USER_URL=http://auth-develop:9999/user`, `MINDFORGE_AUTH_CALLBACK_URL=http://127.0.0.1:11001/v1/auth/callback`, `MINDFORGE_AUTH_REDIRECT_URLS=mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback` e `MINDFORGE_LISTEN_ADDR=:8080`; ele nunca imprime a URL ou a senha. Não passe valores sensíveis como argumentos.

O pacote é apenas preparação. Construção da imagem e execução do serviço na VM ficam para B04 e dependem de autorização de deploy. A aplicação usará `--env-file .env.runtime`; não registre a saída de `docker compose config`, pois ela pode conter o DSN privado.

## Túnel local

O arquivo SSH guarda somente host, porta e usuário. A autenticação usa chave/agent ou prompt do OpenSSH, sem senha armazenada:

```sh
cp ssh-tunnel.env.example ssh-tunnel.env.private
chmod 600 ssh-tunnel.env.private
$EDITOR ssh-tunnel.env.private
python3 scripts/validate-config.py ssh --file ssh-tunnel.env.private
python3 scripts/open-develop-tunnels.py --file ssh-tunnel.env.private
```

O túnel só escuta em `127.0.0.1`: porta local `11001` → `127.0.0.1:18087` na VM. OpenSSH exige host key já confiável e `ExitOnForwardFailure`; se a porta local estiver ocupada, o comando falha e não encerra o processo que a ocupa. Não há encaminhamento para Auth, comando remoto, encaminhamento reverso ou execução SSH dentro do app.

## Rollback coordenado

O rollback futuro deve restaurar em conjunto a configuração/release da API, o Web, os callbacks e o túnel anterior, mantendo as credenciais antigas disponíveis durante a transição. Remova somente os artefatos aditivos desta unidade, como o projeto dedicado, o site Nginx novo e os arquivos privados de configuração correspondentes, depois valide/recarregue o Nginx conforme o runbook do owner.

Não apagar contas, dados, rascunhos, chaves Google ou volumes. Não executar `down -v`, `--remove-orphans`, downgrade de migration ou alterações em main, NSR, Studio, PostgreSQL e volumes protegidos. O rollback volta temporariamente à fronteira antiga e precisa ser registrado pelo owner.

## Verificação local

```sh
./tests/run.sh
```

Os testes usam valores sintéticos, parser YAML local, Compose `config` quando disponível, Go falso para conferir os flags reais do builder e SSH falso para conferir os argumentos do túnel. Não iniciam containers, não abrem rede, não acessam Google Cloud, VM ou banco.
