import 'dart:async';

import 'auth_secret_store.dart';

abstract interface class SecureKeyValueStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
  Future<void> delete(String key);
}

class NamespacedAuthSecretStore implements AuthSecretStore {
  NamespacedAuthSecretStore({
    required String namespace,
    Iterable<String> legacyNamespaces = const <String>[],
    required SecureKeyValueStore keychain,
  }) : _namespace = _validateNamespace(namespace),
       _legacyNamespaces = legacyNamespaces
           .map(_validateNamespace)
           .where((value) => value != namespace)
           .toSet(),
       _keychain = keychain;

  static const String _sessionSuffix = '.session.v1';
  static const String _attemptSuffix = '.oauth-attempt.v1';
  static const String _pkceNameSuffix = '.pkce-key.v1';
  static const String _pkceValueSuffix = '.pkce-verifier.v1';

  final String _namespace;
  final Set<String> _legacyNamespaces;
  final SecureKeyValueStore _keychain;
  Future<void> _writeQueue = Future<void>.value();

  @override
  Future<String?> readSession() => _keychain.read('$_namespace$_sessionSuffix');

  @override
  Future<void> writeSession(String? value) => _serialized(() async {
    final key = '$_namespace$_sessionSuffix';
    if (value == null) {
      await _keychain.delete(key);
    } else {
      await _keychain.write(key, value);
    }
  });

  @override
  Future<String?> readPendingAttempt() =>
      _keychain.read('$_namespace$_attemptSuffix');

  @override
  Future<void> writePendingAttempt(String? value) => _serialized(() async {
    final key = '$_namespace$_attemptSuffix';
    if (value == null) {
      await _keychain.delete(key);
    } else {
      await _keychain.write(key, value);
    }
  });

  @override
  Future<String?> getItem({required String key}) async {
    final safeKey = _validateSdkKey(key);
    final storedKey = await _keychain.read('$_namespace$_pkceNameSuffix');
    if (storedKey != safeKey) return null;
    return _keychain.read('$_namespace$_pkceValueSuffix');
  }

  @override
  Future<void> setItem({required String key, required String value}) =>
      _serialized(() async {
        final safeKey = _validateSdkKey(key);
        await _keychain.write('$_namespace$_pkceNameSuffix', safeKey);
        await _keychain.write('$_namespace$_pkceValueSuffix', value);
      });

  @override
  Future<void> removeItem({required String key}) => _serialized(() async {
    final safeKey = _validateSdkKey(key);
    final keyName = '$_namespace$_pkceNameSuffix';
    if (await _keychain.read(keyName) == safeKey) {
      await _deleteBestEffort(<String>[
        '$_namespace$_pkceValueSuffix',
        keyName,
      ]);
    }
  });

  @override
  Future<void> clearPkce() => _serialized(() async {
    await _deleteBestEffort(<String>[
      '$_namespace$_pkceValueSuffix',
      '$_namespace$_pkceNameSuffix',
    ]);
  });

  @override
  Future<void> clearAuthMaterial() async {
    Object? firstError;
    StackTrace? firstStackTrace;
    for (final cleanup in <Future<void> Function()>[
      () => writeSession(null),
      clearOAuthMaterial,
      clearLegacyAuthMaterial,
    ]) {
      try {
        await cleanup();
      } on Object catch (error, stackTrace) {
        firstError ??= error;
        firstStackTrace ??= stackTrace;
      }
    }
    if (firstError != null) {
      Error.throwWithStackTrace(firstError, firstStackTrace!);
    }
  }

  @override
  Future<void> clearLegacyAuthMaterial() => _serialized(() async {
    final keys = <String>[
      for (final namespace in _legacyNamespaces) ..._authKeys(namespace),
    ];
    await _deleteBestEffort(keys);
  });

  @override
  Future<void> clearOAuthMaterial() async {
    Object? firstError;
    StackTrace? firstStackTrace;
    for (final cleanup in <Future<void> Function()>[
      () => writePendingAttempt(null),
      clearPkce,
    ]) {
      try {
        await cleanup();
      } on Object catch (error, stackTrace) {
        firstError ??= error;
        firstStackTrace ??= stackTrace;
      }
    }
    if (firstError != null) {
      Error.throwWithStackTrace(firstError, firstStackTrace!);
    }
  }

  Future<void> _serialized(Future<void> Function() action) {
    final result = _writeQueue.then((_) => action());
    _writeQueue = result.then<void>(
      (_) {},
      onError: (Object _, StackTrace __) {},
    );
    return result;
  }

  Future<void> _deleteBestEffort(Iterable<String> keys) async {
    Object? firstError;
    StackTrace? firstStackTrace;
    for (final key in keys) {
      try {
        await _keychain.delete(key);
      } on Object catch (error, stackTrace) {
        firstError ??= error;
        firstStackTrace ??= stackTrace;
      }
    }
    if (firstError != null) {
      Error.throwWithStackTrace(firstError, firstStackTrace!);
    }
  }

  static List<String> _authKeys(String namespace) => <String>[
    '$namespace$_sessionSuffix',
    '$namespace$_attemptSuffix',
    '$namespace$_pkceValueSuffix',
    '$namespace$_pkceNameSuffix',
  ];

  static String _validateNamespace(String namespace) {
    if (namespace.isEmpty || namespace.contains('\n')) {
      throw const FormatException(
        'A non-empty auth storage namespace is required.',
      );
    }
    return namespace;
  }

  static String _validateSdkKey(String key) {
    if (key.isEmpty || key.contains('/') || key.contains('..')) {
      throw const FormatException('The GoTrue storage key is invalid.');
    }
    return key;
  }
}
