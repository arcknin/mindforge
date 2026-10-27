import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/foundation.dart';

import '../auth/auth_gateway.dart';
import 'document_models.dart';
import 'document_repository.dart';

enum DocumentsPhase { idle, loading, ready, failed }

class DocumentsState {
  DocumentsState({
    List<LocalDocument> items = const <LocalDocument>[],
    this.selected,
    this.syncing = false,
    this.saving = false,
    this.localPending = false,
    this.message,
    this.layerId,
    this.subtree = false,
    this.trash = false,
    this.lastSyncAt,
    this.phase = DocumentsPhase.idle,
  }) : items = List<LocalDocument>.unmodifiable(items);

  final List<LocalDocument> items;
  final LocalDocument? selected;
  final bool syncing;
  final bool saving;
  final bool localPending;
  final String? message;
  final String? layerId;
  final bool subtree;
  final bool trash;
  final DateTime? lastSyncAt;
  final DocumentsPhase phase;
}

class DocumentsController extends ChangeNotifier {
  DocumentsController({
    required AuthGateway auth,
    required DocumentRepository repository,
    required DocumentLocalStore localStore,
    this.autosaveInterval = const Duration(seconds: 1),
    this.syncInterval = const Duration(minutes: 30),
    DateTime Function()? now,
    String Function()? idGenerator,
    Set<String> Function(String layerId)? subtreeLayerIds,
  }) : _auth = auth,
       _repository = repository,
       _localStore = localStore,
       _now = now ?? DateTime.now,
       _idGenerator = idGenerator ?? _newUuidV4,
       _subtreeLayerIds = subtreeLayerIds {
    _auth.addListener(_handleAuthChanged);
  }

  final AuthGateway _auth;
  final DocumentRepository _repository;
  final DocumentLocalStore _localStore;
  final DateTime Function() _now;
  final String Function() _idGenerator;
  final Set<String> Function(String layerId)? _subtreeLayerIds;

  final Duration autosaveInterval;
  final Duration syncInterval;

  final List<LocalDocument> _documents = <LocalDocument>[];
  final Set<String> _lifecycleOnly = <String>{};
  final Set<String> _newDocumentIds = <String>{};
  final Map<String, RemoteDocument> _conflictRemote =
      <String, RemoteDocument>{};

  DocumentsState _state = DocumentsState();
  String? _activeUserId;
  int? _activeGeneration;
  String? _selectedId;
  String? _viewLayerId;
  bool _viewSubtree = false;
  bool _viewTrash = false;
  Set<String> _visibleLayerIds = <String>{};
  bool _storageReady = false;
  bool _localDirty = false;
  int _mutationEpoch = 0;
  bool _disposed = false;
  DateTime? _nextAutomaticSyncAt;
  Future<void>? _accountLoad;
  Future<void>? _syncInFlight;
  Timer? _autosaveTimer;
  Timer? _syncTimer;

  DocumentsState get state => _state;

  /// All cached documents, including layers outside the current filtered view.
  List<LocalDocument> get knownDocuments =>
      List<LocalDocument>.unmodifiable(_documents);

  Future<void> initialize() async {
    await _auth.initialize();
    _handleAuthChanged();
    await _accountLoad;
  }

  Future<void> loadLayer(
    String layerId, {
    bool subtree = false,
    bool trash = false,
  }) async {
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (userId == null ||
        generation == null ||
        !_isCurrent(userId, generation)) {
      return;
    }
    try {
      await flushLocal();
    } on Object {
      return;
    }
    if (!_isCurrent(userId, generation)) return;
    _viewLayerId = layerId;
    _viewSubtree = subtree;
    _viewTrash = trash;
    _visibleLayerIds = <String>{layerId};
    final subtreeLayerIds = _subtreeLayerIds;
    if (subtree && subtreeLayerIds != null) {
      _visibleLayerIds.addAll(subtreeLayerIds(layerId));
    }
    _publish(phase: DocumentsPhase.loading, message: null);
    try {
      final metadata = <String, DocumentMetadata>{};
      String? after;
      final seenCursors = <String>{};
      do {
        final page = await _repository.list(
          layerId,
          subtree: subtree,
          trash: trash,
          after: after,
        );
        if (!_isCurrent(userId, generation)) return;
        for (final item in page.documents) {
          metadata[item.id] = item;
          _visibleLayerIds.add(item.layerId);
        }
        after = page.nextCursor;
        if (after != null && !seenCursors.add(after)) {
          throw const DocumentResponseException();
        }
      } while (after != null);
      _mergeMetadata(metadata.values);
      _publish(phase: DocumentsPhase.ready, message: null);
    } on Object catch (error) {
      if (!_isCurrent(userId, generation)) return;
      _publish(phase: DocumentsPhase.failed, message: _messageFor(error));
    }
  }

