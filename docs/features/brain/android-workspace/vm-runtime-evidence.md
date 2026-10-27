# VM develop — backup, migration0007 e API documental

Executado em 2026-10-13, entre 17:47 e 17:52 UTC, exclusivamente na VM indicada pelo owner. Autoridade: `owner-android-workspace-develop-20261013`; escopo definido em [vm-develop-plan.md](vm-develop-plan.md), [spec Android](spec-01-main.md) e [spec backend documental](../layer-content/spec-02-backend.md). Nenhuma instalação no aparelho, promoção para main, alteração Auth/Studios/NSR ou operação de restore foi executada.

## Release e configuração preservada

- VM Linux `x86_64`; binários Linux/amd64 compatíveis. Os17 arquivos do manifest do pacote tiveram tamanho/SHA-256 conferidos antes da execução.
- Release privado: `/opt/mindforge/releases/documents-develop-20261013-r1/mindforge-documents-develop-runtime`. Evidências privadas ficam no diretório pai, com acesso restrito. O campo `authorization` do manifest preparado anteriormente ainda diz `pending_owner_vm_approval`; a autoridade operacional usada pelo operador é a aprovação específica acima, não esse metadado histórico do pacote.
- Operador: `mindforge-infra` do pacote, executado como utilitário no image PostgreSQL já existente, sem criar servidor/banco temporário; rede `infra_default`, config privada existente `/etc/mindforge/develop-studio/infra.env`, catálogo versionado `/workspace/source-catalog`.
- API Compose: projeto `mindforge-brain-engine-develop`, único serviço `brain-engine-develop`; env privado existente `/etc/mindforge/manual-online/api.env.private`. Valores de env preservados por comparação privada; nenhuma credencial consta nesta evidência.
- Bind preservado `127.0.0.1:18087:8080`, rede `infra_default`. Google habilitado/email desabilitado preservados; allowlist real contém o callback aprovado `mindforge-dev://auth/callback` e o callback Web existente. Não houve mudança no Auth.

## Backup antes da aplicação

O operador executou `backup --target mindforge_develop --output-dir /backups` com `--source-root /workspace/source-catalog`, operator ID `codex-android-workspace` e referência de autorização acima. `/backups` é somente o mount do diretório privado existente `/home/kali/mindforge-backups` na VM. O trabalho pesado permaneceu na VM.

- Artifact: `/home/kali/mindforge-backups/mindforge-infra-mindforge_develop-20261013T174753.565734160Z.dump`.
- Manifest: mesmo path acrescido de `.manifest.json`.
- SHA-256: `ee010a089a4cf42dde7adb865564e2075c26404effeff56f4921964d4eecc7e2`; tamanho99.537 bytes, hash reconferido após aplicação.
- Custom archive e listing `pg_restore --list` validados pelo operador: `archive_listing_verified=true`. `restore_proof=not_verified`; backup válido não é prova de restauração.

## Migration e prova SQL real

`preflight --scope feature --target mindforge_develop` passou antes do backup e novamente imediatamente antes de `apply`. Referência pré-aplicação: `preflight/mindforge_develop/b739d556a8578b7416308e4c87d767a03d232f4c08f77affa10153614483d7a1`.

`apply --scope feature --target mindforge_develop --preflight-ref <referência acima> --verification-ref documents-backend-local-tests-20261012` retornou somente `applied=0007 target=mindforge_develop`. O operador manteve DDL, checks canônicos e registro no ledger na mesma transação. A referência de behavior aponta aos testes locais existentes; ela não representa CRUD autenticado na VM.

