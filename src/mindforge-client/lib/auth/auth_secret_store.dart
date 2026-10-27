import 'package:gotrue/gotrue.dart';

abstract interface class AuthSecretStore implements GotrueAsyncStorage {
  Future<String?> readSession();
  Future<void> writeSession(String? value);
  Future<String?> readPendingAttempt();
  Future<void> writePendingAttempt(String? value);
  Future<void> clearPkce();
  Future<void> clearOAuthMaterial();
  Future<void> clearAuthMaterial();
  Future<void> clearLegacyAuthMaterial();
}
