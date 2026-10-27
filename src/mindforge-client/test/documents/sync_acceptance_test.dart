import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/documents/document_models.dart';
import 'package:mindforge_client/documents/document_repository.dart';
import 'package:mindforge_client/documents/documents_controller.dart';

const _defaultLayerId = '00000001-1111-4111-8111-111111111111';

void main() {
  const layerId = '00000001-1111-4111-8111-111111111111';
  const documentId = '00000002-1111-4111-8111-111111111111';

  test(
    'new document body survives placeholder create and final sync',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final repository = _FakeDocumentRepository();
      final controller = _controller(
        auth: auth,
        repository: repository,
        store: _FakeDocumentStore(),
        idGenerator: () => documentId,
      );

      await controller.initialize();
      await controller.createDocument(layerId, title: 'A note');
      await controller.editDocument(markdown: 'body survives create');
      await controller.sync();

      expect(repository.createRequests, hasLength(1));
      expect(repository.createRequests.single.markdown, isEmpty);
      expect(repository.updateRequests, hasLength(1));
      expect(repository.updateRequests.single.id, documentId);
      expect(repository.updateRequests.single.markdown, 'body survives create');
      expect(controller.state.selected!.dirty, isFalse);
      expect(controller.state.selected!.remoteVersion, BigInt.two);
      controller.dispose();
    },
  );

  test(
    'second manual sync works and automatic plus manual sync is single-flight',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final repository = _FakeDocumentRepository();
      final store = _FakeDocumentStore(<String, List<LocalDocument>>{
        'user-a': <LocalDocument>[
          _local(
            id: documentId,
            layerId: layerId,
            title: 'A note',
            markdown: 'first',
            version: 1,
            localRevision: 1,
            dirty: true,
          ),
        ],
      });
      final controller = _controller(
        auth: auth,
        repository: repository,
        store: store,
        syncInterval: Duration.zero,
      );

      await controller.initialize();
      await controller.openDocument(documentId);
      await controller.sync();
      await controller.editDocument(markdown: 'second');
      await controller.sync();

      expect(
        repository.updateRequests.map((request) => request.markdown),
        <String>['first', 'second'],
      );

      final gate = Completer<RemoteDocument>();
      final updateStarted = Completer<void>();
      repository.updateHandler = (request) {
        if (!updateStarted.isCompleted) updateStarted.complete();
        return gate.future;
      };
      await controller.editDocument(markdown: 'third');
      repository.updateRequests.clear();

      final automatic = controller.onResume();
      await updateStarted.future;
      final manual = controller.sync();
      await Future<void>.delayed(Duration.zero);

      expect(repository.updateRequests, hasLength(1));

      gate.complete(
        _remote(
          id: documentId,
          layerId: layerId,
          title: 'A note',
          markdown: 'third',
          version: 3,
        ),
      );
      await automatic;
      await manual;
      controller.dispose();
    },
  );

  test(
    'CAS conflict keeps the draft and a second sync does not overwrite it',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final repository = _FakeDocumentRepository();
      repository.updateHandler = (_) async {
        throw const DocumentApiException(
          statusCode: 409,
          code: 'conflict',
          message: 'stale version',
        );
      };
      repository.getHandler = (_) async => _remote(
        id: documentId,
        layerId: layerId,
        title: 'A note',
        markdown: 'server version',
        version: 2,
      );
      final controller = _controller(
        auth: auth,
        repository: repository,
        store: _FakeDocumentStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[
            _local(
              id: documentId,
              layerId: layerId,
              title: 'A note',
              markdown: 'local draft',
              version: 1,
              localRevision: 1,
              dirty: true,
            ),
          ],
        }),
        syncInterval: Duration.zero,
      );

      await controller.initialize();
      await controller.openDocument(documentId);
      await controller.onResume();

      expect(controller.state.selected!.conflict, isTrue);
      expect(controller.state.selected!.markdown, 'local draft');
      expect(repository.updateRequests, hasLength(1));

      await controller.sync();

      expect(repository.updateRequests, hasLength(1));
      expect(controller.state.selected!.markdown, 'local draft');
      expect(controller.state.selected!.conflict, isTrue);
      controller.dispose();
    },
  );

  test(
    'editing while update is pending keeps the new revision dirty',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final response = Completer<RemoteDocument>();
      final updateStarted = Completer<void>();
      final repository = _FakeDocumentRepository()
        ..updateHandler = (_) {
          updateStarted.complete();
          return response.future;
        };
      final controller = _controller(
        auth: auth,
        repository: repository,
        store: _FakeDocumentStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[
            _local(
              id: documentId,
              layerId: layerId,
              title: 'A note',
              markdown: 'original',
              version: 1,
              localRevision: 1,
              dirty: false,
            ),
          ],
        }),
      );

      await controller.initialize();
      await controller.openDocument(documentId);
      await controller.editDocument(markdown: 'first revision');
      final sync = controller.sync();
      await updateStarted.future;
      await controller.editDocument(markdown: 'new revision');
      response.complete(
        _remote(
          id: documentId,
          layerId: layerId,
          title: 'A note',
          markdown: 'first revision',
          version: 2,
        ),
      );
      await sync;

      expect(controller.state.selected!.markdown, 'new revision');
      expect(controller.state.selected!.dirty, isTrue);
      expect(controller.state.selected!.localRevision, 3);
      expect(controller.state.selected!.remoteVersion, BigInt.two);
      controller.dispose();
    },
  );

  test(
    'account A response cannot mutate account B after a generation switch',
    () async {
      final auth = _FakeAuthGateway('user-a');
      final response = Completer<RemoteDocument>();
      final updateStarted = Completer<void>();
      final repository = _FakeDocumentRepository()
        ..updateHandler = (_) {
          if (!updateStarted.isCompleted) updateStarted.complete();
          return response.future;
        };
      final controller = _controller(
        auth: auth,
        repository: repository,
        store: _FakeDocumentStore(<String, List<LocalDocument>>{
          'user-a': <LocalDocument>[
            _local(
              id: documentId,
              layerId: layerId,
              title: 'A note',
              markdown: 'account A',
              version: 1,
              localRevision: 1,
              dirty: false,
            ),
          ],
          'user-b': <LocalDocument>[
            _local(
              id: documentId,
              layerId: layerId,
              title: 'B note',
              markdown: 'account B',
              version: 7,
              localRevision: 2,
              dirty: true,
            ),
          ],
        }),
      );

      await controller.initialize();
      await controller.openDocument(documentId);
      await controller.editDocument(markdown: 'A pending update');
      final sync = controller.sync();
      await updateStarted.future;

      auth.switchTo('user-b');
      await controller.initialize();
      expect(controller.state.items.single.markdown, 'account B');
      expect(controller.state.items.single.remoteVersion, BigInt.from(7));

      response.complete(
        _remote(
          id: documentId,
          layerId: layerId,
          title: 'A note',
          markdown: 'A server response',
          version: 2,
        ),
      );
      await sync;

      expect(controller.state.items.single.title, 'B note');
      expect(controller.state.items.single.markdown, 'account B');
      expect(controller.state.items.single.remoteVersion, BigInt.from(7));
      controller.dispose();
    },
  );
}

