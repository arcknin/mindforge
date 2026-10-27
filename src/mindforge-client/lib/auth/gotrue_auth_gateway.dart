import 'dart:async';
import 'dart:convert';

import 'package:gotrue/gotrue.dart';
import 'package:http/http.dart' as http;

import 'auth_gateway.dart';
import 'auth_secret_store.dart';
import 'socket_exception_stub.dart'
    if (dart.library.io) 'socket_exception_io.dart'
    as platform;

enum GoTrueEventKind {
  signedIn,
  signedOut,
  tokenRefreshed,
  userUpdated,
  initial,
}

class GoTrueEvent {
  const GoTrueEvent(this.kind, this.session);

  final GoTrueEventKind kind;
  final Session? session;
}

abstract interface class GoTruePort {
  Stream<GoTrueEvent> get events;
  Session? get current;
  Future<Uri> createGoogleSignInUrl(String redirectUri);
  Future<Session?> exchangeCodeForSession(String code);
  Future<Session?> getSession();
  Future<Session?> refreshSession();
  Future<Session?> restoreSession(String serializedSession);
  Future<void> signOutLocal();
  void startAutoRefresh();
  void stopAutoRefresh();
  void dispose();
}

class GoTrueAuthGateway extends AuthGateway {
  GoTrueAuthGateway({
    required GoTruePort authApi,
    required AuthSecretStore secretStore,
    required Uri callbackUri,
    required Uri authorizationOrigin,
    required Future<bool> Function(Uri uri) launchExternalBrowser,
    DateTime Function()? now,
    Duration attemptLifetime = const Duration(minutes: 10),
  }) : _authApi = authApi,
       _secretStore = secretStore,
       _callbackUri = callbackUri,
       _authorizationOrigin = authorizationOrigin,
       _launchExternalBrowser = launchExternalBrowser,
       _now = now ?? DateTime.now,
       _attemptLifetime = attemptLifetime {
    _eventsSubscription = _authApi.events.listen(
      _handleAuthEvent,
      onError: _handleAuthStreamError,
    );
  }

  final GoTruePort _authApi;
  final AuthSecretStore _secretStore;
  final Uri _callbackUri;
  final Uri _authorizationOrigin;
  final Future<bool> Function(Uri uri) _launchExternalBrowser;
  final DateTime Function() _now;
  final Duration _attemptLifetime;
  late final StreamSubscription<GoTrueEvent> _eventsSubscription;

  AuthViewState _state = const AuthViewState(
    phase: AuthPhase.initializing,
    generation: 0,
  );
  Session? _session;
  _PendingAttempt? _pendingAttempt;
  _PendingAttempt? _processingAttempt;
  Future<void>? _initialization;
  Future<AuthCredential>? _refreshInFlight;
  int _generation = 0;
  int _attemptCounter = 0;
  bool _disposed = false;

  @override
  AuthViewState get state => _state;

  @override
  Future<void> initialize() {
    final existing = _initialization;
    if (existing != null && _state.phase != AuthPhase.unavailable) {
      return existing;
    }
    if (existing != null) {
      _setState(
        AuthViewState(phase: AuthPhase.initializing, generation: _generation),
      );
    }
    return _initialization = _initialize();
  }

