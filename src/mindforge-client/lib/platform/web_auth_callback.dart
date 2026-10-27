class WebAuthCallbackCapture {
  const WebAuthCallbackCapture._({
    required this.callbackUri,
    required this.cleanedUrl,
  });

  final Uri callbackUri;
  final Uri cleanedUrl;

  static WebAuthCallbackCapture? capture(Uri currentUri, Uri callbackUri) {
    final isExpectedCallback =
        currentUri.scheme == callbackUri.scheme &&
        currentUri.host == callbackUri.host &&
        currentUri.port == callbackUri.port &&
        currentUri.path == callbackUri.path &&
        currentUri.userInfo.isEmpty;
    if (!isExpectedCallback ||
        (!currentUri.hasQuery && !currentUri.hasFragment)) {
      return null;
    }

    return WebAuthCallbackCapture._(
      callbackUri: currentUri,
      cleanedUrl: Uri(
        scheme: currentUri.scheme,
        host: currentUri.host,
        port: currentUri.hasPort ? currentUri.port : null,
        path: currentUri.path,
      ),
    );
  }
}
