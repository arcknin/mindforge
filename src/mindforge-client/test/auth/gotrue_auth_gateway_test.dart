import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gotrue/gotrue.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/auth/auth_secret_store.dart';
import 'package:mindforge_client/auth/gotrue_auth_gateway.dart';

void main() {
  group('GoTrueAuthGateway', () {
    late _MemorySecretStore store;
    late _FakeGoTruePort authApi;
    late List<Uri> launchedUrls;
    late DateTime now;
    late GoTrueAuthGateway gateway;

    setUp(() {
      store = _MemorySecretStore();
      authApi = _FakeGoTruePort();
      launchedUrls = <Uri>[];
      now = DateTime.utc(2026, 10, 8, 12);
      gateway = _gateway(
        store: store,
        authApi: authApi,
        launchedUrls: launchedUrls,
        now: () => now,
      );
    });

    tearDown(() => gateway.dispose());

    test(
      'accepts one exact pending PKCE callback and strips Google tokens',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();

        expect(launchedUrls, hasLength(1));
        expect(
          launchedUrls.single,
          Uri.parse('https://auth.example/v1/auth/authorize'),
        );
        expect(gateway.state.phase, AuthPhase.signingIn);

        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=one-time-code'),
        );

        expect(authApi.exchangedCodes, <String>['one-time-code']);
        expect(
          gateway.state.phase,
          AuthPhase.authenticated,
          reason: gateway.state.message,
        );
        expect(gateway.state.userId, 'user-a');
        expect(store.pendingAttempt, isNull);
        expect(store.pkceValues, isEmpty);
        expect(store.sessionJson, isNot(contains('google-access-secret')));
        expect(store.sessionJson, isNot(contains('google-refresh-secret')));

        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=one-time-code'),
        );
        expect(authApi.exchangedCodes, <String>['one-time-code']);
      },
    );

    test(
      'persists web PKCE material before same-origin HTTP navigation and accepts callback after reload',
      () async {
        gateway.dispose();
        authApi = _FakeGoTruePort();
        authApi.signInUrl = Uri.parse(
          'http://web-mind.home.arpa/api/v1/auth/authorize?provider=google',
        );
        authApi.beforeCreateGoogleSignInUrl = () =>
            store.setItem(key: 'sdk-verifier', value: 'persisted-verifier');
        var navigationCalled = false;
        gateway = GoTrueAuthGateway(
          authApi: authApi,
          secretStore: store,
          callbackUri: Uri.parse('http://web-mind.home.arpa/login/callback'),
          authorizationOrigin: Uri.parse(
            'http://web-mind.home.arpa/api/v1/auth',
          ),
          now: () => now,
          launchExternalBrowser: (uri) async {
            navigationCalled = true;
            expect(uri, authApi.signInUrl);
            expect(store.pendingAttempt, isNotNull);
            expect(
              await store.getItem(key: 'sdk-verifier'),
              'persisted-verifier',
            );
            return true;
          },
        );

        await gateway.initialize();
        await gateway.beginGoogleSignIn();

        expect(navigationCalled, isTrue);
        expect(authApi.redirectUris, <String>[
          'http://web-mind.home.arpa/login/callback',
        ]);
        await gateway.handleCallback(
          Uri.parse(
            'http://web-mind.home.arpa/login/callback?code=returned-after-reload',
          ),
        );

        expect(authApi.exchangedCodes, <String>['returned-after-reload']);
        expect(
          gateway.state.phase,
          AuthPhase.authenticated,
          reason: gateway.state.message,
        );
      },
    );

    test(
      'clears a successful attempt before accepting later refresh events',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();
        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=login-code'),
        );

        authApi.emit(
          GoTrueEvent(
            GoTrueEventKind.tokenRefreshed,
            _session(accessToken: 'event-access'),
          ),
        );
        await pumpEventQueue();

        final credential = await gateway.credential();
        expect(credential.accessToken, 'event-access');
        expect(store.sessionJson, contains('event-access'));
        expect(store.pkceValues, isEmpty);
      },
    );

    test(
      'rejects callbacks outside the registered scheme, host, and path',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();

        await gateway.handleCallback(
          Uri.parse('mindforge-dev://attacker/callback?code=stolen'),
        );
        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/other?code=stolen'),
        );

        expect(authApi.exchangedCodes, isEmpty);
        expect(gateway.state.phase, AuthPhase.signedOut);
        expect(store.pendingAttempt, isNull);
        expect(store.pkceValues, isEmpty);
      },
    );

    test(
      'does not exchange a callback when there is no pending sign-in',
      () async {
        await gateway.initialize();
        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=unsolicited'),
        );

        expect(authApi.exchangedCodes, isEmpty);
        expect(gateway.state.phase, AuthPhase.signedOut);
      },
    );

    test('rejects expired attempts without exchanging the code', () async {
      await gateway.initialize();
      await gateway.beginGoogleSignIn();
      now = now.add(const Duration(minutes: 11));

      await gateway.handleCallback(
        Uri.parse('mindforge-dev://auth/callback?code=expired-code'),
      );

      expect(authApi.exchangedCodes, isEmpty);
      expect(gateway.state.phase, AuthPhase.signedOut);
      expect(store.pendingAttempt, isNull);
      expect(store.pkceValues, isEmpty);
    });

    test('restores a pending callback across a process restart', () async {
      store.pendingAttempt = jsonEncode(<String, Object?>{
        'id': 'previous-process-attempt',
        'generation': 12,
        'expires_at': now
            .add(const Duration(minutes: 4))
            .millisecondsSinceEpoch,
      });
      store.pkceValues['sdk-verifier'] = 'pending-verifier';

      await gateway.initialize();
      expect(gateway.state.phase, AuthPhase.signingIn);
      await gateway.handleCallback(
        Uri.parse('mindforge-dev://auth/callback?code=returned-after-restart'),
      );

      expect(authApi.exchangedCodes, <String>['returned-after-restart']);
      expect(gateway.state.phase, AuthPhase.authenticated);
    });

    test('treats OAuth denial as cancellation and clears the verifier', () async {
      await gateway.initialize();
      await gateway.beginGoogleSignIn();

      await gateway.handleCallback(
        Uri.parse(
          'mindforge-dev://auth/callback?error=access_denied&error_description=cancelled',
        ),
      );

      expect(authApi.exchangedCodes, isEmpty);
      expect(gateway.state.phase, AuthPhase.signedOut);
      expect(gateway.state.message, contains('cancel'));
      expect(store.pendingAttempt, isNull);
      expect(store.pkceValues, isEmpty);
    });

    test('rejects a callback when the PKCE verifier is missing', () async {
      await gateway.initialize();
      await gateway.beginGoogleSignIn();
      authApi.exchangeError = const AuthPKCEGrantCodeExchangeError(
        'Verifier missing.',
      );

      await gateway.handleCallback(
        Uri.parse('mindforge-dev://auth/callback?code=without-verifier'),
      );

      expect(gateway.state.phase, AuthPhase.signedOut);
      expect(gateway.state.userId, isNull);
      expect(store.sessionJson, isNull);
      expect(store.pendingAttempt, isNull);
      expect(store.pkceValues, isEmpty);
    });

    test('keeps a restored session during a transient Auth outage', () async {
      final previousSession = _session();
      store.sessionJson = jsonEncode(previousSession.toJson());
      authApi.restoreResult = previousSession;
      authApi.getSessionError = AuthRetryableFetchException(message: 'offline');

      await gateway.initialize();

      expect(gateway.state.phase, AuthPhase.authenticated);
      expect(gateway.state.userId, 'user-a');
      expect(store.sessionJson, isNotNull);
      expect(gateway.state.message, isNotEmpty);
    });

    test(
      'preserves Keychain session when restore refresh fails and retries explicitly',
      () async {
        final previousSession = _session();
        store.sessionJson = jsonEncode(previousSession.toJson());
        authApi.restoreError = AuthRetryableFetchException(message: 'offline');

        await gateway.initialize();

        expect(gateway.state.phase, AuthPhase.unavailable);
        expect(store.sessionJson, isNotNull);
        expect(authApi.signOutCalls, 0);

        authApi.restoreError = null;
        authApi.restoreResult = previousSession;
        await gateway.initialize();

        expect(gateway.state.phase, AuthPhase.authenticated);
        expect(gateway.state.userId, 'user-a');
        expect(store.sessionJson, isNotNull);
      },
    );

    test(
      'clears a stored session after definitive rejection during restore',
      () async {
        store.sessionJson = jsonEncode(_session().toJson());
        authApi.restoreError = AuthApiException(
          'Invalid refresh token',
          statusCode: '400',
          code: 'refresh_token_not_found',
        );

        await gateway.initialize();

        expect(gateway.state.phase, AuthPhase.signedOut);
        expect(gateway.state.message, contains('rejeitada'));
        expect(store.sessionJson, isNull);
        expect(authApi.signOutCalls, 1);
      },
    );

    test('clears malformed stored session JSON', () async {
      store.sessionJson = '{invalid-json';

      await gateway.initialize();

      expect(gateway.state.phase, AuthPhase.signedOut);
      expect(store.sessionJson, isNull);
      expect(authApi.signOutCalls, 1);
    });

    test(
      'clears a restored session after definitive credential rejection',
      () async {
        final previousSession = _session();
        store.sessionJson = jsonEncode(previousSession.toJson());
        authApi.restoreResult = previousSession;
        authApi.getSessionError = AuthApiException(
          'Invalid refresh token',
          statusCode: '400',
          code: 'refresh_token_not_found',
        );

        await gateway.initialize();

        expect(gateway.state.phase, AuthPhase.signedOut);
        expect(store.sessionJson, isNull);
        expect(authApi.signOutCalls, 1);
      },
    );

    test(
      'logout clears local state and Keychain data when Auth sign-out fails',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();
        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=login-code'),
        );
        authApi.signOutError = AuthRetryableFetchException(message: 'offline');

        await gateway.signOut();

        expect(gateway.state.phase, AuthPhase.signedOut);
        expect(gateway.state.userId, isNull);
        expect(store.sessionJson, isNull);
        expect(store.pendingAttempt, isNull);
        expect(store.pkceValues, isEmpty);
        expect(gateway.state.message, isNotEmpty);
      },
    );

    test(
      'coalesces concurrent 401 refreshes for the same session generation',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();
        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=login-code'),
        );
        final rejected = await gateway.credential();
        final refreshResult = Completer<Session?>();
        authApi.refreshResult = refreshResult.future;

        final first = gateway.refreshAfterUnauthorized(rejected);
        final second = gateway.refreshAfterUnauthorized(rejected);
        await Future<void>.delayed(Duration.zero);
        expect(authApi.refreshCalls, 1);

        refreshResult.complete(_session(accessToken: 'fresh-access'));
        final credentials = await Future.wait(<Future<AuthCredential>>[
          first,
          second,
        ]);

        expect(
          credentials.map((credential) => credential.accessToken).toSet(),
          <String>{'fresh-access'},
        );
        expect(
          credentials.map((credential) => credential.generation).toSet(),
          <int>{rejected.generation},
        );
      },
    );

    test(
      'discards a callback that completes after the sign-in generation is cancelled',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();
        final exchange = Completer<Session?>();
        authApi.exchangeFuture = exchange.future;

        final callback = gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=stale-code'),
        );
        await pumpEventQueue();
        await gateway.cancelGoogleSignIn();

        exchange.complete(_session(accessToken: 'stale-access'));
        await callback;

        expect(gateway.state.phase, AuthPhase.signedOut);
        expect(gateway.state.userId, isNull);
        expect(store.sessionJson, isNull);
        expect(authApi.signOutCalls, 1);
      },
    );

    test(
      'does not apply a refresh response after a newer sign-out generation',
      () async {
        await gateway.initialize();
        await gateway.beginGoogleSignIn();
        await gateway.handleCallback(
          Uri.parse('mindforge-dev://auth/callback?code=login-code'),
        );
        final rejected = await gateway.credential();
        final refresh = Completer<Session?>();
        authApi.refreshResult = refresh.future;

        final pending = gateway.refreshAfterUnauthorized(rejected);
        await pumpEventQueue();
        await gateway.signOut();
        refresh.complete(_session(accessToken: 'late-refresh'));

        await expectLater(pending, throwsA(isA<StaleAuthOperationException>()));
        expect(gateway.state.phase, AuthPhase.signedOut);
        expect(gateway.state.userId, isNull);
        expect(store.sessionJson, isNull);
      },
    );
  });
}

