---
schema_version: 1
managed_by: spec-drive
feature: brain
task: android-workspace
revision: 1
status: implemented_local_device_pending
---

# Correção AND-001 e AND-002 no client

## Contexto e fontes

Pedido do owner em2026-10-14: corrigir os dois erros registrados, usando spec-drive. [Discovery](discovery.md), [registro](known-issues.md), [spec anterior](spec-01-main.md) e regra de layouts adaptativos continuam referências. Checkout misto `feat/brain-core-foundation` preservado. Esta spec delimita o reparo solicitado; não redesenha o produto.

## Objetivos e exclusões

Eliminar overflow na lista vazia ao abrir teclado/formulário. Confirmar e fechar formulário de criação/rename sem controller descartado em uso ou assertion, abrindo o documento editável e preservando autosave local. Cobrir cancelamento e submissão repetida. Manter contratos, dados, namespace, sync30min/manual e Auth API-only.

Fora: VM/SQL/deploy, limpeza do Cérebro órfão, endpoints, pacotes/SDK novos, documentos Web, instalação/aparelho/QA visual, commit/push e refatoração geral.

## Caminho e execução

1. Acrescentar testes widget para teclado/insets em tela compacta e confirmação/fechamento animado. Registrar falhas reais antes de alterar implementação.
2. Ajustar somente componentes responsáveis pela altura e lifecycle dos formulários. Controllers do formulário devem viver até a desmontagem da rota; evitar delays arbitrários. Proteger confirmação duplicada enquanto operação está em andamento e não criar novamente após sucesso.
3. Regressões: cancelamento sem documento, título inválido preserva formulário, rename, edição/autosave e reabertura do documento; preservar navegação e layouts existentes.
4. Rodar suíte Flutter, analyze e build APK debug com configuração API-only develop atual, sem upgrade de dependências. Registrar hash do artefato e limites da prova.

Arquivos: `lib/ui/workspace_screen.dart`, widgets documentais necessários, `test/widget/adaptive_workspace_documents_test.dart`, documentação desta tarefa. Principal é responsável pelo lote; sem reviewer ou delegação automática.

## Compatibilidade, falhas e rollback

Estado/editor e documentos continuam controlados pelos controllers existentes. Erro de criação permanece visível e não descarta formulário. Nenhuma alteração de formato persistido ou transporte. Rollback limitado a este reparo, preservando todo o trabalho anterior e dados locais/remotos.

## Aceite e validação

- [x] Lista vazia com teclado em viewport compacta não gera overflow e continua rolável/acessível.
- [x] Confirmar criação, via botão ou teclado, fecha modal sem exceção durante animação e abre um único documento editável.
- [x] Cancelamento e rename não deixam controller descartado enquanto TextField permanece montado.
- [x] Edição/autosave local e reabertura preservam ID/texto; nenhuma criação HTTP adicional no fluxo local.
- [x] Testes existentes, analyze e APK debug passam.

TDD: **yes**, mantido da entrega aprovada; RED → GREEN → regressões. Testes widget usam insets/tamanhos determinísticos, sem browser/device. Mock/store em memória para fluxo UI não representam prova de disco/servidor; testes existentes do store continuam cobrindo persistência. APK local não comprova desaparecimento do bug no aparelho do owner.

## Automação e autoridade

Usar testes e comandos Flutter existentes; não criar script operacional novo para duas correções locais. Pedido direto de correção autoriza implementação delimitada; escolhas técnicas de lifecycle/scroll não requerem nova decisão de produto. TDD e build local mantidos do escopo Android anterior. Operações externas e instalação continuam separadas. Endpoints não mudam; seus estados permanecem intactos.

## Aprovação e retomada

Autorização: pedido explícito do owner nesta conversa em2026-10-14; execução deste reparo local autorizada. Sem mudança de contrato. Checkpoint: `ui-fixes-checkpoint.yaml`; resultados: `ui-fixes-evidence.md`. Parar somente se surgir alteração material de contrato/persistência/autoridade. Próxima etapa: reproduzir antes de corrigir.

## Resultado da execução

125 testes, analyze e APK debug PASS; assinatura v2 verificada. [Evidência](ui-fixes-evidence.md) registra diagnóstico, arquivos, hash e limites. Correção local concluída, aceite no aparelho pendente.
