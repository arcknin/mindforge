---
schema_version: 1
managed_by: spec-drive
feature: infra
task: develop-studio
revision: 1
status: complete
---

# Spec integrada: develop persistente e portal de Studios

## Contexto e fontes

Owner aprovou Studios distintos, entrada comum `db.home.arpa`, execução local
e TDD `yes`; main terá acesso administrativo. [Discovery](discovery.md)
registra as fontes decisivas e os limites atuais. Esta spec adiciona a
administração Studio ao contrato infra r6, mantendo Brain Core r4 e o fluxo
de migration develop-first como autoridade de seus comportamentos.

## Objetivos

- Disponibilizar portal com NSR, Mindforge main e Mindforge develop em URLs
  distintas por hostname, preservando o NSR atual.
- Concluir a preparação e operação autorizada de develop persistente com
  Auth próprio, SQL infra e Brain Core 0006, com evidência por alvo.
- Preparar/validar acesso administrativo Studio aos dois bancos Mindforge
  sem unir dados, autenticação ou clusters dos projetos.
- Provar acesso a tabelas/SQL e separar essa prova da persistência do Engine.

## Não objetivos

- Seletor multi-project/fork do Studio, stack Supabase completo, novos clusters,
  união de Auth, migração do NSR, APIs/client Brain Engine ou funcionalidades
  agênticas.
- Promover Brain Core para main, aplicar SQL infrastructure em main, alterar
  dados de main/NSR, recriar seus DB/Auth ou remover volumes.
- Commit/push/release e exposição pública na Internet.

## Desenho e caminhos de execução

### Comportamento escolhido

- Entrada `db.home.arpa`; links `nsr.db.home.arpa`, `mindforge.db.home.arpa` e
  `develop.db.home.arpa` como nomes propostos para os Studios.
- NSR continua no backend Studio/gateway existente. Cada novo Studio tem Meta
  pareado e alvo explícito; imagens fixadas na versão inspecionada da VM.
- Portal/rotas protegidos por autenticação administrativa. Meta não publica
  porta. Credenciais de DB e chaves criptográficas ficam em configuração
  privada, com par Studio/Meta correspondente e chaves distintas por ambiente.
- Main é administrativo por decisão do owner, identificado como produção.
  Preparação da interface não executa SQL de alteração em main.
- O Nginx do host já serve `db.home.arpa` na porta 80, encaminhando ao gateway
  NSR loopback 18084. Preparar um vhost substituto: portal em novo root
  `/var/www/mindforge-db-portal`, NSR em `nsr.db.home.arpa` ainda pelo gateway
  18084, main/develop em loopback 18085/18086. Não criar outro Nginx/ingress.
- Studios usam Compose separado, com Meta interno, conectado ao cluster
  Mindforge existente. Não recriar DB/Auth ou conectar Meta ao NSR.

### Estado, falhas, compatibilidade e rollback

- Falta de banco, login superuser, acesso ao volume/role correto ou divergência
  no preflight bloqueia provisionamento/migration; nunca cria ambiente local
  alternativo. `nsrdb` é contexto passivo, nunca target.
- Operador host usa destino de cliente explícito distinto do alias interno
  do Auth; testes devem provar que não muda os destinos/URLs de Auth.
- `POSTGRES_CLIENT_HOST` é obrigatório para conexões executadas no host;
  `POSTGRES_HOST=db` continua sendo o destino interno Auth. Reutilizar o login
  cluster-admin já existente, inclusive quando explicitamente igual ao main
  owner; develop deve continuar distinto. Isso não cria nem amplia privilégio.
- O primeiro `CREATE EXTENSION vector` exige o administrador existente.
  `bootstrap-develop-extensions` será restrito a develop: bookkeeping criado
  pelo owner develop, payload canônico 0001 e seu ledger em uma transação pelo
  admin. Não conceder SUPERUSER ao owner develop, criar ledger paralelo ou
  mudar SQL/hash 0001. O default psql host é 18.4. O helper
  `run-vm-develop-operator.sh` usa o cliente16 da imagem pgvector já fixada na
  VM: entrypoint é o binário Go, nunca o daemon PostgreSQL. Não cria banco
  temporário, servidor ou volume. Source/config montados read-only e `/tmp`
  privado do processo; alvo fixo develop e autorização obrigatória. Essa
  execução fica restrita à VM e depende de aprovação externa.
