# Verificação read-only de sincronização — 2026-10-14

## Escopo e autoridade

- Objetivo: verificar se a sincronização manual observada na UI chegou ao PostgreSQL develop.
- Ambiente: VM indicada pelo owner no AGENTS; banco `mindforge_develop` apenas. Endpoint SSH permanece fora deste registro.
- Execução: sessão SSH interativa autorizada pelo owner; o ControlPath histórico estava ausente.
- Consulta: [sync-check-20261014.sql](sync-check-20261014.sql), com `BEGIN TRANSACTION READ ONLY`, somente `SELECT` e `ROLLBACK`.
- Final da consulta na VM: `2026-10-14T15:27:26Z` (UTC).
- Nenhum corpo Markdown, e-mail, token, segredo ou ID completo foi registrado.

## Saída sanitizada observada

```text
{"current_database" : "mindforge_develop", "current_user" : "mindforge_develop"}
{"total_users" : 1}
{"total_brains" : 2, "distinct_brain_owners" : 2, "nonblank_owner_subjects" : 2, "owners_with_auth_user" : 1, "owners_without_auth_user" : 1}
{"total_documents" : 2, "active_documents" : 2, "trash_documents" : 0, "expired_trash_rows" : 0, "tombstoned_rows" : 0, "documents_without_brain" : 0, "documents_without_owner" : 0, "documents_without_valid_layer" : 0, "owner_coherent" : true}
{"document_id_short" : "72a98fa5", "title" : "Teste", "version" : 2, "created_at_utc" : "2026-10-14T15:17:16.312015Z", "updated_at_utc" : "2026-10-14T15:17:16.676312Z", "markdown_state" : "nonempty", "markdown_bytes" : 20, "markdown_chars" : 20, "state" : "active", "owner_coherence" : "coherent"}
{"document_id_short" : "7757879d", "title" : "Outro teste", "version" : 2, "created_at_utc" : "2026-10-14T15:17:16.490066Z", "updated_at_utc" : "2026-10-14T15:17:16.845409Z", "markdown_state" : "empty", "markdown_bytes" : 0, "markdown_chars" : 0, "state" : "active", "owner_coherence" : "coherent"}
```

## Leitura

O documento `Teste` chegou ao banco com conteúdo não vazio: 20 bytes e 20 caracteres, versão 2, ativo, criado e atualizado em UTC no intervalo `15:17:16Z`. Isso confirma chegada de uma versão persistida no servidor; não prova igualdade byte a byte com um rascunho local porque o corpo não foi exportado nem comparado.

`Outro teste` também está persistido, mas tem Markdown vazio. Deve ser tratado como placeholder/documento vazio, não como prova de upload de conteúdo.

Os dois documentos têm Brain, owner Auth existente e layer válido (`owner_coherence=coherent`). O agregado documental é coerente: nenhum documento órfão, sem owner, sem layer válido, em lixeira ou tombstone.

Há uma anomalia separada de ownership global: existem 2 Brains e 2 `owner_subject` distintos para 1 conta em `auth.users`; somente 1 owner corresponde a uma conta Auth. Os documentos encontrados pertencem ao owner correspondente, então isso não invalida a chegada dos dois documentos, mas impede declarar o estado global de Brains como perfeitamente coerente.

## Limites

- Não houve login, chamada HTTP, mutação, deploy, restart, instalação ou QA visual.
- Não foi lido o corpo Markdown integral.
- Não foi feita comparação com o rascunho local; a evidência é presença, metadados e tamanho/estado no banco.
- AND-001 e AND-002 permanecem adiados conforme o checkpoint; esta verificação não os diagnostica nem corrige.
