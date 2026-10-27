import 'dart:collection';

import '../brain/brain_json.dart';

enum DocumentLinkStatus { available, trash, missing }

class DocumentLink {
  const DocumentLink({
    required this.targetId,
    required this.title,
    required this.layerId,
    required this.status,
  });

  final String targetId;
  final String title;
  final String layerId;
  final DocumentLinkStatus status;

  factory DocumentLink.fromJson(Object? value, String path) {
    final object = _object(
      value,
      path,
      allowed: const <String>{'target_id', 'title', 'layer_id', 'status'},
      required: const <String>{'target_id', 'title', 'layer_id', 'status'},
    );
    final status = switch (_string(object['status'], '$path.status')) {
      'available' => DocumentLinkStatus.available,
      'trash' => DocumentLinkStatus.trash,
      'missing' => DocumentLinkStatus.missing,
      _ => throw FormatException('$path.status is invalid.'),
    };
    return DocumentLink(
      targetId: _nonEmptyString(object['target_id'], '$path.target_id'),
      title: _string(object['title'], '$path.title'),
      layerId: _string(object['layer_id'], '$path.layer_id'),
      status: status,
    );
  }

  Map<String, Object?> toJson() => <String, Object?>{
    'target_id': targetId,
    'title': title,
    'layer_id': layerId,
    'status': status.name,
  };
}

class DocumentMetadata {
  const DocumentMetadata({
    required this.id,
    required this.layerId,
    required this.title,
    required this.version,
    required this.createdAt,
    required this.updatedAt,
    this.deletedAt,
    this.expiresAt,
  });

  final String id;
  final String layerId;
  final String title;
  final BigInt version;
  final DateTime createdAt;
  final DateTime updatedAt;
  final DateTime? deletedAt;
  final DateTime? expiresAt;

  bool get isTrash => deletedAt != null;

  factory DocumentMetadata.fromJson(Object? value, String path) {
    final object = _documentObject(value, path);
    return DocumentMetadata(
      id: _nonEmptyString(object['id'], '$path.id'),
      layerId: _nonEmptyString(object['layer_id'], '$path.layer_id'),
      title: _string(object['title'], '$path.title'),
      version: _documentVersion(object['version'], '$path.version'),
      createdAt: _date(object['created_at'], '$path.created_at'),
      updatedAt: _date(object['updated_at'], '$path.updated_at'),
      deletedAt: _nullableDate(object['deleted_at'], '$path.deleted_at'),
      expiresAt: _nullableDate(object['expires_at'], '$path.expires_at'),
    );
  }

  LocalDocument toLocal() => LocalDocument(
    id: id,
    layerId: layerId,
    title: title,
    markdown: '',
    remoteVersion: version,
    localRevision: 0,
    dirty: false,
    deletedAt: deletedAt,
    expiresAt: expiresAt,
    contentLoaded: false,
    remoteDeleted: isTrash,
    links: const <DocumentLink>[],
  );
}

class RemoteDocument extends DocumentMetadata {
  RemoteDocument({
    required super.id,
    required super.layerId,
    required super.title,
    required super.version,
    required super.createdAt,
    required super.updatedAt,
    super.deletedAt,
    super.expiresAt,
    required this.markdown,
    required List<DocumentLink> links,
  }) : links = List<DocumentLink>.unmodifiable(links);

  final String markdown;
  final List<DocumentLink> links;

  factory RemoteDocument.fromEnvelope(Object? value) {
    final envelope = _object(
      value,
      'response',
      allowed: const <String>{'document'},
      required: const <String>{'document'},
    );
    return RemoteDocument.fromJson(envelope['document'], 'response.document');
  }