GoTrueAuthGateway _gateway({
  required _MemorySecretStore store,
  required _FakeGoTruePort authApi,
  required List<Uri> launchedUrls,
  required DateTime Function() now,
}) {
  return GoTrueAuthGateway(
    authApi: authApi,
    secretStore: store,
    callbackUri: Uri.parse('mindforge-dev://auth/callback'),
    authorizationOrigin: Uri.parse('https://auth.example/v1/auth'),
    now: now,
    launchExternalBrowser: (uri) async {
      launchedUrls.add(uri);
      return true;
    },
  );
}

Session _session({
  String userId = 'user-a',
  String accessToken = 'mindforge-access',
}) {
  return Session(
    accessToken: accessToken,
    refreshToken: 'mindforge-refresh',
    tokenType: 'bearer',
    expiresIn: 3600,
    providerToken: 'google-access-secret',
    providerRefreshToken: 'google-refresh-secret',
    user: User(
      id: userId,
      appMetadata: const <String, dynamic>{'provider': 'google'},
      userMetadata: const <String, dynamic>{'name': 'Mindforge User'},
      aud: 'authenticated',
      createdAt: '2026-10-08T12:00:00Z',
    ),
  );
}

class _MemorySecretStore implements AuthSecretStore {
  String? sessionJson;
  String? pendingAttempt;
  final Map<String, String> pkceValues = <String, String>{};

