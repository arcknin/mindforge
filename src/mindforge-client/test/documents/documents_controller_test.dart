import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/documents/document_models.dart';
import 'package:mindforge_client/documents/document_repository.dart';
import 'package:mindforge_client/documents/documents_controller.dart';

void main() {
  test('paginates metadata and preserves a dirty local draft', () async {
    final auth = _FakeAuthGateway('user-a');
    final local = _local(
      id: _id(1),
      layerId: _id(2),
      title: 'Rascunho local',
      markdown: 'novo',
      version: BigInt.one,
      dirty: true,
      contentLoaded: true,
    );
    final store = _FakeStore(<String, List<LocalDocument>>{
      'user-a': <LocalDocument>[local],
    });
    final repository = _FakeRepository()
      ..listPages = <DocumentPage>[
        DocumentPage(
          documents: <DocumentMetadata>[_metadata(_id(3), _id(2), 'A')],
          nextCursor: _id(3),
        ),
        DocumentPage(
          documents: <DocumentMetadata>[_metadata(_id(4), _id(2), 'B')],
        ),
      ];
    final controller = DocumentsController(
      auth: auth,
      repository: repository,
      localStore: store,
      autosaveInterval: const Duration(hours: 1),
      syncInterval: const Duration(hours: 1),
    );

    await controller.initialize();
    await controller.loadLayer(_id(2));

    expect(repository.listAfter, <String?>[null, _id(3)]);
    expect(
      controller.state.items.map((item) => item.id),
      containsAll(<String>[_id(3), _id(4)]),
    );
    expect(
      controller.state.items.singleWhere((item) => item.id == _id(1)).markdown,
      'novo',
    );
    expect(
      controller.state.items.singleWhere((item) => item.id == _id(1)).dirty,
      isTrue,
    );
    controller.dispose();
  });

  test(
    'includes local child documents in a subtree before remote upload',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final child = _local(
        id: _id(5),
        layerId: _id(6),
        title: 'Filho local',
        markdown: 'draft',
        version: BigInt.one,
        dirty: true,
        contentLoaded: true,
      );
      final controller = DocumentsController(
        auth: auth,
        repository: _FakeRepository()
          ..listPages = <DocumentPage>[
            DocumentPage(documents: <DocumentMetadata>[]),
          ],
        localStore: _FakeStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[child],
        }),
        subtreeLayerIds: (layerId) => <String>{layerId, _id(6)},
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      await controller.loadLayer(_id(2), subtree: true);

      expect(controller.state.items.single.id, _id(5));
      controller.dispose();
    },
  );

  test(
    'opens a server-only target by ID and caches its remote snapshot',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final source = _local(
        id: _id(1),
        layerId: _id(2),
        title: 'Source',
        markdown: '[[doc:${_id(9)}]]',
        version: BigInt.one,
        dirty: false,
        contentLoaded: true,
      );
      final repository = _FakeRepository()
        ..getResult = Future<RemoteDocument>.value(
          _remote(_id(9), _id(8), 'Target', 'server body', 1),
        );
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: _FakeStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[source],
        }),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      await controller.openDocument(_id(1));
      await controller.openDocument(_id(9));

      expect(controller.state.selected!.id, _id(9));
      expect(
        controller.knownDocuments.any((item) => item.id == _id(9)),
        isTrue,
      );
      expect(repository.getCalls, contains(_id(9)));
      controller.dispose();
    },
  );

  test(
    'a missing server-only target does not replace the current draft',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final source = _local(
        id: _id(1),
        layerId: _id(2),
        title: 'Source',
        markdown: 'local draft',
        version: BigInt.one,
        dirty: true,
        contentLoaded: true,
      );
      final repository = _FakeRepository()
        ..getError = const DocumentApiException(
          statusCode: 404,
          code: 'not_found',
          message: 'missing',
        );
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: _FakeStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[source],
        }),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      await controller.openDocument(_id(1));
      await controller.openDocument(_id(9));

      expect(controller.state.selected!.id, _id(1));
      expect(controller.state.selected!.markdown, 'local draft');
      expect(
        controller.knownDocuments.where((item) => item.id == _id(9)),
        isEmpty,
      );
      controller.dispose();
    },
  );

  test(
    'partial create keeps the failed ID pending and does not blindly retry it',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final repository = _FakeRepository()
        ..createError = const DocumentTransportException()
        ..getError = const DocumentTransportException();
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: _FakeStore(),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
        idGenerator: _idGenerator(<String>[_id(1), _id(2)]),
      );

      await controller.initialize();
      await controller.createDocument(_id(3), title: 'A');
      await controller.createDocument(_id(3), title: 'B');
      await controller.sync();

      expect(repository.createCalls, 2);
      expect(repository.getCalls, contains(_id(1)));
      expect(
        controller.knownDocuments
            .firstWhere((item) => item.id == _id(1))
            .pendingMutation,
        isNotNull,
      );
      await controller.sync();
      expect(repository.createCalls, 2);
      controller.dispose();
    },
  );

  test(
    'a higher-version empty create collision remains an explicit conflict',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final repository = _FakeRepository()
        ..createError = const DocumentApiException(
          statusCode: 409,
          code: 'conflict',
          message: 'id already exists',
        )
        ..getResult = Future<RemoteDocument>.value(
          _remote(_id(1), _id(2), 'Collision', '', 2),
        );
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: _FakeStore(),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
        idGenerator: () => _id(1),
      );

      await controller.initialize();
      await controller.createDocument(_id(2), title: 'Collision');
      await controller.sync();

      expect(controller.state.selected!.conflict, isTrue);
      expect(controller.state.selected!.uncertain, isTrue);
      expect(controller.state.selected!.dirty, isTrue);
      expect(repository.getCalls, contains(_id(1)));
      controller.dispose();
    },
  );

  test(
    'a persisted pending create with GET 404 gets one controlled retry',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final document = LocalDocument(
        id: _id(1),
        layerId: _id(2),
        title: 'Offline',
        markdown: 'full body',
        remoteVersion: null,
        localRevision: 1,
        dirty: true,
        uncertain: true,
        contentLoaded: true,
        pendingMutation: PendingDocumentMutation(
          operation: 'create',
          layerId: '00000002-1111-4111-8111-111111111111',
          title: 'Offline',
          markdown: '',
          localRevision: 1,
        ),
      );
      final repository = _FakeRepository()
        ..getError = const DocumentApiException(
          statusCode: 404,
          code: 'not_found',
          message: 'not created',
        );
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: _FakeStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[document],
        }),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      await controller.sync();

      expect(repository.getCalls, contains(_id(1)));
      expect(repository.createCalls, 1);
      expect(controller.state.selected, isNull);
      expect(controller.knownDocuments.single.pendingMutation, isNull);
      expect(controller.knownDocuments.single.dirty, isFalse);
      controller.dispose();
    },
  );

  test('a 409 create followed by GET 404 remains a visible conflict', () async {
    final auth = _FakeAuthGateway('user-a');
    final repository = _FakeRepository()
      ..createError = const DocumentApiException(
        statusCode: 409,
        code: 'conflict',
        message: 'collision',
      )
      ..getError = const DocumentApiException(
        statusCode: 404,
        code: 'not_found',
        message: 'not created',
      );
    final controller = DocumentsController(
      auth: auth,
      repository: repository,
      localStore: _FakeStore(),
      autosaveInterval: const Duration(hours: 1),
      syncInterval: const Duration(hours: 1),
      idGenerator: () => _id(1),
    );

    await controller.initialize();
    await controller.createDocument(_id(2), title: 'Collision');
    await controller.sync();
    await controller.sync();

    expect(repository.createCalls, 1);
    expect(controller.knownDocuments.single.conflict, isTrue);
    expect(controller.knownDocuments.single.uncertain, isTrue);
    expect(controller.knownDocuments.single.dirty, isTrue);
    controller.dispose();
  });

  test(
    'periodic autosave commits local state while automatic sync respects its interval',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final store = _FakeStore();
      final repository = _FakeRepository();
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: store,
        autosaveInterval: const Duration(milliseconds: 5),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      final document = await controller.createDocument(_id(2));
      expect(document.id, hasLength(36));
      await Future<void>.delayed(const Duration(milliseconds: 20));
      expect(store.writeCalls, greaterThanOrEqualTo(1));

      await controller.onResume();
      expect(repository.createCalls, 0);
      controller.dispose();
    },
  );

  test(
    'manual sync is single-flight and creates cyclic new documents in two phases',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final store = _FakeStore();
      final repository = _FakeRepository();
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: store,
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
        idGenerator: _idGenerator(<String>[_id(1), _id(2)]),
      );

      await controller.initialize();
      await controller.createDocument(_id(3), title: 'A');
      await controller.editDocument(markdown: '[[doc:${_id(2)}]]');
      await controller.createDocument(_id(3), title: 'B');
      await controller.editDocument(markdown: '[[doc:${_id(1)}]]');

      final first = controller.sync();
      final second = controller.sync();
      expect(identical(first, second), isTrue);
      await Future.wait(<Future<void>>[first, second]);

      expect(repository.createdMarkdown, <String>['', '']);
      expect(
        repository.updatedMarkdown,
        containsAll(<String>['[[doc:${_id(2)}]]', '[[doc:${_id(1)}]]']),
      );
      expect(controller.state.items.every((item) => !item.dirty), isTrue);
      controller.dispose();
    },
  );

  test(
    'reconciles an uncertain update with GET and keeps edits made during response',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final store = _FakeStore();
      final repository = _FakeRepository()
        ..updateError = const DocumentTransportException()
        ..getResult = Future<RemoteDocument>.value(
          _remote(_id(1), _id(2), 'A', 'body', 2),
        );
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: store,
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );
      await controller.initialize();
      await controller.loadLayer(_id(2));
      await controller.openDocument(_id(1));
      await controller.editDocument(markdown: 'body');

      await controller.sync();

      expect(repository.getCalls, contains(_id(1)));
      expect(controller.state.selected!.uncertain, isFalse);
      expect(controller.state.selected!.dirty, isFalse);

      final delayed = Completer<RemoteDocument>();
      repository.updateResult = delayed.future;
      await controller.editDocument(markdown: 'first');
      final sync = controller.sync();
      await Future<void>.delayed(Duration.zero);
      await controller.editDocument(markdown: 'second');
      delayed.complete(_remote(_id(1), _id(2), 'A', 'first', 3));
      await sync;

      expect(controller.state.selected!.markdown, 'second');
      expect(controller.state.selected!.dirty, isTrue);
      controller.dispose();
    },
  );

  test('conflict preserves draft until explicit local resolution', () async {
    final auth = _FakeAuthGateway('user-a');
    final repository = _FakeRepository()
      ..updateError = const DocumentApiException(
        statusCode: 409,
        code: 'conflict',
        message: 'stale',
      )
      ..getResult = Future<RemoteDocument>.value(
        _remote(_id(1), _id(2), 'Server', 'server', 9),
      );
    final controller = DocumentsController(
      auth: auth,
      repository: repository,
      localStore: _FakeStore(),
      autosaveInterval: const Duration(hours: 1),
      syncInterval: const Duration(hours: 1),
    );
    await controller.initialize();
    await controller.loadLayer(_id(2));
    await controller.openDocument(_id(1));
    await controller.editDocument(title: 'Local', markdown: 'local');
    await controller.sync();

    expect(controller.state.selected!.conflict, isTrue);
    expect(controller.state.selected!.markdown, 'local');
    await controller.resolveConflict(_id(1), useLocal: true);
    expect(controller.state.selected!.conflict, isFalse);
    expect(controller.state.selected!.dirty, isTrue);
    await controller.sync();
    expect(controller.state.selected!.dirty, isFalse);
    controller.dispose();
  });

  test('persists mutation intent before an uncertain request', () async {
    final auth = _FakeAuthGateway('user-a');
    final store = _FakeStore(<String, List<LocalDocument>>{
      'user-a': <LocalDocument>[
        _local(
          id: _id(1),
          layerId: _id(2),
          title: 'A',
          markdown: 'draft',
          version: BigInt.one,
          dirty: true,
          contentLoaded: true,
        ),
      ],
    });
    final repository = _FakeRepository()
      ..updateError = const DocumentTransportException()
      ..getError = const DocumentTransportException();
    final controller = DocumentsController(
      auth: auth,
      repository: repository,
      localStore: store,
      autosaveInterval: const Duration(hours: 1),
      syncInterval: const Duration(hours: 1),
    );

    await controller.initialize();
    await controller.loadLayer(_id(2));
    await controller.openDocument(_id(1));
    await controller.sync();

    expect(store.values['user-a']!.single.pendingMutation, isNotNull);
    expect(repository.getCalls, contains(_id(1)));
    controller.dispose();
  });

  test(
    'account switch prevents old callbacks from changing the new account',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final response = Completer<RemoteDocument>();
      final repository = _FakeRepository()..getResult = response.future;
      final controller = DocumentsController(
        auth: auth,
        repository: repository,
        localStore: _FakeStore(),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );
      await controller.initialize();
      await controller.loadLayer(_id(2));
      final opening = controller.openDocument(_id(1));
      auth.switchTo('user-b');
      await Future<void>.delayed(Duration.zero);
      response.complete(_remote(_id(1), _id(2), 'A', 'old-account', 2));
      await opening;

      expect(controller.state.selected, isNull);
      expect(controller.state.items, isEmpty);
      controller.dispose();
    },
  );

  test(
    'a local edit made while the snapshot is loading is not overwritten',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final store = _FakeStore()..readGate = Completer<List<LocalDocument>>();
      final controller = DocumentsController(
        auth: auth,
        repository: _FakeRepository(),
        localStore: store,
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );
      final initialization = controller.initialize();
      await Future<void>.delayed(Duration.zero);
      final created = await controller.createDocument(_id(2));
      store.readGate!.complete(<LocalDocument>[]);
      await initialization;

      expect(controller.knownDocuments.single.id, created.id);
      controller.dispose();
    },
  );

  test(
    'a failed snapshot read blocks writes until the account is recovered',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final store = _FakeStore()
        ..readError = const DocumentLocalStoreException('corrupt');
      final controller = DocumentsController(
        auth: auth,
        repository: _FakeRepository(),
        localStore: store,
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      await controller.createDocument(_id(2));
      await controller.sync();

      expect(store.writeCalls, 0);
      expect(controller.state.message, contains('não foram carregados'));
      controller.dispose();
    },
  );

  test('repeated trash keeps the original deletion time and TTL', () async {
    final auth = _FakeAuthGateway('user-a');
    final now = DateTime.utc(2026, 10, 13, 12);
    final controller = DocumentsController(
      auth: auth,
      repository: _FakeRepository(),
      localStore: _FakeStore(),
      now: () => now,
      autosaveInterval: const Duration(hours: 1),
      syncInterval: const Duration(hours: 1),
    );

    await controller.initialize();
    final document = await controller.createDocument(_id(2));
    await controller.trashDocument(document.id);
    final first = controller.knownDocuments.single;
    await controller.trashDocument(document.id);
    final second = controller.knownDocuments.single;

    expect(second.localRevision, first.localRevision);
    expect(second.deletedAt, first.deletedAt);
    expect(second.expiresAt, first.expiresAt);
    controller.dispose();
  });

  test(
    'restore at the exact TTL boundary cannot resurrect a trash item',
    () async {
      final auth = _FakeAuthGateway('user-a');
      var now = DateTime.utc(2026, 10, 13, 12);
      final controller = DocumentsController(
        auth: auth,
        repository: _FakeRepository(),
        localStore: _FakeStore(),
        now: () => now,
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      final document = await controller.createDocument(_id(2));
      await controller.trashDocument(document.id);
      final expiresAt = controller.knownDocuments.single.expiresAt!;
      now = expiresAt;
      await controller.restoreDocument(document.id);

      expect(controller.knownDocuments.single.isTrash, isTrue);
      expect(controller.knownDocuments.single.expiresAt, expiresAt);
      expect(controller.state.message, contains('expirou'));
      controller.dispose();
    },
  );

  test(
    'expired purge removes plaintext from the persisted tombstone',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final expired = LocalDocument(
        id: _id(1),
        layerId: _id(2),
        title: 'segredo título',
        markdown: 'segredo corpo',
        remoteVersion: BigInt.one,
        localRevision: 4,
        dirty: true,
        deletedAt: DateTime.utc(2026, 10, 1),
        expiresAt: DateTime.utc(2026, 10, 2),
        contentLoaded: true,
        links: const <DocumentLink>[
          DocumentLink(
            targetId: '33333333-3333-4333-8333-333333333333',
            title: 'segredo link',
            layerId: '44444444-4444-4444-8444-444444444444',
            status: DocumentLinkStatus.available,
          ),
        ],
        pendingMutation: PendingDocumentMutation(
          operation: 'update',
          layerId: '22222222-2222-4222-8222-222222222222',
          title: 'segredo título',
          markdown: 'segredo corpo',
          expectedVersion: BigInt.from(1),
          localRevision: 4,
        ),
      );
      final store = _FakeStore(<String, List<LocalDocument>>{
        'user-a': <LocalDocument>[expired],
      });
      final controller = DocumentsController(
        auth: auth,
        repository: _FakeRepository(),
        localStore: store,
        now: () => DateTime.utc(2026, 10, 3),
        autosaveInterval: const Duration(hours: 1),
        syncInterval: const Duration(hours: 1),
      );

      await controller.initialize();
      await controller.flushLocal();
      final stored = store.values['user-a']!.single;

      expect(stored.title, isEmpty);
      expect(stored.markdown, isEmpty);
      expect(stored.links, isEmpty);
      expect(stored.pendingMutation!.title, isEmpty);
      expect(stored.pendingMutation!.markdown, isEmpty);
      final serialized = jsonEncode(stored.toJson());
      expect(serialized, isNot(contains('segredo título')));
      expect(serialized, isNot(contains('segredo corpo')));
      expect(serialized, isNot(contains('segredo link')));
      controller.dispose();
    },
  );
}

