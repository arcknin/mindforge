import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/namespaced_auth_secret_store.dart';
import 'package:mindforge_client/auth/web_session_storage_store.dart';

void main() {
  test(
    'keeps PKCE and session material in the tab storage namespace',
    () async {
      final tab = _MemoryBrowserStorage();
      final keyValueStore = BrowserSessionStorageStore(sessionStorage: tab);
      final webStore = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:web:http://web-mind.home.arpa',
        keychain: keyValueStore,
      );
      final otherOriginStore = NamespacedAuthSecretStore(
        namespace: 'mindforge:develop:web:https://web-mind.home.arpa',
        keychain: keyValueStore,
      );

      await webStore.writeSession('mindforge-session');
      await webStore.writePendingAttempt('pending-pkce-attempt');
      await webStore.setItem(key: 'sdk-verifier', value: 'pkce-verifier');
      await otherOriginStore.writeSession('other-origin-session');

      expect(await webStore.readSession(), 'mindforge-session');
      expect(await webStore.readPendingAttempt(), 'pending-pkce-attempt');
      expect(await webStore.getItem(key: 'sdk-verifier'), 'pkce-verifier');
      expect(tab.values.keys, everyElement(contains('mindforge:develop:web:')));

      await webStore.clearAuthMaterial();

      expect(await webStore.readSession(), isNull);
      expect(await webStore.readPendingAttempt(), isNull);
      expect(await webStore.getItem(key: 'sdk-verifier'), isNull);
      expect(await otherOriginStore.readSession(), 'other-origin-session');
      expect(tab.values.values, isNot(contains('mindforge-session')));
    },
  );
}

class _MemoryBrowserStorage implements BrowserStorage {
  final Map<String, String> values = <String, String>{};

  @override
  void removeItem(String key) => values.remove(key);

  @override
  String? getItem(String key) => values[key];

  @override
  void setItem(String key, String value) => values[key] = value;
}