  @override
  Future<void> clearAuthMaterial() async {
    sessionJson = null;
    pendingAttempt = null;
    pkceValues.clear();
  }

  @override
  Future<void> clearLegacyAuthMaterial() async {}

  @override
  Future<void> clearPkce() async => pkceValues.clear();

  @override
  Future<void> clearOAuthMaterial() async {
    await writePendingAttempt(null);
    await clearPkce();
  }

  @override
  Future<String?> getItem({required String key}) async => pkceValues[key];

  @override
  Future<String?> readPendingAttempt() async => pendingAttempt;

  @override
  Future<String?> readSession() async => sessionJson;

  @override
  Future<void> removeItem({required String key}) async =>
      pkceValues.remove(key);

  @override
  Future<void> setItem({required String key, required String value}) async {
    pkceValues[key] = value;
  }

  @override
  Future<void> writePendingAttempt(String? value) async =>
      pendingAttempt = value;

  @override
  Future<void> writeSession(String? value) async => sessionJson = value;
}

class _FakeGoTruePort implements GoTruePort {
  final StreamController<GoTrueEvent> _events =
      StreamController<GoTrueEvent>.broadcast();
  final List<String> exchangedCodes = <String>[];
  final List<String> redirectUris = <String>[];
  Uri signInUrl = Uri.parse('https://auth.example/v1/auth/authorize');
  Future<void> Function()? beforeCreateGoogleSignInUrl;
  Session? currentSession;
  Session? restoreResult;
  Session? exchangeResult = _session();
  Future<Session?>? exchangeFuture;
  Future<Session?>? refreshResult;
  Object? getSessionError;
  Object? restoreError;
  Object? exchangeError;
  Object? signOutError;
  int refreshCalls = 0;
  int signOutCalls = 0;

