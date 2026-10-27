import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_http_repository.dart';
import 'package:mindforge_client/brain/brain_repository.dart';

void main() {
  group('BrainHttpRepository', () {
    test(
      'bootstraps and observes through the approved routes with bearer auth',
      () async {
        final calls = <http.Request>[];
        final client = MockClient((request) async {
          calls.add(request);
          if (request.method == 'PUT') {
            return http.Response(
              jsonEncode(<String, Object?>{
                'brain': <String, Object?>{
                  'id': 'brain-a',
                  'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
                },
              }),
              200,
            );
          }
          return http.Response(
            jsonEncode(<String, Object?>{
              'brain': <String, Object?>{
                'id': 'brain-a',
                'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
              },
              'layers': <Object?>[],
            }),
            200,
          );
        });
        final auth = _FakeAuthGateway();
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://127.0.0.1:11001'),
          auth: auth,
          client: client,
        );

        final created = await repository.bootstrap();
        final hierarchy = await repository.observe();

        expect(created.id, 'brain-a');
        expect(hierarchy.brainId, 'brain-a');
        expect(hierarchy.topLevelLayers, isEmpty);
        expect(calls.map((request) => request.method), <String>['PUT', 'GET']);
        expect(calls.map((request) => request.url.path), <String>[
          '/v1/me/brain',
          '/v1/me/brain',
        ]);
        for (final request in calls) {
          expect(request.headers['authorization'], 'Bearer session-access');
          expect(request.body, isEmpty);
        }
        expect(auth.credentialCalls, 2);
        client.close();
      },
    );

    test(
      'creates layers with opaque parent IDs and preserves duplicate names',
      () async {
        late http.Request sent;
        final client = MockClient((request) async {
          sent = request;
          return http.Response(
            jsonEncode(<String, Object?>{
              'layer': <String, Object?>{
                'id': 'layer-new',
                'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
                'name': 'Personal',
                'version': 1,
              },
            }),
            201,
          );
        });
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://127.0.0.1:11001'),
          auth: _FakeAuthGateway(),
          client: client,
        );

        final layer = await repository.createLayer(
          parent: const BrainParent(kind: ParentKind.root, id: 'root-a'),
          name: 'Personal',
        );

        expect(sent.method, 'POST');
        expect(sent.url.path, '/v1/me/brain/layers');
        expect(sent.headers['content-type'], contains('application/json'));
        expect(jsonDecode(sent.body), <String, Object?>{
          'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
          'name': 'Personal',
        });
        expect(layer.name, 'Personal');
        expect(layer.id, 'layer-new');
        client.close();
      },
    );

    test('preserves the same-origin /api prefix on web requests', () async {
      late http.Request sent;
      final client = MockClient((request) async {
        sent = request;
        return http.Response(
          jsonEncode(<String, Object?>{
            'brain': <String, Object?>{
              'id': 'brain-a',
              'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
            },
          }),
          200,
        );
      });
      final repository = BrainHttpRepository(
        baseUri: Uri.parse('http://web-mind.home.arpa/api'),
        auth: _FakeAuthGateway(),
        client: client,
      );

      await repository.bootstrap();

      expect(sent.url, Uri.parse('http://web-mind.home.arpa/api/v1/me/brain'));
      client.close();
    });

    test(
      'round-trips an Int64 CAS version above JavaScript exact integers',
      () async {
        late http.Request sent;
        final client = MockClient((request) async {
          sent = request;
          return http.Response(
            '{"layer":{"id":"layer-a","parent":{"kind":"root",'
            '"id":"root-a"},"name":"Renamed",'
            '"version":9007199254740993}}',
            200,
          );
        });
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://web-mind.home.arpa/api'),
          auth: _FakeAuthGateway(),
          client: client,
        );

        final renamed = await repository.renameLayer(
          layerId: 'layer-a',
          name: 'Renamed',
          expectedVersion: BigInt.parse('9007199254740993'),
        );

        expect(sent.url.path, '/api/v1/me/brain/layers/layer-a');
        expect(
          sent.body,
          '{"name":"Renamed","expected_version":9007199254740993}',
        );
        expect(renamed.version, BigInt.parse('9007199254740993'));
        client.close();
      },
    );

    test(
      'refreshes once after a definite 401 and retries with the new token',
      () async {
        final authorizationHeaders = <String?>[];
        var calls = 0;
        final client = MockClient((request) async {
          calls++;
          authorizationHeaders.add(request.headers['authorization']);
          if (calls == 1) {
            return http.Response(
              jsonEncode(<String, Object?>{
                'error': <String, Object?>{
                  'code': 'unauthenticated',
                  'message': 'Authentication required.',
                },
                'request_id': 'req-1',
              }),
              401,
            );
          }
          return http.Response(
            jsonEncode(<String, Object?>{
              'brain': <String, Object?>{
                'id': 'brain-a',
                'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
              },
            }),
            200,
          );
        });
        final auth = _FakeAuthGateway();
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://127.0.0.1:11001'),
          auth: auth,
          client: client,
        );

        await repository.bootstrap();

        expect(calls, 2);
        expect(authorizationHeaders, <String?>[
          'Bearer session-access',
          'Bearer refreshed-access',
        ]);
        expect(auth.refreshCalls, 1);
        client.close();
      },
    );

    test(
      'does not retry a create after a server error or expose raw details',
      () async {
        var calls = 0;
        final client = MockClient((_) async {
          calls++;
          return http.Response('private sql detail', 503);
        });
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://127.0.0.1:11001'),
          auth: _FakeAuthGateway(),
          client: client,
        );

        await expectLater(
          repository.createLayer(
            parent: const BrainParent(kind: ParentKind.root, id: 'root-a'),
            name: 'Personal',
          ),
          throwsA(
            isA<BrainApiException>()
                .having((error) => error.statusCode, 'statusCode', 503)
                .having(
                  (error) => error.message,
                  'message',
                  isNot(contains('private')),
                ),
          ),
        );

        expect(calls, 1);
        client.close();
      },
    );

    test(
      'does not send a request with a session from a different account',
      () async {
        var calls = 0;
        final client = MockClient((_) async {
          calls++;
          return http.Response('{}', 200);
        });
        final auth = _FakeAuthGateway();
        final delayedCredential = Completer<AuthCredential>();
        auth.delayedCredential = delayedCredential;
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://127.0.0.1:11001'),
          auth: auth,
          client: client,
        );

        final request = repository.bootstrap();
        await Future<void>.delayed(Duration.zero);
        auth.state = const AuthViewState(
          phase: AuthPhase.authenticated,
          generation: 2,
          userId: 'user-b',
        );
        delayedCredential.complete(
          const AuthCredential(
            accessToken: 'user-b-token',
            userId: 'user-b',
            generation: 2,
          ),
        );

        await expectLater(request, throwsA(isA<StaleAuthOperationException>()));
        expect(calls, 0);
        client.close();
      },
    );

    test(
      'expires the session after the single allowed 401 retry is rejected',
      () async {
        var calls = 0;
        final client = MockClient((_) async {
          calls++;
          return http.Response(
            jsonEncode(<String, Object?>{
              'error': <String, Object?>{
                'code': 'unauthenticated',
                'message': 'Authentication required.',
              },
              'request_id': 'req-1',
            }),
            401,
          );
        });
        final auth = _FakeAuthGateway();
        final repository = BrainHttpRepository(
          baseUri: Uri.parse('http://127.0.0.1:11001'),
          auth: auth,
          client: client,
        );

        await expectLater(
          repository.bootstrap(),
          throwsA(
            isA<BrainApiException>().having(
              (error) => error.statusCode,
              'statusCode',
              401,
            ),
          ),
        );

        expect(calls, 2);
        expect(auth.refreshCalls, 1);
        expect(auth.expireCalls, 1);
        client.close();
      },
    );

    test('encodes layer IDs as a single path segment on rename', () async {
      late http.Request sent;
      final client = MockClient((request) async {
        sent = request;
        return http.Response(
          jsonEncode(<String, Object?>{
            'layer': <String, Object?>{
              'id': 'opaque/id',
              'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
              'name': 'Renamed',
              'version': 2,
            },
          }),
          200,
        );
      });
      final repository = BrainHttpRepository(
        baseUri: Uri.parse('http://127.0.0.1:11001'),
        auth: _FakeAuthGateway(),
        client: client,
      );

      await repository.renameLayer(
        layerId: 'opaque/id',
        name: 'Renamed',
        expectedVersion: BigInt.one,
      );

      expect(sent.method, 'PATCH');
      expect(sent.url.pathSegments.last, 'opaque/id');
      expect(jsonDecode(sent.body), <String, Object?>{
        'name': 'Renamed',
        'expected_version': 1,
      });
      client.close();
    });
  });
}

class _FakeAuthGateway extends AuthGateway {
  int credentialCalls = 0;
  int refreshCalls = 0;
  int expireCalls = 0;
  Completer<AuthCredential>? delayedCredential;

  @override
  AuthViewState state = const AuthViewState(
    phase: AuthPhase.authenticated,
    generation: 1,
    userId: 'user-a',
  );

  @override
  Future<AuthCredential> credential() async {
    credentialCalls++;
    final delayed = delayedCredential;
    if (delayed != null) return delayed.future;
    return AuthCredential(
      accessToken: refreshCalls == 0 ? 'session-access' : 'refreshed-access',
      userId: 'user-a',
      generation: 1,
    );
  }

  @override
  Future<AuthCredential> refreshAfterUnauthorized(
    AuthCredential rejected,
  ) async {
    refreshCalls++;
    return AuthCredential(
      accessToken: 'refreshed-access',
      userId: rejected.userId,
      generation: rejected.generation,
    );
  }

  @override
  Future<void> expireAfterUnauthorized(AuthCredential rejected) async {
    expireCalls++;
  }

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