  Future<void> openDocument(String id) async {
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (userId == null ||
        generation == null ||
        !_isCurrent(userId, generation)) {
      return;
    }
    var index = _indexOf(id);
    if (index < 0) {
      try {
        final remote = await _repository.get(id);
        if (!_isCurrent(userId, generation)) return;
        _documents.add(remote.toLocal());
        _mutationEpoch++;
        _localDirty = true;
        index = _documents.length - 1;
        _selectedId = id;
        _publish(phase: DocumentsPhase.ready, message: null);
        return;
      } on Object catch (error) {
        if (_isCurrent(userId, generation)) {
          _publish(message: _messageFor(error));
        }
        return;
      }
    }
    _selectedId = id;
    final existing = _documents[index];
    _publish(
      phase: _state.phase == DocumentsPhase.idle
          ? DocumentsPhase.ready
          : _state.phase,
    );
    if (existing.contentLoaded || existing.dirty) return;
    final capturedRevision = existing.localRevision;
    final capturedGeneration = generation;
    try {
      final remote = await _repository.get(id);
      if (!_isCurrent(userId, capturedGeneration)) return;
      final currentIndex = _indexOf(id);
      if (currentIndex < 0) return;
      final current = _documents[currentIndex];
      if (current.localRevision != capturedRevision || current.dirty) return;
      _documents[currentIndex] = remote.toLocal(
        localRevision: capturedRevision,
      );
      _localDirty = true;
      _publish(phase: DocumentsPhase.ready, message: null);
    } on Object catch (error) {
      if (_isCurrent(userId, generation)) {
        _publish(message: _messageFor(error));
      }
    }
  }

  Future<LocalDocument> createDocument(
    String layerId, {
    String title = 'Sem título',
  }) async {
    _requireCurrentSession();
    final normalizedTitle = title.trim();
    _validateTitle(normalizedTitle);
    final document = LocalDocument(
      id: _idGenerator(),
      layerId: layerId,
      title: normalizedTitle,
      markdown: '',
      remoteVersion: null,
      localRevision: 1,
      dirty: true,
      contentLoaded: true,
    );
    _documents.add(document);
    _newDocumentIds.add(document.id);
    _mutationEpoch++;
    _selectedId = document.id;
    _localDirty = true;
    _publish(phase: DocumentsPhase.ready, message: null);
    return document;
  }

  Future<void> editDocument({String? title, String? markdown}) async {
    final index = _selectedIndex();
    if (index < 0) return;
    final current = _documents[index];
    if (!current.contentLoaded) {
      _publish(message: 'Abra o documento antes de editar o conteúdo.');
      return;
    }
    if (title != null && !_isEditableTitle(title)) {
      _publish(
        message:
            'O título deve ter no máximo 255 caracteres e não conter controles.',
      );
      return;
    }
    if (markdown != null && !_isValidMarkdown(markdown)) {
      _publish(
        message: 'O Markdown excede 1 MiB ou contém um caractere inválido.',
      );
      return;
    }
    _documents[index] = current.copyWith(
      title: title ?? current.title,
      markdown: markdown ?? current.markdown,
      localRevision: current.localRevision + 1,
      dirty: true,
      conflict: false,
      uncertain: false,
      contentLoaded: true,
    );
    _lifecycleOnly.remove(current.id);
    _mutationEpoch++;
    _localDirty = true;
    _publish(message: null);
  }

  Future<void> moveDocument(String id, String layerId) async {
    _requireCurrentSession();
    final index = _indexOf(id);
    if (index < 0 || layerId.trim().isEmpty) return;
    final current = _documents[index];
    _documents[index] = current.copyWith(
      layerId: layerId,
      localRevision: current.localRevision + 1,
      dirty: true,
      conflict: false,
      uncertain: false,
    );
    _lifecycleOnly.remove(id);
    _mutationEpoch++;
    _localDirty = true;
    _publish(message: null);
  }

  Future<void> trashDocument(String id) async {
    _requireCurrentSession();
    final index = _indexOf(id);
    if (index < 0) return;
    final current = _documents[index];
    if (current.isTrash) return;
    final deletedAt = _now().toUtc();
    _documents[index] = current.copyWith(
      localRevision: current.localRevision + 1,
      dirty: true,
      deletedAt: deletedAt,
      expiresAt: deletedAt.add(const Duration(hours: 240)),
      conflict: false,
      uncertain: false,
    );
    _lifecycleOnly.add(id);
    _mutationEpoch++;
    _localDirty = true;
    _publish(message: null);
  }

  Future<void> restoreDocument(String id, {String? layerId}) async {
    _requireCurrentSession();
    final index = _indexOf(id);
    if (index < 0 || (layerId != null && layerId.trim().isEmpty)) return;
    final current = _documents[index];
    if (!current.isTrash) {
      _publish(message: 'O documento já está ativo.');
      return;
    }
    final expiresAt = current.expiresAt;
    if (expiresAt != null && !_now().toUtc().isBefore(expiresAt.toUtc())) {
      _purgeExpiredDocuments();
      _publish(message: 'A lixeira deste documento expirou.');
      return;
    }
    _documents[index] = current.copyWith(
      layerId: layerId ?? current.layerId,
      localRevision: current.localRevision + 1,
      dirty: true,
      deletedAt: null,
      expiresAt: null,
      conflict: false,
      uncertain: false,
    );
    _lifecycleOnly.add(id);
    _mutationEpoch++;
    _localDirty = true;
    _publish(message: null);
  }