  Future<void> _initialize() async {
    String? storedSession;
    try {
      await _secretStore.clearLegacyAuthMaterial();
      storedSession = await _secretStore.readSession();
    } on Object {
      _setState(
        AuthViewState(
          phase: AuthPhase.unavailable,
          generation: _generation,
          message:
              'O armazenamento da sessão não está disponível. O acesso foi interrompido.',
        ),
      );
      return;
    }
    if (storedSession != null) {
      try {
        final decoded = jsonDecode(storedSession);
        if (decoded is! Map<String, dynamic>) {
          throw const FormatException('Stored session must be an object.');
        }
        final sanitized = _sanitizeSessionJson(decoded);
        final session = await _authApi.restoreSession(jsonEncode(sanitized));
        if (session == null) {
          await _clearDefinitively(
            'A sessão anterior expirou. Entre novamente.',
          );
          return;
        }
        await _acceptSession(session, expectedGeneration: _generation);
        try {
          final current = await _authApi.getSession();
          if (current == null) {
            await _clearDefinitively(
              'A sessão anterior expirou. Entre novamente.',
            );
            return;
          }
          await _acceptSession(current, expectedGeneration: _generation);
          _authApi.startAutoRefresh();
        } on Object catch (error) {
          if (_isDefinitiveSessionError(error)) {
            await _clearDefinitively(
              'A sessão anterior foi rejeitada. Entre novamente.',
            );
            return;
          } else {
            _setState(
              _state.copyWith(
                message:
                    'A autenticação está indisponível. A sessão local foi mantida.',
              ),
            );
          }
        }
        if (_state.phase == AuthPhase.authenticated) return;
      } on Object catch (error) {
        if (_isTransientAuthFailure(error)) {
          _setState(
            AuthViewState(
              phase: AuthPhase.unavailable,
              generation: _generation,
              message:
                  'A autenticação está temporariamente indisponível. A sessão salva foi mantida; tente novamente.',
            ),
          );
          return;
        }
        if (_isDefinitiveSessionError(error)) {
          await _clearDefinitively(
            'A sessão anterior foi rejeitada. Entre novamente.',
          );
          return;
        }
        await _clearLocalMaterial();
        _session = null;
        _setState(
          AuthViewState(
            phase: AuthPhase.signedOut,
            generation: _generation,
            message: 'A sessão salva não pôde ser lida. Entre novamente.',
          ),
        );
        return;
      }
    }

    try {
      final pending = await _readPendingAttempt();
      if (pending != null && pending.expiresAt.isAfter(_now())) {
        _pendingAttempt = pending.withGeneration(_generation);
        await _secretStore.writePendingAttempt(
          jsonEncode(_pendingAttempt!.toJson()),
        );
        _setState(
          AuthViewState(
            phase: AuthPhase.signingIn,
            generation: _generation,
            message: 'Aguardando o retorno do Google.',
          ),
        );
        return;
      }
      if (pending != null) {
        await _secretStore.clearOAuthMaterial();
      }
    } on Object {
      _setState(
        AuthViewState(
          phase: AuthPhase.unavailable,
          generation: _generation,
          message:
              'O armazenamento da sessão não está disponível. O acesso foi interrompido.',
        ),
      );
      return;
    }
    _setState(
      AuthViewState(phase: AuthPhase.signedOut, generation: _generation),
    );
  }

  @override
  Future<void> beginGoogleSignIn() async {
    await initialize();
    if (_state.phase == AuthPhase.authenticated) {
      throw StateError('Sign out before starting another Google sign-in.');
    }
    if (_pendingAttempt != null || _processingAttempt != null) return;

    _setState(
      AuthViewState(
        phase: AuthPhase.signingIn,
        generation: _generation,
        message: 'Abrindo o Google no navegador…',
      ),
    );
    try {
      final signInUri = await _authApi.createGoogleSignInUrl(
        _callbackUri.toString(),
      );
      if (!_isAllowedAuthorizationUrl(signInUri)) {
        await _secretStore.clearOAuthMaterial();
        _setState(
          AuthViewState(
            phase: AuthPhase.signedOut,
            generation: _generation,
            message: 'O endereço de autenticação recebido é inválido.',
          ),
        );
        return;
      }

      final attempt = _PendingAttempt(
        id: '${_now().microsecondsSinceEpoch}-${_attemptCounter++}',
        generation: _generation,
        expiresAt: _now().add(_attemptLifetime),
      );
      _pendingAttempt = attempt;
      await _secretStore.writePendingAttempt(jsonEncode(attempt.toJson()));
      final launched = await _launchExternalBrowser(signInUri);
      if (!launched) {
        _pendingAttempt = null;
        await _secretStore.clearOAuthMaterial();
        _setState(
          AuthViewState(
            phase: AuthPhase.signedOut,
            generation: _generation,
            message: 'Não foi possível abrir o navegador.',
          ),
        );
        return;
      }
      _setState(
        AuthViewState(
          phase: AuthPhase.signingIn,
          generation: _generation,
          message: 'Conclua o acesso no navegador e aguarde o retorno.',
        ),
      );
    } on Object {
      _pendingAttempt = null;
      await _secretStore.clearOAuthMaterial();
      _setState(
        AuthViewState(
          phase: AuthPhase.signedOut,
          generation: _generation,
          message: 'Não foi possível iniciar o acesso com Google.',
        ),
      );
    }
  }

