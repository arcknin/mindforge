import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mindforge_client/auth/api_auth_http_client.dart';

void main() {
  group('ApiAuthHttpClient', () {
    test(
      'rejects a different origin and a path outside the Auth allowlist',
      () async {
        final delegate = MockClient((request) async {
          return http.Response('{}', 200);
        });
        final client = ApiAuthHttpClient(
          apiUrl: Uri.parse('https://api.example.test'),
          allowedRedirectUri: Uri.parse('mindforge-dev://auth/callback'),
          inner: delegate,
        );

        await expectLater(
          client.get(Uri.parse('https://attacker.example.test/v1/auth/user')),
          throwsA(isA<http.ClientException>()),
        );
        await expectLater(
          client.get(Uri.parse('https://api.example.test/v1/auth/admin/users')),
          throwsA(isA<http.ClientException>()),
        );

        client.close();
      },
    );

    test(
      'forces token, user, and logout requests to never follow redirects',
      () async {
        final redirectModes = <String, bool>{};
        final delegate = MockClient((request) async {
          redirectModes['${request.method} ${request.url.path}'] =
              request.followRedirects;
          return http.Response('{}', 200);
        });
        final client = ApiAuthHttpClient(
          apiUrl: Uri.parse('https://api.example.test'),
          allowedRedirectUri: Uri.parse('mindforge-dev://auth/callback'),
          inner: delegate,
        );

        await client.post(
          Uri.parse(
            'https://api.example.test/v1/auth/token?grant_type=refresh_token',
          ),
          headers: const {'content-type': 'application/json'},
          body: '{"refresh_token":"refresh"}',
        );
        await client.get(
          Uri.parse('https://api.example.test/v1/auth/user'),
          headers: const {'authorization': 'Bearer access'},
        );
        await client.post(
          Uri.parse('https://api.example.test/v1/auth/logout?scope=local'),
          headers: const {'authorization': 'Bearer access'},
          body: '{}',
        );

        expect(redirectModes, <String, bool>{
          'POST /v1/auth/token': false,
          'GET /v1/auth/user': false,
          'POST /v1/auth/logout': false,
        });
        client.close();
      },
    );

    test(
      'rejects an authorize redirect outside the exact API Auth path',
      () async {
        final client = ApiAuthHttpClient(
          apiUrl: Uri.parse('https://api.example.test'),
          allowedRedirectUri: Uri.parse('mindforge-dev://auth/callback'),
          inner: MockClient((request) async => http.Response('{}', 200)),
        );

        await expectLater(
          client.get(
            Uri.parse(
              'https://api.example.test/auth/authorize?provider=google&flow_type=pkce&redirect_to=app%3A%2F%2Fcallback&code_challenge=${'a' * 43}&code_challenge_method=s256',
            ),
          ),
          throwsA(isA<http.ClientException>()),
        );
        client.close();
      },
    );

    test('rejects a redirect target outside the configured callback', () async {
      final client = ApiAuthHttpClient(
        apiUrl: Uri.parse('https://api.example.test'),
        allowedRedirectUri: Uri.parse('mindforge-dev://auth/callback'),
        inner: MockClient((request) async => http.Response('{}', 200)),
      );

      await expectLater(
        client.get(
          Uri.parse(
            'https://api.example.test/v1/auth/authorize?provider=google&flow_type=pkce&redirect_to=https%3A%2F%2Fattacker.example%2Fcallback&code_challenge=${'a' * 43}&code_challenge_method=s256',
          ),
        ),
        throwsA(isA<http.ClientException>()),
      );
      client.close();
    });
  });
}
