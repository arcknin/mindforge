import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_repository.dart';
import 'package:mindforge_client/brain/brain_workspace_controller.dart';

void main() {
  group('BrainWorkspaceController', () {
    test('bootstraps once, then observes the empty hierarchy', () async {
      final auth = _FakeAuthGateway.signedIn('user-a');
      final repository = _FakeBrainRepository();
      final controller = BrainWorkspaceController(
        auth: auth,
        repository: repository,
      );

      await controller.initialize();

      expect(repository.bootstrapCalls, 1);
      expect(repository.observeCalls, 1);
      expect(controller.state.phase, WorkspacePhase.ready);
      expect(controller.state.hierarchy!.topLevelLayers, isEmpty);
      expect(controller.state.selectedLayerId, isNull);
      controller.dispose();
      auth.dispose();
    });

    test('allows only one create request at a time', () async {
      final auth = _FakeAuthGateway.signedIn('user-a');
      final repository = _FakeBrainRepository();
      final createResult = Completer<BrainLayer>();
      repository.createLayerResult = createResult.future;
      final controller = BrainWorkspaceController(
        auth: auth,
        repository: repository,
      );
      await controller.initialize();

      final first = controller.createLayer(
        parent: const BrainParent(kind: ParentKind.root, id: 'root-a'),
        name: 'Personal',
      );
      final second = await controller.createLayer(
        parent: const BrainParent(kind: ParentKind.root, id: 'root-a'),
        name: 'Personal',
      );

      expect(second.status, MutationStatus.alreadyInFlight);
      expect(repository.createCalls, 1);

      createResult.complete(_layer('new', 'root-a', 'Personal', 1));
      expect((await first).status, MutationStatus.success);
      controller.dispose();
      auth.dispose();
    });

    test(
      'does not repeat a create after timeout and reports uncertain outcome',
      () async {
        final auth = _FakeAuthGateway.signedIn('user-a');
        final repository = _FakeBrainRepository()
          ..createLayerError = const BrainTransportException();
        final controller = BrainWorkspaceController(
          auth: auth,
          repository: repository,
        );
        await controller.initialize();

        final result = await controller.createLayer(
          parent: const BrainParent(kind: ParentKind.root, id: 'root-a'),
          name: 'Personal',
        );

        expect(result.status, MutationStatus.uncertain);
        expect(repository.createCalls, 1);
        expect(repository.observeCalls, 2);
        expect(controller.state.uncertainOperation, contains('Personal'));
        expect(controller.state.phase, WorkspacePhase.ready);
        controller.dispose();
        auth.dispose();
      },
    );

    test(
      'keeps the rename draft on 409 and retries only after explicit choice',
      () async {
        final auth = _FakeAuthGateway.signedIn('user-a');
        final repository = _FakeBrainRepository()
          ..currentHierarchy = _hierarchyWithServerLayer()
          ..renameLayerError = const BrainApiException(
            statusCode: 409,
            code: 'conflict',
            message: 'The layer changed.',
          );
        final controller = BrainWorkspaceController(
          auth: auth,
          repository: repository,
        );
        await controller.initialize();

        final conflict = await controller.renameLayer(
          layerId: 'top',
          draft: 'Local draft',
          expectedVersion: BigInt.one,
        );

        expect(conflict.status, MutationStatus.conflict);
        expect(controller.state.renameConflict!.draft, 'Local draft');
        expect(
          controller.state.renameConflict!.latestLayer!.version,
          BigInt.from(2),
        );
        expect(repository.renameVersions, <BigInt>[BigInt.one]);
        expect(
          controller.state.hierarchy!.layerById('top')!.name,
          'Server name',
        );

        repository.renameLayerError = null;
        final retried = await controller.retryRenameAfterConflict();

        expect(retried.status, MutationStatus.success);
        expect(repository.renameVersions, <BigInt>[BigInt.one, BigInt.from(2)]);
        expect(repository.renameNames, <String>['Local draft', 'Local draft']);
        expect(controller.state.renameConflict, isNull);
        controller.dispose();
        auth.dispose();
      },
    );

    test(
      'clears the old tree immediately and discards its late response on account switch',
      () async {
        final auth = _FakeAuthGateway.signedIn('user-a');
        final repository = _FakeBrainRepository();
        final oldObserve = Completer<BrainHierarchy>();
        repository.bootstrapResults
          ..add(BrainSummary(id: 'brain-A', rootId: 'root-A'))
          ..add(BrainSummary(id: 'brain-B', rootId: 'root-B'));
        repository.observeResults
          ..add(oldObserve.future)
          ..add(Future<BrainHierarchy>.value(_hierarchy(userLabel: 'B')));
        final controller = BrainWorkspaceController(
          auth: auth,
          repository: repository,
        );

        final oldInitialization = controller.initialize();
        await Future<void>.delayed(Duration.zero);
        auth.switchTo('user-b');
        expect(controller.state.hierarchy, isNull);
        expect(controller.state.phase, WorkspacePhase.loading);

        await Future<void>.delayed(Duration.zero);
        await Future<void>.delayed(Duration.zero);
        expect(
          controller.state.hierarchy!.topLevelLayers.single.name,
          'User B',
        );
        oldObserve.complete(_hierarchy(userLabel: 'A'));
        await oldInitialization;
        await Future<void>.delayed(Duration.zero);

        expect(
          controller.state.hierarchy!.topLevelLayers.single.name,
          'User B',
        );
        controller.dispose();
        auth.dispose();
      },
    );
  });
}