  Future<void> flushLocal() async {
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (userId == null || generation == null || !_isCurrent(userId, generation))
      return;
    if (!_storageReady) {
      const failure = DocumentLocalStoreException(
        'the account snapshot was not loaded successfully',
      );
      _publish(
        saving: false,
        message:
            'Os documentos locais não foram carregados; o snapshot foi preservado.',
      );
      throw failure;
    }
    final snapshot = List<LocalDocument>.unmodifiable(_documents);
    final epoch = _mutationEpoch;
    final revisions = <String, int>{
      for (final document in snapshot) document.id: document.localRevision,
    };
    _publish(saving: true);
    try {
      await _localStore.write(userId, snapshot);
      if (_isCurrent(userId, generation)) {
        _localDirty = _mutationEpoch != epoch;
        if (!_localDirty) {
          for (final entry in revisions.entries) {
            final index = _indexOf(entry.key);
            if (index >= 0 && _documents[index].localRevision != entry.value) {
              _localDirty = true;
            }
          }
        }
        _publish(saving: false, message: null);
      }
    } on Object catch (error) {
      if (_isCurrent(userId, generation)) {
        _publish(saving: false, message: _messageFor(error));
      }
      rethrow;
    }
  }

  Future<void> sync() {
    return _startSync(automatic: false);
  }

  Future<void> _startSync({required bool automatic}) {
    final existing = _syncInFlight;
    if (existing != null) return existing;
    final run = _runSync(automatic: automatic);
    late Future<void> shared;
    shared = run.then<void>(
      (_) => _clearSyncFuture(shared),
      onError: (Object error, StackTrace stack) {
        _clearSyncFuture(shared);
        Error.throwWithStackTrace(error, stack);
      },
    );
    _syncInFlight = shared;
    return shared;
  }

  Future<void> onResume() async {
    _purgeExpiredDocuments();
    final due =
        _nextAutomaticSyncAt == null || !_now().isBefore(_nextAutomaticSyncAt!);
    if (due) await _runAutomaticSync();
  }

  Future<void> resolveConflict(String id, {required bool useLocal}) async {
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (userId == null ||
        generation == null ||
        !_isCurrent(userId, generation)) {
      return;
    }
    final index = _indexOf(id);
    if (index < 0) return;
    var remote = _conflictRemote[id];
    if (!useLocal) {
      try {
        remote = await _repository.get(id);
        if (!_isCurrent(userId, generation)) return;
      } on Object catch (error) {
        _publish(message: _messageFor(error));
        return;
      }
      _documents[index] = remote.toLocal();
      _conflictRemote.remove(id);
      _lifecycleOnly.remove(id);
      _localDirty = true;
      _publish(message: null);
      return;
    }
    final current = _documents[index];
    _documents[index] = current.copyWith(
      remoteVersion: remote?.version ?? current.remoteVersion,
      remoteDeleted: remote?.isTrash ?? current.remoteDeleted,
      conflict: false,
      uncertain: false,
      dirty: true,
      pendingMutation: null,
    );
    _conflictRemote.remove(id);
    _lifecycleOnly.remove(id);
    _localDirty = true;
    _publish(message: null);
  }

  Future<bool> prepareLogout({
    Duration budget = const Duration(seconds: 12),
  }) async {
    if (_activeUserId == null) return true;
    var syncSucceeded = true;
    try {
      await sync().timeout(budget);
    } on Object {
      syncSucceeded = false;
    }
    try {
      await flushLocal().timeout(budget);
    } on Object {
      return false;
    }
    if (!syncSucceeded) return false;
    return !_documents.any(
      (document) =>
          document.dirty ||
          document.uncertain ||
          document.conflict ||
          document.pendingMutation != null,
    );
  }

  @override
  void dispose() {
    _disposed = true;
    _auth.removeListener(_handleAuthChanged);
    _autosaveTimer?.cancel();
    _syncTimer?.cancel();
    super.dispose();
  }

  void _handleAuthChanged() {
    if (_disposed) return;
    final authState = _auth.state;
    if (authState.phase == AuthPhase.authenticated &&
        authState.userId != null) {
      if (_activeUserId == authState.userId &&
          _activeGeneration == authState.generation) {
        return;
      }
      _activeUserId = authState.userId;
      _activeGeneration = authState.generation;
      _documents.clear();
      _mutationEpoch = 0;
      _storageReady = false;
      _selectedId = null;
      _viewLayerId = null;
      _viewSubtree = false;
      _viewTrash = false;
      _visibleLayerIds = <String>{};
      _localDirty = false;
      _conflictRemote.clear();
      _lifecycleOnly.clear();
      _newDocumentIds.clear();
      _nextAutomaticSyncAt = _now().add(syncInterval);
      _publish(phase: DocumentsPhase.loading, message: null);
      _startTimers();
      _accountLoad = _loadAccount(authState.userId!, authState.generation);
      return;
    }
    _activeUserId = null;
    _activeGeneration = null;
    _documents.clear();
    _mutationEpoch = 0;
    _selectedId = null;
    _viewLayerId = null;
    _localDirty = false;
    _storageReady = false;
    _conflictRemote.clear();
    _lifecycleOnly.clear();
    _newDocumentIds.clear();
    _nextAutomaticSyncAt = null;
    _autosaveTimer?.cancel();
    _syncTimer?.cancel();
    _publish(phase: DocumentsPhase.idle, message: authState.message);
  }

