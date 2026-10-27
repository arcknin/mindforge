# Discovery — client Android com layouts adaptativos e documentos

Owner autorizou iniciar tarefa específica, implementar a arquitetura de layout, preservar tudo que o client já faz e acrescentar documentos para teste conjunto; TDD=yes. Em seguida ampliou explicitamente para rascunho local1s, sync automático30min e botão manual.

Fontes reconciliadas: AGENTS, arquitetura/stack geral/front/back, regra adaptive-layouts, registry/OpenAPI, specs layer-content01/02/checkpoint e código Flutter atual. Client Flutter3.38.7/Dart3.10.7 tem54 testes passando na base, Google-only GoTrue/PKCE, sessão segura nativa, árvore/create/rename/conflitos e Web. Não há módulo Android nem documentos no início. UI concentra apresentação em workspace_screen.dart e já usa breakpoint860; controller/repos separados. Config nativa exige loopback11000/11001 e namespace macOS. Android SDK34..36, buildtools até36.1 e NDK28.2 já presentes; Java17/21 disponíveis.

Backend documental tem6 contratos approved/implemented/runtime pending; migration0007 versionada, não aplicada. Não assumir DB/serviço publicado. Arquivos/volume main/NSR fora do escopo. Túnel Mac/ADB reverse permite teste privado Android com callbacks loopback existentes; não implica acesso direto pela LAN nem Google OAuth comprovado no aparelho.

Armazenamento Android não pode usar IndexedDB Web. Usar arquivos privados do app, snapshots JSON com commit atômico e namespace ambiente/origem/usuário; path_provider2.1.6 já está transitivamente no lock. Preview usará flutter_markdown_plus1.0.12 (Dart>=3.4), renderer sem inlineHTML; fontes primárias: https://pub.dev/packages/flutter_markdown_plus e https://pub.dev/documentation/path_provider/latest/path_provider/getApplicationSupportDirectory.html. Não renderizar imagens remotas/arquivos nem executar protocolos arbitrários.

Somente código/build/testes locais autorizados. Instalação no aparelho, screenshots/browser QA, VM/SQL/deploy/credenciais/commit/push permanecem separados. Android deve produzir APK privado se o toolchain existente permitir, sem upgrade Flutter ou instalações grandes implícitas.

## Retomada AND-001/AND-002 — 2026-10-14

Owner pediu corrigir os erros registrados atuando como spec-drive. Recorte: UI Flutter documental, testes e APK local; manter TDD=yes da entrega. Auth/API/SQL/VM/dados existentes fora da alteração. Branch `feat/brain-core-foundation`, checkout misto preservado. Sem AGENTS adicionais sob src.

Fontes decisivas: `known-issues.md`, checkpoint, spec01, `lib/ui/workspace_screen.dart` (`_showCreateDocumentDialog`, `_renameDocument`), `lib/ui/workspace/widgets/document_panes.dart` (`_EmptyDocuments`) e testes em `test/widget/adaptive_workspace_documents_test.dart`. Formulários descartam TextEditingController no finally do Future de showDialog (antes de desmontar a rota animada). Estado vazio usa Center/Padding/Column sem scroll e sem adaptação à altura. São hipóteses a reproduzir, não diagnóstico derivado somente dos screenshots.

Documento Teste já confirmado no servidor com conteúdo na checagem anterior; não recriar/remover dados para corrigir UI. Reproduzir em widgets as transições com teclado, confirmação, cancelamento e rename, incluindo autosave local e reabertura. Testes do framework não equivalem ao aceite no aparelho. Spec ativa desta retomada: `spec-02-ui-fixes.md`.

Reprodução TDD confirmada: viewport400x760/inset340 gerou overflow de53px; log diagnóstico local apontou Column de `WorkspaceDocumentListPane` com altura disponível123px. Na confirmação do formulário, o primeiro erro foi `A TextEditingController was used after being disposed` em workspace_screen.dart:520; cancelamento/rename também atingiram lifecycle inválido. Não atribuir a assertion `_dependents.isEmpty` isolada ao backend. Após correção,9 testes focais passam, incluindo banner de rename, scroll acessível, submissão por botão/teclado, erro/retry, título inválido, dupla submissão bloqueada, cancelamento, rename com corpo preservado e autosave/reabertura via store fake.
