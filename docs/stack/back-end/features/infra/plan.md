---
schema_version: 1
flow: orchestration-flow
feature: infra
revision: 3
contract_revision: 2
status: ready
tdd: "yes"
implementation_authorized_at: "2026-09-08T14:21:28-03:00"
repositories:
  - root checkout
---

# Code Plan: Initial reduced Supabase infrastructure

## Execution summary

Corrigir e comprovar o pacote existente de PostgreSQL, Supabase Auth/GoTrue e `pgvector` sem ampliar o contrato. Um unico executor deve executar os cinco lotes sequenciais porque `compose.yaml`, os helpers Bash, o parser de configuracao e o harness de testes sao compartilhados. Cada lote mantem seu proprio ciclo expected-red -> implementacao real -> relevant-green; nao ha fase separada de testes nem gate por lote.

Os testes static/unit usam Bash e utilitarios basicos, com Docker falso para os limites operacionais e sem exigir Bats, jq ou yq. Runtime so comeca depois de todos os greens static/unit: primeiro em projetos Docker locais descartaveis e exclusivos; somente depois, na VM de teste autorizada, tambem em projetos, portas e volumes exclusivos. O stack ja existente nunca e alvo de restore ou replay. Permanecem fora de autoridade: producao, Internet, firewall, DNS, TLS, Coolify, dados reais e exclusao de volumes.

## Tasks