  Future<void> _loadAccount(String userId, int generation) async {
    final epochBeforeRead = _mutationEpoch;
    try {
      final restored = await _localStore.read(userId);
      if (!_isCurrent(userId, generation)) return;
      final recovered = _purgeExpired(restored)
          .map(
            (document) => document.pendingMutation == null
                ? document
                : document.copyWith(uncertain: true),
          )
          .toList(growable: false);
      if (_mutationEpoch == epochBeforeRead) {
        _documents
          ..clear()
          ..addAll(recovered);
      } else {
        final existingIds = _documents.map((document) => document.id).toSet();
        for (final document in recovered) {
          if (existingIds.add(document.id)) _documents.add(document);
        }
      }
      if (recovered.any((document) => document.pendingMutation != null)) {
        _localDirty = true;
      }
      _storageReady = true;
      _publish(phase: DocumentsPhase.ready, message: null);
    } on Object catch (error) {
      if (!_isCurrent(userId, generation)) return;
      _storageReady = false;
      _publish(phase: DocumentsPhase.failed, message: _messageFor(error));
    }
  }

  void _startTimers() {
    _autosaveTimer?.cancel();
    _syncTimer?.cancel();
    if (autosaveInterval > Duration.zero) {
      _autosaveTimer = Timer.periodic(autosaveInterval, (_) {
        if (_localDirty && _activeUserId != null) {
          unawaited(_runAutosave());
        }
      });
    }
    _scheduleSyncTimer();
  }

  void _scheduleSyncTimer() {
    _syncTimer?.cancel();
    _syncTimer = null;
    if (syncInterval <= Duration.zero || _activeUserId == null) return;
    final now = _now();
    final due = _nextAutomaticSyncAt ?? now.add(syncInterval);
    var delay = due.difference(now);
    if (delay <= Duration.zero) delay = syncInterval;
    _syncTimer = Timer(delay, () {
      unawaited(_runAutomaticSync().whenComplete(_scheduleSyncTimer));
    });
  }

  Future<void> _runAutosave() async {
    try {
      await flushLocal();
    } on Object {
      // The state already exposes the failure; retain the in-memory draft.
    }
  }

  Future<void> _runAutomaticSync() async {
    final next = _nextAutomaticSyncAt;
    if (next != null && _now().isBefore(next)) return;
    if (_syncInFlight != null) return;
    try {
      await _startSync(automatic: true);
    } on Object {
      // A failed/partial sync remains dirty and will be retried after the next interval.
    }
  }