BrainHierarchy _hierarchy({String userLabel = 'A'}) =>
    BrainHierarchy.fromJson(<String, Object?>{
      'brain': <String, Object?>{
        'id': 'brain-$userLabel',
        'root': <String, Object?>{'kind': 'root', 'id': 'root-$userLabel'},
      },
      'layers': <Object?>[
        <String, Object?>{
          'id': 'top',
          'parent': <String, Object?>{'kind': 'root', 'id': 'root-$userLabel'},
          'name': 'User $userLabel',
          'version': 1,
        },
      ],
    });

BrainHierarchy _emptyHierarchy() => BrainHierarchy.fromJson(<String, Object?>{
  'brain': <String, Object?>{
    'id': 'brain-a',
    'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
  },
  'layers': <Object?>[],
});

BrainHierarchy _hierarchyWithServerLayer() =>
    BrainHierarchy.fromJson(<String, Object?>{
      'brain': <String, Object?>{
        'id': 'brain-a',
        'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
      },
      'layers': <Object?>[
        <String, Object?>{
          'id': 'top',
          'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
          'name': 'Server name',
          'version': 2,
        },
      ],
    });

BrainLayer _layer(String id, String parentId, String name, Object version) =>
    BrainLayer.fromEnvelope(<String, Object?>{
      'layer': <String, Object?>{
        'id': id,
        'parent': <String, Object?>{'kind': 'root', 'id': parentId},
        'name': name,
        'version': version,
      },
    });

class _FakeAuthGateway extends AuthGateway {
  _FakeAuthGateway.signedIn(String userId)
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
  Future<void> signOut() async {
    state = AuthViewState(
      phase: AuthPhase.signedOut,
      generation: state.generation + 1,
    );
    notifyListeners();
  }
}

class _FakeBrainRepository implements BrainRepository {
  int bootstrapCalls = 0;
  int observeCalls = 0;
  int createCalls = 0;
  final List<Future<BrainHierarchy>> observeResults =
      <Future<BrainHierarchy>>[];
  final List<BrainSummary> bootstrapResults = <BrainSummary>[];
  Object? createLayerError;
  Object? renameLayerError;
  Future<BrainLayer>? createLayerResult;
  final List<BigInt> renameVersions = <BigInt>[];
  final List<String> renameNames = <String>[];
  BrainHierarchy currentHierarchy = _emptyHierarchy();

  @override
  Future<BrainSummary> bootstrap() async {
    bootstrapCalls++;
    if (bootstrapResults.isNotEmpty) return bootstrapResults.removeAt(0);
    return BrainSummary(
      id: currentHierarchy.brainId,
      rootId: currentHierarchy.rootId,
    );
  }

  @override
  Future<BrainHierarchy> observe() async {
    observeCalls++;
    if (observeResults.isNotEmpty) return observeResults.removeAt(0);
    return currentHierarchy;
  }

  @override
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  }) async {
    createCalls++;
    final error = createLayerError;
    if (error != null) throw error;
    return createLayerResult ?? _layer('new', currentHierarchy.rootId, name, 1);
  }

  @override
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  }) async {
    renameVersions.add(expectedVersion);
    renameNames.add(name);
    final error = renameLayerError;
    if (error != null) throw error;
    return _layer(
      layerId,
      currentHierarchy.rootId,
      name,
      expectedVersion + BigInt.one,
    );
  }
}