  factory RemoteDocument.fromJson(Object? value, String path) {
    final object = _documentObject(
      value,
      path,
      additional: const <String>{'markdown', 'links'},
    );
    final rawLinks = object['links'];
    if (rawLinks is! List<Object?>) {
      throw FormatException('$path.links must be an array.');
    }
    return RemoteDocument(
      id: _nonEmptyString(object['id'], '$path.id'),
      layerId: _nonEmptyString(object['layer_id'], '$path.layer_id'),
      title: _string(object['title'], '$path.title'),
      version: _documentVersion(object['version'], '$path.version'),
      createdAt: _date(object['created_at'], '$path.created_at'),
      updatedAt: _date(object['updated_at'], '$path.updated_at'),
      deletedAt: _nullableDate(object['deleted_at'], '$path.deleted_at'),
      expiresAt: _nullableDate(object['expires_at'], '$path.expires_at'),
      markdown: _string(object['markdown'], '$path.markdown'),
      links: <DocumentLink>[
        for (var index = 0; index < rawLinks.length; index++)
          DocumentLink.fromJson(rawLinks[index], '$path.links[$index]'),
      ],
    );
  }

  LocalDocument toLocal({
    int localRevision = 0,
    bool dirty = false,
    bool conflict = false,
    bool uncertain = false,
  }) => LocalDocument(
    id: id,
    layerId: layerId,
    title: title,
    markdown: markdown,
    remoteVersion: version,
    localRevision: localRevision,
    dirty: dirty,
    deletedAt: deletedAt,
    expiresAt: expiresAt,
    conflict: conflict,
    uncertain: uncertain,
    contentLoaded: true,
    remoteDeleted: isTrash,
    links: links,
  );
}

class DocumentPage {
  DocumentPage({required List<DocumentMetadata> documents, this.nextCursor})
    : documents = List<DocumentMetadata>.unmodifiable(documents);

  final List<DocumentMetadata> documents;
  final String? nextCursor;

  factory DocumentPage.fromJson(Object? value) {
    final object = _object(
      value,
      'response',
      allowed: const <String>{'documents', 'next_cursor'},
      required: const <String>{'documents', 'next_cursor'},
    );
    final rawDocuments = object['documents'];
    if (rawDocuments is! List<Object?>) {
      throw const FormatException('response.documents must be an array.');
    }
    final cursor = object['next_cursor'];
    if (cursor != null && cursor is! String) {
      throw const FormatException(
        'response.next_cursor must be a string or null.',
      );
    }
    return DocumentPage(
      documents: <DocumentMetadata>[
        for (var index = 0; index < rawDocuments.length; index++)
          DocumentMetadata.fromJson(
            rawDocuments[index],
            'response.documents[$index]',
          ),
      ],
      nextCursor: cursor as String?,
    );
  }
}

class LocalDocument {
  const LocalDocument({
    required this.id,
    required this.layerId,
    required this.title,
    required this.markdown,
    required this.remoteVersion,
    required this.localRevision,
    required this.dirty,
    this.deletedAt,
    this.expiresAt,
    this.conflict = false,
    this.uncertain = false,
    required this.contentLoaded,
    this.remoteDeleted = false,
    this.links = const <DocumentLink>[],
    this.pendingMutation,
  });

  final String id;
  final String layerId;
  final String title;
  final String markdown;
  final BigInt? remoteVersion;
  final int localRevision;
  final bool dirty;
  final DateTime? deletedAt;
  final DateTime? expiresAt;
  final bool conflict;
  final bool uncertain;
  final bool contentLoaded;

  /// Whether the last confirmed server snapshot is in the trash.
  ///
  /// This is kept separately from [deletedAt] so a local restore can retain
  /// the intent to call the dedicated restore endpoint.
  final bool remoteDeleted;
  final List<DocumentLink> links;
  final PendingDocumentMutation? pendingMutation;

  bool get isTrash => deletedAt != null;

  LocalDocument copyWith({
    String? id,
    String? layerId,
    String? title,
    String? markdown,
    Object? remoteVersion = _keep,
    int? localRevision,
    bool? dirty,
    Object? deletedAt = _keep,
    Object? expiresAt = _keep,
    bool? conflict,
    bool? uncertain,
    bool? contentLoaded,
    bool? remoteDeleted,
    List<DocumentLink>? links,
    Object? pendingMutation = _keep,
  }) => LocalDocument(
    id: id ?? this.id,
    layerId: layerId ?? this.layerId,
    title: title ?? this.title,
    markdown: markdown ?? this.markdown,
    remoteVersion: identical(remoteVersion, _keep)
        ? this.remoteVersion
        : remoteVersion as BigInt?,
    localRevision: localRevision ?? this.localRevision,
    dirty: dirty ?? this.dirty,
    deletedAt: identical(deletedAt, _keep)
        ? this.deletedAt
        : deletedAt as DateTime?,
    expiresAt: identical(expiresAt, _keep)
        ? this.expiresAt
        : expiresAt as DateTime?,
    conflict: conflict ?? this.conflict,
    uncertain: uncertain ?? this.uncertain,
    contentLoaded: contentLoaded ?? this.contentLoaded,
    remoteDeleted: remoteDeleted ?? this.remoteDeleted,
    links: links ?? this.links,
    pendingMutation: identical(pendingMutation, _keep)
        ? this.pendingMutation
        : pendingMutation as PendingDocumentMutation?,
  );

