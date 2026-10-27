# Web local — implementation and local proof

Owner replaced paused Android direction with Flutter web build and Docker VM deployment. Source task spec-01-main.md governs this bounded adaptation. All previous dirty work preserved; no commit/push.

Client now composes native Keychain or browser tab sessionStorage through conditional imports, navigates OAuth in the same tab, restores persisted PKCE on callback reload, clears callback query/fragment, and preserves /auth + /api prefixes. Browser storage access is lazy; unavailable storage follows sanitized failure handling. Flutter starts at application root after callback capture.

Versions retain the approved positive Int64 range through BigInt, lexical preservation of large JSON integer tokens without rewriting strings, collision-free internal markers, and exact numeric expected_version serialization. Backend/OpenAPI behavior unchanged.

Worker local proof:54 Flutter tests PASS, flutter analyze clean, clean releaseweb build PASS. Root inspected composition/config/PKCE/gateway/JSON/HTTP source. Build command:

```sh
cd /Users/arcknin/Desktop/mindforge/src/mindforge-client
flutter build web --release --no-web-resources-cdn --pwa-strategy=none \
  --dart-define=MINDFORGE_ENV=develop \
  --dart-define=MINDFORGE_WEB_ORIGIN=http://web-mind.home.arpa
```

CanvasKit and normalUI Roboto font files are local. Bootstrap does not configure a service worker; generatedworker is empty. Optional engine fallback for missing Unicode glyphs can still use fonts.gstatic.com. Font Apache license/NOTICE retained in source; generated NOTICES contains Apache license text.

Infra leaf implemented only additive web-local subtree: pinned official Nginx Dockerfile, readonly container, loopback18088, existingnetwork, fixedupstreams/allowlistedroutes/methods, logs disabled/sanitized and nocache. Eight tests/render passed; root repeated eight after NATvhost correction. ActualVM build/nginx validation described in runtime-evidence.md. No localDocker daemon, DB, SQL or disposabletestinfra.

Package contains40files, compressed10,803,454bytes; noGoogleClientSecret matched in anyfile. Transport archiveSHA256 `258b0772e523b10ce98ef9c6eb7acae667becca72f8f369ce64c6499b055035f`; main.dart.jsSHA256 `dbf844a3acfe2cfc88c61ddf54c350e9c22c03423be963ed4598d896c47fb493`. Original archive is immutable; later hostvhost NATcorrection is a separately versioned artifact (runtime evidence).

No browser/visual QA, realGoogle login/codeexchange or authenticatedBrainoperation performed. Productendpoint runtime remains pending.