```yaml
- id: B01
  goal: "Fazer configuracao insegura falhar antes do startup e tornar a topologia Compose isolavel e coerente."
  repository: root checkout
  depends_on: []
  allowed_scope:
    - src/brain-engine/features/infra/.env.example
    - src/brain-engine/features/infra/compose.yaml
    - src/brain-engine/features/infra/scripts/validate-config.sh
    - src/brain-engine/features/infra/scripts/lib/compose.sh
    - src/brain-engine/features/infra/tests/run.sh
    - src/brain-engine/features/infra/tests/lib/assert.sh
    - src/brain-engine/features/infra/tests/lib/fixture.sh
    - src/brain-engine/features/infra/tests/lib/fake-docker.sh
    - src/brain-engine/features/infra/tests/test_config_compose.sh
  changes:
    - "Expected-red: criar casos que provem falha nao-zero e sem vazamento de valores para variaveis obrigatorias ausentes/vazias, replace-with-*, JWT curto, GOTRUE_API_EXTERNAL_URL ausente/invalida, redirect com wildcard e bind LAN sem opt-in explicito."
    - "Expected-red: provar que uma porta host Auth alternativa nao pode alterar a porta interna/healthcheck, que container_name fixo impede projetos simultaneos e que a tag major-floating do PostgreSQL viola o pinning do contrato."
    - "Expected-red: provar que POSTGRES_DB, POSTGRES_USER ou senha divergentes do GOTRUE_DB_DATABASE_URL sao rejeitados antes de Docker."
    - "Implementacao: adicionar parser dotenv restrito e validacao fail-closed reutilizavel, sem source/eval e sem imprimir segredos; tornar GOTRUE_API_EXTERNAL_URL e demais entradas requeridas explicitas no exemplo."
    - "Implementacao: centralizar chamadas Compose em helper que sempre valida configuracao, exige project name explicito e preserva loopback por padrao."
    - "Implementacao: remover container_name, fixar portas internas db/auth em 5432/9999, deixar apenas portas host configuraveis e substituir pgvector/pgvector:pg16 por tag completa ou digest compativel comprovado."
    - "Relevant-green: executar exatamente os casos vermelhos e a renderizacao Compose sintetica ate todos passarem sem daemon."
  validation:
    - "bash src/brain-engine/features/infra/tests/run.sh static config"
    - "bash src/brain-engine/features/infra/tests/run.sh static compose"
    - "bash -n src/brain-engine/features/infra/scripts/validate-config.sh src/brain-engine/features/infra/scripts/lib/compose.sh src/brain-engine/features/infra/tests/run.sh src/brain-engine/features/infra/tests/test_config_compose.sh"
    - "Use an environment file in a temporary directory outside the checkout; docker compose --env-file <temporary-env-file> -f src/brain-engine/features/infra/compose.yaml config --quiet."
  done_when:
    - "Nenhuma operacao suportada chega a Docker com configuracao ausente, placeholder, incoerente ou insegura."
    - "Binds default continuam em 127.0.0.1; LAN exige opt-in explicito."
    - "Projetos Compose simultaneos nao colidem por nome de container, rede, porta interna ou volume."
    - "As duas imagens usam versoes explicitas e o PostgreSQL/pgvector nao usa tag major-floating."
  parallelizable: false

- id: B02
  goal: "Entregar bootstrap Auth funcional e migrations de infraestrutura deterministicas, auditaveis e reaplicaveis em volumes existentes."
  repository: root checkout
  depends_on:
    - B01
  allowed_scope:
    - src/brain-engine/features/infra/compose.yaml
    - src/brain-engine/features/infra/postgres/migrations/2026xx01_infra_init.sql
    - src/brain-engine/features/infra/postgres/migrations/0001_infra_init.sql
    - src/brain-engine/features/infra/postgres/migrations/0002_auth_bootstrap.sql
    - src/brain-engine/features/infra/scripts/lib/compose.sh
    - src/brain-engine/features/infra/scripts/up.sh
    - src/brain-engine/features/infra/scripts/migrate.sh
    - src/brain-engine/features/infra/tests/run.sh
    - src/brain-engine/features/infra/tests/lib/fake-docker.sh
    - src/brain-engine/features/infra/tests/test_migrations.sh
  changes:
    - "Expected-red: provar que o mount atual em docker-entrypoint-initdb.d nao aplica migration futura a volume existente, nao possui ledger e ordena 2026xx01 depois de um nome numerico real."
    - "Expected-red: provar que hash alterado, ID duplicado, nome nao canonico ou migration pendente anterior ao maior ID aplicado nao sao detectados hoje."
    - "Expected-red: provar que /health HTTP isolado nao satisfaz Auth quando schema, tabelas e marcador de migrations GoTrue estao ausentes."
    - "Implementacao: renomear a migration placeholder para ID numerico zero-padded preservando apenas uuid-ossp, vector e citext; retirar o mount de initdb como executor de migrations."
    - "Implementacao: criar migrate.sh com ordem lexical canonica, ledger exclusivamente de infraestrutura contendo id/hash SHA-256/applied_at, aplicacao transacional com ON_ERROR_STOP, skip somente de id+hash identicos e rejeicao fail-closed de drift, duplicata ou backfill fora de ordem."
    - "Implementacao: criar apenas o schema/role/configuracao minima requerida pelo Auth e executar gotrue migrate como etapa one-shot da imagem fixada antes de gotrue serve; falha de migration impede Auth de iniciar."
    - "Implementacao: criar up.sh na ordem validate -> db healthy -> migrations infra -> migrations GoTrue -> Auth healthy, usando o helper Compose e sem criar tabelas de dominio Mindforge."
    - "Relevant-green: repetir os casos de ledger/ordem/drift e validar, com Docker falso, a sequencia e a propagacao de qualquer falha."
  validation:
    - "bash src/brain-engine/features/infra/tests/run.sh unit migrations"
    - "bash -n src/brain-engine/features/infra/scripts/up.sh src/brain-engine/features/infra/scripts/migrate.sh src/brain-engine/features/infra/tests/test_migrations.sh"
    - "Validar no teste que o comando suportado gotrue migrate da imagem exata precede gotrue serve e que seu erro retorna nao-zero."
    - "Validar estaticamente que nenhuma migration cria tabela/schema de produto e que contracts/openapi/registry.yaml permanece sem endpoint Mindforge."
  done_when:
    - "Uma migration nova e aplicada em volume existente sem depender de initdb."
    - "Reexecucao identica e idempotente; alteracao de migration aplicada e ordem invalida falham antes de nova DDL."
    - "Auth so fica pronto depois de seu bootstrap/migrations, com schema e marcadores verificaveis."
    - "O conjunto de migrations continua limitado a bookkeeping, suporte Auth e extensoes aprovadas."
  parallelizable: false

- id: B03
  goal: "Tornar backup, restore e inspect fail-closed, privados, portaveis e incapazes de atingir o stack existente."
  repository: root checkout
  depends_on:
    - B02
  allowed_scope:
    - src/brain-engine/features/infra/scripts/lib/compose.sh
    - src/brain-engine/features/infra/scripts/validate-config.sh
    - src/brain-engine/features/infra/scripts/backup.sh
    - src/brain-engine/features/infra/scripts/restore.sh
    - src/brain-engine/features/infra/scripts/inspect.sh
    - src/brain-engine/features/infra/tests/run.sh
    - src/brain-engine/features/infra/tests/lib/fake-docker.sh
    - src/brain-engine/features/infra/tests/test_operations.sh
  changes:
    - "Expected-red: reproduzir backup sem senha dentro do container, dump parcial remanescente, destino default dentro do checkout, permissao 0644 e colisao de timestamp."
    - "Expected-red: reproduzir restore aceitando env/banco primario, target-id divergente, Auth ativo ou projeto/volume nao exclusivo; provar que dump invalido pode deixar estado parcial e que credenciais nao chegam ao pg_restore."
    - "Expected-red: provar incompatibilidade Bash 3.2 da confirmacao e inspect retornando zero quando db, Auth ou extensao falham."
    - "Expected-red: provar que .env contendo comando lateral e executado atualmente pelos tres scripts."
    - "Implementacao: reutilizar o parser dotenv restrito; backup exige diretorio externo explicito, umask 077, nome sem sobrescrita, arquivo temporario com cleanup, credencial transmitida somente ao processo interno, pg_restore --list e rename atomico."
    - "Implementacao: restore exige target-id e confirmacao exata vinculados a env descartavel gerado pelo runner, project name/volume/banco exclusivos e Auth parado; preflight valida dump antes de mutacao e pg_restore usa credenciais explicitas e single transaction."
    - "Implementacao: inspect consulta os endpoints/containers configurados, valida separadamente db, Auth, migrations Auth e extensoes/versoes esperadas, acumula falhas e termina nao-zero diante de qualquer vermelho."
    - "Implementacao: manter scripts compativeis com Bash 3.2 e jamais chamar down -v, docker volume rm ou prune."
    - "Relevant-green: repetir todos os casos com Docker falso, inclusive falhas parciais e env malicioso, e comprovar que nenhum dump parcial e apresentado como valido."
  validation:
    - "bash src/brain-engine/features/infra/tests/run.sh unit operations"
    - "bash -n src/brain-engine/features/infra/scripts/backup.sh src/brain-engine/features/infra/scripts/restore.sh src/brain-engine/features/infra/scripts/inspect.sh src/brain-engine/features/infra/tests/test_operations.sh"
    - "Executar o teste de falha do produtor e verificar ausencia de arquivo final/parcial."
    - "Executar o teste de restore recusado e verificar que nenhuma chamada mutavel ao Docker ocorreu."
  done_when:
    - "Backup valido e privado so aparece por promocao atomica em destino externo explicito."
    - "Restore so aceita destino descartavel exclusivo e deixa o destino inalterado diante de input/preflight/restore falho."
    - "Credenciais chegam aos clientes PostgreSQL sem serem impressas ou executadas como shell."
    - "Inspect nao produz falso verde para db, Auth, migrations ou extensoes."
  parallelizable: false

- id: B04
  goal: "Reconciliar runbook, protecao local e evidencia estatica sem depender de tracking ou commit local."
  repository: root checkout
  depends_on:
    - B03
  allowed_scope:
    - src/brain-engine/features/infra/README.md
    - .gitignore
    - src/brain-engine/features/infra/.env.example
    - src/brain-engine/features/infra/tests/run.sh
    - src/brain-engine/features/infra/tests/test_runbook_hygiene.sh
    - src/brain-engine/features/infra/tests/versioned-surface.txt
    - docs/stack/readme.md
    - docs/stack/back-end/readme.md
  changes:
    - "Expected-red: fixar casos para GOTRUE_API_EXTERNAL_URL ausente, restore documentado como pasta_infra, comandos que expandem variaveis nao carregadas, prerequisito curl omitido, volume hardcoded divergente e ausencia de ignore/manifesto."
    - "Expected-red: provar que o runbook ainda descreve migrations somente por initdb, backup no checkout, remocao direta de volume e estado documental anterior a implementacao/contrato rev. 2."
    - "Implementacao: documentar validate/up/down/status/logs/inspect, conexoes PostgreSQL/Auth, porta host versus interna, migrations infra+GoTrue, backup externo, restore exclusivo, rotacao de segredos e recuperacao sem comando destrutivo automatico."
    - "Implementacao: corrigir a assinatura real dos scripts, prerequisitos e exemplos para funcionarem em shell limpo; documentar o nome logico e a resolucao Compose do volume sem presumir prefixo fixo."
    - "Implementacao: adicionar .gitignore raiz para .env, dumps, backups e artefatos temporarios; adicionar manifesto ordenado da superficie versionavel e teste que rejeita segredos, dumps, tabelas de dominio, endpoints Mindforge ou servicos Supabase extras."
    - "Implementacao: gerar hashes sanitizados da superficie listada somente como evidencia de execucao em diretorio temporario fora do checkout; nao apresentar manifesto como prova de tracking, commit, branch ou publicacao."
    - "Implementacao: atualizar docs/stack/readme.md e docs/stack/back-end/readme.md para reconhecer o pacote atual e contrato rev. 2 aprovado sem declarar runtime/produção que nao foram provados."
    - "Relevant-green: repetir os checks de runbook/higiene e validar que nenhuma referencia antiga ou placeholder estrutural permanece."
  validation:
    - "bash src/brain-engine/features/infra/tests/run.sh static runbook-hygiene"
    - "Verificar que src/brain-engine/features/infra/tests/versioned-surface.txt esta ordenado, sem caminho secreto/backup, e que todos os caminhos listados existem."
    - "Gerar em diretorio temporario fora do checkout um manifesto SHA-256 sanitizado dos caminhos listados e comprovar que nenhum arquivo fora do allowlist entra na evidencia."
    - "Executar varredura estatica por schema/tabela de dominio, endpoint Mindforge e servico Supabase fora de db/auth."
  done_when:
    - "Runbook e READMEs descrevem comandos, argumentos, estado e limites reais da revisao 2."
    - "Configuracao real, backups, dumps e temporarios estao cobertos pelo ignore local e ausentes do manifesto versionavel."
    - "A higiene e comprovavel por allowlist/hash de execucao, com a limitacao de tracking, commit e publicacao declarada."
    - "Nenhuma instrucao de recuperacao apaga volume automaticamente ou omite o gate do owner."
  parallelizable: false

- id: B05
  goal: "Executar a matriz de aceitacao em ambientes descartaveis exclusivos e consolidar evidencia local antes da VM de teste."
  repository: root checkout
  depends_on:
    - B04
  allowed_scope:
    - src/brain-engine/features/infra/.env.example
    - .gitignore
    - src/brain-engine/features/infra/README.md
    - src/brain-engine/features/infra/compose.yaml
    - src/brain-engine/features/infra/postgres/migrations
    - src/brain-engine/features/infra/scripts
    - src/brain-engine/features/infra/tests
  changes:
    - "Expected-red: criar teste do runner ausente e de suas barreiras; ele deve recusar project name reutilizado, bind nao-loopback, volume existente, env primario, target-id divergente ou modo VM antes dos greens locais."
    - "Implementacao: criar acceptance-disposable.sh e integration_disposable.sh com IDs imprevisiveis e tres projetos exclusivos derivados de mf-accept-RUN_ID para src, restore e replay, portas host dinamicas/loopback, redes e volumes proprios e manifesto sanitizado em /tmp."
    - "Implementacao: no projeto src, executar validate/start/readiness, confirmar schema/migrations Auth, signup e sign-in/password grant com identidade sintetica, token sem expor segredo, operacao vector transacional com rollback e ausencia posterior, e marcador sintetico preservado por stop/start."
    - "Implementacao: gerar backup externo do src, restaurar no projeto/volume/db restore limpo e comprovar o marcador; no projeto replay, aplicar migrations limpas, repetir idempotentemente e verificar ledger, Auth e ausencia de dominio."
    - "Implementacao: parar containers/redes ao final sem -v e registrar os volumes exclusivos remanescentes; nunca selecionar, parar, restaurar, migrar ou limpar o stack observado/existente."
    - "Relevant-green: depois dos reds do runner, executar todas as suites static/unit e entao a integracao Docker local descartavel; qualquer falha volta ao arquivo/lote proprietario como correcao in-contract."
    - "Runtime VM: somente com todos os greens locais registrados, executar a mesma matriz em area/projetos descartaveis exclusivos da VM autorizada; manter loopback, dados sinteticos e nenhum firewall/DNS/TLS/Coolify."
  validation:
    - "bash src/brain-engine/features/infra/tests/run.sh unit acceptance-runner"
    - "bash src/brain-engine/features/infra/tests/run.sh static all"
    - "bash src/brain-engine/features/infra/tests/run.sh unit all"
    - "bash src/brain-engine/features/infra/tests/run.sh integration --local-disposable"
    - "Depois dos greens locais: bash src/brain-engine/features/infra/tests/run.sh integration --vm-disposable na area descartavel autorizada, nunca no stack existente."
  done_when:
    - "PostgreSQL e Auth ficam healthy e Auth completa signup/sign-in/token no ambiente descartavel."
    - "Vector funciona sem deixar schema/dado de produto; marcador sintetico sobrevive ao restart."
    - "Backup e restore separado recuperam o marcador sem tocar a origem; replay limpo e segunda execucao idempotente passam."
    - "Catalogo final contem somente suporte infra/Auth/extensoes e nenhum endpoint Mindforge foi criado."
    - "Evidencia separa static/unit, Docker local, VM descartavel e producao, sem inferir uma camada da anterior."
  parallelizable: false
```

