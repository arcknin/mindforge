import 'dart:async';
import 'dart:convert';

import 'package:gotrue/gotrue.dart';
import 'package:http/http.dart' as http;

import 'api_auth_http_client.dart';
import 'auth_secret_store.dart';
import 'gotrue_auth_gateway.dart';
import 'owned_pkce_storage.dart';

/// Adapter around gotrue-dart's supported PKCE/session machinery.
///
/// The SDK remains an implementation detail. Its base URL points at the
/// Brain Engine Auth facade and every network request crosses
/// [ApiAuthHttpClient].
class GoTrueSdkPort implements GoTruePort {
  GoTrueSdkPort({
    required Uri apiUrl,
    required Uri callbackUri,
    required AuthSecretStore secretStore,
    http.Client? httpClient,
  }) : _callbackUri = callbackUri,
       _apiOrigin = apiUrl.origin,
       _secretStore = secretStore {
    _restrictedClient = ApiAuthHttpClient(
      apiUrl: apiUrl,
      allowedRedirectUri: callbackUri,
      inner: httpClient,
    );
    _authFacadeUrl = ApiAuthHttpClient.authApiUrlFor(apiUrl);
    _installClient();
  }

  final Uri _callbackUri;
  final String _apiOrigin;
  final AuthSecretStore _secretStore;
  final StreamController<GoTrueEvent> _events =
      StreamController<GoTrueEvent>.broadcast();
  late final ApiAuthHttpClient _restrictedClient;
  late final Uri _authFacadeUrl;
  late GoTrueClient _client;
  late StreamSubscription<AuthState> _sdkSubscription;
  Session? _verifiedSession;
  int _explicitOperationDepth = 0;
  int _operationEpoch = 0;
  int _sessionEpoch = 0;
  bool _disposed = false;

  void _installClient() {
    final client = GoTrueClient(
      url: _authFacadeUrl.toString(),
      httpClient: _restrictedClient,
      asyncStorage: OwnedPkceStorage(_secretStore),
      autoRefreshToken: true,
      flowType: AuthFlowType.pkce,
    );
    _client = client;
    _sdkSubscription = client.onAuthStateChange.listen(
      (state) => _handleSdkEvent(client, state),
      onError: (Object error, StackTrace stackTrace) {
        if (!_events.isClosed && identical(client, _client)) {
          _events.addError(error, stackTrace);
        }
      },
    );
  }

  void _replaceClientForNewLogin() {
    _operationEpoch++;
    _sessionEpoch++;
    _verifiedSession = null;
    unawaited(_sdkSubscription.cancel());
    _client.dispose();
    _installClient();
  }

  @override
  Stream<GoTrueEvent> get events => _events.stream;

  @override
  Session? get current => _verifiedSession ?? _client.currentSession;

  @override
  Future<Uri> createGoogleSignInUrl(String redirectUri) async {
    if (redirectUri != _callbackUri.toString()) {
      throw const AuthApiException(
        'The Auth callback is not configured for this client.',
        statusCode: '400',
        code: 'invalid_request',
      );
    }
    _replaceClientForNewLogin();
    final client = _client;
    final response = await client.getOAuthSignInUrl(
      provider: OAuthProvider.google,
      redirectTo: redirectUri,
    );
    final uri = Uri.parse(response.url);
    if (uri.origin != _apiOrigin ||
        uri.path != '${_authFacadeUrl.path}/authorize' ||
        uri.userInfo.isNotEmpty ||
        uri.hasFragment) {
      throw const AuthApiException(
        'The Auth authorization URL is invalid.',
        statusCode: '503',
        code: 'auth_unavailable',
      );
    }
    return uri;
  }

  @override
  Future<Session?> exchangeCodeForSession(String code) async {
    return _runSessionOperation((client) async {
      final response = await _withSdkEventsSuppressed(
        () => client.exchangeCodeForSession(code),
      );
      return response.session;
    });
  }

  @override
  Future<Session?> getSession() async {
    return _runSessionOperation(
      (client) => _withSdkEventsSuppressed(client.getSession),
    );
  }

