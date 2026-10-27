---
schema_version: 1
managed_by: spec-drive
feature: brain
task: web-local
revision: 1
status: owner_confirmed_manual_flow
---
# Flutter web develop on owner VM

## Context and outcome
Owner explicitly requests adaptation/build/VM Docker publication at web-mind.home.arpa and Mac hosts command, then delegates best current OAuth route. Discovery: discovery.md. Historical predecessor ../manual-online/spec-01-main.md; Android/macOS build deferred.

## Goals and chosen behavior
Reuse existing Flutter manual interface and four approved Brain Core operations with Google-only GoTruePKCE. Same-origin /auth routes proxy to existing develop GoTrue; /api prefix strips to existing HTTP contract. Application callback /login/callback reloadsSPA, consumes PKCE once, removes sensitive query/fragment before further navigation. Persist PKCE/session only in tab sessionStorage onweb; retain Mac Keychain composition separately. Browser tokens never administrativecredentials or persistedGoogleprovidercredentials. Layer versions preserve the full positive Int64 contract through BigInt/lossless JSON decoding and exact numeric CAS serialization; no JavaScript precision downgrade.

Web accessible on privateLAN HTTP web-mind.home.arpa. This chosen lab origin is not encrypted; do not claim Internet/production readiness. Scope limits hostvhost to loopback, confirmed ownerMac VirtualBox NAT gateway10.0.2.2 and ownerLAN; no publicDNS/firewall/ingress. Use no service-worker token cache; package CanvasKit and normal UI fonts locally; retain engine optional Unicode font fallback where a glyph is absent from packaged fonts. Do not configure nonexistent fallback resource paths.

Google callback remains registered http://127.0.0.1:11000/callback; MacSSH forwards11000 to VM10000. Add precise webcallback to developGoTrue SITE_URL/URI_ALLOW_LIST while preserving existing custom callback and DB/JWT/providercredentials. ExternalAuthURL unchanged. Only auth-develop may be selectively recreated with backup/rollback. This avoids unapproved publicdomain and GoogleCloudchanges.

## Non-goals
New product endpoints, backendcode/DB/migrations, main/NSR/Studio changes, nativeAndroid/macOSbuilds, newAuthprovider, SMTP/password, Internetpublication, real-user test writes, commit/push, browser/visualQA.

## Order and ownership
1. Client leaf owns src/mindforge-client/: TDD config/storage/callback/platform fixes, runner, analyze/tests/build.
2. Infra leaf owns new src/brain-engine/features/infra/web-local/: Docker/staticproxy, compose, hostvhost, bounded tests/runbook.
3. Root integrates, builds/stages immutable manifest archive without secrets; liveVM preflight captures container IDs/startups/mounts and existingNginx/config hashes, verifies18088free.
4. Root installs new release, builds/publishes web container loopback18088 onexistingnetwork; nginxsyntax check thenreload isolatednewsite. Backup/preparedrollback Authdeveloponly thenaddcallback selectively.
5. HTTP proof staticindex/JS/CanvasKit/callbackSPA sameoriginAuthhealth/settings/Google302 and fourAPI401. Compare protected containers/configs. Open only11000SSHforward; give hostscommand plusURL and login caveat.

## Authority and rollback
User's direct request authorizes clientcode/build and bounded VM Docker/Nginx/developAuthconfiguration changes necessary for web, includingSSHforward; no additional permission barrier. Credentials stay outsideGit/logs/args/buildcontext. Preserve all unrelated dirtyfiles/services. Backup current authenv/composeconfiguration privately before recreation; restore onlypreviousAuthdevelopconfiguration ifnewcallback fails. Remove onlynewvhost/container for webrollback, never volumes/DB. Unknown conflictingport/vhost/currentprotectedstate => reconcile before mutation.

## Acceptance and validation
Analyze and existing meaningful tests pass plus focusedweb regressions; releasewebbuild succeeds withoutXcode/Android. Container actual nginxconfig validates, root/index/callback and packagedJS return200 fromMac viaHostheader, Authhealth200/settingsGoogleenabled, authorize302 correctGooglecallback and webredirect allowlisted, APIs401 unauthenticated beforeeffect. Protected services/config preserveidentity/startup/mounts (except authorizedauthdevelop recreation). Endpoint runtime remains pending untilreal authenticatedcompatiblecall; nofake login/acceptance.

