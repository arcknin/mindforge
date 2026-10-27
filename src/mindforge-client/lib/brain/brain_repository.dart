import 'brain_hierarchy.dart';

abstract interface class BrainRepository {
  Future<BrainSummary> bootstrap();
  Future<BrainHierarchy> observe();
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  });
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  });
}

class BrainApiException implements Exception {
  const BrainApiException({
    required this.statusCode,
    required this.code,
    required this.message,
    this.requestId,
  });

  final int statusCode;
  final String code;
  final String message;
  final String? requestId;

  @override
  String toString() => 'Brain API returned $statusCode ($code).';
}

class BrainTransportException implements Exception {
  const BrainTransportException();

  @override
  String toString() => 'The Brain Engine could not be reached.';
}

class BrainResponseException implements Exception {
  const BrainResponseException();

  @override
  String toString() => 'The Brain Engine returned an invalid response.';
}
