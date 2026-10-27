# Front-end

## Stack escolhida

- **Framework:** Flutter
- **Linguagem:** Dart

## Por que Flutter

- Suporta oficialmente Android, iOS/iPadOS, macOS, Windows e Linux.
- Permite compartilhar a maior parte da interface e da lógica do client entre as plataformas.
- Já existe experiência prévia com Flutter, reduzindo curva de aprendizado e risco de entrega.
- Oferece integração com recursos nativos por plugins, platform channels e, quando realmente necessário, FFI.
- Adequa-se aos modos manual, assistido e conversacional previstos para o Mindforge.

## Limites arquiteturais

- Flutter será responsável pelo client e pelo **Prefrontal Cortex**, não pelo Brain Engine.
- Regras, permissões, persistência autoritativa e efeitos reais permanecem no Brain Engine.
- O Prefrontal Cortex utilizará tools tipadas; FFI será apenas um detalhe interno de implementações que realmente precisem de código nativo.
- Leitura completa de Markdown e o gráfico universal de relações são evoluções futuras e não condicionam o primeiro frontend.

Esta é a decisão atual e pode ser revista caso validações práticas revelem limitações relevantes.


## Regra atual de layout e prioridade do client

Owner aprovou em 2026-10-13 o padrão de [layouts adaptativos com estado e componentes compartilhados](adaptive-layouts.md). A próxima experiência será desenhada primeiro para celular/toque no Android e depois refinada para telas amplas, no mesmo projeto Flutter. A regra detalha seleção por espaço, composições compacta/ampla, preservação de estado, evolução incremental dos arquivos e fontes oficiais.

Agentes devem consultar essa regra antes de planejar ou alterar a UI. Não há adoção de novo framework/gerenciador de estado nem autorização de refatoração por este registro. As entregas abaixo são históricas e não substituem a prioridade atual; seus resultados precisam ser reconciliados com checkpoints ativos antes de afirmar estado de runtime.

## Histórico — entrega local macOS

`src/mindforge-client/` contém o primeiro client macOS manual: Google via GoTrue/PKCE, sessão no Keychain, árvore, criação e rename com conflitos. [Evidência](../../features/brain/manual-online/implementation-evidence.md) registra41 testes e análise limpa. O build nativo ficou limitado por `xcodebuild` ausente; login/callback/Keychain reais e integração com a API na VM seguem pendentes. Não há offline, LLM, acesso direto ao PostgreSQL ou login simulado.

## Histórico — continuação web

Owner substituiu a direção Android/macOS pelo client Flutter Web. [Web develop](../../features/brain/web-local/runtime-evidence.md) foi compilada e publicada em Docker na VM, em `http://web-mind.home.arpa`, com Auth/API na mesma origem e callback Google existente pelo túnel Mac11000.54 testes Flutter, análise/build e HTTP real passaram; login no navegador e operações autenticadas continuam pendentes. Xcode não condiciona esta entrega. Main/NSR/Studios preservados; nenhuma migration ou promoção executada.

## Entrega Android atual — 2026-10-13

A [tarefa Android workspace](../../features/brain/android-workspace/spec-01-main.md) implementa layouts adaptativos, preserva Google/camadas e acrescenta documentos Markdown com rascunho local1s e sync30min/manual.99 testes, analyze, APK e regressão de build Web passaram. A [evidência](../../features/brain/android-workspace/implementation-evidence.md) separa fonte/build, SQL/deploy real e aceite no aparelho. Backend develop atualizado com0007; Google/CRUD/sync autenticados no Android e validação visual continuam com o owner. Nenhum framework/gerenciamento de estado substituído.

## Fronteira Auth API-only — implementação local

PD-022 e a [spec API Auth](../../features/brain/api-auth/spec-01-main.md) agora têm implementação local com TDD: client usa somente a API para login/sessão também,117 testes e analyze passaram, APK/Web reconstruídos. Credenciais Auth usam namespace novo; dados documentais mantêm o namespace anterior. Web/API/Auth develop aplicados com22 checks operacionais; login autenticado aguarda URI Google/owner. Prova real detalhada no [checkpoint](../../features/brain/api-auth/api-auth-checkpoint.yaml); não reutilizar comandos Auth11000 dos guias históricos.