  void emit(GoTrueEvent event) => _events.add(event);

  @override
  Stream<GoTrueEvent> get events => _events.stream;

  @override
  Future<Uri> createGoogleSignInUrl(String redirectUri) async {
    redirectUris.add(redirectUri);
    await beforeCreateGoogleSignInUrl?.call();
    return signInUrl;
  }

  @override
  Session? get current => currentSession;

  @override
  Future<Session?> exchangeCodeForSession(String code) async {
    exchangedCodes.add(code);
    final error = exchangeError;
    if (error != null) throw error;
    final result =
        await (exchangeFuture ?? Future<Session?>.value(exchangeResult));
    currentSession = result;
    if (result != null) {
      _events.add(GoTrueEvent(GoTrueEventKind.signedIn, result));
    }
    return result;
  }

  @override
  Future<Session?> getSession() async {
    final error = getSessionError;
    if (error != null) throw error;
    return currentSession ?? restoreResult;
  }

  @override
  Future<Session?> refreshSession() {
    refreshCalls++;
    final future =
        refreshResult ??
        Future<Session?>.value(_session(accessToken: 'fresh-access'));
    return future.then((session) {
      currentSession = session;
      return session;
    });
  }

  @override
  Future<Session?> restoreSession(String json) async {
    final error = restoreError;
    if (error != null) throw error;
    currentSession =
        restoreResult ??
        Session.fromJson(jsonDecode(json) as Map<String, dynamic>);
    return currentSession;
  }

  @override
  Future<void> signOutLocal() async {
    signOutCalls++;
    final error = signOutError;
    currentSession = null;
    if (error != null) throw error;
  }

  @override
  void startAutoRefresh() {}

  @override
  void stopAutoRefresh() {}

  @override
  void dispose() {
    unawaited(_events.close());
  }
}
