import 'package:flutter/foundation.dart';

enum AuthPhase {
  initializing,
  signedOut,
  signingIn,
  authenticated,
  unavailable,
}

class AuthViewState {
  const AuthViewState({
    required this.phase,
    required this.generation,
    this.userId,
    this.displayName,
    this.message,
  });

  final AuthPhase phase;
  final int generation;
  final String? userId;
  final String? displayName;
  final String? message;
}

class AuthCredential {
  const AuthCredential({
    required this.accessToken,
    required this.userId,
    required this.generation,
  });

  final String accessToken;
  final String userId;
  final int generation;
}

abstract class AuthGateway extends ChangeNotifier {
  AuthViewState get state;

  Future<void> initialize();
  Future<void> beginGoogleSignIn();
  Future<void> cancelGoogleSignIn();
  Future<void> handleCallback(Uri uri);
  Future<void> signOut();
  Future<AuthCredential> credential();
  Future<AuthCredential> refreshAfterUnauthorized(AuthCredential rejected);
  Future<void> expireAfterUnauthorized(AuthCredential rejected);
}

class UnauthenticatedException implements Exception {
  const UnauthenticatedException();

  @override
  String toString() => 'No authenticated session is available.';
}

class StaleAuthOperationException implements Exception {
  const StaleAuthOperationException();

  @override
  String toString() => 'The request belongs to an older session.';
}
