import 'namespaced_auth_secret_store.dart';

abstract interface class BrowserStorage {
  String? getItem(String key);
  void setItem(String key, String value);
  void removeItem(String key);
}

class BrowserSessionStorageStore implements SecureKeyValueStore {
  BrowserSessionStorageStore({required BrowserStorage sessionStorage})
    : _sessionStorage = sessionStorage;

  final BrowserStorage _sessionStorage;

  @override
  Future<void> delete(String key) async => _sessionStorage.removeItem(key);

  @override
  Future<String?> read(String key) async => _sessionStorage.getItem(key);

  @override
  Future<void> write(String key, String value) async =>
      _sessionStorage.setItem(key, value);
}