  Future<void> _runSync({required bool automatic}) async {
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (userId == null || generation == null || !_isCurrent(userId, generation))
      return;
    if (automatic &&
        _nextAutomaticSyncAt != null &&
        _now().isBefore(_nextAutomaticSyncAt!)) {
      return;
    }
    _purgeExpiredDocuments();
    try {
      await flushLocal();
    } on Object {
      return;
    }
    if (!_isCurrent(userId, generation)) return;
    _publish(syncing: true, message: null);
    var complete = true;
    try {
      if (!await _refreshCurrentViewMetadata(userId, generation)) {
        complete = false;
      }
      final persistedIntents = List<LocalDocument>.of(
        _documents.where((document) => document.pendingMutation != null),
      );
      for (final document in persistedIntents) {
        final pendingMutation = document.pendingMutation;
        if (pendingMutation == null) continue;
        if (document.conflict) {
          complete = false;
          continue;
        }
        final recovered = await _reconcile(
          document.id,
          _intentFromPending(document, pendingMutation),
          userId,
          generation,
          recoverMissingCreate: true,
        );
        if (!recovered) complete = false;
      }
      final pending = List<LocalDocument>.of(
        _documents.where(
          (document) => document.dirty && document.pendingMutation == null,
        ),
      );
      final newDocuments = pending
          .where((document) => document.remoteVersion == null)
          .toList();
      for (final document in newDocuments) {
        final index = _indexOf(document.id);
        if (index < 0 || !_isCurrent(userId, generation)) return;
        final current = _documents[index];
        final validationMessage = _validationMessage(current);
        if (validationMessage != null) {
          complete = false;
          _publish(message: validationMessage);
          continue;
        }
        final intent = _SyncIntent(
          kind: _SyncIntentKind.create,
          id: current.id,
          layerId: current.layerId,
          title: current.title,
          markdown: '',
          localRevision: current.localRevision,
        );
        if (!await _recordIntent(current.id, intent, userId, generation)) {
          complete = false;
          continue;
        }
        if (!_isCurrent(userId, generation)) return;
        final currentAfterIntent = _documents[_indexOf(current.id)];
        try {
          final remote = await _repository.create(
            id: currentAfterIntent.id,
            layerId: currentAfterIntent.layerId,
            title: currentAfterIntent.title,
            markdown: '',
          );
          if (!_isCurrent(userId, generation)) return;
          _acceptRemote(
            currentAfterIntent.id,
            remote,
            intent.localRevision,
            keepDirty: true,
          );
        } on Object catch (error) {
          final recoverMissingCreate =
              error is DocumentTransportException ||
              error is DocumentResponseException ||
              (error is DocumentApiException && error.statusCode >= 500);
          if (error is DocumentApiException && error.statusCode == 409) {
            _markUncertain(
              current.id,
              'O ID do documento já está em uso. Confira a versão remota antes de reaplicar.',
              conflict: true,
            );
          }
          final reconciled = await _reconcile(
            current.id,
            intent,
            userId,
            generation,
            recoverMissingCreate: recoverMissingCreate,
          );
          if (!reconciled) complete = false;
        }
      }

      final afterCreate = List<LocalDocument>.of(
        _documents.where((document) => document.dirty),
      );
      for (final document in afterCreate) {
        if (!_isCurrent(userId, generation)) return;
        final index = _indexOf(document.id);
        if (index < 0) continue;
        var current = _documents[index];
        if (current.conflict ||
            current.uncertain ||
            current.pendingMutation != null) {
          complete = false;
          continue;
        }
        if (current.remoteVersion == null) {
          complete = false;
          continue;
        }
        final validationMessage = _validationMessage(current);
        if (validationMessage != null) {
          complete = false;
          _publish(message: validationMessage);
          continue;
        }

        if (current.remoteDeleted && current.deletedAt == null) {
          final restoreIntent = _SyncIntent.restore(current);
          if (!await _recordIntent(
            current.id,
            restoreIntent,
            userId,
            generation,
          )) {
            complete = false;
            continue;
          }
          current = _documents[_indexOf(current.id)];
          try {
            final remote = await _repository.restore(
              id: current.id,
              expectedVersion: current.remoteVersion!,
              layerId: current.layerId,
            );
            if (!_isCurrent(userId, generation)) return;
            _acceptRemote(current.id, remote, current.localRevision);
            current = _documents[_indexOf(current.id)];
          } on Object catch (_) {
            final recovered = await _reconcile(
              current.id,
              restoreIntent,
              userId,
              generation,
            );
            if (!recovered) complete = false;
            continue;
          }
        }

        if (!_dependenciesReady(current)) {
          complete = false;
          _markUncertain(
            current.id,
            'Um link aponta para um documento novo que ainda não foi confirmado.',
          );
          continue;
        }

        final needsSnapshot =
            !_lifecycleOnly.contains(current.id) ||
            current.remoteVersion == null ||
            _newDocumentIds.contains(current.id);
        if (needsSnapshot) {
          final intent = _SyncIntent.update(current);
          if (!await _recordIntent(current.id, intent, userId, generation)) {
            complete = false;
            continue;
          }
          current = _documents[_indexOf(current.id)];
          try {
            final remote = await _repository.update(
              id: current.id,
              layerId: current.layerId,
              title: current.title,
              markdown: current.markdown,
              expectedVersion: current.remoteVersion!,
            );
            if (!_isCurrent(userId, generation)) return;
            _acceptRemote(current.id, remote, intent.localRevision);
            if (!_documents[_indexOf(current.id)].dirty) {
              _newDocumentIds.remove(current.id);
            }
          } on Object catch (_) {
            final recovered = await _reconcile(
              current.id,
              intent,
              userId,
              generation,
            );
            if (!recovered) complete = false;
            continue;
          }
        }

        final currentAfterSnapshot = _documents[_indexOf(current.id)];
        if (currentAfterSnapshot.deletedAt != null &&
            !currentAfterSnapshot.remoteDeleted) {
          final intent = _SyncIntent.trash(currentAfterSnapshot);
          if (!await _recordIntent(
            currentAfterSnapshot.id,
            intent,
            userId,
            generation,
          )) {
            complete = false;
            continue;
          }
          try {
            final remote = await _repository.trash(
              id: currentAfterSnapshot.id,
              expectedVersion: currentAfterSnapshot.remoteVersion!,
            );
            if (!_isCurrent(userId, generation)) return;
            _acceptRemote(
              currentAfterSnapshot.id,
              remote,
              intent.localRevision,
            );
            if (!_documents[_indexOf(currentAfterSnapshot.id)].dirty) {
              _newDocumentIds.remove(currentAfterSnapshot.id);
            }
          } on Object catch (_) {
            final recovered = await _reconcile(
              currentAfterSnapshot.id,
              intent,
              userId,
              generation,
            );
            if (!recovered) complete = false;
          }
        } else if (currentAfterSnapshot.deletedAt == null &&
            currentAfterSnapshot.remoteDeleted) {
          // A failed restore may leave the document dirty; keep it pending.
          complete = false;
        }
        final settledIndex = _indexOf(current.id);
        if (settledIndex >= 0) {
          final lifecycleSettled = _documents[settledIndex];
          if (_lifecycleOnly.contains(current.id) &&
              ((lifecycleSettled.deletedAt != null &&
                      lifecycleSettled.remoteDeleted) ||
                  (lifecycleSettled.deletedAt == null &&
                      !lifecycleSettled.remoteDeleted))) {
            _documents[settledIndex] = lifecycleSettled.copyWith(
              dirty: false,
              conflict: false,
              uncertain: false,
              pendingMutation: null,
            );
            _lifecycleOnly.remove(current.id);
            _mutationEpoch++;
            _localDirty = true;
          }
        }
      }
      if (!await _refreshCurrentViewMetadata(userId, generation)) {
        complete = false;
      }
      if (_documents.any((document) => document.dirty)) complete = false;
      if (complete) {
        _nextAutomaticSyncAt = _now().add(syncInterval);
        _state = DocumentsState(
          items: _visibleDocuments(),
          selected: _selectedDocument(),
          syncing: true,
          saving: _state.saving,
          localPending: _localDirty,
          message: null,
          layerId: _viewLayerId,
          subtree: _viewSubtree,
          trash: _viewTrash,
          lastSyncAt: _now().toUtc(),
          phase: DocumentsPhase.ready,
        );
        if (!_disposed) notifyListeners();
      }
    } finally {
      if (_isCurrent(userId, generation)) {
        _publish(syncing: false, message: complete ? null : _state.message);
        if (_localDirty) {
          try {
            await flushLocal();
          } on Object {
            // Keep remote result in memory and surface the store failure through state.
          }
        }
        _scheduleSyncTimer();
      }
    }
  }

