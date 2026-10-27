# Decisões de produto

Ledger conciso de decisões confirmadas pelo owner.

Este documento é pré-contrato. Ele registra decisões de produto, mas não autoriza implementação, criação de schema, persistência, endpoint, plano de entrega ou mudança de arquitetura. Detalhes não registrados como fechados permanecem pendentes em [`open-decisions.md`](open-decisions.md).

## Decisões fechadas

### PD-001 — Um Cérebro por usuário, camadas neurais e Hive Minds

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** cada usuário possui exatamente um Cérebro, criado automaticamente no primeiro login. Dentro dele, a organização é hierárquica, em camadas neurais e subcamadas. Contextos distintos pertencem ao mesmo Cérebro e não criam Cérebros separados.
- **Hive Mind:** uma Hive Mind não é outro Cérebro nem um Cérebro coletivo. É uma conexão/rede neural coletiva entre Cérebros de usuários diferentes, ligando somente camadas neurais ou subcamadas explicitamente selecionadas em cada Cérebro participante. Exemplos como `Relacionamento` ou `Trabalho > Empresa` ilustram a conexão, mas não definem camadas obrigatórias ou default.
- **Fronteira de compartilhamento:** o restante de cada Cérebro permanece fora da Hive Mind e do compartilhamento correspondente. Conectar uma camada não autoriza acesso implícito a outras camadas do mesmo Cérebro.
- **Limites:** esta decisão não define detalhes de transação ou bootstrap do primeiro login, schema, storage, APIs, papéis e permissões detalhados, ownership técnico dos dados compartilhados, sincronização ou lifecycle da Hive Mind. A flexibilidade da hierarquia e a direção de seleção para Funções Cognitivas são complementadas por PD-005.

### PD-002 — Superfície inicial de dados orientada pela finalidade

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** a superfície de dados da finalidade inicial deve contemplar todas as operações adequadas à finalidade.
- **Limites:** toda operação continua sujeita a autorização e regras próprias. A decisão não inventa uma lista final de operações, semântica, falhas, efeitos ou interface.
- **Próximo registro necessário:** um contrato futuro deve enumerar explicitamente as operações e suas regras; essa enumeração permanece aberta em `open-decisions.md`.

### PD-003 — Exclusão para lixeira temporária

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** exclusão envia o item para uma lixeira temporária por 10 dias.
- **Limites:** não estão decididos o início exato da contagem, restauração, purge, retenção legal, clock, visibilidade, relações, anexos ou jobs. Esses detalhes permanecem abertos em `open-decisions.md`.

### PD-004 — Nome técnico da entrada bruta permanece em aberto

- **Status:** owner-approved; fechado em 2026-09-09.
- **Origem:** decisão explícita do owner nesta conversa.
- **Decisão:** `capture`, `note`, `document` ou outro nome técnico de entrada bruta não é uma decisão de produto por si só.
- **Limites:** o nome deve ser escolhido posteriormente para refletir corretamente a semântica aprovada. Nenhum desses termos é promovido aqui a entidade, capability, endpoint ou contrato.

### PD-005 — Hierarquia livre e seleção flexível para Funções Cognitivas

- **Status:** owner-approved; fechado em 2026-09-12.
- **Origem:** decisões explícitas do owner nesta conversa.
- **Hierarquia livre:** o usuário pode criar livremente camadas e camadas dentro de camadas no próprio Cérebro, organizando a hierarquia conforme entender. O aninhamento pode continuar além dos níveis usados nos exemplos, sem estabelecer uma taxonomia obrigatória ou um conjunto fixo de níveis.
- **Seleção flexível:** uma Função Cognitiva poderá considerar ou atuar sobre uma camada, subcamadas específicas ou um grupo explicitamente escolhido de camadas. O grupo pode combinar camadas de ramos distintos, inclusive com pais diferentes.
- **Descendentes:** selecionar uma camada-pai não estabelece uma regra universal de inclusão automática obrigatória de todos os descendentes. O usuário poderá escolher subcamadas específicas; os modos exatos de inclusão e exclusão ainda dependem da futura definição de cada relação.
- **Exemplos:** nomes e árvores usados na documentação são apenas ilustrações e não criam defaults, categorias obrigatórias ou uma estrutura inicial fechada.
- **Limites:** esta decisão não define a natureza exata da relação entre Função Cognitiva e camada — contexto, vínculo, alcance, participação, execução ou outra modelagem permanecem em aberto. Também não decide se grupos são temporários, salvos, reutilizáveis ou nomeados; os efeitos de mover, renomear, arquivar ou excluir camadas; a estratégia técnica de identidade e referências; nem a integração desse modelo com Hive Mind.