TDD yes inherited explicit ownerdecision for same manualflow. Test beforeimplementation forchangedwebbehavior; noDBtests outsideVM. Pure infraconfig tests plus actualVMnginxcheck replace meretextassertions. No automaticreviewgate.

## Automation and resume
Use Dockerfile/Compose/config for reproducibledeploy; stage only explicitartifact/configfiles withhashmanifest; temporaryprivateoperational scripts outsideGit where needed. New taskcheckpoint web-local-checkpoint.yaml; evidence implementation-evidence.md and runtime-evidence.md. PublicHTTPS/domain and humanGoogleconsent deferred. No furtherproductdecision pending for boundedchosenroute.

## Delivery evidence
Implementation/build and bounded service deployment complete; see implementation-evidence.md and runtime-evidence.md. Owner browser login acceptance and authenticated endpoint proof pending.

## Proposed repair — authenticated401 after real Google login
Owner error screenshot prompted live diagnosis: missing GoTrue audience/defaultgroup created one confirmed, nonanonymous Google account with emptyaud/role. API correctly rejects it. Proposed bounded operation: apply localauthGoogleoverlay defaults authenticated; privatelysnapshot currentAuthconfig and accountid/aud/role; repair only that account's twoemptyfields via guarded VM-only operationalSQL; preserveUUID/Googleidentity/Brain/email. Rollback restores exacttwofields and Authconfig; no deletion/recreation, credentials/main/NSR/schema/migrations changes. Restart onlyauthdevelop thenweb (cachedupstreamDNS), verifyclaims/config/health/preservedservices, ownerrepeatlogin. RepairSQL source: src/brain-engine/features/infra/web-local/operations/repair-empty-auth-claims.sql.

Localconfig regression RED→GREEN; webinfra9tests and authhelpers22tests pass. Applyingrealaccountrepair needs explicitowner extension because previousscope excluded real-data/databasewrites; pending. No loosening ofapprovedAuth/APIcontract. No repair applied yet.

## Authorized correction + account recreation — supersedes field repair
Owner explicitly authorized defaults correction and deletion of the affected account to recreate/test Google signup. Apply twoAuthdefaults authenticated; privatelysnapshot target/dependentAuthrecords; verify exactsinglemalformedconfirmednonanonymousGoogle account, noassociatedBrain; pauseAuth during guardeddeletion on ownerVMdevelop; startcorrectedAuth and restartwebcachedDNS; verifyabsence/health/preservation. ExistingaccountUUID/identity are intentionallyremoved, futureGooglelogin createsnewaccount. NooldUUIDpreservationpromise; earliermetadatarepairSQL is superseded and mustnotexecute. NoBrain/main/NSR/schema/migration/backend/clientcodechanges. Authorization comes from ownermessage “pode fazer a correção e o que eu pedi” and selected “apague o registro para recriarmos a conta e ver se deu certo”. No further approval needed within that exact scope. RealnewGoogleaccount/token and fourproductendpointpositiveproof remain ownerlogin-dependent.

## Applied owner-authorized Auth correction and test-account deletion
Freshpreflight confirmed exactmalformedGoogleaccount, noBrain, oneidentity and zerosessions. ExecutionusedGoTrueadminAPI harddelete (not eitherpreparedSQLrepair/deletion), preserving Authaudit/transaction/FKbehavior; ownerrequest supersedes priorUUIDpreservation. Privatefulluser/identity/sessionsbackup retained outsideGit. Deployedevelop-only claims patch operations/compose.auth-claims.yaml, bothdefaults authenticated, onlythose2envkeys changed; allothercredentials/DBURL/providers/callbacks preserved. Authrecreated/webrestarted;26othercontainers preserved. Account/identity/sessions/BrainforoldUUID allzero; noactualnewaccountcreatedbyagent. Health and Macweb/Auth/tunnel PASS. Next: ownerfreshGooglelogin createsnewUUID; verifynewclaims and authenticatedproductproof. NoBrainEnginecodechange or productruntimepromotion. Earlierfieldrepairproposal/SQL superseded and neverapplied.

Owner2026-10-09 functionalacceptance: Googleentry andthreelevelhierarchy confirmedbyrealworkspace screenshot; remainingrename/CAS/session andHTTPwireproof notclaimed. See runtime-evidence.md.

Ownerrenameacceptance2026-10-09: reportsall3levels renamed and repeatedparent/childnames; screenshotselectedlayer version2. RealconcurrentCAS/session/HTTPwireproof remainsunclaimed.
