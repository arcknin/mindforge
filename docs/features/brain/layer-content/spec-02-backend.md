---
schema_version: 1
managed_by: spec-drive
feature: brain
task: layer-content
revision: 1
status: implemented_local_runtime_pending
---

# Documentos Markdown — backend primeiro

## Contexto e fontes

O owner autorizou fechar a spec com TDD e iniciar pelo backend em 2026-10-12; confirmou antecipar API/persistência PostgreSQL, deixando client e sincronização para depois. Esta decisão substitui a ordem local-primeiro de PD-016, sem autorizar upload periódico/logout nem VM. Fontes: [discovery](discovery.md), [direção documental](spec-01-main.md), [ledger](../../../product-decisions.md), [registry](../../../../contracts/openapi/registry.yaml), `features/brain` e `api/handler.go` do Brain Engine. Checkout misto preexistente preservado.

Esta spec separa o lote backend porque seu contrato HTTP, persistência, clock e prova são diferentes do editor/IndexedDB. A spec01 conserva a direção de produto e o planejamento do client, ainda sem execução. Este lote não depende da implementação do client.

## Objetivos e exclusões

Entregar código backend para criar, listar, abrir, editar/renomear/mover, enviar à lixeira e restaurar documentos privados do próprio Brain. Corpo Markdown como texto, um layer por documento, ID estável, títulos repetidos, consulta direta/subárvore, CAS por versão e referências internas estáveis.

Fora: client/editor/IndexedDB, sincronização e reconciliação entre dispositivos, funções cognitivas, grafo visual, LLM, anexos, import/export, aliases, embeds e links a blocos/headings, mover/excluir camadas, deploy, SQL executado, banco temporário, credenciais, commit/push. Nenhum pacote/framework novo.

## Comportamento fechado para este lote

- Brain e layer devem existir; sem bootstrap implícito. Identidade autenticada GoTrue existente, nunca owner vindo do request. Recurso alheio/ausente: mesmo404.
- Create recebe UUID canônico minúsculo fornecido pelo solicitante (permite identidade prévia futura); não reutilizar ID, inclusive após purge. Criação repetida:409, não tratar como replay bem-sucedido. Não há protocolo de sync/deduplicação neste lote.
- Título é trim, 1..255 codepoints Unicode; sem controles, incluindo newline/NUL. Corpo pode ser vazio, UTF-8 válido, sem NUL, até1MiB em bytes. Limite de request documental8MiB cobre escape JSON de um corpo UTF-8 de até1MiB. Limite não é promessa de capacidade total por conta.
- Update envia snapshot completo de layer_id/title/markdown + expected_version inteiro1..MaxInt64. Create começa1; toda mutação válida avança versão, inclusive same-value; overflow rejeita409. Não sobrescrever edição concorrente nem executar retry cego após transporte/5xx.
- List recebe scope=direct|subtree (default direct), state=active|trash (default active), limit1..100 (default50), after=UUID opcional. Ordenação UUID ascendente, limit+1 para next_cursor, cursor exclusivo e sem snapshot entre páginas; novos registros podem exigir reiniciar a consulta. Resultado sem corpo/links, com camada de origem. Subárvore inclui camada consultada e descendentes; root estrutural não é layer.
- Get aceita ativo ou lixeira ainda válida e inclui corpo e referências resolvidas. Expirado/purged:404.
- Trash usa CAS. Em item já na lixeira, versão corrente permite repetir sem renovar prazo nem incrementar versão. Restore requer versão corrente e item na lixeira antes do vencimento; destino explícito opcional, default camada de origem, sempre camada válida do mesmo Brain. Ativo/restauração repetida:409.
- No servidor, prazo240h calculado pelo relógio PostgreSQL, nunca pelo client. Vencimento em agora>=expires_at; restore rejeita404. Limpeza oportunista por conta durante operações documentais remove texto/título/referências de origem, conservando tombstone mínimo de identidade/ownership/versão para impedir recriação. Sem job global, promessa de purge com app parado ou apagamento em backups. Lixeira local do client permanece contrato separado.
- Referência canônica no corpo: `[[doc:UUID]]`. A UI futura oferece seleção `[[Título]]` e grava identidade; texto `[[Título]]` não resolvido permanece texto pendente, sem resolver pelo primeiro nome ou criar documento. Não há alias personalizado neste lote. Referências em código fenced/inline e escapes não são vínculos. Parser documenta subconjunto suportado, sem alegar implementação completa CommonMark.
- Corpo preserva tokens por ID; resposta links projeta título atual/layer/status do destino (available/trash/missing). Rename muda nome projetado, move conserva ID, sem reescrever conteúdo de outros documentos nem aumentar sua versão. Destino alheio/inexistente em token canônico rejeita404; destino conhecido na lixeira/tombstone pode continuar referenciado como indisponível. Targets distintos são indexados atomicamente com corpo, sem duas fontes independentes. Não reinterpretar vínculos existentes por nome.

