---
schema_version: 1
managed_by: spec-drive
feature: brain
task: android-workspace
revision: 2
status: implemented_local_runtime_pending
---

# Client Android — layouts adaptativos, camadas e documentos

## Contexto e fontes

Em 2026-10-13, o owner autorizou iniciar esta tarefa, implementar os layouts adaptativos, preservar o client existente e acrescentar documentos para teste conjunto. TDD foi confirmado. Depois confirmou rascunho local a cada segundo, sincronização automática a cada 30 minutos e botão manual.

Fontes: [discovery](discovery.md), [regra de layout](../../../stack/front-end/adaptive-layouts.md), [produto documental](../layer-content/spec-01-main.md), [backend](../layer-content/spec-02-backend.md), [OpenAPI](../../../../contracts/openapi/documents.openapi.yaml) e registry. Arquivos atuais prevalecem sobre desenhos históricos. Checkout misto preservado; sem branch, commit ou publicação automática.

## Objetivos e exclusões

No mesmo Flutter, preservar Google/GoTrue/PKCE, sessão, árvore, criação e rename de camadas, erros e conflitos; acrescentar documentos e sync; gerar APK privado para Android. Compartilhar estado/componentes entre composições compacta e ampla.

Fora: funções cognitivas, grafo, LLM, Hive Mind/Inbox, anexos, import/export, vault público, aliases/embeds/links para blocos, outro login, novos endpoints, polimento PC, build iOS, publicação e instalação no aparelho. Documentos Web/IndexedDB pertencem a outra entrega; Web existente preserva Auth/Core/layout e não recebe falso storage persistente em memória. SQL/VM/deploy requerem decisão separada no [plano VM](vm-develop-plan.md).

## Composição e experiência

- Usar a largura disponível ao workspace, em pixels lógicos: compacta abaixo de 860, ampla a partir de 860. Testar 859/860/861 com a mesma medida usada pelo seletor e pelas ações. Nunca selecionar por sistema operacional.
- Compacta: camadas → documentos → editor, com voltar preservando seleção/texto. Ampla: painéis simultâneos e componentes comuns. Estado e controllers de texto ficam acima da troca de composição, vinculados ao ID do documento.
- Evoluir `lib/ui/` incrementalmente; árvore/panes compartilhados em widgets próprios e layouts em arquivos separados. Preservar `auth/`, `brain/` e contratos. Material3 atual, safe area, conteúdo rolável, ações em menus e tratamento de teclado/texto ampliado.
- Documento pertence a uma camada; lista direta ou subárvore preserva origem, sem cópia. Lixeira é filtro próprio. Criar, abrir, editar, renomear, mover, excluir e restaurar documentos; não alterar lifecycle das camadas.
- Editor fonte Markdown + prévia. Corpo vazio permitido; título normalizado de 1 a 255 codepoints, sem controles; Markdown UTF-8 até 1 MiB, sem NUL. Não permitir editar corpo ainda não carregado. Mudanças durante request/resize não podem ser substituídas por resposta antiga.
- Mostrar gravação local pendente, commit local confirmado, pendência de envio, sync, conflito e falha. “Salvo localmente” somente após commit; “sincronizado” somente após confirmação do estado remoto desejado.
- Digitar `[[` oferece escolha explícita entre destinos conhecidos por título/origem. Gravar `[[doc:UUID]]`; manter ID por rename/move e projetar nome atual na prévia. Não escolher primeiro homônimo nem criar destino automático. Destinos remotos fora do cache podem ser abertos por GET autorizado. Links inexistentes/na lixeira são sinalizados.
- Código/escapes não criam links navegáveis. Prévia sem inline HTML e imagens/anexos; protocolos externos permitidos somente http/https. Links internos permanecem no app.

## Persistência, conta e lixeira

UUIDv4 seguro local. Revisão local e versão remota BigInt/Int64 são dimensões diferentes. Native storage: snapshot JSON no ApplicationSupportDirectory, namespace reversível por ambiente/origem/usuário, operações serializadas e commit por arquivo temporário, flush e rename. Falha de leitura/corrupção bloqueia escrita destrutiva; não apagar nem substituir snapshot ilegível como fallback. Temporários falhos são limpos sem tocar no arquivo canônico. Tokens/PKCE continuam em secure storage separado.

Autosave nominal periódico de 1 segundo enquanto houver execução, com flush ao trocar de documento, pausar, sincronizar e sair. Falha conserva texto em memória e informa que não houve commit. Reabrir recupera commits por conta; respostas/callbacks de A não alteram B. Não prometer salvar após crash nem app inteiramente offline: Auth/camadas iniciais ainda podem exigir rede.

Lixeira local: 240 horas, clock do dispositivo. Repetir exclusão não renova prazo/revisão. Restore no instante ou depois do vencimento é rejeitado; restore de ativo é no-op visível. Purge oportunista elimina corpo, título, links e cópias textuais do intent, conservando somente identidade mínima quando necessária à reconciliação. Lixeira remota usa o clock PostgreSQL. Não prometer job com app suspenso, apagamento físico exato ou remoção em backups.

## Sincronização autorizada