### PD-006 — Inbox é uma porta de entrada, não uma camada neural

- **Status:** owner-approved; fechado em 2026-09-16.
- **Origem:** decisão explícita do owner durante a definição do primeiro Brain Core.
- **Decisão:** Inbox é uma funcionalidade de captura rápida para registrar informação sem exigir que o usuário escolha previamente uma camada. Ela não é uma camada neural nem cria uma camada default.
- **Direção assistida:** uma IA poderá futuramente, quando autorizada a acessar a entrada e o contexto adequado do usuário, sugerir em qual camada o registro se encaixa.
- **Limites:** ainda não estão decididos o nome final da funcionalidade, o nome técnico da entrada bruta, quando a IA pode acessá-la, se a sugestão exige confirmação, quem executa a classificação, o destino quando não houver classificação e o lifecycle da entrada.

### PD-007 — Fronteira inicial do Brain Core e identidade estável

- **Status:** owner-approved; fechado em 2026-09-16.
- **Origem:** decisões explícitas do owner durante a definição do primeiro Brain Core.
- **Operações iniciais:** o owner pode observar a própria hierarquia, criar uma camada na raiz ou sob qualquer camada do mesmo Cérebro e renomear uma camada. Move, archive, delete, restore e purge ficam fora desta primeira feature.
- **Hierarquia:** o produto não impõe uma quantidade fixa de níveis nem uma taxonomia; limites técnicos de proteção de recursos, caso necessários, não podem redefinir a árvore como um conjunto fixo de níveis.
- **Autorização:** somente o usuário autenticado proprietário pode observar ou alterar o núcleo privado do próprio Cérebro nesta primeira feature.
- **Bootstrap e identidade:** acessos repetidos ou concorrentes do mesmo usuário devem convergir para um único Cérebro. Brain e camadas usam identidade estável independente de nome de exibição e posição hierárquica; renomear não altera essa identidade nem referências existentes.
- **Limites:** formato do identificador, persistência, transação, concorrência, erros, API, schema e mecanismo de autorização permanecem para contratos e planejamento posteriores.

### PD-008 — Escopo de Hive Mind por raiz de subárvore

- **Status:** owner-approved; fechado em 2026-09-16.
- **Origem:** refinamento explícito do owner sobre a fronteira definida em PD-001.
- **Decisão:** para uma futura Hive Mind, cada camada ou subcamada escolhida é a raiz de um recorte compartilhado. O recorte inclui essa raiz e tudo que estiver dentro de sua subárvore. Camadas ancestrais, irmãs e demais ramos do Cérebro permanecem privados.
- **Separação conceitual:** esta regra pertence ao compartilhamento por Hive Mind. Ela não altera PD-005: Funções Cognitivas continuam podendo selecionar camadas e subcamadas específicas sem uma regra universal de inclusão automática de descendentes.
- **Limites:** ainda não está decidido se descendentes criados depois da conexão entram automaticamente no recorte, nem consentimento, ACL, leitura, escrita, administração, revogação, desconexão, histórico, ownership, persistência, sincronização ou retrieval. Esta decisão não autoriza implementar Hive Mind no Brain Core.