DocumentsController _controller({
  required _FakeAuthGateway auth,
  required _FakeDocumentRepository repository,
  required _FakeDocumentStore store,
  Duration syncInterval = const Duration(hours: 1),
  String Function()? idGenerator,
}) => DocumentsController(
  auth: auth,
  repository: repository,
  localStore: store,
  autosaveInterval: const Duration(hours: 1),
  syncInterval: syncInterval,
  idGenerator: idGenerator,
);

LocalDocument _local({
  required String id,
  required String layerId,
  required String title,
  required String markdown,
  required int version,
  required int localRevision,
  required bool dirty,
}) => LocalDocument(
  id: id,
  layerId: layerId,
  title: title,
  markdown: markdown,
  remoteVersion: BigInt.from(version),
  localRevision: localRevision,
  dirty: dirty,
  contentLoaded: true,
);

RemoteDocument _remote({
  required String id,
  required String layerId,
  required String title,
  required String markdown,
  required int version,
}) => RemoteDocument(
  id: id,
  layerId: layerId,
  title: title,
  version: BigInt.from(version),
  createdAt: DateTime.utc(2026, 10, 13),
  updatedAt: DateTime.utc(2026, 10, 13),
  markdown: markdown,
  links: const <DocumentLink>[],
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

class _FakeDocumentStore implements DocumentLocalStore {
  _FakeDocumentStore([Map<String, List<LocalDocument>>? initial])
    : values = <String, List<LocalDocument>>{
        for (final entry
            in (initial ?? <String, List<LocalDocument>>{}).entries)
          entry.key: List<LocalDocument>.of(entry.value),
      };

  final Map<String, List<LocalDocument>> values;

  @override
  Future<List<LocalDocument>> read(String userId) async =>
      List<LocalDocument>.of(values[userId] ?? const <LocalDocument>[]);

  @override
  Future<void> write(String userId, List<LocalDocument> documents) async {
    values[userId] = List<LocalDocument>.of(documents);
  }
}

class _CreateRequest {
  const _CreateRequest({
    required this.id,
    required this.layerId,
    required this.title,
    required this.markdown,
  });

  final String id;
  final String layerId;
  final String title;
  final String markdown;
}

class _UpdateRequest {
  const _UpdateRequest({
    required this.id,
    required this.layerId,
    required this.title,
    required this.markdown,
    required this.expectedVersion,
  });

  final String id;
  final String layerId;
  final String title;
  final String markdown;
  final BigInt expectedVersion;
}

class _FakeDocumentRepository implements DocumentRepository {
  final List<_CreateRequest> createRequests = <_CreateRequest>[];
  final List<_UpdateRequest> updateRequests = <_UpdateRequest>[];

  Future<RemoteDocument> Function(_CreateRequest request)? createHandler;
  Future<RemoteDocument> Function(_UpdateRequest request)? updateHandler;
  Future<RemoteDocument> Function(String id)? getHandler;

  @override
  Future<DocumentPage> list(
    String layerId, {
    bool subtree = false,
    bool trash = false,
    String? after,
  }) async => DocumentPage(documents: <DocumentMetadata>[]);

  @override
  Future<RemoteDocument> get(String id) async =>
      getHandler?.call(id) ??
      _remote(
        id: id,
        layerId: _defaultLayerId,
        title: 'A note',
        markdown: 'remote body',
        version: 1,
      );

  @override
  Future<RemoteDocument> create({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
  }) async {
    final request = _CreateRequest(
      id: id,
      layerId: layerId,
      title: title,
      markdown: markdown,
    );
    createRequests.add(request);
    return createHandler?.call(request) ??
        _remote(
          id: id,
          layerId: layerId,
          title: title,
          markdown: markdown,
          version: 1,
        );
  }

  @override
  Future<RemoteDocument> update({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
    required BigInt expectedVersion,
  }) async {
    final request = _UpdateRequest(
      id: id,
      layerId: layerId,
      title: title,
      markdown: markdown,
      expectedVersion: expectedVersion,
    );
    updateRequests.add(request);
    return updateHandler?.call(request) ??
        _remote(
          id: id,
          layerId: layerId,
          title: title,
          markdown: markdown,
          version: expectedVersion.toInt() + 1,
        );
  }

  @override
  Future<RemoteDocument> trash({
    required String id,
    required BigInt expectedVersion,
  }) async => _remote(
    id: id,
    layerId: _defaultLayerId,
    title: 'A note',
    markdown: '',
    version: expectedVersion.toInt() + 1,
  );

  @override
  Future<RemoteDocument> restore({
    required String id,
    required BigInt expectedVersion,
    String? layerId,
  }) async => _remote(
    id: id,
    layerId: layerId ?? _defaultLayerId,
    title: 'A note',
    markdown: 'remote body',
    version: expectedVersion.toInt() + 1,
  );
}