String _id(int number) =>
    '${number.toString().padLeft(8, '0')}-1111-4111-8111-111111111111';

String Function() _idGenerator(List<String> ids) {
  var index = 0;
  return () => ids[index++];
}

DocumentMetadata _metadata(
  String id,
  String layerId,
  String title, [
  int version = 1,
]) => DocumentMetadata(
  id: id,
  layerId: layerId,
  title: title,
  version: BigInt.from(version),
  createdAt: DateTime.utc(2026, 10, 13),
  updatedAt: DateTime.utc(2026, 10, 13),
);

RemoteDocument _remote(
  String id,
  String layerId,
  String title,
  String markdown,
  int version,
) => RemoteDocument(
  id: id,
  layerId: layerId,
  title: title,
  version: BigInt.from(version),
  createdAt: DateTime.utc(2026, 10, 13),
  updatedAt: DateTime.utc(2026, 10, 13),
  markdown: markdown,
  links: const <DocumentLink>[],
);

LocalDocument _local({
  required String id,
  required String layerId,
  required String title,
  required String markdown,
  required BigInt version,
  required bool dirty,
  required bool contentLoaded,
}) => LocalDocument(
  id: id,
  layerId: layerId,
  title: title,
  markdown: markdown,
  remoteVersion: version,
  localRevision: dirty ? 1 : 0,
  dirty: dirty,
  contentLoaded: contentLoaded,
);