### PD-009 — Conteúdo das camadas: Markdown e dados das Funções Cognitivas

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** conversa sobre o próximo recorte após a entrega web; owner pediu o registro das decisões e planejamento apenas.
- **Decisão:** camadas e subcamadas reúnem o conteúdo textual acumulado pelo usuário e os dados produzidos pelos usos de Funções Cognitivas associados a elas. Esse conjunto compõe o contexto daquela camada.
- **Texto:** anotações, escritas pelo usuário ou por um agente autorizado, são preservadas em formato Markdown; o client interpreta e apresenta esse conteúdo como `.md`. O esclarecimento inicial fechou o formato; PD-012 posteriormente definiu o corpo como texto no PostgreSQL. Diretório local, importação/exportação e sincronização continuam sem aprovação.
- **Dados funcionais:** a organização também contempla informações como tarefas, sessões/horas de foco, pontos e agendamentos. Esses exemplos expressam tipos esperados de informação, sem aprovar catálogo, campos, fórmulas, estados, jobs ou operações específicas.
- **Exemplo ilustrativo:** `Estudos > Curso de investimento` pode reunir anotações Markdown, tarefas, horas estudadas, pontos e agendamentos. Esses nomes não criam camadas default.
- **Limites:** conteúdo pertencente a uma camada e contexto selecionado para um agente são conceitos distintos. Herança de contexto dos descendentes, seleção, autorização de leitura/escrita por agentes e projeções permanecem sujeitos a PD-005 e às decisões futuras. Não há ingestão automática de toda a árvore.

### PD-010 — Funções Cognitivas desenvolvidas pela equipe e combináveis pelo usuário

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** owner esclareceu que, por enquanto, somente a equipe desenvolverá as funções; o usuário poderá combinar as existentes.
- **Decisão:** Funções Cognitivas são capacidades reutilizáveis de produto fornecidas pela equipe. O usuário decide onde usar as funções disponíveis e pode configurá-las e combiná-las dentro das possibilidades que elas oferecerem.
- **Liberdade de uso:** o tema ou nome da camada não restringe a função escolhida; uma função de Finanças pode ser usada tanto em `Finanças` quanto em `Estudos`.
- **Limites:** criação de novas funções, operações ou regras executáveis pelo usuário fica fora desta direção inicial. A composição não aprova um editor de código, workflow engine, automação por eventos, scripts, SQL, shell ou execução livre de cron. A forma concreta de combinação será debatida depois.
- **Autoridade:** funções são linguagem de produto; capabilities continuam sendo interfaces técnicas do Brain Engine. O mapeamento entre ambas não precisa ser um-para-um. As regras e os efeitos permanecem sob o Brain Engine, inclusive quando solicitados por agentes.

### PD-011 — Cada uso de uma Função Cognitiva mantém seu próprio estado

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** owner confirmou usos da mesma ferramenta em camadas diferentes, com estados distintos, e pediu planejamento incluindo “cada uso com seu próprio estado”.
- **Decisão:** a definição reutilizável de uma função é distinta de cada uso dessa função. Cada uso associado a uma camada possui configuração e estado próprios; seus dados não são o estado global da definição nem o estado dos usos em outras camadas.
- **Exemplo ilustrativo:** usar Finanças em `Finanças` e em `Estudos` resulta em dois usos com estados separados. Uma camada também pode reunir usos de funções diferentes.
- **Terminologia:** “uso” é o termo adotado neste registro; “instância” pode ser uma expressão técnica futura, sem impor esse nome à interface ou à persistência.
- **Limites:** quantidade de usos da mesma função na mesma camada, cópia/movimentação, desativação, exclusão/restauração, evolução da definição, referências entre usos e agregação de estados ainda não estão decididas. Selecionar várias camadas para consulta não funde os respectivos estados.
- **Próxima discussão:** o owner adiou expressamente a escolha/detalhamento da primeira função. Registrar o conceito não aprova Tarefas, Foco, Finanças ou outra como primeira implementação.