  @override
  Future<Session?> refreshSession() async {
    return _runSessionOperation((client) async {
      final response = await _withSdkEventsSuppressed(client.refreshSession);
      return response.session;
    });
  }

  @override
  Future<Session?> restoreSession(String serializedSession) async {
    return _runSessionOperation((client) async {
      final response = await _withSdkEventsSuppressed(
        () => client.recoverSession(serializedSession),
      );
      return response.session;
    });
  }

  @override
  Future<void> signOutLocal() async {
    _operationEpoch++;
    _sessionEpoch++;
    _verifiedSession = null;
    final client = _client;
    await _withSdkEventsSuppressed(
      () => client.signOut(scope: SignOutScope.local),
    );
  }

  @override
  void startAutoRefresh() => _client.startAutoRefresh();

  @override
  void stopAutoRefresh() => _client.stopAutoRefresh();

  Future<T> _withSdkEventsSuppressed<T>(Future<T> Function() action) async {
    _explicitOperationDepth++;
    try {
      return await action();
    } finally {
      _explicitOperationDepth--;
    }
  }

  Future<Session?> _runSessionOperation(
    Future<Session?> Function(GoTrueClient client) operation,
  ) async {
    final epoch = ++_operationEpoch;
    final sessionEpoch = _sessionEpoch;
    final client = _client;
    final expectedUserId =
        _verifiedSession?.user.id ?? client.currentSession?.user.id;
    final previousSession = _verifiedSession;
    final session = await operation(client);
    final verified = await _verifySession(client, session);
    if (verified != null &&
        expectedUserId != null &&
        verified.user.id != expectedUserId) {
      if (sessionEpoch == _sessionEpoch && identical(client, _client)) {
        await _restorePreviousSession(client, previousSession);
      }
      throw _unavailable('Auth session changed identity.');
    }
    if (_disposed || epoch != _operationEpoch) {
      if (session != null && sessionEpoch != _sessionEpoch) {
        await _discardSdkSessionIfCurrent(client, session);
      }
      return verified;
    }
    _verifiedSession = verified;
    return verified;
  }

  Future<void> _restorePreviousSession(
    GoTrueClient client,
    Session? previousSession,
  ) async {
    if (previousSession == null) return;
    try {
      await _withSdkEventsSuppressed(
        () => client.recoverSession(jsonEncode(previousSession.toJson())),
      );
    } on Object {
      // A later refresh will surface the real session state if restoration is
      // no longer possible.
    }
  }

  Future<void> _discardSdkSessionIfCurrent(
    GoTrueClient client,
    Session stale,
  ) async {
    if (client.currentSession?.accessToken != stale.accessToken) return;
    try {
      await _withSdkEventsSuppressed(
        () => client.signOut(scope: SignOutScope.local),
      );
    } on Object {
      // The caller's generation remains authoritative if cleanup is unavailable.
    }
  }

  Future<Session?> _verifySession(GoTrueClient client, Session? session) async {
    if (session == null) return null;
    if (session.accessToken.isEmpty ||
        session.tokenType.toLowerCase() != 'bearer') {
      throw _unavailable('Auth returned an invalid session.');
    }
    final response = await client.getUser(session.accessToken);
    final user = response.user;
    if (user == null) throw _unavailable('Auth returned no user.');
    if (user.id != session.user.id) {
      throw _unavailable('Auth session and user do not match.');
    }
    final projected = _projectEligibleUser(user);
    return Session(
      accessToken: session.accessToken,
      refreshToken: session.refreshToken,
      tokenType: 'bearer',
      expiresIn: session.expiresIn,
      user: projected,
    );
  }

