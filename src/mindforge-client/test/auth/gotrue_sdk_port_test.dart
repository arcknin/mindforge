import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gotrue/gotrue.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/auth/auth_secret_store.dart';
import 'package:mindforge_client/auth/gotrue_auth_gateway.dart';
import 'package:mindforge_client/auth/gotrue_sdk_port.dart';
import 'package:mindforge_client/auth/owned_pkce_storage.dart';

void main() {
  test(
    'PKCE storage ownership prevents an older SDK from deleting a newer verifier',
    () async {
      final store = _MemorySecretStore();
      final oldClient = OwnedPkceStorage(store);
      final newClient = OwnedPkceStorage(store);

      await oldClient.setItem(
        key: 'supabase.auth.token-code-verifier',
        value: 'old-verifier',
      );
      await newClient.setItem(
        key: 'supabase.auth.token-code-verifier',
        value: 'new-verifier',
      );

      expect(
        await oldClient.getItem(key: 'supabase.auth.token-code-verifier'),
        'old-verifier',
      );
      await oldClient.removeItem(key: 'supabase.auth.token-code-verifier');
      expect(
        await newClient.getItem(key: 'supabase.auth.token-code-verifier'),
        'new-verifier',
      );

      await newClient.removeItem(key: 'supabase.auth.token-code-verifier');
      expect(
        await store.getItem(key: 'supabase.auth.token-code-verifier'),
        isNull,
      );
    },
  );

  test(
    'serializes the PKCE compare-and-delete against a newer verifier write',
    () async {
      final store = _InterleavingSecretStore();
      final oldClient = OwnedPkceStorage(store);
      final newClient = OwnedPkceStorage(store);
      const key = 'supabase.auth.token-code-verifier';

      await oldClient.setItem(key: key, value: 'old-verifier');
      store.delayNextRead = true;
      final oldRemove = oldClient.removeItem(key: key);
      await store.readStarted.future;
      final newWrite = newClient.setItem(key: key, value: 'new-verifier');
      expect(store.values[key], 'old-verifier');

      store.releaseRead.complete();
      await Future.wait(<Future<void>>[oldRemove, newWrite]);
      expect(await newClient.getItem(key: key), 'new-verifier');
    },
  );

  test('uses only API Auth paths with the real GoTrue SDK traffic', () async {
    final requests = <http.BaseRequest>[];
    final store = _MemorySecretStore();
    await store.setItem(
      key: 'supabase.auth.token-code-verifier',
      value: 'v' * 43,
    );
    final delegate = MockClient((request) async {
      requests.add(request);
      if (request.url.path == '/v1/auth/token') {
        return http.Response(
          jsonEncode(_sessionJson()),
          200,
          headers: const {'content-type': 'application/json'},
        );
      }
      if (request.url.path == '/v1/auth/user') {
        return http.Response(
          jsonEncode(_userJson()),
          200,
          headers: const {'content-type': 'application/json'},
        );
      }
      return http.Response('{}', 404);
    });
    final port = GoTrueSdkPort(
      apiUrl: Uri.parse('https://api.example.test'),
      callbackUri: Uri.parse('mindforge-dev://auth/callback'),
      secretStore: store,
      httpClient: delegate,
    );

    final signIn = await port.createGoogleSignInUrl(
      'mindforge-dev://auth/callback',
    );
    expect(signIn.origin, 'https://api.example.test');
    expect(signIn.path, '/v1/auth/authorize');
    expect(signIn.queryParameters['provider'], 'google');
    expect(signIn.queryParameters['flow_type'], 'pkce');

    final session = await port.exchangeCodeForSession('one-time-code');
    expect(session, isNotNull);
    expect(session!.user.id, '11111111-1111-4111-8111-111111111111');
    expect(requests.map((request) => request.url.path), <String>[
      '/v1/auth/token',
      '/v1/auth/user',
    ]);
    expect(requests.first.url.queryParameters, <String, String>{
      'grant_type': 'pkce',
    });
    expect(requests.first, isA<http.Request>());
    expect((requests.first as http.Request).body, contains('one-time-code'));

    port.dispose();
  });

  test(
    'retains GoTrue single-flight refresh behavior on real API traffic',
    () async {
      final store = _MemorySecretStore();
      var refreshRequests = 0;
      final port = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: store,
        httpClient: MockClient((request) async {
          if (request.url.path == '/v1/auth/token') {
            refreshRequests++;
            return http.Response(
              jsonEncode(_sessionJsonForAccess('a', 'access-a-rotated')),
              200,
            );
          }
          if (request.url.path == '/v1/auth/user') {
            return http.Response(jsonEncode(_userJsonFor('a')), 200);
          }
          return http.Response('{}', 404);
        }),
      );
      await port.restoreSession(jsonEncode(_sessionJsonFor('a')));

      final refreshed = await Future.wait(<Future<Session?>>[
        port.refreshSession(),
        port.refreshSession(),
      ]);

      expect(refreshRequests, 1);
      expect(refreshed, everyElement(isNotNull));
      expect(refreshed.map((session) => session!.user.id).toSet(), <String>{
        '11111111-1111-4111-8111-111111111111',
      });
      expect(refreshed.map((session) => session!.accessToken).toSet(), <String>{
        'access-a-rotated',
      });
      port.dispose();
    },
  );

  test(
    'rejects a refresh that changes the authenticated account and preserves the old session',
    () async {
      final store = _MemorySecretStore();
      var refreshCount = 0;
      final port = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: store,
        httpClient: MockClient((request) async {
          if (request.url.path == '/v1/auth/token') {
            refreshCount++;
            final session = refreshCount == 1
                ? _sessionJsonFor('b')
                : _sessionJsonForAccess('a', 'access-a-rotated');
            return http.Response(jsonEncode(session), 200);
          }
          if (request.url.path == '/v1/auth/user') {
            final access = request.headers['authorization'];
            return http.Response(
              jsonEncode(
                access == 'Bearer access-a' ||
                        access == 'Bearer access-a-rotated'
                    ? _userJsonFor('a')
                    : _userJsonFor('b'),
              ),
              200,
            );
          }
          if (request.url.path == '/v1/auth/logout') {
            return http.Response('', 204);
          }
          return http.Response('{}', 404);
        }),
      );
      await port.restoreSession(jsonEncode(_sessionJsonFor('a')));

      await expectLater(
        port.refreshSession(),
        throwsA(
          isA<AuthApiException>().having(
            (error) => error.statusCode,
            'statusCode',
            '503',
          ),
        ),
      );
      expect(port.current?.user.id, '11111111-1111-4111-8111-111111111111');

      final recovered = await port.refreshSession();
      expect(recovered?.user.id, '11111111-1111-4111-8111-111111111111');
      expect(recovered?.accessToken, 'access-a-rotated');
      expect(refreshCount, 2);
      port.dispose();
    },
  );

  test(
    'requires remote user validation before restoring a local session',
    () async {
      final store = _MemorySecretStore();
      final local = _sessionJson();
      final requests = <String>[];
      final port = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: store,
        httpClient: MockClient((request) async {
          requests.add(request.url.path);
          return http.Response(jsonEncode(_userJson()), 200);
        }),
      );

      final restored = await port.restoreSession(jsonEncode(local));
      expect(restored, isNotNull);
      expect(requests, <String>['/v1/auth/user']);
      port.dispose();
    },
  );

  test(
    'preserves transient validation errors and distinguishes definitive rejection',
    () async {
      final store = _MemorySecretStore();
      final transientPort = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: store,
        httpClient: MockClient((request) async {
          return http.Response(
            '{"code":"auth_unavailable","error_code":"auth_unavailable","message":"unavailable","request_id":"r"}',
            503,
          );
        }),
      );
      await expectLater(
        transientPort.restoreSession(jsonEncode(_sessionJson())),
        throwsA(
          isA<AuthRetryableFetchException>().having(
            (error) => error.statusCode,
            'statusCode',
            '503',
          ),
        ),
      );
      transientPort.dispose();

      final definitivePort = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: store,
        httpClient: MockClient((request) async {
          return http.Response(
            '{"code":"unauthenticated","error_code":"unauthenticated","message":"unauthenticated","request_id":"r"}',
            401,
          );
        }),
      );
      await expectLater(
        definitivePort.restoreSession(jsonEncode(_sessionJson())),
        throwsA(
          isA<AuthApiException>().having(
            (error) => error.statusCode,
            'statusCode',
            '401',
          ),
        ),
      );
      definitivePort.dispose();
    },
  );

  test(
    'does not emit an authenticated event before remote validation succeeds',
    () async {
      final events = <GoTrueEvent>[];
      final port = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: _MemorySecretStore(),
        httpClient: MockClient((request) async {
          return http.Response(
            '{"code":"auth_unavailable","error_code":"auth_unavailable","message":"unavailable","request_id":"r"}',
            503,
          );
        }),
      );
      final subscription = port.events.listen(events.add);

      await expectLater(
        port.restoreSession(jsonEncode(_sessionJson())),
        throwsA(isA<AuthRetryableFetchException>()),
      );
      await pumpEventQueue();

      expect(events, isEmpty);
      await subscription.cancel();
      port.dispose();
    },
  );

  test(
    'does not let a late real-SDK validation from account A replace account B',
    () async {
      final store = _MemorySecretStore();
      final aTokenStarted = Completer<void>();
      final releaseAToken = Completer<void>();
      final logoutTokens = <String>[];
      final refreshTokens = <String>[];
      final launched = <Uri>[];
      final port = GoTrueSdkPort(
        apiUrl: Uri.parse('https://api.example.test'),
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        secretStore: store,
        httpClient: MockClient((request) async {
          if (request.url.path == '/v1/auth/token') {
            final body = jsonDecode(request.body) as Map<String, dynamic>;
            final code = body['auth_code'];
            if (code == 'code-a') {
              if (!aTokenStarted.isCompleted) aTokenStarted.complete();
              await releaseAToken.future;
            }
            if (request.url.queryParameters['grant_type'] == 'refresh_token') {
              refreshTokens.add(body['refresh_token'] as String);
            }
            return http.Response(
              jsonEncode(
                code == 'code-a' ? _sessionJsonFor('a') : _sessionJsonFor('b'),
              ),
              200,
            );
          }
          if (request.url.path == '/v1/auth/user') {
            final authorization = request.headers['authorization'];
            if (authorization == 'Bearer access-a') {
              return http.Response(jsonEncode(_userJsonFor('a')), 200);
            }
            return http.Response(jsonEncode(_userJsonFor('b')), 200);
          }
          if (request.url.path == '/v1/auth/logout') {
            logoutTokens.add(request.headers['authorization'] ?? '');
            return http.Response('', 204);
          }
          return http.Response('{}', 404);
        }),
      );
      final gateway = GoTrueAuthGateway(
        authApi: port,
        secretStore: store,
        callbackUri: Uri.parse('mindforge-dev://auth/callback'),
        authorizationOrigin: Uri.parse('https://api.example.test/v1/auth'),
        launchExternalBrowser: (uri) async {
          launched.add(uri);
          return true;
        },
      );

      await gateway.initialize();
      await gateway.beginGoogleSignIn();
      final callbackA = gateway.handleCallback(
        Uri.parse('mindforge-dev://auth/callback?code=code-a'),
      );
      await aTokenStarted.future;

      await gateway.signOut();
      await gateway.beginGoogleSignIn();
      releaseAToken.complete();
      await callbackA;

      final callbackB = gateway.handleCallback(
        Uri.parse('mindforge-dev://auth/callback?code=code-b'),
      );
      await callbackB;
      expect(gateway.state.phase, AuthPhase.authenticated);
      expect(gateway.state.userId, '22222222-2222-4222-8222-222222222222');

      final credentialB = await gateway.credential();
      final refreshedB = await gateway.refreshAfterUnauthorized(credentialB);
      expect(refreshedB.userId, '22222222-2222-4222-8222-222222222222');
      expect(refreshTokens, <String>['refresh-b']);

      expect(gateway.state.phase, AuthPhase.authenticated);
      expect(gateway.state.userId, '22222222-2222-4222-8222-222222222222');
      expect(store.sessionJson, contains('access-b'));
      expect(logoutTokens, isNot(contains('Bearer access-b')));
      expect(launched, hasLength(2));

      gateway.dispose();
    },
  );
}