class _FakeAuthGateway extends AuthGateway {
  _FakeAuthGateway(String userId)
    : state = AuthViewState(
        phase: AuthPhase.authenticated,
        generation: 1,
        userId: userId,
      );

  @override
  AuthViewState state;

  void switchTo(String userId) {
    state = AuthViewState(
      phase: AuthPhase.authenticated,
      generation: state.generation + 1,
      userId: userId,
    );
    notifyListeners();
  }

  @override
  Future<AuthCredential> credential() async => AuthCredential(
    accessToken: 'token-${state.userId}',
    userId: state.userId!,
    generation: state.generation,
  );

  @override
  Future<AuthCredential> refreshAfterUnauthorized(
    AuthCredential rejected,
  ) async => rejected;
  @override
  Future<void> expireAfterUnauthorized(AuthCredential rejected) async {}
  @override
  Future<void> initialize() async {}
  @override
  Future<void> beginGoogleSignIn() async {}
  @override
  Future<void> cancelGoogleSignIn() async {}
  @override
  Future<void> handleCallback(Uri uri) async {}
  @override
  Future<void> signOut() async {}
}

class _FakeStore implements DocumentLocalStore {
  _FakeStore([Map<String, List<LocalDocument>>? initial])
    : values = <String, List<LocalDocument>>{
        for (final entry
            in (initial ?? <String, List<LocalDocument>>{}).entries)
          entry.key: List<LocalDocument>.of(entry.value),
      };

