import 'package:gotrue/gotrue.dart';

import 'auth_secret_store.dart';

/// Gives one GoTrueClient ownership of the verifier it generated.
///
/// GoTrue removes its verifier after an exchange. During a cancelled OAuth
/// attempt an older client can finish after a newer client has written a new
/// verifier to the shared platform store, so removal must be compare-and-delete.
class OwnedPkceStorage implements GotrueAsyncStorage {
  OwnedPkceStorage(this._delegate);

  static final Expando<_PkceQueue> _queues = Expando<_PkceQueue>();

  final AuthSecretStore _delegate;
  String? _ownedKey;
  String? _ownedValue;

  @override
  Future<String?> getItem({required String key}) async {
    if (_ownedKey == key && _ownedValue != null) return _ownedValue;
    return _serialized(() async {
      if (_ownedKey == key && _ownedValue != null) return _ownedValue;
      final value = await _delegate.getItem(key: key);
      _ownedKey = key;
      _ownedValue = value;
      return value;
    });
  }

  @override
  Future<void> setItem({required String key, required String value}) async {
    await _serialized(() async {
      _ownedKey = key;
      _ownedValue = value;
      await _delegate.setItem(key: key, value: value);
    });
  }

  @override
  Future<void> removeItem({required String key}) async {
    await _serialized(() async {
      final ownedKey = _ownedKey;
      final ownedValue = _ownedValue;
      if (ownedKey == key && ownedValue != null) {
        final current = await _delegate.getItem(key: key);
        if (current == ownedValue) {
          await _delegate.removeItem(key: key);
        }
      }
      if (_ownedKey == key) {
        _ownedKey = null;
        _ownedValue = null;
      }
    });
  }

  Future<T> _serialized<T>(Future<T> Function() action) {
    final queue = _queues[_delegate] ??= _PkceQueue();
    final result = queue.tail.then<T>((_) => action());
    queue.tail = result.then<void>(
      (_) {},
      onError: (Object _, StackTrace __) {},
    );
    return result;
  }
}

class _PkceQueue {
  Future<void> tail = Future<void>.value();
}