  Map<String, Object?> toJson() => <String, Object?>{
    'id': id,
    'layer_id': layerId,
    'title': title,
    'markdown': markdown,
    'remote_version': remoteVersion?.toString(),
    'local_revision': localRevision,
    'dirty': dirty,
    'deleted_at': deletedAt?.toUtc().toIso8601String(),
    'expires_at': expiresAt?.toUtc().toIso8601String(),
    'conflict': conflict,
    'uncertain': uncertain,
    'content_loaded': contentLoaded,
    'remote_deleted': remoteDeleted,
    'links': links.map((link) => link.toJson()).toList(growable: false),
    'pending_mutation': pendingMutation?.toJson(),
  };

  factory LocalDocument.fromJson(Object? value) {
    final object = _object(
      value,
      'document',
      allowed: const <String>{
        'id',
        'layer_id',
        'title',
        'markdown',
        'remote_version',
        'local_revision',
        'dirty',
        'deleted_at',
        'expires_at',
        'conflict',
        'uncertain',
        'content_loaded',
        'remote_deleted',
        'links',
        'pending_mutation',
      },
      required: const <String>{
        'id',
        'layer_id',
        'title',
        'markdown',
        'remote_version',
        'local_revision',
        'dirty',
        'deleted_at',
        'expires_at',
        'conflict',
        'uncertain',
        'content_loaded',
      },
    );
    final localRevision = object['local_revision'];
    if (localRevision is! int || localRevision < 0) {
      throw const FormatException(
        'document.local_revision must be non-negative.',
      );
    }
    return LocalDocument(
      id: _nonEmptyString(object['id'], 'document.id'),
      layerId: _nonEmptyString(object['layer_id'], 'document.layer_id'),
      title: _string(object['title'], 'document.title'),
      markdown: _string(object['markdown'], 'document.markdown'),
      remoteVersion: _nullableVersion(
        object['remote_version'],
        'document.remote_version',
      ),
      localRevision: localRevision,
      dirty: _bool(object['dirty'], 'document.dirty'),
      deletedAt: _nullableDate(object['deleted_at'], 'document.deleted_at'),
      expiresAt: _nullableDate(object['expires_at'], 'document.expires_at'),
      conflict: _bool(object['conflict'], 'document.conflict'),
      uncertain: _bool(object['uncertain'], 'document.uncertain'),
      contentLoaded: _bool(object['content_loaded'], 'document.content_loaded'),
      remoteDeleted: object['remote_deleted'] == null
          ? false
          : _bool(object['remote_deleted'], 'document.remote_deleted'),
      links: _localLinks(object['links']),
      pendingMutation: object['pending_mutation'] == null
          ? null
          : PendingDocumentMutation.fromJson(object['pending_mutation']),
    );
  }

  bool hasSameSnapshotAs({
    required String layerId,
    required String title,
    required String markdown,
    required bool trash,
  }) =>
      this.layerId == layerId &&
      this.title == title &&
      this.markdown == markdown &&
      isTrash == trash;
}

class PendingDocumentMutation {
  const PendingDocumentMutation({
    required this.operation,
    required this.layerId,
    required this.title,
    required this.markdown,
    required this.localRevision,
    this.expectedVersion,
    this.trash = false,
  });

  final String operation;
  final String layerId;
  final String title;
  final String markdown;
  final BigInt? expectedVersion;
  final int localRevision;
  final bool trash;