  final Map<String, List<LocalDocument>> values;
  int writeCalls = 0;
  Object? readError;
  Object? writeError;
  Completer<List<LocalDocument>>? readGate;

  @override
  Future<List<LocalDocument>> read(String userId) async {
    final error = readError;
    if (error != null) throw error;
    final gate = readGate;
    if (gate != null) return gate.future;
    return List<LocalDocument>.of(values[userId] ?? const <LocalDocument>[]);
  }

  @override
  Future<void> write(String userId, List<LocalDocument> documents) async {
    writeCalls++;
    final error = writeError;
    if (error != null) throw error;
    values[userId] = List<LocalDocument>.of(documents);
  }
}

class _FakeRepository implements DocumentRepository {
  List<DocumentPage> listPages = <DocumentPage>[
    DocumentPage(documents: <DocumentMetadata>[_metadata(_id(1), _id(2), 'A')]),
  ];
  final List<String?> listAfter = <String?>[];
  final List<String> createdMarkdown = <String>[];
  final List<String> updatedMarkdown = <String>[];
  final List<String> getCalls = <String>[];
  int createCalls = 0;
  Object? createError;
  Object? updateError;
  Object? getError;
  Future<RemoteDocument>? updateResult;
  Future<RemoteDocument>? getResult;

  @override
  Future<DocumentPage> list(
    String layerId, {
    bool subtree = false,
    bool trash = false,
    String? after,
  }) async {
    listAfter.add(after);
    return listPages.removeAt(0);
  }