- Deploy é aditivo e restrito a serviços novos/develop. `up.sh` global não
  será usado para bootstrap develop porque também muda main.
- Rollback de portal/Studios restaura configuração de roteamento anterior e
  remove apenas os novos serviços, sem apagar dados/volumes/bancos. SQL já
  aplicado não é revertido silenciosamente; falha de schema exige decisão.
- A prova comportamental do Engine em develop usa principal sintético
  identificado; suas escritas duráveis precisam de autorização explícita.

### Execução ordenada

1. Confirmar roteamento HTTP/DNS atual e privilégios/admin/ferramentas sem
   mutação; registrar operação externa candidata.
2. TDD: distinguir conexão psql host de Auth interno; preparar configuração,
   portal, Compose e helpers somente para serviços/targets desta unidade.
3. Rodar checks locais, construir binários Linux e preparar pacote com hashes,
   sem credenciais e preservando o dirty work anterior.
4. Apresentar a operação concreta: entrega de arquivos, provisionamento dev,
   SQL 0001–0005/0006 em develop, Auth develop, deploy Studio/portal, rotas e
   resolução de nomes. Aprovação externa é exigida antes desses efeitos.
5. Após autorização: provisionar develop, bootstrap de extensões pelo operador,
   infra 0001–0005 como owner develop, GoTrue migrate/start só em develop,
   então Brain Core 0006 e prova Engine. Subir serviços administrativos e validar
   cada host/alvo, autenticação e preservação do NSR/main.

### Arquivos e ownership

| Caminho | Mudança | Owner |
| --- | --- | --- |
| docs/features/infra/develop-studio/ | discovery, spec, checkpoint e evidência sanitizada | principal |
| src/brain-engine/features/infra/studio/ | portal, Compose administrativo, templates, helpers e TDD | worker delimitado após congelamento |
| infra/tools/internal/config + fixtures afetadas | host psql vs Auth interno | worker delimitado após congelamento |
| Brain runtime verifier | prova comportamental explícita develop | worker delimitado após congelamento |

Ownership fechado para a preparação local: worker `brain_core` modifica
config/config tests, `.env.example`, shell validator, fixtures/assertions
afetadas e novos packages `tools/cmd/verify-brain-core`/`internal/brainproof`.
Worker `produto_client` possui somente o novo subtree `infra/studio/`.
Principal possui docs desta unidade e o helper de pacote
`scripts/prepare-develop-studio-package.py` com seu teste. `infra` continua
somente leitura da VM durante a descoberta. Depois dela, possui os novos
`operations/bootstrap_extensions.go`/teste e o dispatch específico no CLI.
Principal possui `operations.go:metadataQuery`/`metadata_bootstrap_test.go`.
Essa query passou validação READ ONLY real na VM, inclusive uma variante com
relações ausentes, sem criar objetos. Scopes não se sobrepõem; agentes folha.

## Dependências e autoridade

- Reusar Docker/Compose, PostgreSQL/pgvector/GoTrue existentes e versões
  Studio/Meta inspecionadas. Formato do proxy HTTP será fechado pelo inventário.
- O pedido atual autoriza implementação/preparação local do desenho escolhido.
- SQL, superuser provisioning, entrega/restart de serviços, roteamento LAN,
  alteração de hosts/DNS e dados sintéticos develop serão explicitados para
  aprovação antes da aplicação. Não inferir autorização de produção/NSR.

### Operação externa proposta para aprovação

Depois do pacote Linux e seus hashes serem fechados, a operação será:

1. Entregar o release em `/opt/mindforge/releases/`, verificar hash do arquivo
   e `source-sha256.txt`, sem substituir a implantação DB/Auth existente.
2. Preparar configuração privada em `/etc/mindforge/develop-studio/` com
   credenciais main/admin existentes, novo role/senha develop, Auth develop
   independente e chaves Studio/Meta por ambiente. Nenhum segredo em Git,
   stdout ou argumentos de processo. Dentro do helper, client host é `db` e
   porta5432, resolvidos na rede autorizada da VM.
3. Executar o helper **na VM**, na ordem abaixo, com operator/authorization
   references definidos no momento da aprovação. Ele usa só cliente16 e
   operador, sem iniciar PostgreSQL:

