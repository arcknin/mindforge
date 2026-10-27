# Web local — deployed services and Mac HTTP proof

Applied2026-10-09 to owner-indicated VM under direct web/Docker deployment authorization. Private baseline and exact Authbackup/rollback retained in /etc/mindforge/web-local (directory700/files600); values omitted.

Release /opt/mindforge/releases/web-local-20261009-r1;40transportfiles/hashmanifest verified before application. Docker project mindforge-web-develop, container mindforge-web-develop-web-1. Imageid sha256:f861d950f3e647bd428359cb17f68a6ad9e8bd8f4cda4101a3ce6ec99bf4154e; healthy, running,0restarts. Bind127.0.0.1:18088→80; no volumes or credentials in new image. Officialbase nginx1.30.5-alpine3.24 digest8f84ed99befc3891b8f329c5c202785278a2cfb7c25107d57fb2a134a3117433 pulled/verified on VM.

Authdevelop selectively recreated after rendering/validating exact environment. Only GOTRUE_SITE_URL and GOTRUE_URI_ALLOW_LIST changed. SITE_URL=http://web-mind.home.arpa; exact newcallback=http://web-mind.home.arpa/login/callback; previous nativecallback preserved. DB/JWT/Googlecredentials/externalAPIURL/Googlecallback unchanged. Provider health200; noSQL/migration/newDB/role/promotion or user/Brain testwrites.

Nginx container effectiveconfig test and host syntax/reload PASS. Newhostvhost /etc/nginx/sites-available/web-mind.home.arpa, enabled by symlink. InitialLAN ACL blocked Mac because VirtualBox presents it as10.0.2.2: actualheldTCPconnection and ss proved peer10.0.2.2 on VM10.0.2.15. Added only that gateway alongside127.0.0.1 and192.168.0.0/24, followed by denyall. Versioned correction /opt/mindforge/releases/web-local-vhost-20261009-r2/web-mind.home.arpa.conf; live/sourceSHA25670722a6f0fa39640ad110db068c0c34f9e16575753cb81f6638a496320422e19. No firewall/DNS/publicdomain changes.

After correction actualMac LAN HTTP requests with Host=web-mind.home.arpa: /, /login/callback, /flutter_bootstrap.js, /main.dart.js, /auth/health and /auth/settings return200. Downloaded mainJS hash matches builtartifact. AuthGoogle enabled. VMalso served localCanvasKit wasm200/hashmatched; fourAPIroutes GET/PUT/POST/PATCH unauthenticated401; admin/signup/outsideAPIpaths404. /auth/authorize Google302, registeredGooglecallback stillhttp://127.0.0.1:11000/callback. Redirect Location/state/codes omitted.

Preservation PASS after application and NATcorrection:26othercontainers IDs/starttimes/runningstates/mountcontents unchanged (Docker mountarray order normalized);32previousNginxfiles hashunchanged. CoversmainAuth/DB, API, NSR and Studios. Onlyauthdevelop was intentionally recreated; webcontainer added.

MacSSH callback tunnel now open only127.0.0.1:11000→VM127.0.0.1:10000, strict hostkey. Owner Mac callbackhealth200. Session89612 is runtime-specific, not an autostart mechanism. Tunnel needs reopen after process/Macrestart; website/API are directLAN and neednoSSH.

```sh
ssh -N -T -o ExitOnForwardFailure=yes \
  -L 127.0.0.1:11000:127.0.0.1:10000 \
  -p 2222 kali@192.168.0.111
```

Hosts is deliberately not modified automatically; ownerrequestedcode. Map192.168.0.111 web-mind.home.arpa, thenopenhttp://web-mind.home.arpa.

Services/static/proxy/Googleauthorizationredirect/negativeAPI proved; humanGooglecodeexchange, sessionrestore in realbrowser and authenticatedBrainactions remain pending. No browserQA or realGoogleaccount actions. Do not promote fourendpoint runtime_status until corresponding realauthorizedproof. HTTPorigin is privateLAN; publicHTTPS/domain evolution separate.

Rollback: remove onlynewhostsymlink/site and nginxsyntaxcheck/reload, stoponlymindforge-web-develop composeproject; restoreprivateAuthrollback throughsavedcommands if revertingwebcallback. Neverremovevolumes/DB/main/NSR. Preserve privatebackups and immutable releases for recovery.