Map<String, dynamic> _sessionJson() => <String, dynamic>{
  'access_token': 'access-token',
  'refresh_token': 'refresh-token',
  'token_type': 'bearer',
  'expires_in': 3600,
  'user': _userJson(),
};

Map<String, dynamic> _sessionJsonFor(String account) => <String, dynamic>{
  'access_token': 'access-$account',
  'refresh_token': 'refresh-$account',
  'token_type': 'bearer',
  'expires_in': 3600,
  'user': _userJsonFor(account),
};

Map<String, dynamic> _sessionJsonForAccess(
  String account,
  String accessToken,
) => <String, dynamic>{
  'access_token': accessToken,
  'refresh_token': 'refresh-$account',
  'token_type': 'bearer',
  'expires_in': 3600,
  'user': _userJsonFor(account),
};

Map<String, dynamic> _userJson() => <String, dynamic>{
  'id': '11111111-1111-4111-8111-111111111111',
  'aud': 'authenticated',
  'role': 'authenticated',
  'is_anonymous': false,
  'created_at': '2026-10-13T12:00:00Z',
  'email_confirmed_at': '2026-10-13T12:00:00Z',
  'email': 'user@example.test',
  'app_metadata': <String, dynamic>{
    'provider': 'google',
    'providers': <String>['google'],
  },
  'user_metadata': <String, dynamic>{'name': 'Mindforge User'},
};

