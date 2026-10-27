# Evidência de aplicação na VM

Owner autorizou a operação preparada em 08/10/2026: “beleza agora pode aplicar então la na vm por favor”. Principal executou os efeitos; agente produto_client pesquisou rotas de validação em leitura.

## Release aplicado

- Diretório VM: `/opt/mindforge/releases/develop-studio-20261008-r2`.
- SHA-256 do arquivo: `2bd55dcb2ea932e16e6dff0e404aa8216ab95dc6c7ebde3547d464bffc0752d0`.
- Arquivo e todos os fontes do manifesto conferidos após transporte.
- Prova sanitizada preservada privadamente em `/etc/mindforge/develop-studio/runtime-proof.json`.
- Configuração privada: `/etc/mindforge/develop-studio/`, diretório700/arquivos600; sem credenciais em Git ou stdout.
- Primeiro provision recusou `--source-root` antes de mutação. Helper corrigido localmente com regressão TDD red/green; pacote r2 substitui o pacote inicialmente preparado. Binários/SQL canônicos não mudaram.

## Banco e Engine

- `mindforge_develop` criado com role próprio; owner sem superuser/createdb/createrole.
- Extensões canônicas0001 pelo admin existente; demais migrations0002–0005 pelo operador develop; Auth migrate/start seletivos; Brain Core0006 aplicado pelo ledger único.
- Ledger develop0001–0006; Auth70 migrations. Hash0006 `1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9`.
- Engine + PostgresStore reais: bootstrap idempotente, own observe, isolamento, camada aninhada, identidade estável no rename e conflito stale: todos PASS.
- Run `develop-studio-20261008`: um Brain sintético, uma raiz e três camadas duráveis. Não houve limpeza.

## Studio e roteamento

- Projeto `mindforge-studio`: Studio+Meta por ambiente, quatro serviços healthy. Meta sem porta publicada; Studio18085/18086 e Auth develop10000 loopback.
- Portal em `/var/www/mindforge-db-portal`; Nginx existente com quatro vhosts. `nginx -t` e reload PASS. Basic Auth usa o login Dashboard NSR existente; gateway NSR não alterado.
- Backup recuperável do vhost anterior em `/etc/mindforge/develop-studio/nginx-vhost.before.conf`; target resolvido registrado privadamente ao lado.
- `db.home.arpa`, `nsr.db.home.arpa`, `mindforge.db.home.arpa`, `develop.db.home.arpa`: HTTP401 anônimo e HTTP200 com login. Portal contém três links; profile e projects dos três Studios retornaram200 e nomes corretos.
- Consultas SELECT através de `/api/platform/pg-meta/default/query` confirmam: NSR→`postgres`/`supabase_admin`; main→`mindforge`/`mindforge`; develop→`mindforge_develop`/`mindforge_develop`.
- Listagem de tabelas através do Studio: develop `brain_core.brains`/`brain_core.nodes`; main sem `brain_core`. SELECT restrito ao run sintético devolveu uma raiz e três camadas no develop.

## Preservação

- DB/Auth main e DB/Studio/gateway NSR preservaram IDs, StartedAt e volumes e estão healthy.
- Volume protegido `infra_mindforge-postgres-data` preservado; nenhum DB/Auth existente recriado.
- Main permanece ledger0001–0004 com os mesmos hashes e Auth70 migrations, sem schema Brain Core. Não houve promoção0006/main, leitura de linhas pessoais NSR ou alteração de dados main/NSR.
- Nenhum novo endpoint do produto; OpenAPI continua `endpoints: []`. Não houve commit/push.

## Limites e confirmação do owner

- Validação real por HTTP/API/SQL e catálogo. Não foi feita QA visual/interação no navegador, nem prometidas todas as abas do dashboard Supabase completo.
- HTTP401 dos quatro vhosts confirmado também a partir do Mac com destino explícito; rede LAN acessível.
- A janela administrativa iniciada pelo agente foi cancelada pelo usuário. Posteriormente foram fornecidos os comandos de hosts e o owner confirmou acesso aos Studios main/develop. Pendência de acesso encerrada por esse relato; não houve nova reinspeção independente de `/etc/hosts` ou QA visual.