## Contrato HTTP aprovado

Base `/v1/me/brain`, Bearer GoTrue existente, JSON estrito, campos desconhecidos/repetidos/null indevido rejeitados. Erro `{error:{code,message},request_id}`, sem SQL/segredos/conteúdo. Operações aprovadas pelo owner após proposta concreta nesta conversa:

| Método/path | Request | Sucesso |
|---|---|---|
| POST /documents | id,layer_id,title,markdown |201 `{document}`|
| GET /layers/{layer_id}/documents | query scope,state,limit,after; sem corpo |200 `{documents,next_cursor}` (cursor null ao terminar)|
| GET /documents/{document_id} | sem query/corpo |200 `{document}`|
| PATCH /documents/{document_id} | expected_version,layer_id,title,markdown |200 `{document}`|
| POST /documents/{document_id}/trash | expected_version |200 `{document}`|
| POST /documents/{document_id}/restore | expected_version,layer_id opcional |200 `{document}`|

Document: id,layer_id,title,markdown,version,created_at,updated_at,deleted_at,expires_at,links. Datas UTC RFC3339; delete/expiry null quando ativo. Link: target_id,title,layer_id,status; missing não revela texto/posição anterior. List usa os mesmos metadados sem markdown/links. Versão Int64 JSON, consumidores futuros devem preservar precisão como no Core. ID inválido no path/query400; payload bem-formado inválido422 invalid_document; estrutura/query400 invalid_request; stale/ID existente409 conflict; auth401/503; alheio/ausente404; método405; limite413; media415; interno500. Sem endpoint público de purge.

## Design, ownership e sequência

Novo módulo `features/documents`, independente das quatro operações do Brain Core. Engine valida entrada e parser; Store realiza ownership, transações, CAS, tempo e projeção. PostgreSQL pgx já existente, sem ORM. Migration canônica documents + checks preflight/verify e entrada global0007 via catálogo infra; não aplicar.

Portas normativas: `Store.Apply(context.Context, brain.Principal, Mutation) (Document,error)`, `Get(context.Context, brain.Principal,string) (Document,error)`, `List(context.Context,brain.Principal,Query) (Page,error)`. Mutation: Kind (create/update/trash/restore), ID,LayerID,Title,Markdown string; ExpectedVersion int64; Targets []string. Document: ID,LayerID,Title,Markdown string; Version int64; CreatedAt,UpdatedAt time.Time; DeletedAt,ExpiresAt *time.Time; Links []Link. Link: TargetID,Title,LayerID,Status string. Query: LayerID,Scope,State,After string; Limit int. Page: Documents []Document; NextCursor string. Engine métodos Create/Update(ctx,owner,Mutation), Trash(ctx,owner,id,version), Restore(ctx,owner,id,version,layer), Get(ctx,owner,id), List(ctx,owner,Query). Erros usuais brain.Err*; validação usa documents.ErrInvalidDocument e ErrInvalidQuery. Targets derivados pelo Engine; store nunca aceita ownership fornecido pelo client.

