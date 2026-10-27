import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_repository.dart';
import 'package:mindforge_client/brain/brain_workspace_controller.dart';
import 'package:mindforge_client/ui/workspace_screen.dart';

void main() {
  testWidgets('empty Brain offers the first layer and sends it to the Engine', (
    tester,
  ) async {
    final auth = _FakeAuthGateway();
    final repository = _FakeBrainRepository();
    final workspace = BrainWorkspaceController(
      auth: auth,
      repository: repository,
    );
    await workspace.initialize();

    await tester.pumpWidget(
      MaterialApp(
        home: WorkspaceScreen(auth: auth, workspace: workspace),
      ),
    );

    expect(find.text('Seu Cérebro está pronto'), findsOneWidget);
    expect(find.byKey(const Key('create-first-layer')), findsOneWidget);

    await tester.tap(find.byKey(const Key('create-first-layer')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Personal');
    await tester.tap(find.text('Criar'));
    await tester.pumpAndSettle();

    expect(repository.createdNames, <String>['Personal']);
    expect(workspace.state.phase, WorkspacePhase.ready);
    workspace.dispose();
    auth.dispose();
  });
}

class _FakeAuthGateway extends AuthGateway {
  @override
  AuthViewState state = const AuthViewState(
    phase: AuthPhase.authenticated,
    generation: 1,
    userId: 'user-a',
  );

  @override
  Future<AuthCredential> credential() async => const AuthCredential(
    accessToken: 'test-token',
    userId: 'user-a',
    generation: 1,
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

class _FakeBrainRepository implements BrainRepository {
  int observeCalls = 0;
  final List<String> createdNames = <String>[];
  final BrainHierarchy hierarchy = BrainHierarchy.fromJson(<String, Object?>{
    'brain': <String, Object?>{
      'id': 'brain-a',
      'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
    },
    'layers': <Object?>[],
  });

  @override
  Future<BrainSummary> bootstrap() async =>
      BrainSummary(id: hierarchy.brainId, rootId: hierarchy.rootId);

  @override
  Future<BrainHierarchy> observe() async {
    observeCalls++;
    return hierarchy;
  }

  @override
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  }) async {
    createdNames.add(name);
    return BrainLayer.fromEnvelope(<String, Object?>{
      'layer': <String, Object?>{
        'id': 'layer-new',
        'parent': <String, Object?>{'kind': parent.kindName, 'id': parent.id},
        'name': name,
        'version': 1,
      },
    });
  }

  @override
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  }) async => throw UnimplementedError();
}