  @override
  Future<void> cancelGoogleSignIn() async {
    await initialize();
    if (_pendingAttempt == null && _processingAttempt == null) return;
    _generation++;
    _pendingAttempt = null;
    _processingAttempt = null;
    _setState(
      AuthViewState(
        phase: AuthPhase.signedOut,
        generation: _generation,
        message: 'Acesso cancelado.',
      ),
    );
    await _secretStore.clearOAuthMaterial();
  }

  @override
  Future<void> handleCallback(Uri uri) async {
    await initialize();
    if (uri.scheme != _callbackUri.scheme) return;
    if (!_isExpectedCallbackUri(uri)) {
      await _cancelInvalidCallback();
      return;
    }
    if (_processingAttempt != null) return;

    final attempt = _pendingAttempt ?? await _readPendingAttempt();
    if (attempt == null || attempt.generation != _generation) {
      if (_state.phase == AuthPhase.signingIn) {
        await _cancelInvalidCallback();
      }
      return;
    }
    if (!attempt.expiresAt.isAfter(_now())) {
      await _cancelInvalidCallback(
        'A tentativa de acesso expirou. Inicie novamente.',
      );
      return;
    }

    final query = uri.queryParametersAll;
    const allowedKeys = <String>{
      'code',
      'error',
      'error_code',
      'error_description',
    };
    if (query.keys.any((key) => !allowedKeys.contains(key)) ||
        query.values.any((values) => values.length != 1)) {
      await _cancelInvalidCallback();
      return;
    }
    final code = query['code']?.single;
    final authError = query['error']?.single;
    if ((code == null || code.isEmpty) ==
        (authError == null || authError.isEmpty)) {
      await _cancelInvalidCallback();
      return;
    }

    _pendingAttempt = null;
    _processingAttempt = attempt;
    await _secretStore.writePendingAttempt(null);
    if (authError != null) {
      final ownsAttempt = _ownsProcessingAttempt(attempt);
      if (ownsAttempt) {
        _processingAttempt = null;
        await _secretStore.clearPkce();
      }
      _setState(
        AuthViewState(
          phase: AuthPhase.signedOut,
          generation: _generation,
          message: authError == 'access_denied'
              ? 'Acesso com Google cancelado.'
              : 'O Google não concluiu a autenticação.',
        ),
      );
      return;
    }

    try {
      final session = await _authApi.exchangeCodeForSession(code!);
      if (_generation != attempt.generation) {
        await _clearSdkSessionIfItIsStale(session);
        return;
      }
      if (session == null) {
        _setState(
          AuthViewState(
            phase: AuthPhase.signedOut,
            generation: _generation,
            message: 'O Google não retornou uma sessão Mindforge.',
          ),
        );
        return;
      }
      await _acceptSession(session, expectedGeneration: attempt.generation);
      _authApi.startAutoRefresh();
    } on Object {
      if (_generation == attempt.generation) {
        _setState(
          AuthViewState(
            phase: AuthPhase.signedOut,
            generation: _generation,
            message:
                'Não foi possível concluir o acesso. Inicie uma nova tentativa.',
          ),
        );
      }
    } finally {
      if (_ownsProcessingAttempt(attempt)) {
        _processingAttempt = null;
        await _secretStore.clearPkce();
      }
    }
  }

  bool _ownsProcessingAttempt(_PendingAttempt attempt) =>
      identical(_processingAttempt, attempt);

