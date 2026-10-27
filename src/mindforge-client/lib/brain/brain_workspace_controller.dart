import 'dart:async';

import 'package:flutter/foundation.dart';

import '../auth/auth_gateway.dart';
import 'brain_hierarchy.dart';
import 'brain_repository.dart';

enum WorkspacePhase { signedOut, loading, ready, failed }

enum MutationStatus { success, rejected, uncertain, conflict, alreadyInFlight }

class RenameConflict {
  const RenameConflict({
    required this.layerId,
    required this.draft,
    required this.latestLayer,
  });

  final String layerId;
  final String draft;
  final BrainLayer? latestLayer;
}

class MutationResult {
  const MutationResult(this.status, {this.message});

  final MutationStatus status;
  final String? message;
}

class WorkspaceViewState {
  const WorkspaceViewState({
    required this.phase,
    this.hierarchy,
    this.selectedLayerId,
    this.message,
    this.uncertainOperation,
    this.renameConflict,
    this.createInFlight = false,
    this.renameInFlight = false,
  });

  final WorkspacePhase phase;
  final BrainHierarchy? hierarchy;
  final String? selectedLayerId;
  final String? message;
  final String? uncertainOperation;
  final RenameConflict? renameConflict;
  final bool createInFlight;
  final bool renameInFlight;
}

class BrainWorkspaceController extends ChangeNotifier {
  BrainWorkspaceController({
    required AuthGateway auth,
    required BrainRepository repository,
  }) : _auth = auth,
       _repository = repository {
    _auth.addListener(_handleAuthChanged);
  }

  final AuthGateway _auth;
  final BrainRepository _repository;
  WorkspaceViewState _state = const WorkspaceViewState(
    phase: WorkspacePhase.signedOut,
  );
  WorkspaceViewState get state => _state;

  String? _activeUserId;
  int? _activeGeneration;
  BrainSummary? _summary;
  Future<void>? _loadFuture;
  bool _createInFlight = false;
  bool _renameInFlight = false;
  bool _disposed = false;

  Future<void> initialize() async {
    await _auth.initialize();
    _handleAuthChanged();
    await _loadFuture;
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
      _summary = null;
      _createInFlight = false;
      _renameInFlight = false;
      _setState(const WorkspaceViewState(phase: WorkspacePhase.loading));
      _loadFuture = _loadForSession(authState.userId!, authState.generation);
      return;
    }

