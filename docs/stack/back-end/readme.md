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

- `infra`: [contract revision6](features/infra/contract.md), governando cluster persistente, targets explícitos `mindforge_develop`/`mindforge`, Auth isolado, migrations/ledger e operações protegidas. [develop-studio](../../features/infra/develop-studio/runtime-evidence.md) registra a extensão administrativa e a prova aplicada.
- Brain Core: [contract r4](../../features/brain/brain-core/brain-core-contract.md), com domínio determinístico/persistência independente de transporte.
- Manual online: [spec integrada](../../features/brain/manual-online/spec-01-main.md), [OpenAPI](../../../contracts/openapi/brain-core.openapi.yaml) e [evidência local](../../features/brain/manual-online/implementation-evidence.md). Auth/API/composição Go implementados localmente; quatro endpoints com contrato aprovado, implementação concluída e runtime pendente.

O servidor inicial é restrito a develop e reutiliza o Engine/PostgresStore. O verificador consulta o GoTrue interno, não aceita identidade informada livremente pelo client e não executa migration ao iniciar. Auth Google/API develop foram aplicados por operação VM autorizada; [evidência](../../features/brain/manual-online/runtime-service-evidence.md) separa configuração/redirect/rejeição real de login e operações autenticadas ainda pendentes. Local fake-backed/race/build não substituem Google/Auth/PostgreSQL/VM reais, promoção a main ou publicação.

## Continuação web atual

Owner substituiu a direção Android/macOS pelo client Flutter Web. [Web develop](../../features/brain/web-local/runtime-evidence.md) foi compilada e publicada em Docker na VM, em `http://web-mind.home.arpa`, com Auth/API na mesma origem e callback Google existente pelo túnel Mac11000.54 testes Flutter, análise/build e HTTP real passaram; login no navegador e operações autenticadas continuam pendentes. Xcode não condiciona esta entrega. Main/NSR/Studios preservados; nenhuma migration ou promoção executada.

## Fachada Auth API-only — implementação local

PD-022 e a [spec API Auth](../../features/brain/api-auth/spec-01-main.md) foram aprovados e implementados localmente com TDD. Cinco rotas em [auth.openapi.yaml](../../../contracts/openapi/auth.openapi.yaml); GoTrue interno, DTO mínimo, PKCE e logout local, sem JWT próprio ou novo schema. Test/race/vet/build passaram; corte develop aplicado nesta tarefa com22 checks e47 containers protegidos preservados; runtime autenticado possui limites próprios no [checkpoint](../../features/brain/api-auth/api-auth-checkpoint.yaml).