Timer de 30 minutos durante execução e botão manual usam o mesmo single-flight. Retomada verifica vencimento; sync manual não pode atrasar a próxima tentativa por quase outro intervalo. Atualizar metadata da visualização atual sem sobrescrever dirty/conflitos; versão remota nova invalida corpo limpo antigo antes de editar. Não prometer réplica global ou protocolo multi-dispositivos completo.

Usar somente os seis contratos documentais existentes:

1. Flush local e persistência de intent antes de cada mutação HTTP. Registrar tipo, snapshot, revisão e versão esperada, sem credenciais.
2. Criar novos IDs primeiro com título/camada e corpo vazio; depois enviar corpo/vínculos quando os IDs dos destinos estiverem confirmados. Essa fase provisória permite ciclos sem endpoint novo e nunca apaga o rascunho ou conta como conteúdo final sincronizado.
3. Aplicar snapshots completos, trash e restore com CAS, encadeando versões confirmadas. Após resposta, confirmar somente a revisão enviada; edição posterior continua dirty.
4. Timeout/5xx/interrupção não autorizam retry cego. Reconciliar por GET; create só adota placeholder correspondente na versão 1. Create pendente + GET 404 libera tentativa controlada, preservando corpo; colisão 409 não entra em loop. Divergência exige escolha explícita e não autoriza overwrite automático.
5. Conflitos/missing de update/trash/restore preservam rascunho e são acionáveis; nunca ressuscitar por edição antiga. Referência para ID ainda não confirmado pode impedir envio e deve ser sinalizada.
6. Sync parcial nunca recebe sucesso global. Intents/revisões persistidos permitem retomar após restart; falha de storage impede enviar sem registro durável.

Logout tenta sync com credencial ativa e orçamento finito, depois flush local. Falha/pendência permite cancelar ou sair conscientemente conservando commits; não garantir upload ao fechar app. Depois da saída, callbacks antigos não aparecem na conta nova.

## Interfaces e implementação

`lib/documents/` contém modelos imutáveis, `DocumentRepository`, `DocumentHttpRepository`, `DocumentLocalStore`, `FileDocumentLocalStore` e `DocumentsController`. O controller recebe AuthGateway, repository, localStore, relógio/gerador de IDs injetáveis e callback para descendentes da árvore conhecida. Expõe estado, documentos conhecidos, operações documentais, flush/sync, resolução de conflitos, prepareLogout e onResume.

`MindforgeApp` inicializa/descarta o controller e encaminha lifecycle; `main` compõe adapters/repositórios/controllers. `WorkspaceScreen` aceita controller documental opcional para preservar consumidores antigos/Web. UI não acessa HTTP ou arquivos diretamente.

Dependências registradas: `path_provider 2.1.6` (já transitiva), `flutter_markdown_plus 1.0.12` e `markdown 7.3.1` para prévia/links. Nenhum framework, gerenciamento de estado ou serviço externo novo. Fontes primárias dos pacotes estão no discovery.

## Android e transporte privado

Scaffold Kotlin gerado pelo Flutter3.38.7; applicationId `app.mindforge.client`, minSDK24, compile/target36, Java17 e NDK existente28.2.13676358. Scheme `mindforge-dev://auth/callback`, app_links cold/warm deduplicado, Flutter deeplinking desabilitado, INTERNET e allowBackup=false. Cleartext negado por padrão, liberado somente para127.0.0.1 no laboratório; sem segredo Google no app.

Aparelho11000/11001 → ADB reverse → Mac11000/11001 → túnelSSH → Auth/API develop existentes. Callback Google loopback atual permanece. Não configurar Google Cloud, expor banco ou criar daemon na VM por inferência. [Guia de teste](android-test-guide.md) contém comandos e aceite manual. Não iniciar emulador, instalar APK ou executar screenshots/browser QA automaticamente.

## Execução, TDD e aceite

TDD=yes: RED/GREEN/refactor para domínio/store/sync, layouts e Android. Baseline54 testes; suíte final99 testes passou. Principal integrou spec/deps/main/app/platform e validação; agentes folha implementaram documentos, UI e shell Android, com ownership registrada no checkpoint. Não houve reviewer obrigatório.

Aceite local: funcionalidades existentes preservadas; navegação compacta/resize/rascunho; CRUD/lixeira/links; sync manual/30min sem sobreposição; falha/conflito/troca de conta sem perda silenciosa; Int64 preservado; commit antes do status de salvo. Testes widget/unitários, analyze, builds APK/Web e inspeção de APK são níveis distintos de prova. Não substituem Google/aparelho, SQL, HTTP real ou QA visual.

## Aprovações, rollback e retomada

Código, testes e builds locais autorizados em2026-10-13. Backend endpoints continuam approved/implemented/runtime pending; consumer Android será registrado com evidência local. VM/SQL/deploy dependem da resposta ao plano específico; instalação do APK, publicação e commit/push não estão autorizados. Rollback preserva alterações anteriores e retira somente este lote; não DROP ou restore automático.

Estado, resultados e próxima ação: [checkpoint](android-workspace-checkpoint.yaml) e [evidência](implementation-evidence.md). Riscos restantes: backend documental ainda não aplicado no momento do build, Google/segurança nativa não comprovados no aparelho, ausência de execução garantida em background e fase provisória do create. Mudança material de comportamento/segurança/autoridade volta ao owner.
