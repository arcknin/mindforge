# B04: serviços develop aplicados, login nativo pendente

Owner autorizou a operação concreta após receber a pergunta de aplicação Auth develop/API/túnel: “beleza temos então o oauth configurado eu acho ne? podemos prosseguir?”. As credenciais foram fornecidas pelo owner e tratadas privadamente. Esta evidência não comprova consentimento/troca de código Google nem aplicativo nativo.

## Aplicação real

- Preflight: main/NSR/DB/Auth/Studios saudáveis; main ledger0001–0004, develop0001–0006; role/owner develop sem superuser/createdb/createrole; rede existente e porta18087 livre. Somente catálogo read-only, sem dados pessoais.
- Release em `/opt/mindforge/releases/manual-online-20261008-r2/`; archive SHA-256 `30b4930983b96a7cfe88d2865c6a99bc2dcda003d7c39223967d2a55b9adf98a`;10 arquivos de transporte verificados.
- Binário API SHA-256 `29e771185d9d19a67c846e5d7f95e4d4eb2d18f716e9a06a8598952b73c62f65`; imagem scratch e usuário65532:65532.
- Configuração em `/etc/mindforge/manual-online/`, dir700/env600. Google Client Secret e DSN develop ficam fora de Git/args/stdout/Docker context. Credencial DB derivada do arquivo privado develop já instalado; não houve criação de role/banco ou migration.
- Backup do ambiente/configuração anterior Auth develop e overlay recuperável em arquivos privados. Compose renderizado em memória validou DB URL/JWT e volume histórico preservados.
- Somente `auth-develop` foi recriado, com Google habilitado e entrada email/telefone/anônimo desligada. Health200. Main/NSR/DB e Studios mantiveram IDs, StartedAt e volumes, todos saudáveis.
- Novo projeto `mindforge-brain-engine-develop`, serviço API em `127.0.0.1:18087→8080`, rede `infra_default`, zero restart. O startup executou o probe read-only previsto e abriu o Store; não há migration automática ou volume novo.
- Túnel real neste Mac: loopback11000→Auth10000 e11001→API18087, host key estrita. Processo deve permanecer ativo; encerrá-lo deixa o callback/local API indisponíveis. Config SSH privada sem senha; processo autenticado com credencial autorizada.

## Verificações reais

- `/settings` GoTrue mostra Google habilitado e métodos próprios/anônimo desligados.
- `/authorize` retorna302 para `accounts.google.com`; Client ID corresponde à configuração privada, callback Google exato `http://127.0.0.1:11000/callback`. Location/state/códigos não foram emitidos.
- Escopos efetivamente solicitados são `email` e `profile`, sem APIs Gmail/Drive/Calendar; compatíveis com a exceção Google para login básico. Nenhuma allowlist manual de emails foi acrescentada. [Regra Google](https://support.google.com/cloud/answer/15549945?hl=en)
- API real retorna401 para token ausente e inválido. Pelo túnel Mac, GET/PUT/POST/PATCH dos paths aprovados sem token também retornam401. Nenhuma dessas chamadas inicializou Brain/camada.
- Health/Auth provider/redirect/callback e API negativa confirmados também a partir do Mac pelo túnel; prova local sanitizada em arquivo temporário privado.
- Prova sanitizada e backups mantidos na VM em `/etc/mindforge/manual-online/runtime-service-proof.json` e arquivos privados adjacentes.

## Limites e próximo passo

- Provider/configuração/serviços/túnel estão aplicados. Client Secret ainda não foi validado pela troca de um código Google após consentimento humano.
- API autenticada, novo Brain do usuário Google, Keychain e retorno ao app macOS permanecem pendentes. Os quatro endpoints continuam `runtime_status: pending`; não promover status com base só em401/302.
- Flutter nativo ainda não compilado: `xcodebuild` ausente. Instalação/configuração Xcode e licença não foram autorizadas por esta operação.
- Nenhuma publicação Internet, DNS/Nginx, mudança main/NSR, promoção0006, backup/restore de dados, criação de usuários reais/Brains de prova, commit ou push nesta aplicação.
- Iniciar o OAuth pode criar estado transitório interno de fluxo no Auth; isso não é conta de usuário ou prova de login.

Próxima fronteira: preparar toolchain macOS e compilar o app, depois owner concluir login Google no browser e consentir na prova autenticada. Para reiniciar o túnel, usar o helper e o arquivo SSH privado, nunca senha em código. Rollback restaura somente Auth develop e remove API/túnel novos, sem apagar banco/usuário/Brain/volume.


## Pausa solicitada pelo owner

Owner adiou continuidade e instalação Xcode, escolhendo Android para a próxima etapa. Serviços na VM preservados; túnel local encerrado para a pausa. Retomada exige reconciliar Android e reabrir a conectividade, sem repetir deployment ou presumir login verificado.