```bash
MF_STUDIO_RELEASE=/opt/mindforge/releases/RELEASE_APROVADO
MF_STUDIO_CONFIG=/etc/mindforge/develop-studio/infra.env
MF_STUDIO_RUN=develop-studio-20261008
MF_STUDIO_OPERATOR=owner-authorized-agent
MF_STUDIO_AUTHORITY=REFERENCIA_DA_APROVACAO_DO_OWNER
MF_STUDIO_HELPER="$MF_STUDIO_RELEASE/src/brain-engine/features/infra/scripts/run-vm-develop-operator.sh"
bash "$MF_STUDIO_HELPER" "$MF_STUDIO_RELEASE" "$MF_STUDIO_CONFIG" "$MF_STUDIO_OPERATOR" "$MF_STUDIO_AUTHORITY" provision "$MF_STUDIO_RUN"
bash "$MF_STUDIO_HELPER" "$MF_STUDIO_RELEASE" "$MF_STUDIO_CONFIG" "$MF_STUDIO_OPERATOR" "$MF_STUDIO_AUTHORITY" extensions "$MF_STUDIO_RUN"
bash "$MF_STUDIO_HELPER" "$MF_STUDIO_RELEASE" "$MF_STUDIO_CONFIG" "$MF_STUDIO_OPERATOR" "$MF_STUDIO_AUTHORITY" infra "$MF_STUDIO_RUN"
```

4. Com Compose `-p infra` e o arquivo do release, executar somente
   `run --rm --no-deps auth-develop-migrate` e
   `up -d --no-deps auth-develop`. Confirmar health de develop; não chamar
   `up.sh`, main Auth, serviço DB ou comandos globais de Compose.
5. Executar os modos `feature` e `verify` do mesmo helper, com as mesmas
   referências. A prova deixa um Brain e camadas sintéticas identificados pelo
   run ID em develop; o owner precisa aceitar essas gravações duráveis.
6. Preparar e validar privadamente `studio.env`; subir somente o novo projeto
   `mindforge-studio` com seus quatro serviços, sem mudar Compose/network do
   NSR nem recriar DB/Auth do Mindforge. Confirmar os backends loopback.
7. Preparar o portal em `/var/www/mindforge-db-portal`, htpasswd privado em
   `/etc/nginx/auth/mindforge-db.htpasswd` e vhost renderizado. Para manter o
   login, obter apenas a credencial Dashboard já configurada do gateway NSR
   em memória, gerar o hash sem expor senha e não alterar aquele gateway.
8. Guardar cópia recuperável do vhost atual, instalar o candidato no target
   resolvido de `/etc/nginx/sites-enabled/db-home-arpa`, rodar `nginx -t` e
   reload somente se passar. NSR usa o mesmo upstream18084 e seu Host original;
   main/develop removem Basic Authorization antes de enviar ao Studio.
9. Acrescentar os três subhosts no `/etc/hosts` do Mac usando a mesma mapping
   existente de `db.home.arpa`, sem inserir IP da máquina nos fontes. Nenhuma
   alteração global de DNS/firewall/TLS ou publicação Internet neste recorte.
10. Validar HTTP401 sem login, portal/links com login, perfis/DB/schema dos três
    destinos e dados sintéticos develop. Não consultar dados de usuário NSR ou
    executar SQL de alteração em main. Falha de interface/rota restaura o vhost
    anterior; nenhum rollback apaga banco, schema, dado, role ou volume.

Owner autorizou esta operação em 08/10/2026: “beleza agora pode aplicar então la na vm por favor”.
Não instalar outro banco temporário, compartilhar Auth, promover0006 para
main ou ampliar privilégios do role develop como alternativa implícita.

## Aceitação

- [x] Portal lista exatamente os três destinos e identifica main como produção.
- [x] Cada Studio/Meta usa seu banco/credencial/chave, sem cruzamento de alvos.
- [x] Somente gateway/proxy administrativo fica acessível; Meta não expõe porta.
- [x] Host psql alcança o alvo correto sem alterar URL interna do Auth.
- [x] Bootstrap de extensões usa admin existente só em develop e mantém SQL
  canônico 0001/ledger atômicos, com owner develop sem superuser.
- [x] Provisionamento dev é aditivo e migration 0006 usa o operador/ledger único.
- [x] Nenhum comando da entrega migra/recria DB/Auth main ou NSR implicitamente.
- [x] Runtime mostra schemas/tabelas/consulta dos bancos corretos, se autorizado.
- [x] Prova Engine em develop cobre bootstrap, isolamento, camada aninhada,
  rename estável e conflito, distinguida da prova do schema.
