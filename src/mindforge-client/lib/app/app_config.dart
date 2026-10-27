import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, kIsWeb, TargetPlatform;

enum AppTargetPlatform { android, macos, web }

class AppConfig {
  AppConfig._({
    required this.apiUrl,
    required this.authApiUrl,
    required this.callbackUri,
    required this.authStorageNamespace,
    required this.documentStorageNamespace,
    required this.isWeb,
    required this.targetPlatform,
  });

  final Uri apiUrl;
  final Uri authApiUrl;
  final Uri callbackUri;
  final String authStorageNamespace;
  final String documentStorageNamespace;
  final bool isWeb;
  final AppTargetPlatform targetPlatform;

  /// Kept as a source-compatible name for document adapters. It is never an
  /// Auth namespace for new sessions.
  @Deprecated('Use documentStorageNamespace or authStorageNamespace.')
  String get storageNamespace => documentStorageNamespace;

  static AppTargetPlatform detectTargetPlatform({
    bool isWeb = kIsWeb,
    TargetPlatform? platform,
  }) {
    if (isWeb) return AppTargetPlatform.web;
    return switch (platform ?? defaultTargetPlatform) {
      TargetPlatform.android => AppTargetPlatform.android,
      TargetPlatform.macOS => AppTargetPlatform.macos,
      _ => throw const FormatException(
        'This native Mindforge client supports Android and macOS only.',
      ),
    };
  }

  factory AppConfig.fromEnvironment({
    bool isWeb = kIsWeb,
    TargetPlatform? platform,
  }) {
    const environment = String.fromEnvironment('MINDFORGE_ENV');
    const apiUrl = String.fromEnvironment('MINDFORGE_API_URL');
    const webOrigin = String.fromEnvironment('MINDFORGE_WEB_ORIGIN');
    if (environment != 'develop') {
      throw const FormatException('MINDFORGE_ENV must be set to develop.');
    }
    final targetPlatform = AppConfig.detectTargetPlatform(
      isWeb: isWeb,
      platform: platform,
    );
    if (targetPlatform == AppTargetPlatform.web) {
      return AppConfig.developmentWeb(webOrigin: webOrigin);
    }
    return AppConfig.development(
      apiUrl: apiUrl,
      targetPlatform: targetPlatform,
    );
  }

  factory AppConfig.development({
    required String apiUrl,
    AppTargetPlatform targetPlatform = AppTargetPlatform.macos,
  }) {
    if (targetPlatform == AppTargetPlatform.web) {
      throw const FormatException(
        'Web configuration must use AppConfig.developmentWeb.',
      );
    }
    final parsedApiUrl = _parseEndpoint(apiUrl, 'MINDFORGE_API_URL');
    if (parsedApiUrl.host != '127.0.0.1' || parsedApiUrl.port != 11001) {
      throw const FormatException(
        'Brain Engine develop must use http://127.0.0.1:11001.',
      );
    }
    final platformName = targetPlatform == AppTargetPlatform.android
        ? 'android-native'
        : 'macos';
    final documentStorageNamespace =
        'mindforge:develop:$platformName:http://127.0.0.1:11000';
    return AppConfig._(
      apiUrl: parsedApiUrl,
      authApiUrl: _authApiUrl(parsedApiUrl),
      callbackUri: Uri(
        scheme: 'mindforge-dev',
        host: 'auth',
        path: '/callback',
      ),
      authStorageNamespace:
          'mindforge:develop:auth:v2:$platformName:${parsedApiUrl.origin}',
      documentStorageNamespace: documentStorageNamespace,
      isWeb: false,
      targetPlatform: targetPlatform,
    );
  }

  factory AppConfig.developmentWeb({required String webOrigin}) {
    final origin = _parseWebOrigin(webOrigin);
    final apiUrl = origin.replace(path: '/api');
    return AppConfig._(
      apiUrl: apiUrl,
      authApiUrl: _authApiUrl(apiUrl),
      callbackUri: origin.replace(path: '/login/callback'),
      authStorageNamespace: 'mindforge:develop:auth:v2:web:${origin.origin}',
      documentStorageNamespace: 'mindforge:develop:web:${origin.origin}',
      isWeb: true,
      targetPlatform: AppTargetPlatform.web,
    );
  }

  static Uri _authApiUrl(Uri apiUrl) {
    final path = apiUrl.path == '/' ? '' : apiUrl.path;
    final normalized = path.endsWith('/')
        ? path.substring(0, path.length - 1)
        : path;
    return apiUrl.replace(path: '$normalized/v1/auth');
  }

  static Uri _parseEndpoint(String value, String name) {
    final uri = Uri.tryParse(value);
    if (uri == null ||
        uri.scheme != 'http' ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        (uri.path.isNotEmpty && uri.path != '/')) {
      throw FormatException('$name must be an HTTP service origin.');
    }
    return uri.replace(path: '');
  }

  static Uri _parseWebOrigin(String value) {
    final uri = Uri.tryParse(value);
    if (uri == null ||
        (uri.scheme != 'http' && uri.scheme != 'https') ||
        uri.host != 'web-mind.home.arpa' ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        (uri.path.isNotEmpty && uri.path != '/')) {
      throw const FormatException(
        'MINDFORGE_WEB_ORIGIN must be an HTTP or HTTPS origin on web-mind.home.arpa.',
      );
    }
    return uri.replace(path: '');
  }
}
