import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../auth/auth_gateway.dart';
import 'brain_json.dart';
import 'brain_hierarchy.dart';
import 'brain_repository.dart';

class BrainHttpRepository implements BrainRepository {
  BrainHttpRepository({
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
  Future<BrainSummary> bootstrap() async {
    final response = await _send(
      method: 'PUT',
      path: '/v1/me/brain',
      expectedStatus: 200,
    );
    return BrainSummary.fromEnvelope(_decodeJson(response));
  }

  @override
  Future<BrainHierarchy> observe() async {
    final response = await _send(
      method: 'GET',
      path: '/v1/me/brain',
      expectedStatus: 200,
    );
    return BrainHierarchy.fromJson(_decodeJson(response));
  }

  @override
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  }) async {
    final response = await _send(
      method: 'POST',
      path: '/v1/me/brain/layers',
      expectedStatus: 201,
      jsonBody: <String, Object?>{
        'parent': <String, Object?>{'kind': parent.kindName, 'id': parent.id},
        'name': name,
      },
    );
    return BrainLayer.fromEnvelope(_decodeJson(response));
  }

  @override
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  }) async {
    final path = Uri(
      pathSegments: <String>['v1', 'me', 'brain', 'layers', layerId],
    ).path;
    final response = await _send(
      method: 'PATCH',
      path: path,
      expectedStatus: 200,
      encodedJsonBody: encodeRenameRequest(
        name: name,
        expectedVersion: expectedVersion,
      ),
    );
    return BrainLayer.fromEnvelope(_decodeJson(response));
  }

  Future<http.Response> _send({
    required String method,
    required String path,
    required int expectedStatus,
    Map<String, Object?>? jsonBody,
    String? encodedJsonBody,
  }) async {
    final requestSession = _auth.state;
    final initialCredential = await _auth.credential();
    if (requestSession.phase != AuthPhase.authenticated ||
        requestSession.userId != initialCredential.userId ||
        requestSession.generation != initialCredential.generation) {
      throw const StaleAuthOperationException();
    }
    var credential = initialCredential;
    var response = await _sendOnce(
      method: method,
      path: path,
      credential: credential,
      jsonBody: jsonBody,
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
        jsonBody: jsonBody,
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
    Map<String, Object?>? jsonBody,
    String? encodedJsonBody,
  }) async {
    final headers = <String, String>{
      'accept': 'application/json',
      'authorization': 'Bearer ${credential.accessToken}',
    };
    if (jsonBody != null && encodedJsonBody != null) {
      throw ArgumentError('Provide only one JSON request body.');
    }
    final body =
        encodedJsonBody ?? (jsonBody == null ? null : jsonEncode(jsonBody));
    if (body != null) headers['content-type'] = 'application/json';
    final routeSegments = Uri.parse(
      path,
    ).pathSegments.where((segment) => segment.isNotEmpty);
    final requestUri = _baseUri.replace(
      pathSegments: <String>[
        ..._baseUri.pathSegments.where((segment) => segment.isNotEmpty),
        ...routeSegments,
      ],
    );
    final request = http.Request(method, requestUri)..headers.addAll(headers);
    if (body != null) request.body = body;
    try {
      final streamed = await _client.send(request).timeout(requestTimeout);
      return await http.Response.fromStream(streamed).timeout(requestTimeout);
    } on TimeoutException {
      throw const BrainTransportException();
    } on http.ClientException {
      throw const BrainTransportException();
    }
  }

  Object? _decodeJson(http.Response response) {
    try {
      return decodeJsonPreservingLargeIntegers(response.body);
    } on FormatException {
      throw const BrainResponseException();
    }
  }

  BrainApiException _apiException(http.Response response) {
    String code = 'request_failed';
    String message = _defaultErrorMessage(response.statusCode);
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
    return BrainApiException(
      statusCode: response.statusCode,
      code: code,
      message: message,
      requestId: requestId,
    );
  }

  String _defaultErrorMessage(int statusCode) => switch (statusCode) {
    401 => 'A sessão precisa ser renovada.',
    404 => 'O recurso não foi encontrado.',
    409 => 'O recurso mudou. Atualize e tente novamente.',
    422 => 'O nome informado não é válido.',
    >= 500 => 'O Brain Engine está indisponível.',
    _ => 'A solicitação não foi aceita.',
  };

  static Uri _validateBaseUri(Uri uri) {
    if (uri.scheme != 'http' && uri.scheme != 'https' ||
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
