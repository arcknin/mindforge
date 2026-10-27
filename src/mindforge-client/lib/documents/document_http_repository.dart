import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../auth/auth_gateway.dart';
import '../brain/brain_json.dart';
import 'document_models.dart';
import 'document_repository.dart';

class DocumentHttpRepository implements DocumentRepository {
  DocumentHttpRepository({
    required Uri baseUri,
    required AuthGateway auth,
    http.Client? client,
    this.requestTimeout = const Duration(seconds: 30),
  }) : _baseUri = _validateBaseUri(baseUri),
       _auth = auth,
       _client = client ?? http.Client();

  final Uri _baseUri;
  final AuthGateway _auth;
  final http.Client _client;
  final Duration requestTimeout;

  @override
  Future<DocumentPage> list(
    String layerId, {
    bool subtree = false,
    bool trash = false,
    String? after,
  }) async {
    final query = <String, String>{
      'scope': subtree ? 'subtree' : 'direct',
      'state': trash ? 'trash' : 'active',
    };
    if (after != null) query['after'] = after;
    final path = Uri(
      pathSegments: <String>[
        'v1',
        'me',
        'brain',
        'layers',
        layerId,
        'documents',
      ],
      queryParameters: query,
    ).toString();
    final response = await _send(
      method: 'GET',
      path: path,
      expectedStatus: 200,
    );
    return DocumentPage.fromJson(_decodeJson(response));
  }

  @override
  Future<RemoteDocument> get(String id) async {
    final response = await _send(
      method: 'GET',
      path: _documentPath(id),
      expectedStatus: 200,
    );
    return RemoteDocument.fromEnvelope(_decodeJson(response));
  }

  @override
  Future<RemoteDocument> create({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
  }) async {
    final response = await _send(
      method: 'POST',
      path: '/v1/me/brain/documents',
      expectedStatus: 201,
      encodedJsonBody: _encodeCreate(
        id: id,
        layerId: layerId,
        title: title,
        markdown: markdown,
      ),
    );
    return RemoteDocument.fromEnvelope(_decodeJson(response));
  }

  @override
  Future<RemoteDocument> update({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
    required BigInt expectedVersion,
  }) async {
    final response = await _send(
      method: 'PATCH',
      path: _documentPath(id),
      expectedStatus: 200,
      encodedJsonBody: _encodeUpdate(
        layerId: layerId,
        title: title,
        markdown: markdown,
        expectedVersion: expectedVersion,
      ),
    );
    return RemoteDocument.fromEnvelope(_decodeJson(response));
  }

  @override
  Future<RemoteDocument> trash({
    required String id,
    required BigInt expectedVersion,
  }) async {
    final response = await _send(
      method: 'POST',
      path: '${_documentPath(id)}/trash',
      expectedStatus: 200,
      encodedJsonBody: _encodeVersion(expectedVersion),
    );
    return RemoteDocument.fromEnvelope(_decodeJson(response));
  }

  @override
  Future<RemoteDocument> restore({
    required String id,
    required BigInt expectedVersion,
    String? layerId,
  }) async {
    final response = await _send(
      method: 'POST',
      path: '${_documentPath(id)}/restore',
      expectedStatus: 200,
      encodedJsonBody: _encodeRestore(
        expectedVersion: expectedVersion,
        layerId: layerId,
      ),
    );
    return RemoteDocument.fromEnvelope(_decodeJson(response));
  }

  Future<http.Response> _send({
    required String method,
    required String path,
    required int expectedStatus,
    String? encodedJsonBody,
  }) async {
    final requestSession = _auth.state;
    final initialCredential = await _auth.credential();
    final currentSession = _auth.state;
    if (requestSession.phase != AuthPhase.authenticated ||
        currentSession.phase != AuthPhase.authenticated ||
        currentSession.userId != initialCredential.userId ||
        currentSession.generation != initialCredential.generation) {
      throw const StaleAuthOperationException();
    }
    var credential = initialCredential;
    var response = await _sendOnce(
      method: method,
      path: path,
      credential: credential,
      encodedJsonBody: encodedJsonBody,
    );
    if (response.statusCode == 401) {
      credential = await _auth.refreshAfterUnauthorized(initialCredential);
      if (credential.generation != initialCredential.generation ||
          credential.userId != initialCredential.userId) {
        throw const StaleAuthOperationException();
      }
      response = await _sendOnce(
        method: method,
        path: path,
        credential: credential,
        encodedJsonBody: encodedJsonBody,
      );
      if (response.statusCode == 401) {
        await _auth.expireAfterUnauthorized(credential);
      }
    }

    if (response.statusCode != expectedStatus) {
      throw _apiException(response);
    }
    return response;
  }

