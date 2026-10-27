# Estrutura do projeto

O Mindforge será mantido inicialmente como um monorepo: existe um único repositório Git na raiz, enquanto client, backend e infraestrutura continuam com builds, processos e ciclos de entrega independentes.

## Estrutura-alvo

```text
mindforge/
├── .git/                                  # Git único, quando a raiz for inicializada
├── README.md                              # visão operacional futura do projeto
├── AGENTS.md                              # direção obrigatória para orquestradores
├── docs/                                  # arquitetura, decisões e stack
│   └── stack/
│       ├── readme.md
│       ├── front-end/readme.md
│       └── back-end/
│           ├── readme.md
│           └── features/infra/            # contrato e estado da feature infra
├── contracts/
│   └── openapi/                           # contratos e estado dos endpoints
├── src/
│   ├── brain-engine/
│   │   └── features/infra/                # infraestrutura operacional inteira
│   └── mindforge-client/                  # src/mindforge-client/, criado somente quando autorizado
│       ├── lib/
│       ├── test/
│       └── pubspec.yaml
```

Os diretórios de código são estrutura-alvo e devem ser criados somente quando o trabalho correspondente começar. Não criar repositórios Git dentro de `src/brain-engine/` ou `src/mindforge-client/`.

## Responsabilidades

- `src/brain-engine/`: serviço Go autoritativo sobre autenticação, autorização, regras, validação, persistência, capabilities e efeitos reais.
- `src/brain-engine/features/infra/`: ambiente VM-only do Brain Engine, incluindo um cluster PostgreSQL persistente com duas bases, dois Auth/GoTrue, migrations, seeds, backup, restore e o operador Go distribuível.
- `src/mindforge-client/`: aplicação Flutter/Dart e Prefrontal Cortex. Pode executar tools locais explicitamente registradas, mas efeitos compartilhados ou autoritativos passam pelo Brain Engine.
- `contracts/openapi/`: fonte canônica dos contratos HTTP aprovados e registro separado de sua implementação e verificação.
- `docs/`: visão arquitetural e decisões. Exemplos conceituais não criam endpoints automaticamente.
- `docs/stack/`: tecnologias escolhidas, limites de cada componente e contratos de feature.

- `src/hidden-cortex/`: processo Go independente para o Hidden Cortex, previsto mas fora do primeiro recorte; seu diretório físico será criado apenas quando houver contrato aprovado para o trabalho em background.

## Estado atual

Infra r6 preserva main físico `mindforge` e develop `mindforge_develop`, com Auth/GoTrue separados no cluster persistente. A unidade [develop-studio](../features/infra/develop-studio/runtime-evidence.md) registra implantação e verificações reais de develop/Brain0006 e portal administrativo; main continua sem promoção0006.

Brain Core r4 implementa bootstrap privado, hierarquia, criação e rename com conflito. A nova [entrega manual-online](../features/brain/manual-online/implementation-evidence.md) acrescenta localmente `auth/`, `api/` e `cmd/brain-engine/`, quatro contratos OpenAPI aprovados e client Flutter macOS com Google-only, Keychain e árvore manual. API usa o Core existente e não aplica migrations no startup.

Testes Go/Flutter, análise e pacote Linux passaram. Auth Google/API develop e túnel estão aplicados conforme [evidência de serviços](../features/brain/manual-online/runtime-service-evidence.md); login Google/troca de código e operações autenticadas ainda não foram comprovados. O build Flutter macOS depende de `xcodebuild`, ausente no ambiente ativo. Contrato aprovado e implementação local não representam deploy/runtime verificado. Setup Google, toolchain nativo e integração B04 são a próxima fronteira; nenhum Hidden Cortex ou fluxo agêntico foi criado.

## Evolução do repositório

O monorepo não obriga versionamento único. Backend, client e contratos poderão possuir tags, releases e pipelines próprios. Se no futuro houver necessidade operacional, um componente poderá ser extraído para outro repositório preservando seu histórico.

## Continuação web atual

Owner substituiu a direção Android/macOS pelo client Flutter Web. [Web develop](../features/brain/web-local/runtime-evidence.md) foi compilada e publicada em Docker na VM, em `http://web-mind.home.arpa`, com Auth/API na mesma origem e callback Google existente pelo túnel Mac11000.54 testes Flutter, análise/build e HTTP real passaram; login no navegador e operações autenticadas continuam pendentes. Xcode não condiciona esta entrega. Main/NSR/Studios preservados; nenhuma migration ou promoção executada.

## Continuação API-only Auth — 2026-10-14

A [tarefa api-auth](../features/brain/api-auth/api-auth-checkpoint.yaml) implementou com TDD login/sessão exclusivamente pelo Brain Engine e aplicou API/Auth/Web develop. Client mantém SDK interno, com endereço API único11001; GoTrue não é endpoint do app.117 testes Flutter/analyze, Go test/race/vet/build e artefatos passaram;22 checks reais e47 containers protegidos preservados. Google Console/aceite autenticado ficam com o owner. Consulte o guia atual antes de reutilizar o transporte Auth11000 das entregas históricas.
