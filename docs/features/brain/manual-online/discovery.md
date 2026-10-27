# Discovery: primeiro fluxo manual no macOS

## Pedido e limite da investigação

- Owner escolheu a opção Auth + API + Flutter mínimo e autorizou **somente planejamento**, com agentes folha. Não autorizou implementação, instalação, execução SQL, acesso/mudança de VM, publicação, commit ou push nesta unidade.
- Decisões confirmadas na conversa: macOS; TDD `yes`; somente Google OAuth, sem senha própria Mindforge. Esta última decisão substitui a interpretação anterior de cadastro/recuperação por email. Primeiro acesso Google estabelece a conta Mindforge; recuperação da conta Google permanece no provedor.
- Repositório: `/Users/arcknin/Desktop/mindforge`, branch `feat/brain-core-foundation`, HEAD `0204ba4`. Dirty work anterior preservado.

## Comportamento atual verificado

- `features/brain/engine.go`: `Bootstrap`, `Observe`, `CreateLayer`, `RenameLayer`. `model.go` define `Principal` opaco, root estrutural separado de layer e versão de rename. `ports.go` mantém o domínio independente de transporte.
- Brain Core r4: um Brain privado por owner; root interno sem starter tree; nesting livre; nomes irmãos repetidos; IDs estáveis e rename condicionado à versão. API, autenticação técnica e client estão fora desse contrato.
- `PostgresStore` é pgx e não aplica migrations ao abrir. `Observe` lê snapshot transacional; criação gera novo ID a cada chamada. Não há deduplicação por chave ou nome para `CreateLayer`.
- A evidência da unidade [develop-studio](../../infra/develop-studio/runtime-evidence.md) registra develop com migrations0001–0006, Auth próprio e seis verificações reais do Engine. Main permaneceu sem0006. Isso não comprova OAuth, API de produto ou Flutter.
- Owner confirmou acesso aos Studios main/develop após a aplicação. Essa confirmação encerra a pendência de acesso do owner; não é uma nova inspeção independente de hosts ou prova de aplicativo.
- Ao iniciar esta pesquisa, registry estava vazio e `src/mindforge-client/` ausente; os executáveis existentes são operadores/verificador. Agora podem ser registrados contratos HTTP **proposed**, sem implementação ou aprovação.
- Compose publica GoTrue diretamente, sem Kong/REST gateway para Mindforge. Auth develop é `supabase/gotrue:v2.196.0`, em loopback da VM. Os fontes não contêm configuração Google nem SMTP. Não foram consultadas configurações privadas ou a VM nesta unidade.
- Neste Mac, comandos `flutter` e `dart` são localizáveis, arquitetura `arm64`; `xcode-select -p` aponta a CommandLineTools; `/Applications/Xcode.app` está ausente e `pod` não foi localizado no PATH. Versões do SDK e outras instalações de Xcode/CocoaPods não foram verificadas; prontidão para build macOS não está estabelecida.

## Fontes e achados delegados

- Principal leu AGENTS, as seis fontes obrigatórias, contrato r4, decisões de produto, fontes decisivas do Engine/Store, registry e evidência da implantação anterior.
- `brain_core`, Luna/xhigh, folha: mapeamento HTTP proposto, erro opaco, root separado, versão de rename e criação sem retry seguro. Principal confirmou em `model.go`, `engine.go`, `ports.go` e `postgres_store.go`.
- `infra`, Luna/xhigh, folha: GoTrue como emissor da sessão e Google como provedor; dois callbacks distintos; configuração Google ausente; callback privado requer conectividade explícita. Principal confirmou Compose/exemplo e fontes oficiais abaixo.
- `produto_client`, Luna/xhigh, folha: SDK standalone compatível com URL raiz; PKCE, browser externo, callback macOS e persistência de sessão exigem integração; estados de conflito e resultado incerto. Principal leu as referências oficiais do SDK, storage e Flutter.
- Nenhum worker alterou arquivos, executou testes, acessou VM/banco ou abriu interface do produto. A pesquisa externa usou fontes oficiais; os retornos não aprovam contratos.

## Fontes externas decisivas

