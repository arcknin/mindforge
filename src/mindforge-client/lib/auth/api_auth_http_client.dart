import 'package:http/http.dart' as http;

/// HTTP boundary used by the embedded GoTrue SDK.
///
/// The SDK still owns PKCE generation, refresh de-duplication and session
/// parsing, but it can only send the small public Auth subset through the
/// Brain Engine origin. A new route cannot be reached accidentally by a
/// future SDK method.
class ApiAuthHttpClient extends http.BaseClient {
  ApiAuthHttpClient({
    required Uri apiUrl,
    required Uri allowedRedirectUri,
    http.Client? inner,
  }) : _apiOrigin = _validateApiUrl(apiUrl).origin,
       _authPath = _authPathFor(apiUrl),
       _allowedRedirectUri = allowedRedirectUri,
       _inner = inner ?? http.Client();

  final String _apiOrigin;
  final String _authPath;
  final Uri _allowedRedirectUri;
  final http.Client _inner;

  static Uri authApiUrlFor(Uri apiUrl) =>
      apiUrl.replace(path: _authPathFor(apiUrl), query: null, fragment: null);

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) {
    _validateRequest(request);
    request.followRedirects = false;
    request.maxRedirects = 0;
    return _inner.send(request);
  }

  @override
  void close() => _inner.close();

  void _validateRequest(http.BaseRequest request) {
    final uri = request.url;
    if (uri.origin != _apiOrigin ||
        uri.userInfo.isNotEmpty ||
        uri.hasFragment) {
      throw _clientError('Auth request origin is outside the configured API.');
    }

    final rawPath = uri
        .toString()
        .substring(uri.origin.length)
        .split('?')
        .first;
    final route = _routeFor(uri.path);
    if (rawPath != '$_authPath/$route') {
      throw _clientError('Auth request path is outside the allowlist.');
    }

    final query = uri.queryParametersAll;
    if (query.values.any((values) => values.length != 1)) {
      throw _clientError('Auth request contains duplicate query fields.');
    }

    switch (route) {
      case 'authorize':
        _expectMethod(request, 'GET');
        _expectQueryKeys(query, const <String>{
          'provider',
          'flow_type',
          'redirect_to',
          'code_challenge',
          'code_challenge_method',
        });
        if (query['provider']?.single != 'google' ||
            query['flow_type']?.single != 'pkce' ||
            query['code_challenge_method']?.single != 's256' ||
            !_isPkceChallenge(query['code_challenge']?.single) ||
            (query['redirect_to']?.single ?? '').isEmpty) {
          throw _clientError('Auth authorize parameters are invalid.');
        }
        if (query['redirect_to']?.single != _allowedRedirectUri.toString()) {
          throw _clientError('Auth redirect is not allowlisted.');
        }
        break;
      case 'callback':
        _expectMethod(request, 'GET');
        _expectQueryKeys(
          query,
          const <String>{
            'state',
            'code',
            'error',
            'error_description',
            'error_uri',
            'scope',
            'authuser',
            'prompt',
            'hd',
            'iss',
          },
          required: const <String>{'state'},
        );
        final hasCode = (query['code']?.single ?? '').isNotEmpty;
        final hasError = (query['error']?.single ?? '').isNotEmpty;
        if (hasCode == hasError) {
          throw _clientError('Auth callback result is invalid.');
        }
        break;
      case 'token':
        _expectMethod(request, 'POST');
        _expectQueryKeys(query, const <String>{'grant_type'});
        final grant = query['grant_type']?.single;
        if (grant != 'pkce' && grant != 'refresh_token') {
          throw _clientError('Auth token grant is not allowed.');
        }
        break;
      case 'user':
        _expectMethod(request, 'GET');
        if (query.isNotEmpty) {
          throw _clientError('Auth user does not accept query parameters.');
        }
        _requireAuthorization(request);
        break;
      case 'logout':
        _expectMethod(request, 'POST');
        _expectQueryKeys(query, const <String>{'scope'});
        if (query['scope']?.single != 'local') {
          throw _clientError('Only local Auth logout is allowed.');
        }
        _requireAuthorization(request);
        break;
      default:
        throw _clientError('Auth request path is outside the allowlist.');
    }

    _sanitizeHeaders(request);
  }

  String _routeFor(String path) {
    if (!path.startsWith('$_authPath/')) return '';
    return path.substring(_authPath.length + 1);
  }

  void _sanitizeHeaders(http.BaseRequest request) {
    const forbidden = <String>{
      'apikey',
      'cookie',
      'origin',
      'referer',
      'x-forwarded-for',
      'x-forwarded-host',
      'x-forwarded-proto',
      'x-use-cookie',
    };
    for (final key in request.headers.keys.toList(growable: false)) {
      final lower = key.toLowerCase();
      if (forbidden.contains(lower)) {
        throw _clientError('Forbidden Auth header.');
      }
      if (lower != 'authorization' &&
          lower != 'accept' &&
          lower != 'content-type') {
        request.headers.remove(key);
      }
    }
  }

  void _requireAuthorization(http.BaseRequest request) {
    final authorization = request.headers.entries
        .where((entry) => entry.key.toLowerCase() == 'authorization')
        .map((entry) => entry.value)
        .singleOrNull;
    if (authorization == null ||
        !RegExp(r'^Bearer [^\s]+$').hasMatch(authorization)) {
      throw _clientError('Auth request requires a bearer token.');
    }
  }

  void _expectMethod(http.BaseRequest request, String method) {
    if (request.method != method) {
      throw _clientError('Auth method is not allowed.');
    }
  }

  void _expectQueryKeys(
    Map<String, List<String>> query,
    Set<String> allowed, {
    Set<String>? required,
  }) {
    final requiredKeys = required ?? allowed;
    if (query.keys.any((key) => !allowed.contains(key))) {
      throw _clientError('Auth query parameters are not allowed.');
    }
    if (requiredKeys.any((key) => !query.containsKey(key))) {
      throw _clientError('Auth query parameters are incomplete.');
    }
  }

  static bool _isPkceChallenge(String? challenge) =>
      challenge != null && RegExp(r'^[A-Za-z0-9_-]{43}$').hasMatch(challenge);

  static Uri _validateApiUrl(Uri apiUrl) {
    if ((apiUrl.scheme != 'http' && apiUrl.scheme != 'https') ||
        apiUrl.host.isEmpty ||
        apiUrl.userInfo.isNotEmpty ||
        apiUrl.hasQuery ||
        apiUrl.hasFragment) {
      throw const FormatException('The Auth adapter requires an API origin.');
    }
    return apiUrl;
  }

  static String _authPathFor(Uri apiUrl) {
    _validateApiUrl(apiUrl);
    final path = apiUrl.path == '/' ? '' : apiUrl.path;
    final normalized = path.endsWith('/')
        ? path.substring(0, path.length - 1)
        : path;
    return '$normalized/v1/auth';
  }

  static http.ClientException _clientError(String message) =>
      http.ClientException(message);
}

extension on Iterable<String> {
  String? get singleOrNull => length == 1 ? first : null;
}