  Future<bool> _refreshCurrentViewMetadata(
    String userId,
    int generation,
  ) async {
    final layerId = _viewLayerId;
    if (layerId == null) return true;
    final metadata = <String, DocumentMetadata>{};
    String? after;
    final seenCursors = <String>{};
    try {
      do {
        final page = await _repository.list(
          layerId,
          subtree: _viewSubtree,
          trash: _viewTrash,
          after: after,
        );
        if (!_isCurrent(userId, generation)) return false;
        for (final item in page.documents) {
          metadata[item.id] = item;
        }
        after = page.nextCursor;
        if (after != null && !seenCursors.add(after)) {
          throw const DocumentResponseException();
        }
      } while (after != null);
      _mergeMetadata(metadata.values);
      _publish(phase: DocumentsPhase.ready, message: null);
      return true;
    } on Object catch (error) {
      if (_isCurrent(userId, generation)) {
        _publish(message: _messageFor(error));
      }
      return false;
    }
  }

  Future<bool> _recordIntent(
    String id,
    _SyncIntent intent,
    String userId,
    int generation,
  ) async {
    final index = _indexOf(id);
    if (index < 0 || !_isCurrent(userId, generation)) return false;
    final current = _documents[index];
    _documents[index] = current.copyWith(
      pendingMutation: PendingDocumentMutation(
        operation: intent.kind.name,
        layerId: intent.layerId,
        title: intent.title,
        markdown: intent.markdown,
        expectedVersion: current.remoteVersion,
        localRevision: intent.localRevision,
        trash: intent.trash,
      ),
    );
    _mutationEpoch++;
    _localDirty = true;
    try {
      await flushLocal();
      return _isCurrent(userId, generation);
    } on Object {
      return false;
    }
  }

  _SyncIntent _intentFromPending(
    LocalDocument document,
    PendingDocumentMutation pending,
  ) => _SyncIntent(
    kind: switch (pending.operation) {
      'create' => _SyncIntentKind.create,
      'update' => _SyncIntentKind.update,
      'trash' => _SyncIntentKind.trash,
      'restore' => _SyncIntentKind.restore,
      _ => _SyncIntentKind.update,
    },
    id: document.id,
    layerId: pending.layerId,
    title: pending.title,
    markdown: pending.markdown,
    localRevision: pending.localRevision,
    trash: pending.trash,
  );

  Future<bool> _reconcile(
    String id,
    _SyncIntent intent,
    String userId,
    int generation, {
    bool recoverMissingCreate = false,
  }) async {
    if (!_isCurrent(userId, generation)) return false;
    try {
      final remote = await _repository.get(id);
      if (!_isCurrent(userId, generation)) return false;
      final matches = _remoteMatchesIntent(remote, intent);
      if (matches) {
        _acceptRemote(
          id,
          remote,
          intent.localRevision,
          keepDirty: intent.kind == _SyncIntentKind.create,
        );
        return true;
      }
      _conflictRemote[id] = remote;
      final index = _indexOf(id);
      if (index >= 0) {
        _documents[index] = _documents[index].copyWith(
          remoteVersion: remote.version,
          remoteDeleted: remote.isTrash,
          conflict: true,
          uncertain: true,
          dirty: true,
        );
        _mutationEpoch++;
        _localDirty = true;
      }
      _publish(
        message:
            'Resultado incerto. Confira a versão remota antes de tentar novamente.',
      );
      return false;
    } on Object catch (error) {
      if (error is DocumentApiException && error.statusCode == 404) {
        if (intent.kind == _SyncIntentKind.create && recoverMissingCreate) {
          _confirmCreateWasNotFound(id);
          return true;
        }
        _markUncertain(
          id,
          intent.kind == _SyncIntentKind.create
              ? 'A criação ficou ambígua e o ID não foi encontrado. Confira o conflito antes de tentar novamente.'
              : 'O documento não foi encontrado para esta operação. O rascunho foi preservado como conflito.',
          conflict: true,
        );
        return false;
      }
      _markUncertain(
        id,
        'Resultado incerto; não houve retry automático. ${_messageFor(error)}',
      );
      return false;
    }
  }

  bool _remoteMatchesIntent(RemoteDocument remote, _SyncIntent intent) =>
      remote.id == intent.id &&
      remote.layerId == intent.layerId &&
      remote.title == intent.title &&
      (intent.kind == _SyncIntentKind.create
          ? remote.version == BigInt.one && remote.markdown.isEmpty
          : remote.markdown == intent.markdown) &&
      remote.isTrash == intent.trash;