- [OAuth self-hosted](https://supabase.com/docs/guides/self-hosting/self-hosted-oauth): configuração do provedor no GoTrue e retorno intermediado pelo Auth.
- [Google OAuth e callbacks locais](https://supabase.com/docs/guides/auth/social-login/auth-google) e [regras Google](https://developers.google.com/identity/protocols/oauth2/web-server): Web OAuth client; callback exato; exceção loopback para desenvolvimento. Não atribuir validade de callback público a nomes home.arpa.
- [GoTrueClient 2.27.2](https://pub.dev/documentation/gotrue/2.27.2/gotrue/GoTrueClient-class.html): URL própria, PKCE e storage do verifier. [Troca de código](https://pub.dev/documentation/gotrue/2.27.2/gotrue/GoTrueClient/exchangeCodeForSession.html) e [recoverSession](https://pub.dev/documentation/gotrue/2.27.2/gotrue/GoTrueClient/recoverSession.html) foram lidos na documentação atual identificada como2.27.2.
- [Auth GET /user](https://supabase.com/docs/reference/self-hosting-auth) e [logout](https://supabase.com/docs/guides/auth/signout): fronteira remota candidata; logout não implica revogação instantânea de todo JWT emitido.
- [Callback macOS](https://pub.dev/packages/app_links) e [browser externo](https://pub.dev/packages/url_launcher): plugins candidatos, configuração/versões ainda sujeitas à compatibilidade do projeto.
- [Flutter architecture](https://docs.flutter.dev/app-architecture/guide): separação pequena de Views/ViewModels e Services/Repositories. [Secure storage](https://pub.dev/packages/flutter_secure_storage): Keychain macOS; [macOS build](https://docs.flutter.dev/platform-integration/macos/building): entitlements de rede e setup nativo precisam ser validados.

## Incertezas e decisões

- Fluxo de produto/macOS/Google-only/TDD definidos. Paths, DTOs, política do verificador, bibliotecas, configuração de callbacks e desenho operacional ainda são propostas da spec.
- Não há evidência de OAuth client Google Cloud existente, consent screen/test users, credenciais privadas ou callbacks cadastrados. Isso é pré-requisito futuro, não convite para criar conta/projeto nesta sessão.
- Candidato privado: túnel SSH do Mac ao Auth/API loopback da VM; callback Google em loopback Mac e callback final em scheme da app. Viabilidade documental, sem túnel ou OAuth real validado.
- Validação remota pelo Auth adiciona uma chamada por request; não promete revogação de access tokens além do comportamento real do GoTrue.
- Toolchain macOS requer inventário/configuração futura. Não instalar SDK/Xcode/CocoaPods por inferência.
- `infra/tests/test_runbook_hygiene.sh:199` ainda exige registry vazio; propostas permitidas por AGENTS agora tornam essa premissa obsoleta. Registrar adaptação no lote de código futuro; nenhum teste foi alterado/executado nesta sessão.
- READMEs/checkpoints antigos ainda contêm r3, `mindforge_main` e provas históricas. Contratos atuais e evidência develop-studio prevalecem; não fazer limpeza geral nesta unidade.

## Continuação

- [Spec integrada](spec-01-main.md) e [checkpoint](manual-online-checkpoint.yaml).
- Próximo passo: discutir o desenho proposto e contratos com owner. Implementação/externalização permanecem fora da autorização atual.


## Complemento: configuração Google OAuth como etapa do processo

Owner pediu incluir configuração da conta/projeto Google e obtenção das chaves. A spec revision2 detalha esse setup em B01: acesso do responsável, projeto develop, Branding/Audience/Data Access, OAuth Web client, callbacks e guarda privada de Client ID/Secret. Contrato congelado permite desenvolver contra fakes; credenciais/configuração reais são obrigatórias antes da integração B04. A revisão não executa Google Cloud, Auth, VM ou aplicação.

Principal conferiu [Google Auth Platform](https://support.google.com/cloud/answer/15544987?hl=en), [Audience](https://support.google.com/cloud/answer/15549945?hl=en) e as fontes oficiais Google/Supabase já relacionadas. As exceções de Testing para identidade básica impedem tratá-lo como uma allowlist de segurança garantida.


## Execução local posterior autorizada

Owner pediu executar o plano completamente até onde fosse possível sem credenciais e deixar cadastro Google para depois. Contratos foram aprovados/formalizados antes da implementação. A [evidência de entrega](implementation-evidence.md) supersede o estado somente-planejamento desta discovery: Auth/API/runtime Go, Flutter macOS e helpers locais concluídos; nenhum setup Google/VM real. O SDK existente foi mantido, e a ausência de `xcodebuild` limitou o artefato nativo Flutter.