  @override
  Future<RemoteDocument> get(String id) async {
    getCalls.add(id);
    final error = getError;
    if (error != null) throw error;
    return getResult ?? _remote(id, _id(2), 'A', 'body', 1);
  }

  @override
  Future<RemoteDocument> create({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
  }) async {
    createCalls++;
    createdMarkdown.add(markdown);
    final error = createError;
    if (error != null) {
      createError = null;
      throw error;
    }
    return _remote(id, layerId, title, markdown, createCalls);
  }

  @override
  Future<RemoteDocument> update({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
    required BigInt expectedVersion,
  }) async {
    updatedMarkdown.add(markdown);
    final error = updateError;
    if (error != null) {
      updateError = null;
      throw error;
    }
    return updateResult ??
        _remote(id, layerId, title, markdown, expectedVersion.toInt() + 1);
  }

  @override
  Future<RemoteDocument> trash({
    required String id,
    required BigInt expectedVersion,
  }) async => _remote(id, _id(2), 'A', '', expectedVersion.toInt() + 1);

  @override
  Future<RemoteDocument> restore({
    required String id,
    required BigInt expectedVersion,
    String? layerId,
  }) async =>
      _remote(id, layerId ?? _id(2), 'A', 'body', expectedVersion.toInt() + 1);
}