  Map<String, Object?> toJson() => <String, Object?>{
    'operation': operation,
    'layer_id': layerId,
    'title': title,
    'markdown': markdown,
    'expected_version': expectedVersion?.toString(),
    'local_revision': localRevision,
    'trash': trash,
  };

  factory PendingDocumentMutation.fromJson(Object? value) {
    final object = _object(
      value,
      'document.pending_mutation',
      allowed: const <String>{
        'operation',
        'layer_id',
        'title',
        'markdown',
        'expected_version',
        'local_revision',
        'trash',
      },
      required: const <String>{
        'operation',
        'layer_id',
        'title',
        'markdown',
        'expected_version',
        'local_revision',
        'trash',
      },
    );
    final revision = object['local_revision'];
    if (revision is! int || revision < 0) {
      throw const FormatException(
        'document.pending_mutation.local_revision must be non-negative.',
      );
    }
    return PendingDocumentMutation(
      operation: _nonEmptyString(
        object['operation'],
        'document.pending_mutation.operation',
      ),
      layerId: _nonEmptyString(
        object['layer_id'],
        'document.pending_mutation.layer_id',
      ),
      title: _string(object['title'], 'document.pending_mutation.title'),
      markdown: _string(
        object['markdown'],
        'document.pending_mutation.markdown',
      ),
      expectedVersion: _nullableVersion(
        object['expected_version'],
        'document.pending_mutation.expected_version',
      ),
      localRevision: revision,
      trash: _bool(object['trash'], 'document.pending_mutation.trash'),
    );
  }
}

const Object _keep = Object();

Map<String, Object?> _documentObject(
  Object? value,
  String path, {
  Set<String> additional = const <String>{},
}) => _object(
  value,
  path,
  allowed: <String>{
    'id',
    'layer_id',
    'title',
    'version',
    'created_at',
    'updated_at',
    'deleted_at',
    'expires_at',
    ...additional,
  },
  required: <String>{
    'id',
    'layer_id',
    'title',
    'version',
    'created_at',
    'updated_at',
    'deleted_at',
    'expires_at',
    ...additional,
  },
);

Map<String, Object?> _object(
  Object? value,
  String path, {
  required Set<String> allowed,
  required Set<String> required,
}) {
  if (value is! Map) throw FormatException('$path must be an object.');
  final object = <String, Object?>{};
  for (final entry in value.entries) {
    if (entry.key is! String || !allowed.contains(entry.key)) {
      throw FormatException('$path contains an unknown field.');
    }
    object[entry.key as String] = entry.value;
  }
  for (final key in required) {
    if (!object.containsKey(key))
      throw FormatException('$path.$key is required.');
  }
  return UnmodifiableMapView<String, Object?>(object);
}

String _string(Object? value, String path) {
  if (value is! String) throw FormatException('$path must be a string.');
  return value;
}

String _nonEmptyString(Object? value, String path) {
  final text = _string(value, path);
  if (text.isEmpty) throw FormatException('$path must not be empty.');
  return text;
}

bool _bool(Object? value, String path) {
  if (value is! bool) throw FormatException('$path must be a boolean.');
  return value;
}

List<DocumentLink> _localLinks(Object? value) {
  if (value == null) return const <DocumentLink>[];
  if (value is! List<Object?>) {
    throw const FormatException('document.links must be an array.');
  }
  return <DocumentLink>[
    for (var index = 0; index < value.length; index++)
      DocumentLink.fromJson(value[index], 'document.links[$index]'),
  ];
}

DateTime _date(Object? value, String path) {
  final text = _string(value, path);
  try {
    return DateTime.parse(text).toUtc();
  } on FormatException {
    throw FormatException('$path must be an ISO date-time.');
  }
}

DateTime? _nullableDate(Object? value, String path) =>
    value == null ? null : _date(value, path);

BigInt _documentVersion(Object? value, String path) {
  try {
    final parsed = value is String ? BigInt.parse(value) : value;
    return parseLayerVersion(parsed, path);
  } on Object {
    throw FormatException('$path must be a positive Int64.');
  }
}

BigInt? _nullableVersion(Object? value, String path) {
  if (value == null) return null;
  return _documentVersion(value, path);
}
