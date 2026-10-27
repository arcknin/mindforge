# Operação proposta — liberar o teste integrado Android

Status: owner autorizou especificamente backup, migration0007 e API develop em2026-10-13; operação concluída, com evidência em vm-runtime-evidence.md. A implementação do client não autoriza executar esta operação. Fontes: spec desta tarefa, backend layer-content/spec02, migration canônica0007 e operador infra existente.

## Alvo e efeitos

Aplicar somente a nova migration global0007 na base física develop e atualizar somente o serviço Brain Engine develop. Main, NSR, Studios, volumes e credenciais não são alvos. Não promover migrations para main, restaurar/copiar banco ou remover dados/volumes. Não alterar Google Cloud nem o método Google-only.

O pacote local `/private/tmp/mindforge-documents-develop-runtime.tar.gz` contém binários Linux/amd64 compilados sem rede, Dockerfile/Compose do único serviço API e catálogo/payload/checks canônicos. Manifest local contém SHA-256, sem credenciais. A arquitetura da VM será conferida antes de usar os binários; se diferente, recompilar localmente para o alvo real. Projeto Compose continua `mindforge-brain-engine-develop`, image tag nova `mindforge/brain-engine:documents-20261013`.

Migration0007: `src/brain-engine/features/documents/migrations/0001_documents.sql`; SHA-256 `37c0378f6a4d18e98505e31baf50ebb8e36da071868beae05e4631a92fed351f`. Pins0001..0006 preservados. DDL, verificação e registro em infra.migration_ledger compartilham a transação do operador.

## Sequência após autorização

1. Inventário somente leitura: VM/arquitetura, container/API develop, imagem atual para rollback, binding18087, cluster e utilitários PostgreSQL. Resolver configuração privada existente sem copiar/imprimir senhas ou tokens. Conferir callback nativo mindforge-dev e Web preservados no Auth develop; divergência não autoriza alterar Auth silenciosamente.
2. Transferir pacote e verificar hashes. Manter snapshot da configuração/imagem anterior e evidência sanitizada em diretório privado da VM.
3. Criar/validar backup develop fora do checkout através do operador existente; backup pesado permanece na VM. Não executar restore nem criar DB temporário.
4. Executar preflight feature develop com catálogo novo. Exigir0001..0006 aplicadas com hashes canônicos,0007 ausente elegível (ou já completa para repetição idempotente). Estado parcial/divergente interrompe aplicação para reconciliação.
5. Aplicar0007 através de mindforge-infra apply com alvo, autorização, preflight e referência de validação explícitos. Validação SQL ocorre somente nesta VM. Falha de schema/checks: transação rollback, corrigir fonte local e seus pins antes de nova tentativa; não alterar hashes de migration já aplicada.
6. Construir imagem scratch com binário e recriar somente brain-engine-develop no projeto existente, mantendo env privado, rede e bindloopback. Não iniciar stack inteira nem recriar Auth/Studios.
7. Verificar startup/schema/ledger, erro401 sanitizado sem token e preservação de serviços protegidos. Falha API: restaurar somente imagem/config anterior; não DROP de schema ou restore do DB por inferência.
8. Teste positivo Google/CRUD é aceite separado no aparelho pelo owner. Teste sem token não promove runtime_status dos endpoints a verified.

## Limites da prova

Backup/DDL/deploy reais ainda não executados ao preparar este plano. Não declarar pronta a integração só por build ou mocks. Timers Android dependem de execução; teste real de consentimento/redirect, sessão segura, toque/teclado e sync requer aparelho, USB/túnel e owner.
