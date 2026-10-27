# Evidência local — backend documental

## Escopo e autoridade

Owner autorizou backend com TDD em 2026-10-12, confirmou API/PostgreSQL antes do client/sync e aprovou seis rotas concretas e lixeira remota240h. [Spec executável](spec-02-backend.md), [contrato](../../../../contracts/openapi/documents.openapi.yaml), [checkpoint](layer-content-checkpoint.yaml). Mudanças preexistentes no checkout misto preservadas; nenhum commit/push.

Implementação: módulo `features/documents` (Engine, parser, adapter pgx); handler/API e startup develop; migration canônica/checks e catálogo global0007. Test-only memória e fake pgx não são persistência de produto. Startup verifica tabelas, não aplica SQL; novo binary depende da migration futura. Runtime/Auth existentes preservados.

## TDD observado

- Domínio RED: símbolos Engine/Mutation/ErrInvalidDocument ausentes; GREEN Engine/parser e cenários de ownership/CAS/subárvore/lixeira/links no fake transacional.
- HTTP RED: `NewHandlerWithDocuments` ausente. GREEN seis operações, JSON estrito, query, Auth, erros opacos e Int64 acima de2^53.
- HTTP regressão RED: restore com layer_id explicitamente vazio era aceito. GREEN distingue ausência de destino de destino inválido. Corpo Markdown1MiB com escapes JSON válido aceito dentro de8MiB de transporte; payload excessivo413.
- Runtime RED: dependências documentais não existiam no probe. GREEN schema documental obrigatório antes de abrir runtime Store.
- Adapter: RED/GREEN de transações e projeções pelo worker; integração principal identificou cursor pulando lookahead, commit com rows abertas, expiração concorrente e divergência de nomes de índices/checks; correções e evidências finais constam no fechamento abaixo.
- Infra RED na suíte completa: catálogo passou a sete migrations mas baseline ainda tinha seis pins. Atualizar apenas pin/catálogo novo, preservando0001..0006.

## Comandos e limitações

Cache Go direcionado a `/private/tmp/mindforge-go-build` porque cache padrão não era gravável no sandbox. Testes HTTP existentes usam servidores loopback; a primeira execução restrita falhou ao bind, e suíte completa foi executada com permissão para esses listeners locais. Não há conexão VM/SQL nesses testes.

Comandos finais planejados: `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...` no módulo `src/brain-engine`; YAML/refs OpenAPI/registry, links e `git diff --check`. Resultado final: todos PASS após integração dos workers.

Nenhum PostgreSQL iniciado, descartável ou temporário. SQL não executado nem validado pelo parser do servidor; locks, triggers, constraints, permissões e integração do driver precisam de validação na VM autorizada. Testes fakes/static não promovem runtime. Nenhum backend novo buildado/deployado na VM, client alterado, sync implementado ou documento real manipulado.

## Próximo gate

Depois de evidência local: preparar/autorizar aplicação e integração na VM develop, com preflight/backup segundo fluxo infra existente. Main, NSR e volumes permanecem fora do lote. Aceite HTTP real e client/editor/sync continuam posteriores, separados.

## Fechamento e resultado final

Em 2026-10-12 (evidência UTC2026-10-13T01:47:00Z):

| Verificação | Resultado |
|---|---|
| `go test ./...` | PASS, módulo completo, incluindo Auth/Core/infra |
| `go test -race ./...` | PASS, módulo completo |
| `go vet ./...` | PASS |
| `go build ./...` | PASS, compilação local |
| `bash src/brain-engine/features/infra/tests/run.sh unit migrations` | PASS, checks estáticos/fakes |
| YAML, seis paths/approvals e refs internas OpenAPI | PASS |
| Links documentais e `git diff --check` | PASS |

Correções integradas: cursor usa último ID retornado, rows fechadas antes do commit; purge usa locks ordenados e recheck de estado/expiração, sem apagar restore concorrente por snapshot antigo; GET tombstone confirma manutenção antes404; links vencidos projetam missing sem título/camada; clock consistente por projeção; restore protege overflow; pool extra limitado4conexões. Parser usa lookahead fixo para UUID e índice de crases para evitar rescans quadráticos; teste de estresse1MiB sem timing frágil. Fakes não provam locks PostgreSQL.

Arquivos principais: `features/documents/{model,engine,links,postgres_store}.go` e testes/migrations; `api/documents.go`, `api/documents_test.go`, ajuste `api/handler.go`; `cmd/brain-engine/{main,runtime,runtime_test}.go`; catálogo e baseline global infra; OpenAPI/registry e documentação desta unidade. Novo código não altera client nem as quatro operações Core. Pins0001..0006 preservados. SQL0007 SHA-256: `37c0378f6a4d18e98505e31baf50ebb8e36da071868beae05e4631a92fed351f`.

Estados das seis operações documentais: `contract_status: approved`, `implementation_status: implemented`, `runtime_status: pending`. Evidência runtime vazia de propósito. O build local não significa serviço atualizado na VM. O novo binary requer as tabelas documentais antes do startup; aplicar a migration e publicar o binary continuam etapas separadas.
