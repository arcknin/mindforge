import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'document_models.dart';
import 'document_repository.dart';

/// Durable native document storage.
///
/// The directory provider is intentionally lazy so composition can construct
/// the adapter before platform path discovery has completed. Every user owns a
/// separate snapshot, and writes are serialized per user.
class FileDocumentLocalStore implements DocumentLocalStore {
  FileDocumentLocalStore({
    required String namespace,
    required Future<Directory> Function() directory,
  }) : _namespace = _safeComponent(namespace, 'namespace'),
       _directory = directory;

  final String _namespace;
  final Future<Directory> Function() _directory;
  final Map<String, Future<void>> _queues = <String, Future<void>>{};

  @override
  Future<List<LocalDocument>> read(String userId) => _queued(userId, () async {
    try {
      final file = await _fileFor(userId);
      if (!await file.exists()) return <LocalDocument>[];
      final decoded = jsonDecode(await file.readAsString());
      if (decoded is! Map || decoded['version'] != 1) {
        throw const FormatException('Unsupported local snapshot version.');
      }
      final rawDocuments = decoded['documents'];
      if (rawDocuments is! List) {
        throw const FormatException('Local snapshot documents must be a list.');
      }
      return <LocalDocument>[
        for (var index = 0; index < rawDocuments.length; index++)
          LocalDocument.fromJson(rawDocuments[index]),
      ];
    } on DocumentLocalStoreException {
      rethrow;
    } on Object catch (error) {
      throw DocumentLocalStoreException('snapshot is unreadable: $error');
    }
  });

  @override
  Future<void> write(String userId, List<LocalDocument> documents) => _queued(
    userId,
    () async {
      File? temporary;
      try {
        final directory = await _directory();
        await directory.create(recursive: true);
        final file = _fileIn(directory, userId);
        temporary = File(
          '${file.path}.${DateTime.now().microsecondsSinceEpoch}.tmp',
        );
        final snapshot = <String, Object?>{
          'version': 1,
          'documents': documents.map((document) => document.toJson()).toList(),
        };
        await temporary.writeAsString(jsonEncode(snapshot), flush: true);
        await temporary.rename(file.path);
      } on Object catch (error) {
        try {
          final failedTemporary = temporary;
          if (failedTemporary != null && await failedTemporary.exists()) {
            await failedTemporary.delete();
          }
        } on Object {
          // Best-effort cleanup must never touch the canonical snapshot.
        }
        throw DocumentLocalStoreException('snapshot commit failed: $error');
      }
    },
  );

  Future<T> _queued<T>(String userId, Future<T> Function() action) {
    final key = _safeComponent(userId, 'userId');
    final previous = _queues[key] ?? Future<void>.value();
    final next = previous.then<T>((_) => action());
    _queues[key] = next.then<void>((_) {}, onError: (Object _) {});
    next.then<void>(
      (_) {
        if (identical(_queues[key], next)) _queues.remove(key);
      },
      onError: (Object _) {
        if (identical(_queues[key], next)) _queues.remove(key);
      },
    );
    return next;
  }

  Future<File> _fileFor(String userId) async {
    final directory = await _directory();
    return _fileIn(directory, userId);
  }

  File _fileIn(Directory directory, String userId) => File(
    '${directory.path}${Platform.pathSeparator}$_namespace.${_safeComponent(userId, 'userId')}.documents.json',
  );

  static String _safeComponent(String value, String label) {
    if (value.isEmpty || value.contains('\u0000')) {
      throw ArgumentError.value(value, label);
    }
    return base64Url.encode(utf8.encode(value)).replaceAll('=', '');
  }
}
