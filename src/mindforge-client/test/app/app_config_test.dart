import 'package:flutter/foundation.dart' show TargetPlatform;
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/app/app_config.dart';

void main() {
  group('AppConfig', () {
    test(
      'requires explicit develop endpoints and has no production fallback',
      () {
        expect(AppConfig.fromEnvironment, throwsA(isA<FormatException>()));
      },
    );

    test('accepts only the explicit loopback API develop service', () {
      final config = AppConfig.development(apiUrl: 'http://127.0.0.1:11001');

      expect(config.apiUrl, Uri.parse('http://127.0.0.1:11001'));
      expect(config.authApiUrl, Uri.parse('http://127.0.0.1:11001/v1/auth'));
      expect(config.callbackUri, Uri.parse('mindforge-dev://auth/callback'));
      expect(config.authStorageNamespace, contains(':auth:v2:'));
      expect(config.authStorageNamespace, contains('127.0.0.1:11001'));
      expect(
        config.documentStorageNamespace,
        'mindforge:develop:macos:http://127.0.0.1:11000',
      );
    });

    test('detects Android separately and uses an Android-native namespace', () {
      expect(
        AppConfig.detectTargetPlatform(
          isWeb: false,
          platform: TargetPlatform.android,
        ),
        AppTargetPlatform.android,
      );
      expect(
        AppConfig.detectTargetPlatform(
          isWeb: false,
          platform: TargetPlatform.macOS,
        ),
        AppTargetPlatform.macos,
      );

      final config = AppConfig.development(
        apiUrl: 'http://127.0.0.1:11001',
        targetPlatform: AppTargetPlatform.android,
      );

      expect(config.targetPlatform, AppTargetPlatform.android);
      expect(config.documentStorageNamespace, contains(':android-native:'));
      expect(config.authStorageNamespace, contains(':android-native:'));
      expect(config.documentStorageNamespace, contains('127.0.0.1:11000'));
    });

    test('keeps the development constructor compatible with macOS callers', () {
      final config = AppConfig.development(apiUrl: 'http://127.0.0.1:11001');

      expect(config.targetPlatform, AppTargetPlatform.macos);
      expect(config.documentStorageNamespace, contains(':macos:'));
    });

    test(
      'rejects an endpoint that can point outside the local develop setup',
      () {
        expect(
          () => AppConfig.development(apiUrl: 'https://api.example.com'),
          throwsA(isA<FormatException>()),
        );
        expect(
          () => AppConfig.development(apiUrl: 'http://localhost:8080'),
          throwsA(isA<FormatException>()),
        );
      },
    );

    test('derives web services and callback from the explicit LAN origin', () {
      final config = AppConfig.developmentWeb(
        webOrigin: 'http://web-mind.home.arpa',
      );

      expect(config.apiUrl, Uri.parse('http://web-mind.home.arpa/api'));
      expect(
        config.authApiUrl,
        Uri.parse('http://web-mind.home.arpa/api/v1/auth'),
      );
      expect(
        config.callbackUri,
        Uri.parse('http://web-mind.home.arpa/login/callback'),
      );
      expect(config.authStorageNamespace, contains(':auth:v2:web:'));
      expect(config.documentStorageNamespace, contains(':web:'));
      expect(
        config.documentStorageNamespace,
        contains('http://web-mind.home.arpa'),
      );
    });

    test('accepts HTTPS and rejects web origins outside the exact host', () {
      final config = AppConfig.developmentWeb(
        webOrigin: 'https://web-mind.home.arpa',
      );
      expect(
        config.callbackUri,
        Uri.parse('https://web-mind.home.arpa/login/callback'),
      );

      for (final invalidOrigin in <String>[
        'http://localhost',
        'http://web-mind.home.arpa.evil.example',
        'http://user@web-mind.home.arpa',
        'http://web-mind.home.arpa/path',
        'http://web-mind.home.arpa?query=1',
        'http://web-mind.home.arpa#fragment',
        'ftp://web-mind.home.arpa',
      ]) {
        expect(
          () => AppConfig.developmentWeb(webOrigin: invalidOrigin),
          throwsA(isA<FormatException>()),
          reason: 'Expected $invalidOrigin to be rejected',
        );
      }
    });
  });
}