### PD-012 — Corpo dos documentos Markdown como texto no PostgreSQL

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** owner propôs guardar o conteúdo “em um campo de texto” e confirmou o registro após a explicação da proposta.
- **Decisão:** o corpo do documento é preservado como texto Markdown no PostgreSQL; o client interpreta e apresenta esse formato. Um documento `.md` é uma unidade lógica de conteúdo e não exige um arquivo físico no disco da VM ou do Mac.
- **Identidade:** documento possui identidade estável, título e conteúdo; o título não é sua identidade. O vínculo com a organização em camadas deve ser preservado independentemente do nome.
- **Autoridade:** leitura, edição, validação e persistência passam pelo Brain Engine; o client não acessa o banco diretamente.
- **Limites:** esta decisão escolhe o meio de persistência do corpo, sem criar tabela/coluna/migration ou escolher variante Markdown, editor, autosave, histórico, limites, anexos, importação/exportação ou sincronização. O armazenamento de dados das Funções Cognitivas continua dependendo dos contratos de cada domínio.

### PD-013 — Links internos por identidade e grafo de documentos

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** owner pediu links funcionais como `[[...]]` ou via ID para correlacionar informações e visualizar ligações entre documentos; confirmou o registro da proposta de referências internas estáveis e grafo derivado.
- **Decisão:** links internos apontam para a identidade estável do documento de destino, com nome/rótulo visível. Renomear o documento não quebra o vínculo.
- **Autoria:** o usuário pode expressar a intenção por título com wikilinks `[[...]]`; escolher o destino deve conservar a referência por ID. Se dois documentos tiverem o mesmo título, a resolução exige escolha explícita, sem selecionar silenciosamente o primeiro.
- **Responsabilidades:** o client interpreta/renderiza o conteúdo e a navegação; o Brain Engine entende e valida as referências, o acesso aos destinos e a disponibilização das relações. A regra se aplica ao conteúdo escrito por pessoa ou agente autorizado.
- **Grafo:** representa documentos e seus links internos e é derivado das referências; não é uma autoridade paralela de conteúdo. A árvore organiza camadas/subcamadas e o grafo mostra conexões entre informações dessa organização. Links podem conectar documentos da mesma camada ou de camadas diferentes respeitando o acesso aos documentos.
- **Limites:** `[[...]]` é uma extensão wikilink, não uma escolha completa de dialeto Markdown. Sintaxe exata do ID/rótulo, resolução/canonicalização, comportamento de links pendentes/quebrados, atualização de rótulos, exclusão/restauração de destinos, índice/materialização, layout/UI do grafo e cobertura de registros funcionais/URLs externos ainda não estão decididos. Nenhum endpoint, biblioteca de renderização/grafo ou serviço novo é aprovado por este registro.

### PD-014 — Pertencimento direto e consulta documental da subárvore

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** owner exemplificou `Trabalho/Nsr-project/docs.md` e confirmou que, em `Trabalho`, pode consultar todos os documentos contidos nos filhos e descendentes.
- **Decisão:** cada documento pertence diretamente a uma camada; consultar a camada-pai permite reunir os documentos próprios e os de todas as suas subcamadas, recursivamente. Esse conjunto está disponível como contexto documental do pai.
- **Organização:** a visualização distingue documentos da camada e, futuramente, usos de Funções Cognitivas daquela camada. Caminhos apresentados na UI descrevem a hierarquia lógica, não arquivos físicos no servidor. Títulos repetidos permanecem diferenciados por identidade.
- **Limites:** consulta herdada não copia documentos para os pais, não muda pertencimento e não funde estados de usos funcionais. Ownership/permissões continuam aplicáveis. A disponibilidade de contexto da subárvore não exige enviar todo o conteúdo em cada prompt; seleção e orçamento de contexto de agentes permanecem para etapa própria.
- **Coerência:** refina a consulta de documentos em OD-014; a seleção de atuação de Funções Cognitivas de PD-005 continua distinta e será detalhada depois.

