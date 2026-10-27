import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/app/app_config.dart';
import 'package:mindforge_client/auth/secure_auth_store.dart';
import 'package:mindforge_client/documents/document_models.dart';
import 'package:mindforge_client/documents/file_document_local_store.dart';

void main() {
  test(
    'isolates session, pending callback, and PKCE keys by app and Auth origin',
    () async {
      final keychain = _MemoryKeychain();
      final develop = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:macos:http://127.0.0.1:11000',
        keychain: keychain,
      );
      final otherOrigin = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:macos:http://127.0.0.1:12000',
        keychain: keychain,
      );

      await develop.writeSession('session-a');
      await develop.writePendingAttempt('attempt-a');
      await develop.setItem(key: 'sdk-verifier', value: 'verifier-a');
      await otherOrigin.writeSession('session-b');
      await otherOrigin.setItem(key: 'sdk-verifier', value: 'verifier-b');

      await develop.clearAuthMaterial();

      expect(await develop.readSession(), isNull);
      expect(await develop.readPendingAttempt(), isNull);
      expect(await develop.getItem(key: 'sdk-verifier'), isNull);
      expect(await otherOrigin.readSession(), 'session-b');
      expect(await otherOrigin.getItem(key: 'sdk-verifier'), 'verifier-b');
      expect(keychain.values.values, isNot(contains('session-a')));
      expect(keychain.values.values, contains('session-b'));
    },
  );

  test(
    'clears only known legacy auth keys while leaving document keys intact',
    () async {
      final keychain = _MemoryKeychain();
      final legacy = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:macos:http://127.0.0.1:11000',
        keychain: keychain,
      );
      final current = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:auth:v2:macos:http://127.0.0.1:11001',
        legacyNamespaces: const <String>[
          'mindforge:develop:macos:http://127.0.0.1:11000',
        ],
        keychain: keychain,
      );
      await legacy.writeSession('legacy-session');
      await legacy.writePendingAttempt('legacy-attempt');
      await legacy.setItem(key: 'sdk-verifier', value: 'legacy-verifier');
      await keychain.write(
        'mindforge:develop:macos:http://127.0.0.1:11000.documents.v1',
        'document-snapshot',
      );

      await current.clearLegacyAuthMaterial();

      expect(await legacy.readSession(), isNull);
      expect(await legacy.readPendingAttempt(), isNull);
      expect(await legacy.getItem(key: 'sdk-verifier'), isNull);
      expect(
        keychain
            .values['mindforge:develop:macos:http://127.0.0.1:11000.documents.v1'],
        'document-snapshot',
      );
    },
  );

  test(
    'attempts to clear callback and PKCE data when session deletion fails',
    () async {
      final keychain = _MemoryKeychain();
      final store = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:macos:http://127.0.0.1:11000',
        keychain: keychain,
      );
      await store.writeSession('session-a');
      await store.writePendingAttempt('attempt-a');
      await store.setItem(key: 'sdk-verifier', value: 'verifier-a');
      keychain.failDeletes.add(
        'mindforge:develop:macos:http://127.0.0.1:11000.session.v1',
      );

      await expectLater(store.clearAuthMaterial(), throwsA(isA<StateError>()));

      expect(await store.readSession(), 'session-a');
      expect(await store.readPendingAttempt(), isNull);
      expect(await store.getItem(key: 'sdk-verifier'), isNull);
    },
  );

  test('uses the single-app macOS Keychain without a sharing entitlement', () {
    final options = FlutterKeychainStore.macOsOptions;
    final debugEntitlements = File(
      'macos/Runner/DebugProfile.entitlements',
    ).readAsStringSync();
    final releaseEntitlements = File(
      'macos/Runner/Release.entitlements',
    ).readAsStringSync();

    expect(options.usesDataProtectionKeychain, isFalse);
    expect(options.groupId, isNull);
    expect(debugEntitlements, isNot(contains('keychain-access-groups')));
    expect(releaseEntitlements, isNot(contains('keychain-access-groups')));
    expect(debugEntitlements, contains('com.apple.security.network.client'));
    expect(releaseEntitlements, contains('com.apple.security.network.client'));
  });

  test(
    'upgrade keeps the historical document namespace and local work',
    () async {
      final config = AppConfig.development(apiUrl: 'http://127.0.0.1:11001');
      final root = await Directory.systemTemp.createTemp(
        'mindforge-auth-upgrade-',
      );
      addTearDown(() async {
        if (root.existsSync()) await root.delete(recursive: true);
      });
      final historical = FileDocumentLocalStore(
        namespace:
            'mindforge:develop:macos:http://127.0.0.1:11000:documents:v1',
        directory: () async => root,
      );
      final document = LocalDocument(
        id: '11111111-1111-4111-8111-111111111111',
        layerId: '22222222-2222-4222-8222-222222222222',
        title: 'Rascunho',
        markdown: 'conteúdo local',
        remoteVersion: BigInt.from(4),
        localRevision: 7,
        dirty: true,
        deletedAt: DateTime.utc(2026, 10, 13, 12),
        expiresAt: DateTime.utc(2026, 10, 20, 12),
        conflict: true,
        uncertain: true,
        contentLoaded: true,
        remoteDeleted: true,
        pendingMutation: PendingDocumentMutation(
          operation: 'update',
          layerId: '22222222-2222-4222-8222-222222222222',
          title: 'Rascunho',
          markdown: 'conteúdo local',
          localRevision: 7,
          expectedVersion: BigInt.from(4),
          trash: true,
        ),
      );
      await historical.write('user-a', <LocalDocument>[document]);

      final afterUpgrade = FileDocumentLocalStore(
        namespace: '${config.documentStorageNamespace}:documents:v1',
        directory: () async => root,
      );
      final restored = (await afterUpgrade.read('user-a')).single;

      expect(
        config.authStorageNamespace,
        isNot(config.documentStorageNamespace),
      );
      expect(config.documentStorageNamespace, contains('127.0.0.1:11000'));
      expect(restored.markdown, 'conteúdo local');
      expect(restored.isTrash, isTrue);
      expect(restored.dirty, isTrue);
      expect(restored.pendingMutation?.trash, isTrue);
      expect(restored.pendingMutation?.expectedVersion, BigInt.from(4));
    },
  );
}

class _MemoryKeychain implements SecureKeyValueStore {
  final Map<String, String> values = <String, String>{};
  final Set<String> failDeletes = <String>{};

  @override
  Future<void> delete(String key) async {
    if (failDeletes.contains(key)) throw StateError('delete failed');
    values.remove(key);
  }

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;
}