  Future<http.Response> _sendOnce({
    required String method,
    required String path,
    required AuthCredential credential,
    String? encodedJsonBody,
  }) async {
    final headers = <String, String>{
      'accept': 'application/json',
      'authorization': 'Bearer ${credential.accessToken}',
    };
    if (encodedJsonBody != null) headers['content-type'] = 'application/json';
    final route = Uri.parse(path);
    final requestUri = _baseUri.replace(
      pathSegments: <String>[
        ..._baseUri.pathSegments.where((segment) => segment.isNotEmpty),
        ...route.pathSegments,
      ],
      queryParameters: route.queryParameters.isEmpty
          ? null
          : route.queryParameters,
    );
    final request = http.Request(method, requestUri)..headers.addAll(headers);
    if (encodedJsonBody != null) request.body = encodedJsonBody;
    try {
      final streamed = await _client.send(request).timeout(requestTimeout);
      return await http.Response.fromStream(streamed).timeout(requestTimeout);
    } on TimeoutException {
      throw const DocumentTransportException();
    } on http.ClientException {
      throw const DocumentTransportException();
    }
  }

  Object? _decodeJson(http.Response response) {
    try {
      return decodeJsonPreservingLargeIntegers(response.body);
    } on FormatException {
      throw const DocumentResponseException();
    }
  }

  DocumentApiException _apiException(http.Response response) {
    var code = 'request_failed';
    var message = _defaultErrorMessage(response.statusCode);
    String? requestId;
    try {
      final decoded = decodeJsonPreservingLargeIntegers(response.body);
      if (decoded is Map<String, dynamic>) {
        final error = decoded['error'];
        if (error is Map<String, dynamic>) {
          if (error['code'] is String && (error['code'] as String).isNotEmpty) {
            code = error['code'] as String;
          }
          if (error['message'] is String &&
              (error['message'] as String).trim().isNotEmpty) {
            message = error['message'] as String;
          }
        }
        if (decoded['request_id'] is String) {
          requestId = decoded['request_id'] as String;
        }
      }
    } on FormatException {
      // Never surface the raw response body.
    }
    return DocumentApiException(
      statusCode: response.statusCode,
      code: code,
      message: message,
      requestId: requestId,
    );
  }

  String _defaultErrorMessage(int statusCode) => switch (statusCode) {
    401 => 'A sessão precisa ser renovada.',
    404 => 'O documento não foi encontrado.',
    409 => 'O documento mudou. Confira a versão atual.',
    422 => 'Os dados do documento não são válidos.',
    >= 500 => 'O Brain Engine está indisponível.',
    _ => 'A solicitação não foi aceita.',
  };

  String _documentPath(String id) =>
      Uri(pathSegments: <String>['v1', 'me', 'brain', 'documents', id]).path;

  static String _encodeCreate({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
  }) => jsonEncode(<String, Object?>{
    'id': id,
    'layer_id': layerId,
    'title': title,
    'markdown': markdown,
  });

  static String _encodeUpdate({
    required String layerId,
    required String title,
    required String markdown,
    required BigInt expectedVersion,
  }) {
    _validateVersion(expectedVersion);
    return '{"expected_version":$expectedVersion,"layer_id":${jsonEncode(layerId)},"title":${jsonEncode(title)},"markdown":${jsonEncode(markdown)}}';
  }

  static String _encodeVersion(BigInt expectedVersion) {
    _validateVersion(expectedVersion);
    return '{"expected_version":$expectedVersion}';
  }

  static String _encodeRestore({
    required BigInt expectedVersion,
    required String? layerId,
  }) {
    _validateVersion(expectedVersion);
    final layer = layerId == null ? '' : ',"layer_id":${jsonEncode(layerId)}';
    return '{"expected_version":$expectedVersion$layer}';
  }

  static void _validateVersion(BigInt version) {
    if (version < minimumLayerVersion || version > maximumLayerVersion) {
      throw ArgumentError.value(version, 'expectedVersion');
    }
  }

  static Uri _validateBaseUri(Uri uri) {
    if ((uri.scheme != 'http' && uri.scheme != 'https') ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        (uri.path.isNotEmpty &&
            uri.path != '/' &&
            uri.path != '/api' &&
            uri.path != '/api/')) {
      throw ArgumentError.value(uri, 'baseUri');
    }
    return uri.replace(path: uri.path.startsWith('/api') ? '/api' : '');
  }
}