### PD-015 — Operações e primeira experiência do editor documental

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** respostas do owner às opções de operações/editor/links/grafo.
- **Operações:** criar, listar, abrir, editar e renomear documentos; mover entre camadas, excluir e restaurar também entram no recorte documental. A exclusão segue a direção de lixeira temporária de 10 dias de PD-003; relógio, prazo operacional e purge ainda precisam de detalhamento.
- **Editor inicial:** texto Markdown com prévia, contemplando títulos, listas, tabelas, código e links; refinamentos de UX podem vir depois. Nenhum renderer/dialeto/pacote específico foi escolhido.
- **Links:** confirma intenção por `[[...]]`, seleção de destino por ID e sinalização de links sem destino, sem criar documentos automaticamente. Rename conserva identidade; sintaxe canônica de IDs/rótulos continua aberta.
- **Adiamentos:** visualização do grafo fica para depois. PD-013 preserva a direção do grafo derivado para evolução futura; navegação/vínculos internos continuam na discussão atual. Funções Cognitivas e seu estado/UI ficam para etapa posterior.
- **Limites:** essas operações referem-se a documentos, não adicionam mover/excluir/restaurar camadas ao contrato atual do Brain Core. Comportamentos de conflitos, lixeira, origem/destino de moves e restauração ainda precisam ser contratados.

### PD-016 — Salvamento documental local a cada segundo; sincronização posterior

- **Status:** owner-approved; fechado em 2026-10-11.
- **Origem:** owner pediu salvamento local a cada segundo e intenção futura de envio a cada 30 minutos. Ao esclarecer o recorte, escolheu explicitamente “Apenas salvamento local agora; sincronização em etapa posterior”.
- **Primeira entrega:** preservar automaticamente as alterações dos documentos no armazenamento local do client, em cadência de um segundo. Salvamento manual não é a estratégia principal escolhida.
- **Servidor posterior:** envio automático ao servidor em cadência de 30 minutos é uma direção para a próxima etapa de sincronização, junto com sincronização manual por botão futuro. Nenhum dos dois entra agora.
- **Persistência:** PD-012 continua como escolha do corpo Markdown no PostgreSQL para a etapa do servidor. A entrega inicial contém documentos locais, ainda não enviados; não deve afirmar sincronização ou disponibilidade em outros dispositivos.
- **Autoridade e privacidade:** dados locais pendentes não são estado remoto confirmado pelo Brain Engine. O planejamento precisa separar conteúdo local, identidade do usuário/ambiente, alterações pendentes e futura versão remota; não usar a sessão de Auth como armazenamento dos documentos.
- **Limites:** engine local, identidade antes do primeiro envio, falha/quota, múltiplas abas, logout/troca de conta, prazo da lixeira local, conflitos futuros e migração para a fase de sincronização permanecem para contrato técnico. A cadência é um objetivo operacional enquanto o client puder executar; navegador suspenso/fechado não executa timers nem salva mudanças não gravadas. Operação offline completa do aplicativo e sincronização da árvore de camadas não estão aprovadas por essa decisão.

### PD-017 — Lixeira local de documentos por 10 dias

- **Status:** owner-approved; confirmado em 2026-10-12.
- **Origem:** owner perguntou se a proposta era lixeira local de 10 dias e aceitou: “se sim ok”.
- **Decisão:** documentos excluídos na primeira entrega vão para lixeira local e podem ser restaurados dentro do prazo de 10 dias. Complementa PD-003/015 para a fase local escolhida.
- **Detalhamento adotado no planejamento:** 10 dias corridos (240 horas), a partir do commit local da exclusão, com timestamp UTC do dispositivo. Excluir novamente um item já na lixeira não renova o prazo. Restaurar antes do vencimento conserva ID, texto e referências; nova exclusão depois da restauração inicia outro período.
- **Vencimento:** a partir de `agora >= expires_at`, restauração deixa de estar disponível. Remoção física local ocorre na próxima oportunidade de execução do aplicativo; não há promessa de job/purge com navegador fechado. A limpeza deve incluir cópias/rascunhos locais daquele documento para não manter conteúdo recuperável depois do purge.
- **Limites:** mudança manual do relógio do dispositivo afeta a avaliação local; relógio autoritativo do servidor pertence à sincronização futura. Destino de restore indisponível e efeitos sobre links continuam no detalhamento do contrato documental. Nenhum dado real foi excluído por este registro.

