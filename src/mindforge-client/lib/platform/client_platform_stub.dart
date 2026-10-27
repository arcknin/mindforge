import 'package:app_links/app_links.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:path_provider/path_provider.dart';

import '../app/app_config.dart';
import '../auth/auth_secret_store.dart';
import '../auth/secure_auth_store.dart';
import '../documents/document_repository.dart';
import '../documents/file_document_local_store.dart';
import 'native_auth_callbacks.dart';

AuthSecretStore createAuthSecretStore(AppConfig config) =>
    NamespacedAuthSecretStore(
      namespace: config.authStorageNamespace,
      legacyNamespaces: <String>[config.documentStorageNamespace],
      keychain: FlutterKeychainStore(),
    );

Future<bool> launchGoogleSignIn(Uri uri) =>
    launchUrl(uri, mode: LaunchMode.externalApplication);

Stream<Uri> incomingAuthCallbacks(Uri callbackUri) {
  final links = AppLinks();
  return NativeAuthCallbackAdapter(
    callbackUri: callbackUri,
    getInitialLink: links.getInitialLink,
    linkStream: links.uriLinkStream,
  ).callbacks;
}

DocumentLocalStore? createDocumentLocalStore(AppConfig config) =>
    FileDocumentLocalStore(
      namespace: '${config.documentStorageNamespace}:documents:v1',
      directory: getApplicationSupportDirectory,
    );
