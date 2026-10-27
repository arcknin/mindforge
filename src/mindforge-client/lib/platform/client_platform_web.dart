import 'dart:async';

import 'package:web/web.dart' as web;

import '../app/app_config.dart';
import '../auth/auth_secret_store.dart';
import '../auth/namespaced_auth_secret_store.dart';
import '../auth/web_session_storage_store.dart';
import '../documents/document_repository.dart';
import 'web_auth_callback.dart';

AuthSecretStore createAuthSecretStore(AppConfig config) =>
    NamespacedAuthSecretStore(
      namespace: config.authStorageNamespace,
      legacyNamespaces: <String>[config.documentStorageNamespace],
      keychain: BrowserSessionStorageStore(
        sessionStorage: const _WindowSessionStorage(),
      ),
    );

Future<bool> launchGoogleSignIn(Uri uri) async {
  try {
    web.window.location.assign(uri.toString());
    return true;
  } on Object {
    return false;
  }
}

Stream<Uri> incomingAuthCallbacks(Uri callbackUri) {
  final capture = WebAuthCallbackCapture.capture(Uri.base, callbackUri);
  if (capture == null) return const Stream<Uri>.empty();

  try {
    web.window.history.replaceState(null, '', capture.cleanedUrl.toString());
  } on Object {
    web.window.location.replace('/');
    return const Stream<Uri>.empty();
  }
  return Stream<Uri>.value(capture.callbackUri);
}

class _WindowSessionStorage implements BrowserStorage {
  const _WindowSessionStorage();

  web.Storage get _storage => web.window.sessionStorage;

  @override
  String? getItem(String key) => _storage.getItem(key);

  @override
  void removeItem(String key) => _storage.removeItem(key);

  @override
  void setItem(String key, String value) => _storage.setItem(key, value);
}

// Web documents need their own IndexedDB adapter. Do not pretend sessionStorage
// or in-memory state is durable document storage. Existing Auth/Core stay enabled.
DocumentLocalStore? createDocumentLocalStore(AppConfig config) => null;
