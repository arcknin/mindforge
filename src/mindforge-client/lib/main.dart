import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import 'app/app_config.dart';
import 'app/mindforge_app.dart';
import 'auth/gotrue_auth_gateway.dart';
import 'auth/gotrue_sdk_port.dart';
import 'brain/brain_http_repository.dart';
import 'brain/brain_workspace_controller.dart';
import 'documents/documents_controller.dart';
import 'documents/document_http_repository.dart';
import 'platform/client_platform_stub.dart'
    if (dart.library.js_interop) 'platform/client_platform_web.dart'
    as client_platform;

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  late final AppConfig config;
  try {
    config = AppConfig.fromEnvironment();
  } on FormatException {
    runApp(const ConfigurationErrorApp());
    return;
  }

  final secretStore = client_platform.createAuthSecretStore(config);
  final auth = GoTrueAuthGateway(
    authApi: GoTrueSdkPort(
      apiUrl: config.apiUrl,
      callbackUri: config.callbackUri,
      secretStore: secretStore,
    ),
    secretStore: secretStore,
    callbackUri: config.callbackUri,
    authorizationOrigin: config.authApiUrl,
    launchExternalBrowser: client_platform.launchGoogleSignIn,
  );
  final engineHttpClient = http.Client();
  final workspace = BrainWorkspaceController(
    auth: auth,
    repository: BrainHttpRepository(
      baseUri: config.apiUrl,
      auth: auth,
      client: engineHttpClient,
    ),
  );

  final documentStore = client_platform.createDocumentLocalStore(config);
  final documents = documentStore == null
      ? null
      : DocumentsController(
          auth: auth,
          repository: DocumentHttpRepository(
            baseUri: config.apiUrl,
            auth: auth,
            client: engineHttpClient,
          ),
          localStore: documentStore,
          subtreeLayerIds: (layerId) => _subtreeLayerIds(workspace, layerId),
        );

  runApp(
    MindforgeApp(
      auth: auth,
      workspace: workspace,
      documents: documents,
      incomingLinks: client_platform.incomingAuthCallbacks(config.callbackUri),
      onDispose: engineHttpClient.close,
    ),
  );
}

Set<String> _subtreeLayerIds(
  BrainWorkspaceController workspace,
  String layerId,
) {
  final root = workspace.state.hierarchy?.layerById(layerId);
  if (root == null) return <String>{layerId};
  final pending = [root];
  final ids = <String>{};
  while (pending.isNotEmpty) {
    final layer = pending.removeLast();
    if (!ids.add(layer.id)) continue;
    pending.addAll(layer.children);
  }
  return ids;
}
