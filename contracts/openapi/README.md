# Contratos HTTP do Brain Engine

Este diretório será a fonte canônica dos endpoints HTTP aprovados do Mindforge. Ele não transforma capabilities conceituais em HTTP e não autoriza implementação por si só.

## Arquivos

- `registry.yaml`: índice operacional de propostas e endpoints, com estados e evidências independentes.
- `*.openapi.yaml`: contratos executáveis criados somente quando existir ao menos um endpoint aprovado.

As quatro operações Brain Core estão aprovadas pelo pedido de execução do owner. O contrato é [brain-core.openapi.yaml](brain-core.openapi.yaml); implementação local concluída e runtime permanece pendente.

O registry acompanha o [primeiro fluxo manual macOS](../../docs/features/brain/manual-online/spec-01-main.md), com Google OAuth via GoTrue. Seus estados de contrato, implementação e runtime são independentes; credenciais Google e integração real serão preparadas depois.

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

A continuação [web-local](../../docs/features/brain/web-local/runtime-evidence.md) publica o client Flutter Web e o proxy develop. Os contratos permanecem iguais e runtime dos quatro endpoints continua pendente de prova autenticada.

As seis operações documentais foram aprovadas explicitamente pelo owner em 2026-10-12 após proposta no registry. Contrato: [documents.openapi.yaml](documents.openapi.yaml); spec: [backend documental](../../docs/features/brain/layer-content/spec-02-backend.md). Client/sincronização posteriores; SQL/VM/deploy dependem de autorização separada. Estados de implementação/runtime constam no registry.

As cinco rotas de Auth API-only foram aprovadas pelo owner em2026-10-13. Contrato: [auth.openapi.yaml](auth.openapi.yaml); [spec](../../docs/features/brain/api-auth/spec-01-main.md). Implementação e runtime possuem estados independentes no registry. GoTrue permanece provedor interno; não confundir aprovação com corte aplicado.