  @override
  Future<void> signOut() async {
    await initialize();
    final nextGeneration = _generation + 1;
    _generation = nextGeneration;
    _session = null;
    _pendingAttempt = null;
    _processingAttempt = null;
    _authApi.stopAutoRefresh();
    _setState(
      AuthViewState(phase: AuthPhase.signedOut, generation: _generation),
    );

    String? localCleanupMessage;
    try {
      await _secretStore.clearAuthMaterial();
    } on Object {
      localCleanupMessage =
          'Não foi possível limpar todos os dados locais da sessão.';
    }
    try {
      await _authApi.signOutLocal();
    } on Object {
      localCleanupMessage ??=
          'A sessão local foi encerrada; o serviço de autenticação está indisponível.';
    }
    if (!_disposed &&
        _generation == nextGeneration &&
        localCleanupMessage != null) {
      _setState(
        AuthViewState(
          phase: AuthPhase.signedOut,
          generation: _generation,
          message: localCleanupMessage,
        ),
      );
    }
  }

  @override
  Future<AuthCredential> credential() async {
    await initialize();
    final session = _session;
    final state = _state;
    if (state.phase != AuthPhase.authenticated ||
        session == null ||
        session.accessToken.isEmpty) {
      throw const UnauthenticatedException();
    }
    return AuthCredential(
      accessToken: session.accessToken,
      userId: session.user.id,
      generation: _generation,
    );
  }

  @override
  Future<AuthCredential> refreshAfterUnauthorized(AuthCredential rejected) {
    if (_generation != rejected.generation ||
        _state.userId != rejected.userId ||
        _session == null) {
      return Future<AuthCredential>.error(const StaleAuthOperationException());
    }
    if (_session!.accessToken != rejected.accessToken) {
      return credential();
    }
    return _refreshInFlight ??= _refresh(rejected);
  }

  Future<AuthCredential> _refresh(AuthCredential rejected) async {
    try {
      final refreshed = await _authApi.refreshSession();
      if (_generation != rejected.generation ||
          _state.userId != rejected.userId) {
        throw const StaleAuthOperationException();
      }
      if (refreshed == null) {
        await expireAfterUnauthorized(rejected);
        throw const UnauthenticatedException();
      }
      await _acceptSession(refreshed, expectedGeneration: rejected.generation);
      return credential();
    } on Object catch (error) {
      if (_isDefinitiveSessionError(error)) {
        await expireAfterUnauthorized(rejected);
      }
      rethrow;
    } finally {
      _refreshInFlight = null;
    }
  }

  @override
  Future<void> expireAfterUnauthorized(AuthCredential rejected) async {
    if (_generation != rejected.generation || _state.userId != rejected.userId)
      return;
    await _clearDefinitively('A sessão foi rejeitada. Entre novamente.');
  }

  Future<void> _acceptSession(
    Session incoming, {
    required int expectedGeneration,
  }) async {
    if (_generation != expectedGeneration) return;
    final session = _withoutProviderTokens(incoming);
    final userId = session.user.id;
    if (userId.isEmpty || session.accessToken.isEmpty) {
      throw const FormatException('Auth returned an incomplete session.');
    }
    if (_state.userId != userId) _generation++;
    final acceptedGeneration = _generation;
    _session = session;
    final displayName = _displayName(session.user);
    _setState(
      AuthViewState(
        phase: AuthPhase.authenticated,
        generation: _generation,
        userId: userId,
        displayName: displayName,
        message: null,
      ),
    );
    try {
      await _secretStore.writeSession(jsonEncode(_sessionJson(session)));
    } on Object {
      if (_generation == acceptedGeneration && _state.userId == userId) {
        _setState(
          _state.copyWith(
            message: 'A sessão está ativa, mas não pôde ser salva localmente.',
          ),
        );
      }
    }
  }

