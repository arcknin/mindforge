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