## Authenticated401 diagnosis and proposed repair
Owner screenshot shows actualpostOAuth session rejection. Live read-only authusers aggregate: exactlyone account, blankaud/role, confirmed, nonanonymous and Googleidentity. Authenvironment omitted defaultaud/group required by API. Local overlayfixed and regressionRED→GREEN;9webinfra/22authhelpertestsPASS. VM preparation validated exactonecandidate, privateaccountid/aud/role and fullAuthconfigbackup/rollback, fullComposerender preserving allotherenv/volume, and PREPARE/DEALLOCATE of UPDATE in read-only transaction. NoSQLupdate or servicechange executed. Operatorplan /etc/mindforge/web-local/auth-claims-repair is private. Ownerapprovalrequested for actualaccounttwofieldrepair plusAuthdefaults/restartAuthdevelop/web only. Noverifierloosened, userdeleted/recreated, email/Googleidentity/Brainmodified or endpointstatepromoted.

## Applied defaults correction + deletion for owner recreation
Ownerexplicitly authorized correction and removal of the malformed Google test account on2026-10-09. FreshVMpreflight target1, Brain0, Googleidentity1,sessions0. Privatefullaccount/identity/sessionbackup stored0600 under /etc/mindforge/web-local/auth-claims-repair. AdministrativeGET checkedexacttarget/blankclaims/nonanonymous/Google; short-lived serveradministrativeJWT retainedonlyinmemory. GoTrueadminDELETE harddelete returned200; read-onlypostcheck confirms olduser/identity/session/Braincountsall0. NoGoogleaccountoutsideMindforge removed, noaccountautomaticallyrecreated.

Appliedpublicpatch /opt/mindforge/releases/auth-claims-20261009-r1/compose.auth-claims.yaml; currentAuthcmd recorded incanonical /etc/mindforge/web-local/commands.private.json. Twoaddedenvkeys GOTRUE_JWT_AUD=authenticated and GOTRUE_JWT_DEFAULT_GROUP_NAME=authenticated; allotherAuthenvexactlypreserved. Webrestarted torefreshDockerDNS. Auth/webhealthy0restarts;26othercontainers IDs/starttimes/mounts/runningpreserved includingBrainAPI/main/NSR/Studios. SQLfieldrepair/deletion notexecuted; onlydiagnosticSQL read-only and officialAuthAPI deletionused.

MacHTTP200 root/callback/Authhealth/settings;Googleenabled; APIwithouttoken401; callbacktunnel11000health200. Root9webinfratests/22AuthhelpertestsPASS; sourceclaimspatchtest checks bothproduceroverlay andappliedpatch. NewrealGooglelogin and approvedendpointpositiveproof remainpending. Refresh/newtab thenGooglelogin; newUUID expected. Proofprivate recreation-proof.json has sanitisedcounts/config, noUUID/email/secret.

## Owner functional acceptance — 2026-10-09
After correctedAuthdefaults and requestedtestaccountdeletion, owner reports “fununciando” and supplies a screenshot of the real Web workspace showing Trabalho → Icarus Studio → NSR, threehierarchylevels and selectedlayer version1. This is owner-provided functional evidence of Googleentry and manualhierarchycreation/observation afterrecreation, rather than agent browserQA. Attachment reference /var/folders/bj/kmbmd31n2ml0rtyj34qx0_ww0000gn/T/codex-clipboard-be7cb44e-457b-4ed4-a120-c7a36bc08714.png (temporaryownerimage, not a durablefiledependency).

Do not extrapolate this screenshot to rename/CAS409, refresh/logout, isolationbetweenaccounts or exactHTTPresponsecontractproof. NoadditionalVMcalls, accounts, mutations or tests performed for this acceptance recording. OpenAPIendpoint runtime_status remainsunchanged; futurewire-levelevidence shouldidentify actualmethod/status/sanitizedresponse. Nextowneracceptance: rename andconflict handling when requested; no automaticnewscope.

## Owner rename and duplicate-name acceptance — 2026-10-09
Owner reports “renomeados testei dois nomes iguais . tudo ai foi renomeado” and supplies actualWebworkspace screenshot. Visiblethreeleveltree Renomeado → Icarus Studio → Icarus Studio; selectedrootlayer showsversion2 andhierarchyretained. Accept owner-reportedrename ofallthreevisiblelayers and duplicate names acrossparent/child. This extends previouslogin/create/observefunctionalacceptance. It doesnot demonstrate concurrent stale-version409, same-parentduplicatenames, sessionlifecycle/isolation or exactHTTPmethod/status/bodytrace. NoagentbrowserQA, VMmutation or extraaccountaction performed. RegistryHTTPproofstatuses unchanged. Temporaryimage reference codex-clipboard-9ffcf941-ff86-49e6-a2e6-105a488e04ef.png.