  void _handleAuthEvent(GoTrueEvent event) {
    if (_disposed) return;
    if (_processingAttempt != null) return;
    if (event.kind == GoTrueEventKind.signedOut) {
      if (_state.phase == AuthPhase.authenticated) {
        unawaited(_clearDefinitively('A sessão foi encerrada.'));
      }
      return;
    }
    final session = event.session;
    if (session == null) return;
    final isCurrentUser = _state.userId == session.user.id;
    final isSignInInProgress =
        _pendingAttempt != null || _state.phase == AuthPhase.initializing;
    if (!isCurrentUser && !isSignInInProgress) return;
    final expectedGeneration = _processingAttempt?.generation ?? _generation;
    unawaited(_acceptSession(session, expectedGeneration: expectedGeneration));
  }

  void _handleAuthStreamError(Object error, StackTrace stackTrace) {
    if (_disposed) return;
    if (_state.phase == AuthPhase.authenticated) {
      _setState(
        _state.copyWith(
          message:
              'A autenticação está indisponível. A sessão local foi mantida.',
        ),
      );
    }
  }

  Future<void> _clearDefinitively(String message) async {
    _generation++;
    _session = null;
    _pendingAttempt = null;
    _processingAttempt = null;
    _authApi.stopAutoRefresh();
    _setState(
      AuthViewState(
        phase: AuthPhase.signedOut,
        generation: _generation,
        message: message,
      ),
    );
    try {
      await _secretStore.clearAuthMaterial();
    } on Object {
      _setState(
        _state.copyWith(
          message:
              'A sessão expirou, mas não foi possível limpar os dados locais.',
        ),
      );
    }
    try {
      await _authApi.signOutLocal();
    } on Object {
      // The local app state and Keychain were already cleared.
    }
  }

  Future<void> _clearLocalMaterial() async {
    try {
      await _secretStore.clearAuthMaterial();
    } on Object {
      // Initialization below reports that the saved session could not be used.
    }
    try {
      await _authApi.signOutLocal();
    } on Object {
      // No local session is exposed after malformed storage.
    }
  }

  Future<void> _cancelInvalidCallback([
    String message =
        'O retorno de autenticação não corresponde a uma tentativa válida.',
  ]) async {
    final hadAttempt = _pendingAttempt != null || _processingAttempt != null;
    _pendingAttempt = null;
    _processingAttempt = null;
    if (_state.phase != AuthPhase.authenticated && hadAttempt) {
      _generation++;
      _setState(
        AuthViewState(
          phase: AuthPhase.signedOut,
          generation: _generation,
          message: message,
        ),
      );
    }
    await _secretStore.clearOAuthMaterial();
  }

  Future<void> _clearSdkSessionIfItIsStale(Session? returnedSession) async {
    if (returnedSession == null ||
        _authApi.current?.accessToken != returnedSession.accessToken) {
      return;
    }
    try {
      await _authApi.signOutLocal();
    } on Object {
      // Keep the current app generation authoritative even if the SDK is late.
    }
  }

