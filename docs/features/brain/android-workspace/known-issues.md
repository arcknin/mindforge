# Erros reportados no Android

Registrados em2026-10-14 por relato e screenshots do owner. Registro inicial: owner pediu apenas registrar para corrigir depois, sem reprodução ou alteração naquela unidade. Em2026-10-14 autorizou corrigir; resultado local abaixo. Versão exata do APK instalado não conferida.

## AND-001 — Overflow com teclado no formulário de documento

- **Estado:** corrigido localmente com TDD; aceite no aparelho pendente. [Spec](spec-02-ui-fixes.md) / [evidência](ui-fixes-evidence.md).
- **Contexto observado:** composição compacta Android, camada sem documentos, modal “Novo documento” aberto, título “Teste” e teclado visível.
- **Sintoma:** faixa de erro no estado vazio da lista, atrás do modal: `BOTTOM OVERFLOWED BY 38 PIXELS`.
- **Impacto:** conteúdo da tela não cabe na altura disponível durante a entrada do título.
- **Evidência:** screenshot do owner `codex-clipboard-64c2d191-5be9-42d7-8d0c-3bba74c56e37.png`, enviado nesta conversa.
- **Retomada:** reproduzir com teclado/insets e altura compacta em teste de widget; localizar o widget responsável antes de definir a correção.

## AND-002 — Tela vermelha ao salvar documento

- **Estado:** corrigido localmente com TDD; aceite no aparelho pendente. [Spec](spec-02-ui-fixes.md) / [evidência](ui-fixes-evidence.md).
- **Gatilho relatado:** inicialmente observado ao confirmar a criação pelo formulário “Novo documento”. Owner esclareceu depois que acontece **sempre ao salvar, apenas nesse momento**; manter esse relato para reprodução, sem restringir o bug à criação inicial. A ação exata de salvamento ainda não foi instrumentada.
- **Sintoma:** tela inteira vermelha com assertion do Flutter:

```text
'package:flutter/src/widgets/framework.dart': Failed assertion:
line 6271 pos 12: '_dependents.isEmpty': is not true.
```

- **Impacto:** falha de interface interrompe o fluxo visível de criação, mas o documento não foi perdido no teste relatado. Owner confirmou em2026-10-14 que, ao sair e entrar novamente, o documento apareceu salvo e pôde ser escrito/editado. O relato não distingue persistência local de sincronização no servidor.
- **Sincronização:** depois de acionar “Sincronizar”, owner pediu conferência na VM. [Consulta somente de leitura](sync-check-20261014.md) em2026-10-14 confirmou `Teste` ativo, versão2, Markdown com20 bytes/20 caracteres no servidor, atualizado às15:17:16UTC (12:17:16 de Brasília). `Outro teste` também existe, versão2, com corpo vazio. Presença remota confirmada; igualdade com o rascunho local não foi comparada. Os bugs de interface continuam adiados.
- **Diagnóstico da retomada:** testes reproduziram TextEditingController descartado antes de terminar o fechamento animado de create/rename; corrigido com ownership no StatefulWidget do formulário. A assertion isolada do screenshot não permitia atribuir a falha ao backend/banco. Evidência e limites no relatório da correção.
- **Evidência:** screenshot do owner `codex-clipboard-7f643c98-3cad-460e-8980-e8f13f40b737.png`, enviado nesta conversa.
- **Retomada:** obter stack trace sanitizado e reproduzir a confirmação/fechamento do modal e abertura do documento em teste de widget com TDD. Preservar a criação já realizada; conferir o estado antes de repetir a operação. Incluir regressão para documento que foi salvo apesar da falha visual e continua editável após sair/entrar.

Prioridade de retomada: AND-002 primeiro, por bloquear o fluxo; AND-001 em seguida. Nenhuma mudança de contrato, migration, limpeza de dados ou deploy autorizada por este registro.

## Correção autorizada — resultado

AND-001: cabeçalho, filtros, lista e estado vazio compartilham scroll para caber com teclado. AND-002: controller do formulário vive até a desmontagem da rota; guardas bloqueiam dupla submissão.125 testes/analyze/APK PASS. Nenhuma mudança de dados, VM ou endpoints. Substituir o APK e confirmar no celular; não marcar aceite device a partir de testes widget. As observações de adiamento acima descrevem a situação antes da nova autorização.
