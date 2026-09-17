# Contratos HTTP do Brain Engine

Este diretório será a fonte canônica dos endpoints HTTP aprovados do Mindforge. Ele não transforma capabilities conceituais em HTTP e não autoriza implementação por si só.

## Arquivos

- `registry.yaml`: índice operacional de propostas e endpoints, com estados e evidências independentes.
- `*.openapi.yaml`: contratos executáveis criados somente quando existir ao menos um endpoint aprovado.

Nenhum endpoint está aprovado ou implementado neste momento. Por isso ainda não existe um documento OpenAPI com `paths`.

Uma proposta pode existir somente em `registry.yaml`. Ela só entra em `paths` de um documento OpenAPI depois da aprovação do owner.

## Fluxo

```text
necessidade identificada
→ proposed
→ decisão do owner
→ contrato OpenAPI approved
→ implementação e validação local
→ implementation_status: implemented
→ chamada em ambiente identificado
→ runtime_status: verified
```

Cada proposta deve registrar, quando aplicável:

- owner e capability;
- método e path;
- autenticação e autorização;
- request, response e códigos de erro;
- idempotência e efeitos;
- produtor e consumidores;
- evidências de implementação e runtime.

## Estados

Cada entrada usa:

- `lifecycle`: resumo derivado como `proposed`, `approved`, `implemented` ou `verified`;
- `contract_status`: `proposed` ou `approved`;
- `implementation_status`: `pending` ou `implemented`;
- `runtime_status`: `pending` ou `verified`;
- `gate`: `null` ou `needs_decision`.

`needs_decision` é um bloqueio, não um quinto estado do lifecycle. O resumo nunca pode esconder as dimensões: contrato aprovado não prova implementação, e implementação não prova runtime.

Campos ainda não decididos devem permanecer explícitos como pendentes. Não usar exemplos de `docs/architecture-overview.md` como contrato e não marcar `verified` com base apenas em documentação, mocks, testes unitários ou build.

## Regra de divergência

Se OpenAPI, Brain Engine, testes e client discordarem, use `gate: needs_decision`. O orquestrador deve apresentar a divergência ao owner antes de alterar contrato ou comportamento.