Map<String, dynamic> _userJsonFor(String account) => <String, dynamic>{
  'id': account == 'a'
      ? '11111111-1111-4111-8111-111111111111'
      : '22222222-2222-4222-8222-222222222222',
  'aud': 'authenticated',
  'role': 'authenticated',
  'is_anonymous': false,
  'created_at': '2026-10-13T12:00:00Z',
  'email_confirmed_at': '2026-10-13T12:00:00Z',
  'email': '$account@example.test',
  'app_metadata': <String, dynamic>{
    'provider': 'google',
    'providers': <String>['google'],
  },
  'user_metadata': <String, dynamic>{'name': 'Mindforge $account'},
};

class _MemorySecretStore implements AuthSecretStore {
  final Map<String, String> values = <String, String>{};
  String? sessionJson;
  String? pendingAttempt;

  @override
  Future<void> clearAuthMaterial() async {
    sessionJson = null;
    pendingAttempt = null;
    values.clear();
  }

  @override
  Future<void> clearLegacyAuthMaterial() async {}

  @override
  Future<void> clearOAuthMaterial() async => pendingAttempt = null;

  @override
  Future<void> clearPkce() async {}

  @override
  Future<String?> getItem({required String key}) async => values[key];

  @override
  Future<String?> readPendingAttempt() async => pendingAttempt;

  @override
  Future<String?> readSession() async => sessionJson;

  @override
  Future<void> removeItem({required String key}) async => values.remove(key);

  @override
  Future<void> setItem({required String key, required String value}) async {
    values[key] = value;
  }

  @override
  Future<void> writePendingAttempt(String? value) async =>
      pendingAttempt = value;

  @override
  Future<void> writeSession(String? value) async => sessionJson = value;
}

class _InterleavingSecretStore extends _MemorySecretStore {
  bool delayNextRead = false;
  final Completer<void> readStarted = Completer<void>();
  final Completer<void> releaseRead = Completer<void>();

  @override
  Future<String?> getItem({required String key}) async {
    final value = await super.getItem(key: key);
    if (!delayNextRead) return value;
    delayNextRead = false;
    readStarted.complete();
    await releaseRead.future;
    return value;
  }
}