## Assumptions

- O contrato revision 2 aprovado autoriza escolher o runner de migrations e o layout de suporte sem nova decisao de produto.
- O comando one-shot oficial `gotrue migrate`, executado pela mesma imagem fixada do Auth, e a fonte das migrations GoTrue; o repositorio versiona seu pinning, configuracao, precondicoes e sequenciamento, nao uma copia divergente do catalogo upstream.
- O executor pode criar schema/role estritamente necessarios ao Auth e um ledger de migrations infra; nenhuma tabela de dominio Mindforge e autorizada.
- A raiz Git existe neste checkpoint na branch `main`, ainda sem commits anteriores a este checkpoint; manifesto/hasheamento prova apenas a superficie observada, nao tracking, commit, upstream ou publicacao.
- Um unico executor recebe B01-B05 em ordem porque helpers e testes sao sobrepostos; nao ha workstream de implementacao paralelo seguro.

## TDD execution

- Em cada lote, escrever primeiro o menor caso que reproduz o defeito real e registrar o expected-red pelo motivo esperado.
- Implementar somente a mudanca causal do lote e repetir o mesmo caso ate relevant-green; falha de fixture ou dependencia nao conta como red valido.
- Rodar as regressões acumuladas ao fechar cada lote. Falhas in-contract retornam ao mesmo executor; apenas mudanca material de contrato, autoridade, seguranca, persistencia ou validation boundary retorna ao owner.
- Nenhuma chamada Docker mutavel ocorre nos reds/static/unit. A integracao local de B05 so inicia apos B01-B04 verdes, e a VM so inicia apos toda a matriz local verde.

## Residual risks

- O daemon Docker local estava indisponivel durante o planejamento; se continuar ausente, B05 deve retornar bloqueado no gate local e nao usar a VM como substituto.
- A imagem fixada precisa suportar a arquitetura do host e da VM; pull/compatibilidade sao evidencias de runtime, nao static/unit.
- Os volumes descartaveis ficam preservados porque exclusao nao foi autorizada; seus nomes devem ser reportados ao owner para uma decisao posterior de limpeza.
- Mesmo uma matriz VM verde prova somente a VM de teste identificada. Producao, rede externa e dados reais permanecem nao autorizados e nao verificados.
