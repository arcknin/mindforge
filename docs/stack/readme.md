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
- `src/brain-engine/features/infra/`: ambiente reproduzível do Brain Engine, incluindo PostgreSQL, migrations, seeds, backup e restore.
- `src/mindforge-client/`: aplicação Flutter/Dart e Prefrontal Cortex. Pode executar tools locais explicitamente registradas, mas efeitos compartilhados ou autoritativos passam pelo Brain Engine.
- `contracts/openapi/`: fonte canônica dos contratos HTTP aprovados e registro separado de sua implementação e verificação.
- `docs/`: visão arquitetural e decisões. Exemplos conceituais não criam endpoints automaticamente.
- `docs/stack/`: tecnologias escolhidas, limites de cada componente e contratos de feature.

- `src/hidden-cortex/`: processo Go independente para o Hidden Cortex, previsto mas fora do primeiro recorte; seu diretório físico será criado apenas quando houver contrato aprovado para o trabalho em background.

## Estado atual

A primeira superfície de infraestrutura existe em `src/brain-engine/features/infra/`: definição reduzida de PostgreSQL, Supabase Auth/GoTrue, `pgvector`, migrations de infraestrutura e helpers operacionais. Os documentos da feature ficam em `docs/stack/back-end/features/infra/`. Ela é regida pelo contrato `infra` revision 2 aprovado. Isso não cria uma API do Mindforge, tabelas de domínio, Brain Engine executável ou client.

O pacote ainda exige validação separada de runtime descartável para cada critério funcional. A presença dos arquivos e os testes static/unit não provam Docker local, VM de teste, produção, rede, dados reais ou publicação. Flutter/Dart continua escolhido para o client; Go continua escolhido para Brain Engine e Hidden Cortex.

## Evolução do repositório

O monorepo não obriga versionamento único. Backend, client e contratos poderão possuir tags, releases e pipelines próprios. Se no futuro houver necessidade operacional, um componente poderá ser extraído para outro repositório preservando seu histórico.
