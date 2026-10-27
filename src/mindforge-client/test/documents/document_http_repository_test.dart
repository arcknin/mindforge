import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/documents/document_http_repository.dart';
import 'package:mindforge_client/documents/document_repository.dart';

void main() {
  test('lists pages with bearer auth and preserves large versions', () async {
    final calls = <http.Request>[];
    final client = MockClient((request) async {
      calls.add(request);
      return http.Response(
        jsonEncode(<String, Object?>{
          'documents': <Object?>[
            <String, Object?>{
              'id': '11111111-1111-4111-8111-111111111111',
              'layer_id': '22222222-2222-4222-8222-222222222222',
              'title': 'A',
              'version': 9007199254740993,
              'created_at': '2026-10-13T00:00:00Z',
              'updated_at': '2026-10-13T00:00:00Z',
              'deleted_at': null,
              'expires_at': null,
            },
          ],
          'next_cursor': '33333333-3333-4333-8333-333333333333',
        }),
        200,
      );
    });
    final auth = _FakeAuthGateway('user-a');
    final repository = DocumentHttpRepository(
      baseUri: Uri.parse('http://127.0.0.1:11001'),
      auth: auth,
      client: client,
    );

    final page = await repository.list(
      '22222222-2222-4222-8222-222222222222',
      subtree: true,
      trash: true,
      after: '44444444-4444-4444-8444-444444444444',
    );

    expect(page.documents.single.version, BigInt.parse('9007199254740993'));
    expect(calls.single.headers['authorization'], 'Bearer token-user-a');
    expect(
      calls.single.url.path,
      '/v1/me/brain/layers/22222222-2222-4222-8222-222222222222/documents',
    );
    expect(calls.single.url.queryParameters, <String, String>{
      'scope': 'subtree',
      'state': 'trash',
      'after': '44444444-4444-4444-8444-444444444444',
    });
    client.close();
  });

  test(
    'refreshes once after 401 and does not retry an uncertain mutation',
    () async {
      final requests = <http.Request>[];
      final client = MockClient((request) async {
        requests.add(request);
        return http.Response(
          jsonEncode(<String, Object?>{
            'error': <String, Object?>{
              'code': 'unauthenticated',
              'message': 'Authentication required.',
            },
            'request_id': 'request-1',
          }),
          401,
        );
      });
      final auth = _FakeAuthGateway('user-a');
      final repository = DocumentHttpRepository(
        baseUri: Uri.parse('http://127.0.0.1:11001'),
        auth: auth,
        client: client,
      );

      await expectLater(
        repository.update(
          id: '11111111-1111-4111-8111-111111111111',
          layerId: '22222222-2222-4222-8222-222222222222',
          title: 'A',
          markdown: 'body',
          expectedVersion: BigInt.parse('9223372036854775807'),
        ),
        throwsA(
          isA<DocumentApiException>().having(
            (error) => error.statusCode,
            'statusCode',
            401,
          ),
        ),
      );
      expect(requests, hasLength(2));
      expect(auth.refreshCalls, 1);
      expect(requests.last.headers['authorization'], 'Bearer refreshed-user-a');
      expect(requests.last.body, contains('9223372036854775807'));
      client.close();
    },
  );

  test(
    'rejects a request whose credential belongs to an older session',
    () async {
      final client = MockClient((_) async => http.Response('{}', 200));
      final auth = _FakeAuthGateway('user-a')
        ..delayedCredential = Completer<AuthCredential>();
      final repository = DocumentHttpRepository(
        baseUri: Uri.parse('http://127.0.0.1:11001'),
        auth: auth,
        client: client,
      );

      final request = repository.get('11111111-1111-4111-8111-111111111111');
      await Future<void>.delayed(Duration.zero);
      auth.switchTo('user-b');
      auth.delayedCredential!.complete(
        const AuthCredential(
          accessToken: 'old',
          userId: 'user-a',
          generation: 1,
        ),
      );

      await expectLater(request, throwsA(isA<StaleAuthOperationException>()));
      client.close();
    },
  );
}

class _FakeAuthGateway extends AuthGateway {
  _FakeAuthGateway(String userId)
    : state = AuthViewState(
        phase: AuthPhase.authenticated,
        generation: 1,
        userId: userId,
      );

  @override
  AuthViewState state;

  int refreshCalls = 0;
  Completer<AuthCredential>? delayedCredential;

  void switchTo(String userId) {
    state = AuthViewState(
      phase: AuthPhase.authenticated,
      generation: state.generation + 1,
      userId: userId,
    );
    notifyListeners();
  }

  @override
  Future<AuthCredential> credential() async =>
      delayedCredential?.future ??
      AuthCredential(
        accessToken: 'token-${state.userId}',
        userId: state.userId!,
        generation: state.generation,
      );

  @override
  Future<AuthCredential> refreshAfterUnauthorized(
    AuthCredential rejected,
  ) async {
    refreshCalls++;
    return AuthCredential(
      accessToken: 'refreshed-${rejected.userId}',
      userId: rejected.userId,
      generation: rejected.generation,
    );
  }

  @override
  Future<void> expireAfterUnauthorized(AuthCredential rejected) async {}

  @override
  Future<void> initialize() async {}

  @override
  Future<void> beginGoogleSignIn() async {}

  @override
  Future<void> cancelGoogleSignIn() async {}

  @override
  Future<void> handleCallback(Uri uri) async {}

  @override
  Future<void> signOut() async {}
}