  Future<_PendingAttempt?> _readPendingAttempt() async {
    final raw = await _secretStore.readPendingAttempt();
    if (raw == null) return null;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) return null;
      return _PendingAttempt.fromJson(decoded);
    } on Object {
      await _secretStore.clearOAuthMaterial();
      return null;
    }
  }

  bool _isAllowedAuthorizationUrl(Uri uri) {
    final localHttp =
        uri.scheme == 'http' &&
        (uri.host == '127.0.0.1' || uri.host == 'web-mind.home.arpa');
    final secureHttps = uri.scheme == 'https';
    return (localHttp || secureHttps) &&
        uri.origin == _authorizationOrigin.origin &&
        uri.path == '${_authorizationOrigin.path}/authorize' &&
        uri.userInfo.isEmpty &&
        !uri.hasFragment;
  }

  bool _isExpectedCallbackUri(Uri uri) =>
      uri.scheme == _callbackUri.scheme &&
      uri.host == _callbackUri.host &&
      uri.path == _callbackUri.path &&
      uri.userInfo.isEmpty &&
      uri.port == _callbackUri.port &&
      !uri.hasFragment;

  static bool _isTransientAuthFailure(Object error) {
    if (error is AuthRetryableFetchException ||
        error is TimeoutException ||
        platform.isSocketException(error) ||
        error is http.ClientException) {
      return true;
    }
    if (error is AuthApiException) {
      final statusCode = int.tryParse(error.statusCode ?? '');
      return statusCode != null && statusCode >= 500;
    }
    return false;
  }

  static bool _isDefinitiveSessionError(Object error) {
    if (error is AuthInvalidJwtException ||
        error is AuthSessionMissingException) {
      return true;
    }
    if (error is AuthApiException) {
      const invalidSessionCodes = <String>{
        'bad_jwt',
        'refresh_token_not_found',
        'refresh_token_already_used',
        'session_expired',
        'session_missing',
        'session_not_found',
      };
      return error.statusCode == '401' ||
          invalidSessionCodes.contains(error.code);
    }
    return false;
  }

  static Map<String, Object?> _sanitizeSessionJson(Map<String, dynamic> input) {
    final session = Session.fromJson(_removeProviderTokens(input));
    if (session == null) {
      throw const FormatException('Stored session has no access token.');
    }
    return _sessionJson(session);
  }

  static Session _withoutProviderTokens(Session session) {
    final sanitized = Session.fromJson(_removeProviderTokens(session.toJson()));
    if (sanitized == null) {
      throw const FormatException(
        'Auth returned a session without an access token.',
      );
    }
    return sanitized;
  }

  static Map<String, Object?> _sessionJson(Session session) {
    final json = _removeProviderTokens(session.toJson());
    return <String, Object?>{
      'access_token': json['access_token'],
      'expires_in': json['expires_in'],
      'expires_at': json['expires_at'],
      'refresh_token': json['refresh_token'],
      'token_type': json['token_type'],
      'user': json['user'],
    }..removeWhere((_, value) => value == null);
  }

  static Map<String, dynamic> _removeProviderTokens(
    Map<String, dynamic> input,
  ) => _stripProviderTokens(input) as Map<String, dynamic>;

  static Object? _stripProviderTokens(Object? value) {
    if (value is Map) {
      final output = <String, dynamic>{};
      for (final entry in value.entries) {
        if (entry.key is! String ||
            entry.key == 'provider_token' ||
            entry.key == 'provider_refresh_token') {
          continue;
        }
        output[entry.key as String] = _stripProviderTokens(entry.value);
      }
      return output;
    }
    if (value is List) {
      return value.map(_stripProviderTokens).toList(growable: false);
    }
    return value;
  }

  static String? _displayName(User user) {
    final metadata = user.userMetadata;
    final name = metadata?['full_name'] ?? metadata?['name'];
    return name is String && name.trim().isNotEmpty ? name : user.email;
  }

  void _setState(AuthViewState state) {
    if (_disposed) return;
    _state = state;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    unawaited(_eventsSubscription.cancel());
    _authApi.dispose();
    super.dispose();
  }
}

extension on AuthViewState {
  AuthViewState copyWith({String? message}) => AuthViewState(
    phase: phase,
    generation: generation,
    userId: userId,
    displayName: displayName,
    message: message,
  );
}

class _PendingAttempt {
  const _PendingAttempt({
    required this.id,
    required this.generation,
    required this.expiresAt,
  });

  final String id;
  final int generation;
  final DateTime expiresAt;

  Map<String, Object?> toJson() => <String, Object?>{
    'id': id,
    'generation': generation,
    'expires_at': expiresAt.millisecondsSinceEpoch,
  };

  factory _PendingAttempt.fromJson(Map<String, dynamic> json) {
    final id = json['id'];
    final generation = json['generation'];
    final expiresAt = json['expires_at'];
    if (id is! String || generation is! int || expiresAt is! int) {
      throw const FormatException('Stored OAuth attempt is invalid.');
    }
    return _PendingAttempt(
      id: id,
      generation: generation,
      expiresAt: DateTime.fromMillisecondsSinceEpoch(expiresAt),
    );
  }

  _PendingAttempt withGeneration(int generation) =>
      _PendingAttempt(id: id, generation: generation, expiresAt: expiresAt);
}