  void _acceptRemote(
    String id,
    RemoteDocument remote,
    int sentRevision, {
    bool keepDirty = false,
  }) {
    final index = _indexOf(id);
    if (index < 0) return;
    final current = _documents[index];
    final changedDuringRequest = current.localRevision != sentRevision;
    final preserveLocalSnapshot = changedDuringRequest || keepDirty;
    _documents[index] = remote
        .toLocal(
          localRevision: current.localRevision,
          dirty: changedDuringRequest || keepDirty,
          conflict: false,
          uncertain: false,
        )
        .copyWith(
          title: preserveLocalSnapshot ? current.title : null,
          markdown: preserveLocalSnapshot ? current.markdown : null,
          layerId: preserveLocalSnapshot ? current.layerId : null,
          deletedAt: preserveLocalSnapshot ? current.deletedAt : null,
          expiresAt: preserveLocalSnapshot ? current.expiresAt : null,
          remoteDeleted: remote.isTrash,
          links: preserveLocalSnapshot ? current.links : remote.links,
        );
    _mutationEpoch++;
    if (!changedDuringRequest && !keepDirty) _lifecycleOnly.remove(id);
    _localDirty = true;
    _publish(message: null);
  }

  bool _dependenciesReady(LocalDocument document) {
    final token = RegExp(r'\[\[doc:([0-9a-fA-F-]{36})\]\]');
    for (final match in token.allMatches(document.markdown)) {
      final target = _documents.where(
        (item) => item.id.toLowerCase() == match.group(1)!.toLowerCase(),
      );
      if (target.isNotEmpty && target.first.remoteVersion == null) return false;
    }
    return true;
  }

  void _confirmCreateWasNotFound(String id) {
    final index = _indexOf(id);
    if (index < 0) return;
    _documents[index] = _documents[index].copyWith(
      pendingMutation: null,
      conflict: false,
      uncertain: false,
      dirty: true,
    );
    _mutationEpoch++;
    _localDirty = true;
    _publish(
      message:
          'A criação anterior não foi encontrada. O documento continua pendente para uma nova tentativa controlada.',
    );
  }

  void _markUncertain(String id, String message, {bool conflict = false}) {
    final index = _indexOf(id);
    if (index >= 0) {
      _documents[index] = _documents[index].copyWith(
        uncertain: true,
        conflict: conflict || _documents[index].conflict,
        dirty: true,
      );
      _mutationEpoch++;
      _localDirty = true;
    }
    _publish(message: message);
  }

  void _mergeMetadata(Iterable<DocumentMetadata> metadata) {
    final incoming = <String, DocumentMetadata>{
      for (final item in metadata) item.id: item,
    };
    final knownLayers = <String>{..._visibleLayerIds};
    if (_viewLayerId != null) knownLayers.add(_viewLayerId!);
    knownLayers.addAll(incoming.values.map((item) => item.layerId));
    _visibleLayerIds = knownLayers;
    for (final item in incoming.values) {
      final index = _indexOf(item.id);
      if (index < 0) {
        _documents.add(item.toLocal());
        _mutationEpoch++;
        _localDirty = true;
        continue;
      }
      final current = _documents[index];
      if (current.dirty || current.conflict || current.uncertain) continue;
      final metadataLocal = item.toLocal();
      final remoteVersionChanged =
          current.remoteVersion != metadataLocal.remoteVersion;
      _documents[index] = current.copyWith(
        layerId: metadataLocal.layerId,
        title: metadataLocal.title,
        remoteVersion: metadataLocal.remoteVersion,
        markdown: remoteVersionChanged ? '' : current.markdown,
        deletedAt: metadataLocal.deletedAt,
        expiresAt: metadataLocal.expiresAt,
        remoteDeleted: metadataLocal.remoteDeleted,
        contentLoaded: remoteVersionChanged ? false : current.contentLoaded,
        links: remoteVersionChanged ? const <DocumentLink>[] : current.links,
      );
      if (remoteVersionChanged) _mutationEpoch++;
    }
  }

  List<LocalDocument> _purgeExpired(Iterable<LocalDocument> documents) {
    final now = _now().toUtc();
    final kept = <LocalDocument>[];
    for (final document in documents) {
      if (document.expiresAt != null &&
          !now.isBefore(document.expiresAt!.toUtc())) {
        if (document.dirty ||
            document.uncertain ||
            document.pendingMutation != null) {
          kept.add(
            document.copyWith(
              title: '',
              markdown: '',
              contentLoaded: false,
              links: const <DocumentLink>[],
              uncertain: true,
              dirty: true,
              pendingMutation: document.pendingMutation == null
                  ? null
                  : PendingDocumentMutation(
                      operation: document.pendingMutation!.operation,
                      layerId: document.pendingMutation!.layerId,
                      title: '',
                      markdown: '',
                      expectedVersion:
                          document.pendingMutation!.expectedVersion,
                      localRevision: document.pendingMutation!.localRevision,
                      trash: document.pendingMutation!.trash,
                    ),
            ),
          );
        }
        _localDirty = true;
        continue;
      }
      kept.add(document);
    }
    return kept;
  }

  void _purgeExpiredDocuments() {
    final before = List<LocalDocument>.of(_documents);
    final after = _purgeExpired(before);
    if (before.length == after.length && _sameDocumentObjects(before, after)) {
      return;
    }
    _documents
      ..clear()
      ..addAll(after);
    _mutationEpoch++;
    _localDirty = true;
    _publish(message: null);
  }

  bool _sameDocumentObjects(
    List<LocalDocument> left,
    List<LocalDocument> right,
  ) {
    if (left.length != right.length) return false;
    for (var index = 0; index < left.length; index++) {
      if (!identical(left[index], right[index])) return false;
    }
    return true;
  }