    _activeUserId = null;
    _activeGeneration = null;
    _summary = null;
    _createInFlight = false;
    _renameInFlight = false;
    _setState(
      WorkspaceViewState(
        phase: WorkspacePhase.signedOut,
        message: authState.message,
      ),
    );
  }

  Future<void> _loadForSession(String userId, int generation) async {
    try {
      final summary = await _repository.bootstrap();
      if (!_isCurrent(userId, generation)) return;
      final hierarchy = await _repository.observe();
      if (!_isCurrent(userId, generation)) return;
      _validateHierarchy(summary, hierarchy);
      _summary = summary;
      _setState(
        WorkspaceViewState(phase: WorkspacePhase.ready, hierarchy: hierarchy),
      );
    } on Object catch (error) {
      if (!_isCurrent(userId, generation)) return;
      _setState(
        WorkspaceViewState(
          phase: WorkspacePhase.failed,
          message: _messageFor(error),
        ),
      );
    }
  }

  Future<void> reload() async {
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (userId == null || generation == null || !_isCurrent(userId, generation))
      return;
    if (_summary == null) {
      final load = _loadForSession(userId, generation);
      _loadFuture = load;
      await load;
      return;
    }
    await _reloadHierarchy(
      userId,
      generation,
      preserveUncertainOperation: _state.uncertainOperation,
    );
  }

  Future<bool> _reloadHierarchy(
    String userId,
    int generation, {
    String? messageOnSuccess,
    String? preserveUncertainOperation,
  }) async {
    if (!_isCurrent(userId, generation)) return false;
    final previous = _state;
    _setState(
      WorkspaceViewState(
        phase: WorkspacePhase.loading,
        hierarchy: previous.hierarchy,
        selectedLayerId: previous.selectedLayerId,
        uncertainOperation: preserveUncertainOperation,
        createInFlight: _createInFlight,
        renameInFlight: _renameInFlight,
      ),
    );
    try {
      final hierarchy = await _repository.observe();
      if (!_isCurrent(userId, generation)) return false;
      final summary = _summary;
      if (summary != null) _validateHierarchy(summary, hierarchy);
      final selectedId =
          hierarchy.layerById(previous.selectedLayerId ?? '') == null
          ? null
          : previous.selectedLayerId;
      final previousConflict = previous.renameConflict;
      final refreshedConflict = previousConflict == null
          ? null
          : RenameConflict(
              layerId: previousConflict.layerId,
              draft: previousConflict.draft,
              latestLayer: hierarchy.layerById(previousConflict.layerId),
            );
      _setState(
        WorkspaceViewState(
          phase: WorkspacePhase.ready,
          hierarchy: hierarchy,
          selectedLayerId: selectedId,
          message: messageOnSuccess,
          uncertainOperation: preserveUncertainOperation,
          renameConflict: refreshedConflict,
          createInFlight: _createInFlight,
          renameInFlight: _renameInFlight,
        ),
      );
      return true;
    } on Object catch (error) {
      if (!_isCurrent(userId, generation)) return false;
      _setState(
        WorkspaceViewState(
          phase: WorkspacePhase.failed,
          hierarchy: previous.hierarchy,
          selectedLayerId: previous.selectedLayerId,
          message: messageOnSuccess == null
              ? _messageFor(error)
              : '$messageOnSuccess ${_messageFor(error)}',
          uncertainOperation: preserveUncertainOperation,
          renameConflict: previous.renameConflict,
        ),
      );
      return false;
    }
  }

  Future<MutationResult> createLayer({
    required BrainParent parent,
    required String name,
  }) async {
    if (_createInFlight) {
      return const MutationResult(MutationStatus.alreadyInFlight);
    }
    final hierarchy = _state.hierarchy;
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (hierarchy == null || userId == null || generation == null) {
      return const MutationResult(
        MutationStatus.rejected,
        message: 'A hierarquia ainda não está carregada.',
      );
    }
    if (!_isParentInHierarchy(parent, hierarchy)) {
      return const MutationResult(
        MutationStatus.rejected,
        message: 'A camada-pai não pertence à hierarquia atual.',
      );
    }

    _createInFlight = true;
    _setState(
      WorkspaceViewState(
        phase: _state.phase,
        hierarchy: hierarchy,
        selectedLayerId: _state.selectedLayerId,
        message: null,
        uncertainOperation: null,
        renameConflict: _state.renameConflict,
        createInFlight: true,
        renameInFlight: _renameInFlight,
      ),
    );
    try {
      await _repository.createLayer(parent: parent, name: name);
      if (!_isCurrent(userId, generation)) {
        return const MutationResult(MutationStatus.rejected);
      }
      await _reloadHierarchy(
        userId,
        generation,
        messageOnSuccess: 'A camada foi criada.',
      );
      return const MutationResult(MutationStatus.success);
    } on Object catch (error) {
      if (!_isCurrent(userId, generation)) {
        return const MutationResult(MutationStatus.rejected);
      }
      if (_isUncertainMutation(error)) {
        final notice =
            'Resultado incerto ao criar “$name”. Confirme na árvore antes de tentar novamente.';
        await _reloadHierarchy(
          userId,
          generation,
          preserveUncertainOperation: notice,
        );
        return MutationResult(MutationStatus.uncertain, message: notice);
      }
      final message = _messageFor(error);
      _setState(
        WorkspaceViewState(
          phase: WorkspacePhase.ready,
          hierarchy: hierarchy,
          selectedLayerId: _state.selectedLayerId,
          message: message,
          uncertainOperation: _state.uncertainOperation,
          renameConflict: _state.renameConflict,
        ),
      );
      return MutationResult(MutationStatus.rejected, message: message);
    } finally {
      _createInFlight = false;
      if (_isCurrent(userId, generation)) {
        _setState(
          WorkspaceViewState(
            phase: _state.phase,
            hierarchy: _state.hierarchy,
            selectedLayerId: _state.selectedLayerId,
            message: _state.message,
            uncertainOperation: _state.uncertainOperation,
            renameConflict: _state.renameConflict,
            createInFlight: false,
            renameInFlight: _renameInFlight,
          ),
        );
      }
    }
  }

  Future<MutationResult> renameLayer({
    required String layerId,
    required String draft,
    required BigInt expectedVersion,
  }) async {
    if (_renameInFlight) {
      return const MutationResult(MutationStatus.alreadyInFlight);
    }
    final hierarchy = _state.hierarchy;
    final userId = _activeUserId;
    final generation = _activeGeneration;
    if (hierarchy == null || userId == null || generation == null) {
      return const MutationResult(
        MutationStatus.rejected,
        message: 'A hierarquia ainda não está carregada.',
      );
    }
    if (hierarchy.layerById(layerId) == null) {
      return const MutationResult(
        MutationStatus.rejected,
        message: 'A camada não existe na hierarquia atual.',
      );
    }
    _renameInFlight = true;
    _setState(
      WorkspaceViewState(
        phase: _state.phase,
        hierarchy: hierarchy,
        selectedLayerId: _state.selectedLayerId,
        uncertainOperation: null,
        createInFlight: _createInFlight,
        renameInFlight: true,
      ),
    );
    try {
      await _repository.renameLayer(
        layerId: layerId,
        name: draft,
        expectedVersion: expectedVersion,
      );
      if (!_isCurrent(userId, generation)) {
        return const MutationResult(MutationStatus.rejected);
      }
      await _reloadHierarchy(
        userId,
        generation,
        messageOnSuccess: 'A camada foi renomeada.',
      );
      return const MutationResult(MutationStatus.success);
    } on BrainApiException catch (error) {
      if (!_isCurrent(userId, generation)) {
        return const MutationResult(MutationStatus.rejected);
      }
      if (error.statusCode == 409) {
        final refreshed = await _reloadHierarchy(userId, generation);
        final latest = refreshed ? _state.hierarchy?.layerById(layerId) : null;
        final resolvedConflict = RenameConflict(
          layerId: layerId,
          draft: draft,
          latestLayer: latest,
        );
        _setState(
          WorkspaceViewState(
            phase: _state.phase,
            hierarchy: _state.hierarchy,
            selectedLayerId: _state.selectedLayerId,
            message: latest == null
                ? 'A camada mudou e não foi encontrada após atualizar. O rascunho foi mantido.'
                : 'A camada mudou. Confira a versão atual antes de reaplicar o rascunho.',
            uncertainOperation: _state.uncertainOperation,
            renameConflict: resolvedConflict,
            createInFlight: _createInFlight,
            renameInFlight: true,
          ),
        );
        return MutationResult(MutationStatus.conflict, message: error.message);
      }
      if (_isUncertainMutation(error)) {
        final notice =
            'Resultado incerto ao renomear. A árvore será recarregada antes de qualquer nova tentativa.';
        final refreshed = await _reloadHierarchy(
          userId,
          generation,
          preserveUncertainOperation: notice,
        );
        final latest = refreshed ? _state.hierarchy?.layerById(layerId) : null;
        _setState(
          WorkspaceViewState(
            phase: _state.phase,
            hierarchy: _state.hierarchy,
            selectedLayerId: _state.selectedLayerId,
            message:
                'Resultado incerto. Confira a versão atual antes de reaplicar o rascunho.',
            uncertainOperation: notice,
            renameConflict: RenameConflict(
              layerId: layerId,
              draft: draft,
              latestLayer: latest,
            ),
            createInFlight: _createInFlight,
            renameInFlight: true,
          ),
        );
        return MutationResult(MutationStatus.uncertain, message: notice);
      }
      final message = _messageFor(error);
      _setState(
        WorkspaceViewState(
          phase: WorkspacePhase.ready,
          hierarchy: hierarchy,
          selectedLayerId: _state.selectedLayerId,
          message: message,
          uncertainOperation: _state.uncertainOperation,
          renameConflict: _state.renameConflict,
        ),
      );
      return MutationResult(MutationStatus.rejected, message: message);
    } on Object catch (error) {
      if (!_isCurrent(userId, generation)) {
        return const MutationResult(MutationStatus.rejected);
      }
      if (_isUncertainMutation(error)) {
        final notice =
            'Resultado incerto ao renomear. A árvore será recarregada antes de qualquer nova tentativa.';
        final refreshed = await _reloadHierarchy(
          userId,
          generation,
          preserveUncertainOperation: notice,
        );
        final latest = refreshed ? _state.hierarchy?.layerById(layerId) : null;
        _setState(
          WorkspaceViewState(
            phase: _state.phase,
            hierarchy: _state.hierarchy,
            selectedLayerId: _state.selectedLayerId,
            message:
                'Resultado incerto. Confira a versão atual antes de reaplicar o rascunho.',
            uncertainOperation: notice,
            renameConflict: RenameConflict(
              layerId: layerId,
              draft: draft,
              latestLayer: latest,
            ),
            createInFlight: _createInFlight,
            renameInFlight: true,
          ),
        );
        return MutationResult(MutationStatus.uncertain, message: notice);
      }
      final message = _messageFor(error);
      _setState(
        WorkspaceViewState(
          phase: WorkspacePhase.ready,
          hierarchy: hierarchy,
          selectedLayerId: _state.selectedLayerId,
          message: message,
          uncertainOperation: _state.uncertainOperation,
          renameConflict: _state.renameConflict,
        ),
      );
      return MutationResult(MutationStatus.rejected, message: message);
    } finally {
      _renameInFlight = false;
      if (_isCurrent(userId, generation)) {
        _setState(
          WorkspaceViewState(
            phase: _state.phase,
            hierarchy: _state.hierarchy,
            selectedLayerId: _state.selectedLayerId,
            message: _state.message,
            uncertainOperation: _state.uncertainOperation,
            renameConflict: _state.renameConflict,
            createInFlight: _createInFlight,
            renameInFlight: false,
          ),
        );
      }
    }
  }

  Future<MutationResult> retryRenameAfterConflict() async {
    final conflict = _state.renameConflict;
    final latest = conflict?.latestLayer;
    if (conflict == null || latest == null) {
      return const MutationResult(
        MutationStatus.rejected,
        message: 'Atualize a árvore antes de reaplicar o rascunho.',
      );
    }
    return renameLayer(
      layerId: conflict.layerId,
      draft: conflict.draft,
      expectedVersion: latest.version,
    );
  }

  void selectLayer(String? layerId) {
    if (layerId != null && _state.hierarchy?.layerById(layerId) == null) return;
    _setState(
      WorkspaceViewState(
        phase: _state.phase,
        hierarchy: _state.hierarchy,
        selectedLayerId: layerId,
        message: _state.message,
        uncertainOperation: _state.uncertainOperation,
        renameConflict: _state.renameConflict,
        createInFlight: _createInFlight,
        renameInFlight: _renameInFlight,
      ),
    );
  }

  bool _isCurrent(String userId, int generation) =>
      !_disposed &&
      _activeUserId == userId &&
      _activeGeneration == generation &&
      _auth.state.phase == AuthPhase.authenticated &&
      _auth.state.userId == userId &&
      _auth.state.generation == generation;

  static bool _isParentInHierarchy(
    BrainParent parent,
    BrainHierarchy hierarchy,
  ) => parent.kind == ParentKind.root
      ? parent.id == hierarchy.rootId
      : hierarchy.layerById(parent.id) != null;

  static bool _isUncertainMutation(Object error) =>
      error is BrainTransportException ||
      error is BrainResponseException ||
      (error is BrainApiException && error.statusCode >= 500);

  static void _validateHierarchy(
    BrainSummary summary,
    BrainHierarchy hierarchy,
  ) {
    if (summary.id != hierarchy.brain.id ||
        summary.rootId != hierarchy.rootId) {
      throw const BrainResponseException();
    }
  }

  static String _messageFor(Object error) => switch (error) {
    BrainApiException() => error.message,
    BrainTransportException() =>
      'Não foi possível alcançar o Brain Engine. Verifique a conexão e tente novamente.',
    BrainResponseException() =>
      'O Brain Engine retornou uma resposta inválida.',
    StaleAuthOperationException() =>
      'A sessão mudou durante a solicitação. Carregue a árvore novamente.',
    UnauthenticatedException() => 'Entre novamente para continuar.',
    _ => 'Não foi possível concluir a solicitação. Tente novamente.',
  };

  void _setState(WorkspaceViewState state) {
    if (_disposed) return;
    _state = state;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _auth.removeListener(_handleAuthChanged);
    super.dispose();
  }
}
