---
schema_version: 1
managed_by: spec-drive
feature: brain
task: layer-content
revision: 7
status: product_closed_backend_first
---

# Documentos Markdown nas camadas — contrato de produto

## Contexto e fontes

Owner fechou direção documental com referência no Obsidian básico, confirmou links acompanhando rename e autorizou backend primeiro com TDD. [PD-009..020](../../../product-decisions.md), [discovery](discovery.md), [histórico](planning-history.md), [backend executável](spec-02-backend.md) e [checkpoint](layer-content-checkpoint.yaml). Esta spec consolida as decisões; revisões antigas não mantêm pendências que foram fechadas depois.

## Objetivos e limites

Documentos Markdown são conteúdo de uma única camada neural, identificados por UUID estável e título separado. Pai pode consultar documentos próprios e de todos os descendentes, sem cópias ou mudança de pertencimento. Títulos repetidos permitidos, com escolha explícita do destino de links. Texto humano e futuro texto produzido por agentes autorizados obedecem às mesmas regras; não executar conteúdo como instrução/efeito.

Operações: criar/listar/abrir/editar/renomear/mover/excluir/restaurar documentos. Não adicionar lifecycle de camadas por inferência. Fora: funções cognitivas e estado por uso (conceito separado em sua própria pasta), grafo visual, IA, Hive Mind/Inbox, anexos, import/export, vault/filesystem, aliases/embeds/links a blocos/headings, histórico navegável e sincronização nesta entrega backend.

## Comportamento e experiência

- Referência de interação: notas básicas do Obsidian, sem compromisso de paridade integral. Editor inicial Markdown fonte + prévia, incluindo títulos/listas/tabelas/código/links. Live Preview inline não é pré-requisito.
- Documento vazio permitido. Backend exige título normalizado1..255 codepoints sem controles e corpo UTF-8 até1MiB sem NUL. Client futuro pode sugerir `Sem título`; títulos duplicados continuam válidos. Limites do transporte constam no contrato HTTP.
- UI futura distingue documentos próprios e consulta da subárvore por seleção explícita; documentos herdados mostram camada/caminho de origem. Abrir navega para documento, mover escolhe camada válida; nenhuma operação cria camada implicitamente. Lixeira é visualização separada.
- Digitar `[[` oferece seleção por título/camada; destino confirmado é UUID, nunca posição/título. Representação canônica Mindforge: `[[doc:UUID]]`; título não resolvido permanece pendente e sinalizado, sem criação automática. UI futura apresenta o nome atual; usuário não precisa administrar IDs. Código fenced/inline e escapes não geram vínculos; subconjunto do parser documentado no backend.
- Rename atualiza nome projetado do destino em seus links; move conserva destino. Corpo canônico conserva token por ID, índice de relações é derivado atomicamente. Documento homônimo não redireciona vínculos. Lixeira torna destino indisponível; restore válido conserva ID/vínculos; purgado permanece missing, sem reutilização da identidade.
- Backend privado é autoridade remota para Auth, ownership, validação, persistência e CAS. Client guarda somente sua cópia/rascunhos locais e não acessa SQL diretamente. Referência/conteúdo de outra conta nunca fica acessível por link.

## Ordem de entrega, dependências e autoridade

**Backend primeiro autorizado:** [spec02](spec-02-backend.md), API e persistência PostgreSQL, seis contratos HTTP aprovados, TDD=yes. Não requer editor/localstore concluído. Migrations versionadas apenas; aplicar SQL e fazer deploy requer autorização futura na VM indicada.

**Client posterior, sem implementação neste lote:** login/árvore existentes; IndexedDB separado do Auth/sessionStorage, namespace por origem/ambiente/usuário. Autosave nominal1s enquanto aplicativo executa, confirmação somente no commit. Transações + revisão local e rascunhos por aba preservam conflito, sem sobrescrita silenciosa; BroadcastChannel é só aviso opcional. Releitura ao retomar foco; upgrade/quota/falha não apagam banco nem alegam salvo. Draft da outra aba nunca reverte move/rename/delete.

**Sincronização posterior:** upload automático30min, botão e tentativa antes do logout são direção futura, sem implementação de envio agora. API CRUD pronta não equivale a protocolo de sincronização. Reconciliação/versões locais vs remotas/deduplicação/autoria e filas serão contratadas nesse lote, sem transportar clock do client como autoridade.

## Lixeira, conta e falhas

Lixeira local futura:240h do commit local da exclusão, clockUTC dispositivo, restore antes do vencimento, purge na próxima execução; mudar relógio altera a avaliação. Backend:240h pelo clock PostgreSQL, CAS e purge oportunista por conta, tombstone mínimo sem corpo/título após vencimento. Não prometer job com aplicativo parado, exatidão física de remoção ou apagamento em backups.

Re-delete corrente não renova prazo. Restore conserva conteúdo/ID antes expiry e exige camada própria válida; se origem indisponível, conservar item e solicitar destino. Outra aba não pode purgar item restaurado nem ressuscitar item removido.

Logout futuro conclui flush local se possível, limpa UI/invalida callbacks da sessão antiga e conserva dados confirmados por conta. Falha permite cancelar ou sair conscientemente sem falso salvo. A→B não mostra nem grava dados de A em B; A ao relogar recupera commits/rascunhos. Envio no logout pertence à fase sync, antes de encerrar credencial; falha conserva pendências.

## Aceite e validação

- Criar/reabrir documento mantém Markdown, UUID e camada; títulos repetidos não fundem documentos.
- Pai consulta subárvore sem cópia; mover altera consulta e preserva links.
- Rename mostra nome atual por UUID; referência pendente/indisponível é sinalizada, sem autocreate.
- Duas alterações na mesma revisão: uma vence, outra conflita sem perda silenciosa. Persistência de corpo/relações é atômica.
- Lixeira/restore/expiry obedecem clock e fase; conta alheia recebe recurso ausente.
- Autosave/client/sync têm prova própria posterior; testes backend não provam editor, IndexedDB ou navegação real.

TDD=yes mantido explicitamente; backend segue spec02 test-first. Provas locais, SQL/VM, runtime HTTP e aceite do owner permanecem separados. Não criar banco temporário para validar SQL. Nenhum script extra requerido nesta consolidação.

## Decisões e retomada

Produto fechado para este recorte. Defaults de implementação podem ser refinados sem alterar as regras; diferenças materiais de comportamento/segurança/escopo/autoridade voltam ao owner. Backend e contratos aprovados; execução client/sync/VM não incluída. Histórico de discussões preservado em planning-history.md. Próximo passo atual: completar spec02 e evidência local, depois planejar validação da migration na VM quando autorizada.
