# Regra de layouts adaptativos do client Flutter

Status: regra aprovada pelo owner em 2026-10-13. Escopo: organização e comportamento da apresentação no mesmo projeto `src/mindforge-client/`. Fonte de aprovação: pedido explícito nesta conversa para registrar o padrão pesquisado para os agentes seguirem. Não autoriza implementação por si só.

## Formato a seguir

O client compartilha estado/controllers, dados, ações e widgets reutilizáveis. Uma composição adaptativa escolhe como apresentá-los conforme o espaço disponível. Para a frente atual, desenhar primeiro a experiência compacta e de toque no Android; refinamentos para PC/telas amplas ficam posteriores, preservando a capacidade de adaptação.

1. Medir o espaço do componente com `LayoutBuilder`; usar `MediaQuery.sizeOf` quando a decisão pertence à janela inteira. A escolha é por dimensões lógicas disponíveis, não pelo sistema operacional ou orientação isolada. Comportamentos nativos específicos continuam em adapters/componentes próprios, sem se tornarem seletores de tamanho.
2. Centralizar a mudança global de composição na tela/shell adaptativo. Um componente pode medir seu próprio espaço quando houver necessidade local; evitar regras globais repetidas e divergentes em vários widgets.
3. Usar condicionais pequenas para espaçamento, densidade e visibilidade simples. Quando muda a estrutura ou a navegação, extrair composições compacta/ampla em widgets e arquivos distintos, recebendo o mesmo estado e callbacks. Não replicar acesso à API, regras ou controller para cada variante.
4. Compartilhar árvore, lista, editor e outros componentes quando o comportamento for comum. Separar somente diferenças justificadas; um arquivo por widget não é obrigação. Android e iOS podem coexistir na mesma composição, com adaptações específicas quando necessárias.
5. Manter progresso do usuário fora da subárvore que será substituída: seleção, documento aberto, rascunho/estado de edição e navegação devem sobreviver à troca de layout. Preservar/restaurar leitura/scroll e foco conforme o fluxo. Não recriar controllers ou disparar carregamentos somente por resize/build.
6. Definir breakpoints na spec da tela pela necessidade do conteúdo. Não tratar600,840 ou o860 atual como valor universal. Usar classe intermediária apenas quando existir um terceiro comportamento real; não criar composições vazias antecipadamente.

Exemplo de comportamento para a futura spec documental: compacto navega entre camadas, documentos e editor; amplo pode mostrar painéis simultâneos. Isso ilustra o padrão, sem congelar o desenho/navegação de uma spec ainda não aprovada.

## Organização incremental dos arquivos

Preservar as separações existentes `lib/auth/`, `lib/brain/` e `lib/ui/`. Dentro da área de UI de uma funcionalidade, extrair `layouts/` e `widgets/` conforme crescerem. A tela conecta estado/eventos; os layouts compõem a apresentação; os widgets implementam partes reutilizáveis.

Nomes como `workspace_compact_layout.dart` e `workspace_expanded_layout.dart` são convenção local aceitável. Não são uma arquitetura formal exigida pelo Flutter, nem motivo para mover todo o projeto. Um novo módulo de documentos deve nascer do trabalho documental autorizado; não é pré-requisito para reorganizar layouts.

Não substituir gerenciamento de estado, introduzir dependências, duplicar o app em pastas Android/iOS/desktop ou criar a estrutura-alvo inteira por causa desta regra. A autoridade remota de regras/permissões/persistência continua no Brain Engine.

## Evidência oficial e limites

Pesquisa realizada por agente folha `adaptive_layout_research` (`gpt-5.6-luna`, esforço `xhigh`) e fontes decisivas conferidas pelo principal nesta conversa. Fontes consultadas em 2026-10-13:

| Fonte primária | O que sustenta |
|---|---|
| [Flutter — General approach to adaptive apps](https://docs.flutter.dev/ui/adaptive-responsive/general) | Compartilhar informação, medir espaço e escolher a composição; `LayoutBuilder` para constraints locais e `MediaQuery.sizeOf` para janela. |
| [Flutter — tutorial de layout adaptativo](https://docs.flutter.dev/learn/pathway/tutorial/adaptive-layout) | Exemplo concreto de navegação em janela estreita e sidebar/detalhe em janela ampla. |
| [Flutter — boas práticas adaptativas](https://docs.flutter.dev/ui/adaptive-responsive/best-practices) | Componentes menores/reutilizáveis, foco inicial em toque e preservação/restauração de estado ao mudar a janela. |
| [Flutter — guia de arquitetura](https://docs.flutter.dev/app-architecture/guide) | Separação entre apresentação, estado e acesso a dados; o documento orienta princípios, sem impor migração do Mindforge para outra arquitetura. |
| [Android/Compose — canonical layouts](https://developer.android.com/develop/adaptive-apps/guides/canonical-layouts) | Padrão equivalente lista/detalhe: painéis simultâneos ou navegação conforme espaço, com estado compartilhado/preservado. É comparação, não dependência do client Flutter. |

As fontes sustentam o padrão de engenharia, não a árvore exata de pastas proposta. Nomes/extrações e prioridade Android são decisões locais do owner. Não há benchmark demonstrando que dois arquivos ou duas árvores serão sempre mais rápidos; desempenho deve ser medido quando houver implementação. Nenhuma tela foi implementada ou validada visualmente por este registro.

## Validação em futuras entregas autorizadas

Manter TDD conforme a decisão da entrega. Testar seletores/composições abaixo, no limite e acima dos breakpoints; transições nos dois sentidos; preservação de seleção/rascunho/navegação; sem carregamentos/efeitos duplicados. Cobrir teclado aberto, texto ampliado, toque, foco/semântica e entradas de mouse/teclado quando o respectivo layout entrar no escopo. Estado preservado em resize e recuperação após encerramento são provas distintas.

Testes widget/unitários não provam a experiência em Android real. Runtime/dispositivo e validação visual seguem a autoridade específica da entrega; esta regra não inicia emulador, browser QA, screenshots, instalação, VM ou deploy. Testes de UI futuros não mudam estados de runtime dos endpoints por inferência.