### PD-018 — Coordenação técnica entre abas e preservação no logout

- **Status:** direção confirmada/delegação técnica do owner em 2026-10-12; mecanismo detalhado no planejamento, sem implementação.
- **Origem:** owner delegou resolver conflitos entre abas e propôs envio no logout; ao esclarecer o alcance, escolheu “Deixar o gatilho de logout para a etapa de sincronização; manter a primeira entrega local”.
- **Primeira entrega:** logout encerra a sessão e limpa o conteúdo visível da conta; documentos, lixeira e rascunhos confirmados localmente são conservados no namespace dessa conta e recuperados quando ela entrar novamente. Trocar de conta não mostra nem aplica pendências da conta anterior.
- **Sincronização futura:** envio antes de concluir logout é comportamento alvo para a etapa remota. Não entra nesta entrega e não substitui autosave local. Falha de envio nunca autoriza apagar dados locais ou afirmar sucesso remoto; UX de retry/cancelar/sair conservando pendências será detalhada nessa etapa.
- **Delegação técnica:** o principal pode definir coordenação entre abas, transações e tratamento de versões/rascunhos sem repassar a escolha de mecanismos ao owner. O planejamento usa controle de concorrência e preservação de ambas as edições, sem last-write-wins silencioso ou merge automático obrigatório.
- **Limites:** essa delegação não autoriza código, instalação, APIs/SQL, mudanças na VM ou criação de sincronização remota. As regras operacionais estão na spec de documentos.

### PD-019 — Obsidian básico como referência documental e links acompanhando renomeações

