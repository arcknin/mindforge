# AND-001/AND-002 — evidência de correção local

Data:2026-10-14. Autorização: pedido explícito do owner para corrigir os erros registrados, usando spec-drive; TDD=yes mantido. [Spec](spec-02-ui-fixes.md), [discovery](discovery.md), [checkpoint](ui-fixes-checkpoint.yaml).

## Reprodução e correção

**AND-001:** teste widget400x760, teclado/inset inferior340 e banner “A camada foi renomeada.”. Antes da correção, overflow. O diagnóstico focal posterior localizou Column de WorkspaceDocumentListPane com altura123px e overflow53px (o screenshot anterior reportava38px; dimensões/fonte variam). Corrigir somente o estado vazio não bastava: cabeçalho/filtros também consumiam altura fixa. Lista agora usa um CustomScrollView com cabeçalho em SliverToBoxAdapter, itens em SliverList e estado vazio em SliverFillRemaining(hasScrollBody:false). Conteúdo rola quando não cabe, sem reduzir texto ou esconder controles. Teste confirma botão do estado vazio alcançável por scroll com teclado.

**AND-002:** teste de confirmação/fechamento animado reproduziu `A TextEditingController was used after being disposed` no TextField documental original em workspace_screen.dart:520. Create e rename descartavam o controller no finally de showDialog; esse Future termina no pop, antes de desmontar a rota que ainda anima/reconstrói o campo. Novo WorkspaceDocumentTitleDialog é StatefulWidget e possui/descarta seu controller no próprio dispose. Submissão e fechamento têm guardas; back/cancelamento ficam bloqueados enquanto uma operação está em andamento. Parent abre o editor após sucesso sem reaplicar criação. Cancelar e rename compartilham lifecycle corrigido. Sem delays, supressão de exceptions ou upgrade de Flutter.

O screenshot do owner contém `_dependents.isEmpty`. O erro primário reproduzido foi o controller descartado em uso, com cascata de assertions; a igualdade exata daquela assertion não foi usada como único critério de prova. A confirmação de desaparecimento da tela vermelha no aparelho continua pendente.

## TDD e validação executada

| Validação | Resultado |
| --- | --- |
| Novos testes antes da implementação | RED: overflow e TextEditingController descartado em uso |
| Testes focais de documentos |9 PASS (8 novos + regressão existente) |
| `flutter test --no-pub --reporter expanded` |125 PASS |
| `flutter analyze --no-pub` |PASS, sem issues |
| `flutter build apk --debug --no-pub --dart-define=MINDFORGE_ENV=develop --dart-define=MINDFORGE_API_URL=http://127.0.0.1:11001` |PASS |
| `apksigner verify --verbose` |PASS, assinatura v2 |
| `aapt dump badging` |app.mindforge.client, version0.1.0/code1, minSDK24/target36 |
| `git diff --check` |PASS (arquivos tracked; lote Flutter continua untracked no checkout herdado) |

Testes focais: teclado com banner e scroll; criação por botão e teclado; cancelamento; rename preservando corpo; título vazio/correção; falha preservando título para retry; submissão repetida bloqueada durante operação e saída; autosave1s e reabertura do mesmo ID/texto via store fake; navegação/resize/rascunho anteriores. A fixture foi encerrada dentro do teste para cancelar timers antes das invariantes do binding. Não confundir store fake com nova prova de disco ou servidor; store real e sync têm regressões próprias na suíte existente.

Logs locais temporários: `/private/tmp/mindforge-ui-fixes-green.log`, `mindforge-ui-fixes-suite.log`, `mindforge-ui-fixes-analyze.log`, `mindforge-ui-fixes-apk.log`. São apoio desta execução, sem garantia de retenção. Resultados acima registrados para continuidade.

## Artefato atual

- APK: `src/mindforge-client/build/app/outputs/flutter-apk/app-debug.apk`
- Bytes:176749955
- SHA256:`ede5314434a57855e0ef26821a24bc196c71bb724f8c9c263de6a6a81b37f911`
- Build privado/debug; configuração develop/API-only11001. Este APK substitui o hash anterior do mesmo path. Namespace documental, assinatura/appID e configuração de transporte preservados.

Atualizar pelo Mac com aparelho conectado:

```sh
adb install -r /Users/arcknin/Desktop/mindforge/src/mindforge-client/build/app/outputs/flutter-apk/app-debug.apk
```

Aceite no aparelho: abrir camada com lista vazia, abrir “Novo documento” com teclado, criar/salvar e editar; voltar, renomear, cancelar e reabrir. Conferir ausência de faixa overflow/tela vermelha e preservação de texto. Usar o transporte API-only já configurado; esta tarefa não verificou nem recriou túnel/ADB.

## Arquivos e limites

Fonte alterada: `lib/ui/workspace_screen.dart`, `lib/ui/workspace/widgets/document_panes.dart`, novo `document_title_dialog.dart`; testes em `test/widget/adaptive_workspace_documents_test.dart`. Documentação da retomada fica nesta tarefa. Checkout misto preservado, sem commit/push.

Nenhuma alteração no backend, Auth, contratos, storage/schema, dados locais/remotos ou VM. Nenhum endpoint adicionado/modificado; estados do registry mantidos. Não houve instalação, emulador, QA visual, screenshot ou HTTP/SQL nesta correção. Runtime/device aceito somente após teste do owner com o APK atualizado.
