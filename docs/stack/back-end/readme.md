# Back-end

## Stack escolhida

- **Brain Engine:** Go
- **Hidden Cortex:** Go
- **Executores especializados futuros:** Rust, somente quando houver necessidade comprovada de processamento pesado

## Por que Go

- Oferece bom equilíbrio entre desempenho, concorrência, simplicidade operacional e consumo de recursos em Linux/VPS.
- Atende APIs, streaming, jobs, eventos, automações e chamadas a serviços externos sem exigir duas stacks desde o início.
- Permite manter Brain Engine e Hidden Cortex na mesma linguagem, reduzindo o custo inicial de desenvolvimento e operação.

## Limites arquiteturais

- Brain Engine e Hidden Cortex serão processos independentes, mesmo usando a mesma linguagem.
- O Brain Engine permanece como autoridade sobre regras, permissões, dados, capabilities e estado durável dos jobs.
- O Hidden Cortex não acessa os dados autoritativos diretamente; ele consome jobs e solicita efeitos pelas capabilities do Brain Engine.
- Processamento server-side de modelos não é foco inicial.
- Rust fica reservado para futuros workers ou bibliotecas de indexação, grafo, sincronização, inferência local ou outros trabalhos computacionalmente pesados.

Esta é a decisão atual e pode ser revista caso necessidades reais de carga ou processamento justifiquem outra composição.

## Feature contracts

- A infraestrutura operacional da feature infra fica em `src/brain-engine/features/infra/`; seus documentos de contrato, plano e checkpoint ficam em `docs/stack/back-end/features/infra/`.
- `infra`: [contract revision 2](features/infra/contract.md), approved by the owner; the active executable plan and continuation state are recorded in [plan.md](features/infra/plan.md) and [checkpoint.yaml](features/infra/checkpoint.yaml).

The `infra` contract prepares a reduced reproducible PostgreSQL, Auth/GoTrue and `pgvector` package with persistence, infrastructure migrations, backup and isolated restore, without defining product tables, Mindforge HTTP endpoints, or implementing the Brain Engine. Static/unit delivery evidence does not establish Docker runtime, test-VM acceptance, production or deployment.