1. Fechar direção/backend e propostas concretas no registry; obter aprovação HTTP conforme AGENTS.
2. TDD Engine/parser e fake transacional somente de teste; nenhum store em memória de produto.
3. TDD adapter pgx e migration/checks estáticos; nenhuma execução SQL.
4. Após aprovação HTTP: OpenAPI completo, TDD handler e composição runtime com dependência documental; preservar construtor/Core existentes e não migrar no startup.
5. Tests/race/vet/build Go e checks documentais; registrar prova local e limites. Integração PostgreSQL e aceitação HTTP real ficam pendentes na VM.

Ownership: principal spec/contratos/API/runtime/integração; worker leaf domínio/parser/testes exclusivamente arquivos de domínio documents; worker leaf adapter/migrations/checks/catálogo exclusivamente persistência. Sem reviewer obrigatório.

## Aceite, testes e TDD

TDD=yes explícito: observar RED antes de implementação comportamental, GREEN e refactor. Casos: vazio/limites/Unicode; IDs repetidos; rename/move/links por ID; código/escape não gera link; duas camadas homônimas sem rerouting; CAS concorrente um vencedor; isolamento ownership inclusive destinos de links; subárvore e paginação; trash/re-delete/restore; instante de expiração; purge remove conteúdo e impede reutilizar ID; falha transacional não grava corpo sem vínculos; protocolo/Auth e precisão Int64.

Validação local: `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...` no módulo Brain Engine; testes do catálogo infra afetados, YAML/OpenAPI/registry e git diff --check. Fakes/static SQL não provam PostgreSQL executar, constraints/locks/permissões/runtime. Aplicação/validação SQL somente VM indicada, com autorização futura. Sem scripts extras: comandos/testes bastam.

## Autoridade, riscos e rollback

Owner autorizou preparação/backend com TDD; client/sync depois. Owner aprovou especificamente os seis métodos/paths e a lixeira remota240h depois de ler a proposta concreta; promover registry/OpenAPI e implementar domínio, persistência e HTTP está autorizado. Sem autoridade VM, instalação, dados reais, execução SQL ou publicação.

Riscos: schema/SQL não executados, dois pools pgx caso necessário para preservar interfaces existentes, relógio servidor difere do local, tokens canônicos específicos Mindforge sem promessa de interoperabilidade, paginação sem snapshot, segurança depende do acesso exclusivo pelo Engine/DB role. Sem expandir para engine de plugins ou parser completo. Stop se exigir novos endpoints/comportamento/autorização ou mudar isolamento.

Rollback de código: retirar somente arquivos/mudanças deste lote; preservar checkout anterior. Migration não aplicada dispensa rollback de dados; depois de aplicação exige plano próprio, nunca DROP automático.

## Aprovações e retomada

- Produto/backend e TDD: owner execution request + resposta explícita nesta conversa, 2026-10-12.
- HTTP: owner aprovou explicitamente o contrato das seis rotas, incluindo lixeira remota240h, em 2026-10-12.
- SQL/VM/deploy: não autorizados.
- Progresso/evidência: [checkpoint](layer-content-checkpoint.yaml); nenhuma alegação de runtime antes de chamada real identificada.

## Fechamento local — 2026-10-12

Código backend concluído no escopo aprovado, com TDD e testes Go/race/vet/build PASS. Seis endpoints approved/implemented/runtime pending. Migration0007 e checks versionados; não aplicados. Evidência e limitações em [implementation-evidence.md](implementation-evidence.md). Client/sync posteriores; próxima autoridade necessária é SQL/VM develop, sem promover esta evidência a runtime.

## Continuação operacional — 2026-10-13

A tarefa [android-workspace](../android-workspace/spec-01-main.md) recebeu autorização específica para backup,0007 e API develop. Migration aplicada/verificada pelo operador e API publicada, conforme [evidência VM](../android-workspace/vm-runtime-evidence.md). O fechamento local anterior é histórico: SQL/schema e deploy agora têm prova real; CRUD/DML/Google/sync autenticados e runtime dos endpoints continuam pendentes do teste Android do owner. Main/NSR/Studios/Auth/volumes preservados.
