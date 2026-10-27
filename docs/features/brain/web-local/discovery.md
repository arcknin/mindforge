# Discovery — Flutter web local

Owner supersedes paused Android direction: adapt existing Flutter client, build web, publish Docker on indicated VM at web-mind.home.arpa, supply Mac hosts command. Existing dirty branch feat/brain-core-foundation preserved.

Mandatory architecture/stack/OpenAPI sources read in order. Four approved API operations retain contract/implementation states and runtime pending. Previous manual-online local proof41 tests, VM develop Auth/API running, human Google login unproved.

Client currently locks Mac localhost/custom scheme/Keychain, imports dart:io, and API URI construction loses /api prefix. Web requires platform storage/browser navigation/callback reload handling and same-origin proxy path preservation. No new Brain API needed.

VM live inventory on2026-10-09: hostNginx port80 manages home.arpa sites; Mindforge DB/Auth and NSR containers exist; existing network infra_default. Web will add a standalone static/proxy container loopback18088 and one isolated hostNginx vhost. Do not touch DB/NSR/main/Studios.

Google Web OAuth requires HTTPS/public suffix domain, except localhost loopback. home.arpa is unsuitable as Google callback. Owner delegated best current solution after explanation: retain registered localhost11000/callback using MacSSH forward to VM10000; final GoTrue redirect is web-mind.home.arpa/login/callback. Website/API do not require tunnel. Public-domain HTTPS deferred; no invented domain/provider.

Sources: ../manual-online/runtime-service-evidence.md; src/mindforge-client/lib/app/app_config.dart; contracts/openapi/registry.yaml; live docker inspect (sanitized) and nginx -T; https://developers.google.com/identity/protocols/oauth2/web-server.

## Owner-reported authenticated401 — 2026-10-09
Screenshot shows twoAPI401 and session rejected. LiveAuthdevelop has no GOTRUE_JWT_AUD or GOTRUE_JWT_DEFAULT_GROUP_NAME; read-only aggregate of auth.users finds exactly1account, blankaud/role, confirmed/nonanonymous; identityGooglecount1. API fixedverifier requires authenticatedaud/role. Cause is producerconfiguration missing, not a reason to weakenverifier. ExplicitGoTrue2.196 configuration source has no defaults for thosefields.

Prepared localGoogledevelopoverlay with requireddefaults and regressiontest (RED missingdefaults thenGREEN). Existingaccount requires narrow aud/role repair preservingUUID/identity; schema/main/NSR unchanged. That real-account repair lies outside priorweb spec non-goal, so apply remains pending owner decision; no runtimechange in this diagnosis.
