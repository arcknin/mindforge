# Mindforge Web develop

Este pacote serve o build estático do client Flutter Web e faz proxy somente para
o Brain Engine develop através da rede Docker já existente `infra_default`.
O projeto Compose é `mindforge-web-develop`; o único bind de host é
`127.0.0.1:18088 → 80`. O Nginx do host publica `web-mind.home.arpa` somente
para loopback, gateway NAT existente `10.0.2.2` e `192.168.0.0/24`, encaminhando todas as requisições a esse
bind local.

O pacote não contém Auth, API, banco, migrations, volume ou credenciais. O
upstream está fixo no container existente
`mindforge-brain-engine-develop-brain-engine-develop-1:8080`. O proxy expõe
somente as cinco rotas Auth aprovadas pela fachada da API e as quatro rotas Brain
aprovadas. Rotas `/auth/*` legadas, administrativas, senha, signup e provedores
que não sejam Google retornam 404. Não existe upstream direto para GoTrue.

O container usa filesystem raiz somente-leitura, tmpfs para estado temporário
do Nginx, nenhuma capability por padrão e somente `NET_BIND_SERVICE`, `SETUID`,
`SETGID` e `CHOWN` para escutar em 80, reduzir a identidade dos workers e
preparar seus diretórios temporários.

## Build do pacote

O contexto de build precisa ter somente este conjunto mínimo:

```text
package/
├── Dockerfile
├── compose.yaml
├── .dockerignore
├── nginx/default.conf
└── build/web/
    ├── index.html
    └── ...arquivos estáticos Flutter Web...
```

Escolha um diretório novo fora do checkout para `package/`. Construa o Flutter
Web com as definições aprovadas pela feature web e grave o resultado em
`package/build/web`. Copie para esse diretório o `Dockerfile`, `compose.yaml`,
`.dockerignore` e `nginx/default.conf` deste pacote. Antes de empacotar, confirme
que `build/web/index.html` existe e que o build usa a origem única
`http://web-mind.home.arpa/api` para Auth, Brain Engine e demais operações da
API. O callback final Web permanece
`http://web-mind.home.arpa/login/callback`. Não copie arquivos `.env`,
configuração privada ou artefatos de outro componente.

O Dockerfile usa a [imagem oficial nginx:1.30.5-alpine3.24](https://hub.docker.com/layers/library/nginx/1.30.5-alpine3.24/images/sha256-8f84ed99befc3891b8f329c5c202785278a2cfb7c25107d57fb2a134a3117433),
fixada no digest Linux/amd64
`sha256:8f84ed99befc3891b8f329c5c202785278a2cfb7c25107d57fb2a134a3117433`.
O `.dockerignore` permite somente o Dockerfile, a configuração Nginx e o
conteúdo `build/web/` no contexto enviado ao builder.

## Dependências operacionais

Antes de iniciar este serviço na VM designada, confirme que a porta loopback
`18088` está livre, `infra_default` existe, os dois nomes DNS Docker fixos
resolvem para os serviços develop esperados e o Nginx do host continua intacto.
O root deve instalar o arquivo `nginx/web-mind.home.arpa.conf` como um novo
site habilitado, sem substituir configurações ou logs compartilhados. O arquivo
reusa `/var/log/nginx/error.log` com nível `crit` e desliga o access log para
não registrar query strings de OAuth, códigos, tokens ou o caminho do callback.

O Google redireciona primeiro à fachada do Brain Engine,
`http://127.0.0.1:11001/v1/auth/callback`, pelo único túnel SSH local. Os
redirects finais exatos são `mindforge-dev://auth/callback` e
`http://web-mind.home.arpa/login/callback`; a alteração correspondente no
Google Console continua pendente da ação do owner. Esta unidade não altera
Google Console, Auth runtime ou qualquer rota de main. HTTPS externo e DNS
público estão fora deste pacote.

## Render e deploy

No diretório do pacote, valide a configuração Compose sem iniciar containers:

```sh
docker compose -f compose.yaml config --quiet
```

Na VM, depois de um preflight específico e da cópia do pacote, construa e inicie
somente o projeto novo:

```sh
docker compose -p mindforge-web-develop -f compose.yaml up -d --build
docker compose -p mindforge-web-develop -f compose.yaml ps
curl --fail --silent --show-error http://127.0.0.1:18088/login/callback -o /dev/null
```

Instale o vhost aditivo e valide/recarregue o Nginx do host somente depois de
confirmar o caminho de sites habilitados e a configuração ativa:

```sh
nginx -t
systemctl reload nginx
```

Verifique pelo hostname que `/` e `/login/callback` servem `index.html`, que
`/api/v1/auth/{authorize,callback,token,user,logout}` alcançam somente o Brain
Engine, e que `/api/v1/me/brain` sem bearer continua retornando 401. `/auth/*`,
`/api/v1/auth/settings`, `/api/v1/auth/health`, `provider` diferente de Google,
`grant_type=password` e uma rota Brain fora das quatro operações devem retornar
404. Esses checks não comprovam login Google nem uma operação Brain autenticada.

Depois que Auth develop for recriado, reinicie este container para atualizar os
endereços DNS dos upstreams. O nome DNS Docker é estável, mas o Nginx resolve o
endereço na inicialização.

## Rollback

Remova somente o novo site habilitado, valide/recarregue o Nginx do host e
encerre o projeto dedicado:

```sh
nginx -t
systemctl reload nginx
docker compose -p mindforge-web-develop -f compose.yaml down
```

Este Compose não declara volumes; `down` não remove dados de Brain, Auth ou
PostgreSQL. Remova somente o site Nginx e o projeto Web adicionados nesta
unidade, coordenando antes a release da API/Auth e os callbacks. Não use
`down -v` nem `--remove-orphans` em um projeto compartilhado; preserve main,
NSR, Studio, `db`, Auth, PostgreSQL, rascunhos, credenciais e volumes.

## Verificação local

```sh
bash tests/run.sh
```

Os testes conferem o escopo Compose, bind loopback, rede externa, allowlist de
rotas/métodos, headers, cache, ausência de logs de acesso e o contrato do
contexto `build/web`. `docker compose config` é apenas renderização: nenhum
daemon, container, serviço, Auth, API ou banco é iniciado por eles.
