import '../brain/brain_repository.dart';
import 'document_models.dart';

abstract interface class DocumentRepository {
  Future<DocumentPage> list(
    String layerId, {
    bool subtree = false,
    bool trash = false,
    String? after,
  });

  Future<RemoteDocument> get(String id);

  Future<RemoteDocument> create({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
  });

  Future<RemoteDocument> update({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
    required BigInt expectedVersion,
  });

  Future<RemoteDocument> trash({
    required String id,
    required BigInt expectedVersion,
  });

  Future<RemoteDocument> restore({
    required String id,
    required BigInt expectedVersion,
    String? layerId,
  });
}

/// API errors retain the same public fields and semantics as Brain Core errors,
/// while giving document tests and callers a domain-specific type to catch.
class DocumentApiException extends BrainApiException {
  const DocumentApiException({
    required super.statusCode,
    required super.code,
    required super.message,
    super.requestId,
  });
}

class DocumentTransportException extends BrainTransportException {
  const DocumentTransportException() : super();
}

class DocumentResponseException extends BrainResponseException {
  const DocumentResponseException() : super();
}

class DocumentValidationException implements Exception {
  const DocumentValidationException(this.message);

  final String message;

  @override
  String toString() => 'Invalid document: $message';
}

class DocumentLocalStoreException implements Exception {
  const DocumentLocalStoreException(this.message);

  final String message;

  @override
  String toString() => 'The local document store failed: $message';
}

abstract interface class DocumentLocalStore {
  Future<List<LocalDocument>> read(String userId);

  Future<void> write(String userId, List<LocalDocument> documents);
}