- Payload0007: `37c0378f6a4d18e98505e31baf50ebb8e36da071868beae05e4631a92fed351f`.
- Ledger develop confirmado por SELECT read-only:0001..0007; hashes0001..0006 continuam canônicos, com datas de aplicação anteriores.
- Evidência0007: `3cc1dada-d73c-4250-8b88-c85ad70e2bbe`, `post_application_result=succeeded`, `sanitized_outcome=succeeded`; aplicação em2026-10-13T17:48:37Z.
- Preflight feature posterior passou: `preflight/mindforge_develop/f3d31e6d622ab2ecfb9b56184347b6f2b2a06bdd5ac4406a60838d78c42c69b0`.
- Contagens após migration: `brain_documents.documents=0`, `brain_documents.document_links=0`. Nenhum documento/usuário de teste foi criado, excluído ou alterado. Nenhuma conexão a main/NSR foi usada.

## Deploy e preservação de serviços

Build scratch da imagem empacotada passou; execução limitada a `docker compose ... build brain-engine-develop` e `up -d --no-deps --no-build --pull never brain-engine-develop`. Não houve `down`, pull/restart global ou mudança de volumes.

- Imagem anterior preservada para rollback: `mindforge/brain-engine:manual-online`, image ID `sha256:a18af7d9ed6729d7670b282e4bef5801ebf2c3a970bab362d3bb7cf19a824bc7`; configuração anterior `/opt/mindforge/releases/manual-online-20261008-r2/compose.yaml`.
- Nova imagem: `mindforge/brain-engine:documents-20261013`, image ID `sha256:9ca5c0d8d41667da03176756573e06def610057cd5047a5ac8b43e7afaffe877`.
- API container anterior: `af05ac7a8a7593a4260558d303ee4de3b9a6513cccf70de7dd5d489a95038120`; novo: `51b1f5ce049f27efd3c3ead48f980a5a52c826662bd0ffc1bbe8d72ac9c289b7`, running após startup/schema guard.
- Baseline50 containers,28 running. Os49 containers protegidos (27 running) preservaram exatamente ID, estado e `StartedAt`; inclui PostgreSQL, Auth main/develop, Studios, Web e serviços NSR. Snapshot privado antes/depois comparado por nome.
- SHA-256 da lista ordenada de identidade/estado/StartedAt protegidos: `aea21ba8f46964b3edada819ae6b4096977d2d2d4318f16bcdddaa9dbffdf824`.
- Provas privadas: `baseline.private.json`, `api-before.private.json`, `backup.private.log`, `preflight-before.private.log`, `apply.private.log`, `preflight-after.private.log`, `sql-proof.json`, `api-build.private.log`, `api-up.private.log`, `api-startup.private.log`, `runtime-proof.json`. Procedimentos sem credenciais: `collect-proof.py`, `deploy-api.py`, no diretório pai do release. Não repetir deploy para mera leitura de evidência.

## HTTP e limites do aceite

Chamadas reais sem token em `http://127.0.0.1:18087` retornaram401 com `error.code=unauthenticated` nos seis métodos/paths documentais:

| Método | Path |
|---|---|
| POST | `/v1/me/brain/documents` |
| GET | `/v1/me/brain/layers/{layer_id}/documents` |
| GET | `/v1/me/brain/documents/{document_id}` |
| PATCH | `/v1/me/brain/documents/{document_id}` |
| POST | `/v1/me/brain/documents/{document_id}/trash` |
| POST | `/v1/me/brain/documents/{document_id}/restore` |

IDs UUID sintáticos foram usados somente nos paths de requisições rejeitadas antes do Engine. O proxy Web existente ainda retorna401 JSON em `/api/v1/me/brain`; rotas documentais Web permanecem404 conforme seu allowlist anterior, fora desta entrega Android. O Android usa o túnel direto para18087 conforme [guia de teste](android-test-guide.md).

SQL canônico executado e serviço iniciado são prova operacional real. As chamadas401 demonstram rejeição sem autenticação; **não** provam Google/PKCE no aparelho, ownership/CAS/CRUD documental autenticado ou sync. Os seis endpoints permanecem `contract_status=approved`, `implementation_status=implemented`, `runtime_status=pending`. Próximo gate: instalação e aceite integrado pelo owner no aparelho, com túnel/ADB e conta real; não executados por esta operação.