- [x] Credenciais/tokens/dumps/IPs da máquina não entram em Git nem outputs.

## Validação

- TDD local: testes com configuração sintética/fakes, mappings e execução
  seletiva; sem daemon Docker/PostgreSQL local ou descartável.
- Build: Go Linux e inspeção estática/render-only de Compose/configuração.
- Runtime: apenas VM designada após aprovação específica, incluindo hosts e
  HTTP autenticado, schema/migration, targets e resultados sanitizados.
- Limite: resultado local não marca runtime, deployment ou endpoint como
  verified. Abas não necessárias do Dashboard não são prometidas sem prova.

## Testes e TDD

- Decisão `yes`, confirmada pelo owner em 08/10/2026.
- Testar primeiro rejeição de mistura de alvo/segredo, ausência de autoridade,
  main como destino de provisionamento e comandos globais que alteram main.
- Verde relevante seguido dos checks de regressão já existentes.
- Expected-red adicional: preflight referenciava diretamente relações
  opcionais, resolvidas por PostgreSQL antes do CASE. Correção adia a query
  fixa após lookup; count/ledger preservados. Verde local e READ ONLY real,
  inclusive variante ausente, passaram. Isso não prova instalação.

## Automação e scripts

Helpers justificáveis: reduzir erro no deploy seletivo, no rendering seguro e
na preparação reproduzível de arquivos/hashes. Nenhum helper esconde SQL,
alvo, autorização ou operação de rollback destrutiva; comandos globais de
limpeza de volumes/bancos e leitura de .env via eval/source são proibidos.

## Riscos e decisões

- HTTP/DNS exatos e credencial superuser são dependências em confirmação.
- Studio administrativo main permite SQL manual: privilégio consciente do
  owner, sem promover esse uso a autorização para agentes alterarem main.
- Table Editor completo e abas Auth/API dependem de versão/permissões;
  validar os fluxos necessários sem instalar stack inteiro por inferência.
- Retornar ao owner para modificar dados/serviços NSR, migrar main, mudar
  segurança/exposição, usar credenciais insuficientes ou alterar este resultado.

## Aprovações

- Resultado/desenho: aprovado na conversa pelo owner em 08/10/2026.
- Preparação/implementação local: solicitada pelo owner; TDD `yes`.
- Preparação local está congelada com os scopes acima. O inventário final de
  paths/config privada alimenta a operação externa, sem ampliar os scopes.
- Operações externas: aprovadas explicitamente pelo owner em 08/10/2026; executar a lista concreta acima.

## Execução e retomada

- Checkpoint: [develop-studio-checkpoint.yaml](develop-studio-checkpoint.yaml).
- Etapa: concluída; VM validada e owner confirmou acesso aos Studios main/develop após configuração manual dos hosts.
- Próxima unidade: planejamento manual-online de Auth/API/Flutter; não repetir migrations/deploy.
- Evidência real: [runtime-evidence.md](runtime-evidence.md).

### Evidência final de preparação local

- Go `test -race -count=1 ./...` e `go vet ./...`: PASS após integração.
- Infra static all e unit all: PASS. Studio: 12 testes PASS, incluindo Compose
  renderizado com dados sintéticos. Pacote: quatro testes PASS; helper VM:
  alvo/autorização/mounts/entrypoint PASS com Docker fake, sem daemon.
- Query metadata e bootstrap identity: execução READ ONLY no banco existente
  da VM passou; nenhum operador/migration/serviço foi aplicado.
- Build Linux/amd64: dois binários ELF estáticos. Arquivo preparado:
  `/private/tmp/mindforge-develop-studio-release-20261008/mindforge-develop-studio.tar.gz`.
  SHA-256 `0eb61dcb59af352eadd2f9ed1a79e721220727f7da4f474607a74a43de232b51`.
  Conteúdo: 81 arquivos de fonte, dois binários e manifesto; hashes comparados
  às fontes atuais, arquivos privados excluídos. Nenhum commit/push.
- Se o arquivo temporário desaparecer, reconstruir com o helper local e
  comparar fontes/testes antes da aplicação; não inferir que um pacote novo
  tem os mesmos bytes aprovados.