- **Status:** direção owner-approved em 2026-10-12; planejamento apenas.
- **Origem:** owner confirmou “Acompanha renomeações sim” para o nome exibido pelos links e pediu que os documentos funcionem “mais ou menos como o obsidian base funciona”.
- **Decisão:** usar a experiência básica de notas do Obsidian como referência para criar/abrir/editar/renomear/mover documentos, Markdown e navegação por links internos. O nome padrão exibido pelo link acompanha a renomeação do documento de destino; a identidade e a ligação permanecem as mesmas.
- **Aplicação no planejamento:** digitar `[[` oferece documentos para seleção; nomes iguais são diferenciados por caminho/camada e identidade, sem escolher o primeiro silenciosamente. Rename atualiza o nome apresentado; move conserva a ligação e atualiza a localização apresentada quando aplicável. Não exigir que o usuário gerencie IDs técnicos.
- **Precedência:** mantêm-se IDs estáveis, títulos repetidos, consulta recursiva dos pais, primeira entrega local com autosave1s, lixeira240h e preservação por conta. Links sem destino continuam sinalizados sem criação automática. O Mindforge não passa a depender de arquivos físicos/vault.
- **Limite da referência:** não implica paridade integral com Obsidian, todos os core plugins, Live Preview inline, aliases, embeds, referências a blocos/headings, anexos, filesystem, import/export, grafo ou sync. O editor texto + prévia já escolhido permanece nesta entrega; recursos adicionais exigem recorte explícito. Rótulos personalizados não entram por inferência nesta entrega.
- **Detalhamento:** o principal deve propor padrões técnicos/UX coerentes com essa referência, distinguindo proposta de decisão aprovada; perguntar ao owner somente por diferença material de comportamento/escopo. Nenhuma implementação ou operação externa autorizada aqui.
- **Fontes de referência:** [links internos](https://obsidian.md/help/links), [gestão de notas](https://obsidian.md/help/manage-notes) e [modos do editor](https://obsidian.md/help/edit-and-read). Essas fontes descrevem Obsidian, não substituem as decisões Mindforge.

### PD-020 — Antecipar o backend documental com TDD

- **Status:** owner-approved em 2026-10-12.
- **Origem:** owner autorizou fechar a spec e iniciar backend com TDD; confirmou explicitamente “Sim, preparar o backend completo; client e sincronização depois”.
- **Decisão:** antecipar API e persistência PostgreSQL dos documentos; client/editor/autosave local e sincronização ficam para depois. Substitui a ordem local-primeiro de PD-016, preservando o comportamento local previamente decidido para a etapa do client.
- **Autoridade:** código/testes e SQL versionado localmente; não executar SQL, alterar dados, instalar ou fazer deploy na VM. Contratos HTTP concretos seguem aprovação específica antes de promoção, conforme AGENTS.
- **Contrato HTTP:** owner aprovou explicitamente as seis rotas propostas e lixeira remota240h nesta conversa, antes de promoção no registry/OpenAPI.
- **Planejamento executável:** [spec backend](features/brain/layer-content/spec-02-backend.md); direção e histórico continuam na spec01.

### PD-021 — Client Android adaptativo com rascunho e sincronização

- **Status:** owner-approved em2026-10-13.
- **Origem:** owner autorizou tarefa específica para implementar layouts adaptativos, manter funcionalidades existentes e acrescentar documentos, TDD=yes. Ao esclarecer envio, escolheu rascunho local, sync automático30min e botão manual.
- **Decisão:** primeira interface de documentos será Android com experiência compacta/toque, mesma lógica/componentes para composição ampla. Autosave local1s, upload automático30min enquanto houver execução e sync manual entram neste lote; substitui o adiamento de envio de PD-016/018/020 apenas para o escopo aqui confirmado.
- **Falhas/autoridade:** rascunho local não significa servidor atualizado; conflito/resultado incerto conserva texto e não autoriza overwrite/retry cego. Brain Engine continua autoridade remota. Não prometer execução de timer em background suspenso nem logout garantir upload.
- **Limites:** código/testes/buildlocal autorizados; SQL/VM/deploy/instalaçãoaparelho/Googleconsole e publicação permanecem separados. Documentos Web/IndexedDB ainda não incluídos. A regra adaptativa aprovada está no AGENTS e docs/stack/front-end/adaptive-layouts.md; nenhuma nova arquitetura de estado/framework adotada.
- **Execução:** [spec Android](features/brain/android-workspace/spec-01-main.md) e checkpoint da tarefa.

### PD-022 — Client usa somente o Brain Engine, inclusive login e sessão

- **Status:** direção/spec/cinco contratos HTTP aprovados pelo owner em2026-10-13; execução backend/client com TDD e corte VM develop concluídos em2026-10-14; aceite Google autenticado pendente.
- **Origem:** owner determinou que a API sempre intermedeie o GoTrue e pediu “monta a spec com tdd para o client falar apenas com a nossa API, inclusive login e sessão”.
- **Decisão:** Flutter usa somente endpoints Brain Engine para login Google, troca PKCE, validação/restauração, refresh e logout, além de Core/documentos. GoTrue permanece provedor interno; consentimento ocorre no navegador Google. Não confundir proxy Nginx direto para Auth com intermediação pelo Brain Engine.
- **Estado constatado na discovery:** native apontava diretamente ao Auth e Web tinha `/auth/*` direto. **Execução posterior:** nova fachada/API/client e proxy aplicados no develop, conforme evidência da tarefa; iniciação/cancelamento OAuth reais passaram, login/sessão autenticados aguardam URI Google e aceite. Substitui essa fronteira de transporte das entregas anteriores; preserva Google-only, contas/IDs, regras, documentos e sincronização.
- **TDD:** yes explícito. [Spec integrada](features/brain/api-auth/spec-01-main.md) inclui cinco rotas propostas, proteção de redirects, elegibilidade, refresh/logout, namespaces e corte coordenado.
- **Autoridade:** owner aprovou spec/cinco rotas e implementação TDD backend/client, com VM develop para mudanças necessárias de API/Auth/Web/túnel. Google Console será ajustado pelo owner ao final; nenhuma alteração main/NSR/Studios/DB/volumes ou operação SQL/destrutiva/commit/push inferida.

### PD-023 — Assistente pessoal contínuo do Cérebro

- **Status:** direção owner-approved; consolidada em 2026-10-18, somente registro documental.
- **Origem:** owner descreveu o Mindforge como assistente pessoal/extensão do seu Cérebro e pediu continuidade natural, inclusive com diferentes modelos; solicitou consolidar a visão geral e decisões fechadas.
- **Decisão:** oferecer experiência conversacional contínua para consultar e organizar informações e solicitar ações. O usuário não deve administrar janelas de contexto; a continuidade deve ser responsabilidade do Mindforge, sem depender de manter um único modelo/provedor.
- **Limites:** preserva um Cérebro por usuário, fronteiras de Hive Mind e autoridade Brain Engine. Não define storage de conversas, retenção, contexto infinito, autonomia de memória, modelos/provedores, ferramentas ou funcionalidades executáveis. “Segunda persona” é metáfora de experiência, não autoridade ou infalibilidade.
- **Visão consolidada:** [RAG, memória, harness e MCP](product/assistant/rag-harness-overview.md); propostas e pendências separadas.

### PD-024 — Trava e aviso para troca de modelo

- **Status:** direção owner-approved; consolidada em 2026-10-18.
- **Origem:** owner determinou que a restrição de troca durante operações deve ser “uma trava do aplicativo” e ter aviso ao usuário.
- **Decisão:** controlar a troca no aplicativo para evitar mudança insegura no meio de uma operação; informar ao usuário a restrição. Não depender apenas de recomendação textual.
- **Limites:** enfileiramento até fim do turno, reconciliação, estados, cancelamento, timeout e recuperação são propostas a detalhar em contrato futuro. Nenhuma implementação autorizada.

### PD-025 — MCP como consumidor e provedor

- **Status:** direção owner-approved; consolidada em 2026-10-18.
- **Origem:** owner afirmou querer “as duas capacidades”: conectar ferramentas externas e oferecer ferramentas do Mindforge a clientes autorizados.
- **Decisão:** contemplar consumo de servidores MCP externos e exposição de capacidades Mindforge por servidor MCP para clientes autorizados.
- **Limites:** servidor MCP não cria autoridade paralela ao Brain Engine. Fluxo do harness no client via MCP interno é proposta arquitetural discutida; não fecha processo/transporte, autenticação, escopos, contratos de tools nem substitui APIs manuais. PD-022 permanece aplicável a login/sessão. Sem autorização para código, instalação ou deploy.

## Limite de interpretação

As definições de um Cérebro por usuário, organização hierárquica em camadas e subcamadas, Hive Mind e sua fronteira inicial estão fechadas em PD-001 e refinadas em PD-008. A liberdade de criação da árvore e a direção de seleção flexível para Funções Cognitivas estão fechadas em PD-005, sem fechar a modelagem dessa relação. Inbox está separada da hierarquia por PD-006, e o primeiro Brain Core está delimitado em PD-007. Exemplos concretos de nomes de camadas, formatos de dados e capabilities em [`architecture-overview.md`](architecture-overview.md) continuam sendo orientação conceitual quando não aparecem nas decisões acima. Este ledger não promove exemplos arquiteturais a contrato nem autoriza implementação.

PD-009..011 registram o refinamento de conteúdos e usos de funções confirmado em 2026-10-11. O [planejamento de conteúdo das camadas](features/brain/layer-content/spec-01-main.md) permanece exclusivamente documental e sujeito a discussão técnica posterior.
