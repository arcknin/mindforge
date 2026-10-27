import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'namespaced_auth_secret_store.dart';

export 'namespaced_auth_secret_store.dart';

class FlutterKeychainStore implements SecureKeyValueStore {
  static const MacOsOptions macOsOptions = MacOsOptions(
    accountName: 'com.mindforge.mindforge_client',
    usesDataProtectionKeychain: false,
  );

  FlutterKeychainStore({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage(mOptions: macOsOptions);

  final FlutterSecureStorage _storage;

  @override
  Future<void> delete(String key) => _storage.delete(key: key);

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);
}