  List<LocalDocument> _visibleDocuments() {
    return _documents
        .where((document) {
          if (_viewLayerId == null) return true;
          final inLayer = _viewSubtree
              ? _visibleLayerIds.contains(document.layerId)
              : document.layerId == _viewLayerId;
          final inState = _viewTrash ? document.isTrash : !document.isTrash;
          return inLayer && inState;
        })
        .toList(growable: false);
  }

  LocalDocument? _selectedDocument() {
    final id = _selectedId;
    if (id == null) return null;
    final index = _indexOf(id);
    return index < 0 ? null : _documents[index];
  }

  int _selectedIndex() => _selectedId == null ? -1 : _indexOf(_selectedId!);

  int _indexOf(String id) =>
      _documents.indexWhere((document) => document.id == id);

  bool _isCurrent(String userId, int generation) {
    final authState = _auth.state;
    return !_disposed &&
        _activeUserId == userId &&
        _activeGeneration == generation &&
        authState.phase == AuthPhase.authenticated &&
        authState.userId == userId &&
        authState.generation == generation;
  }

  void _requireCurrentSession() {
    if (_activeUserId == null ||
        _activeGeneration == null ||
        !_isCurrent(_activeUserId!, _activeGeneration!)) {
      throw const StaleAuthOperationException();
    }
  }

  void _publish({
    DocumentsPhase? phase,
    bool? syncing,
    bool? saving,
    String? message,
  }) {
    if (_disposed) return;
    _state = DocumentsState(
      items: _visibleDocuments(),
      selected: _selectedDocument(),
      syncing: syncing ?? _state.syncing,
      saving: saving ?? _state.saving,
      localPending: _localDirty,
      message: message,
      layerId: _viewLayerId,
      subtree: _viewSubtree,
      trash: _viewTrash,
      lastSyncAt: _state.lastSyncAt,
      phase: phase ?? _state.phase,
    );
    notifyListeners();
  }

  String _messageFor(Object error) {
    if (error is DocumentApiException) return error.message;
    if (error is DocumentValidationException) return error.message;
    if (error is DocumentLocalStoreException)
      return 'Não foi possível salvar os documentos localmente.';
    if (error is StaleAuthOperationException ||
        error is UnauthenticatedException)
      return 'A sessão mudou. Entre novamente.';
    if (error is DocumentTransportException)
      return 'Não foi possível alcançar o Brain Engine.';
    return 'Não foi possível concluir a operação documental.';
  }

  String? _validationMessage(LocalDocument document) {
    try {
      _validateTitle(document.title.trim());
    } on DocumentValidationException catch (error) {
      return error.message;
    }
    if (!_isValidMarkdown(document.markdown)) {
      return 'O Markdown excede 1 MiB ou contém um caractere inválido.';
    }
    return null;
  }

  static void _validateTitle(String title) {
    if (!_isEditableTitle(title) || title.trim().isEmpty) {
      throw const DocumentValidationException(
        'O título deve ter entre 1 e 255 caracteres e não conter controles.',
      );
    }
  }

  static bool _isEditableTitle(String title) {
    if (title.runes.length > 255) return false;
    return !title.runes.any((rune) => rune < 0x20 || rune == 0x7f);
  }

  static bool _isValidMarkdown(String markdown) {
    if (markdown.contains('\u0000')) return false;
    return utf8.encode(markdown).length <= 1024 * 1024;
  }

  void _clearSyncFuture(Future<void> future) {
    if (identical(_syncInFlight, future)) _syncInFlight = null;
  }

  static String _newUuidV4() {
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    String hex(int byte) => byte.toRadixString(16).padLeft(2, '0');
    return '${hex(bytes[0])}${hex(bytes[1])}${hex(bytes[2])}${hex(bytes[3])}-'
        '${hex(bytes[4])}${hex(bytes[5])}-'
        '${hex(bytes[6])}${hex(bytes[7])}-'
        '${hex(bytes[8])}${hex(bytes[9])}-'
        '${hex(bytes[10])}${hex(bytes[11])}${hex(bytes[12])}${hex(bytes[13])}${hex(bytes[14])}${hex(bytes[15])}';
  }
}

enum _SyncIntentKind { create, update, trash, restore }

class _SyncIntent {
  const _SyncIntent({
    required this.kind,
    required this.id,
    required this.layerId,
    required this.title,
    required this.markdown,
    required this.localRevision,
    this.trash = false,
  });

  factory _SyncIntent.update(LocalDocument document) => _SyncIntent(
    kind: _SyncIntentKind.update,
    id: document.id,
    layerId: document.layerId,
    title: document.title,
    markdown: document.markdown,
    localRevision: document.localRevision,
    trash: false,
  );

  factory _SyncIntent.trash(LocalDocument document) => _SyncIntent(
    kind: _SyncIntentKind.trash,
    id: document.id,
    layerId: document.layerId,
    title: document.title,
    markdown: document.markdown,
    localRevision: document.localRevision,
    trash: true,
  );

  factory _SyncIntent.restore(LocalDocument document) => _SyncIntent(
    kind: _SyncIntentKind.restore,
    id: document.id,
    layerId: document.layerId,
    title: document.title,
    markdown: document.markdown,
    localRevision: document.localRevision,
    trash: false,
  );

  final _SyncIntentKind kind;
  final String id;
  final String layerId;
  final String title;
  final String markdown;
  final int localRevision;
  final bool trash;
}