  User _projectEligibleUser(User user) {
    if (!_isEligibleUser(user)) {
      throw const AuthApiException(
        'The Auth user is not eligible for Mindforge.',
        statusCode: '401',
        code: 'unauthenticated',
      );
    }
    final createdAt = DateTime.tryParse(user.createdAt);
    final confirmedAt = DateTime.tryParse(user.emailConfirmedAt!);
    if (createdAt == null || confirmedAt == null) {
      throw _unavailable('Auth returned invalid user dates.');
    }

    final sourceMetadata = user.userMetadata ?? const <String, dynamic>{};
    final metadata = <String, dynamic>{};
    for (final key in const <String>['name', 'full_name']) {
      final value = sourceMetadata[key];
      if (value is String && value.length <= 255) metadata[key] = value;
    }
    final email = user.email;
    if (email != null && email.length > 320) {
      throw _unavailable('Auth returned an invalid user email.');
    }
    return User(
      id: user.id,
      aud: 'authenticated',
      role: 'authenticated',
      isAnonymous: false,
      appMetadata: const <String, dynamic>{
        'provider': 'google',
        'providers': <String>['google'],
      },
      userMetadata: metadata.isEmpty ? null : metadata,
      createdAt: createdAt.toUtc().toIso8601String(),
      emailConfirmedAt: confirmedAt.toUtc().toIso8601String(),
      email: email,
    );
  }

  static bool _isEligibleUser(User user) {
    if (user.id.isEmpty ||
        !RegExp(
          r'^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$',
        ).hasMatch(user.id) ||
        user.aud != 'authenticated' ||
        user.role != 'authenticated' ||
        user.isAnonymous ||
        user.emailConfirmedAt == null) {
      return false;
    }
    final provider = user.appMetadata['provider'];
    final providers = user.appMetadata['providers'];
    return provider == 'google' ||
        providers is List && providers.whereType<String>().contains('google');
  }

  static AuthApiException _unavailable(String message) =>
      AuthApiException(message, statusCode: '503', code: 'auth_unavailable');

  void _handleSdkEvent(GoTrueClient source, AuthState state) {
    if (_disposed || !identical(source, _client)) return;
    if (state.event == AuthChangeEvent.signedOut) {
      _verifiedSession = null;
      _events.add(const GoTrueEvent(GoTrueEventKind.signedOut, null));
      return;
    }
    if (_explicitOperationDepth != 0 ||
        state.event != AuthChangeEvent.tokenRefreshed ||
        state.session == null) {
      return;
    }
    unawaited(_verifyAutoRefresh(source, state.session!));
  }

  Future<void> _verifyAutoRefresh(GoTrueClient source, Session incoming) async {
    final epoch = _operationEpoch;
    final sessionEpoch = _sessionEpoch;
    final expectedUserId =
        _verifiedSession?.user.id ?? source.currentSession?.user.id;
    try {
      final verified = await _verifySession(source, incoming);
      if (verified != null &&
          expectedUserId != null &&
          verified.user.id != expectedUserId) {
        await _restorePreviousSession(source, _verifiedSession);
        throw _unavailable('Auth session changed identity.');
      }
      if (_disposed ||
          epoch != _operationEpoch ||
          sessionEpoch != _sessionEpoch ||
          !identical(source, _client) ||
          source.currentSession?.accessToken != incoming.accessToken) {
        return;
      }
      _verifiedSession = verified;
      _events.add(GoTrueEvent(GoTrueEventKind.tokenRefreshed, verified));
    } on Object catch (error, stackTrace) {
      final stillCurrent =
          !_disposed &&
          epoch == _operationEpoch &&
          sessionEpoch == _sessionEpoch &&
          identical(source, _client) &&
          source.currentSession?.accessToken == incoming.accessToken;
      if (stillCurrent && _isDefinitiveValidationError(error)) {
        _verifiedSession = null;
        _events.add(const GoTrueEvent(GoTrueEventKind.signedOut, null));
      } else if (!_disposed) {
        _events.addError(error, stackTrace);
      }
    }
  }

  static bool _isDefinitiveValidationError(Object error) =>
      error is AuthInvalidJwtException ||
      error is AuthSessionMissingException ||
      error is AuthApiException &&
          (error.statusCode == '401' || error.code == 'unauthenticated');

  @override
  void dispose() {
    if (_disposed) return;
    _disposed = true;
    unawaited(_sdkSubscription.cancel());
    _client.dispose();
    _restrictedClient.close();
    unawaited(_events.close());
  }
}
