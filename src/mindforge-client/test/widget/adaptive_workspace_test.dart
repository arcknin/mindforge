import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_repository.dart';
import 'package:mindforge_client/brain/brain_workspace_controller.dart';
import 'package:mindforge_client/ui/workspace_screen.dart';

void main() {
  testWidgets('uses the compact composition below the workspace breakpoint', (
    tester,
  ) async {
    final fixture = _WorkspaceFixture.withLayer();
    await fixture.mount(tester, const Size(859, 640));

    expect(find.byKey(const Key('workspace-compact-layout')), findsOneWidget);
    expect(find.byKey(const Key('workspace-expanded-layout')), findsNothing);

    fixture.dispose();
  });

  testWidgets('uses the expanded composition at and above the breakpoint', (
    tester,
  ) async {
    for (final width in <double>[860, 861]) {
      final fixture = _WorkspaceFixture.withLayer();
      await fixture.mount(tester, Size(width, 640));

      expect(
        find.byKey(const Key('workspace-expanded-layout')),
        findsOneWidget,
        reason: 'width=$width must select the wide composition',
      );
      expect(find.byKey(const Key('workspace-compact-layout')), findsNothing);

      fixture.dispose();
    }
  });
}

class _WorkspaceFixture {
  _WorkspaceFixture(this.auth, this.workspace);

  final _FakeAuthGateway auth;
  final BrainWorkspaceController workspace;

  factory _WorkspaceFixture.withLayer() {
    final auth = _FakeAuthGateway();
    final repository = _FakeBrainRepository();
    final workspace = BrainWorkspaceController(
      auth: auth,
      repository: repository,
    );
    return _WorkspaceFixture(auth, workspace);
  }

  Future<void> mount(WidgetTester tester, Size size) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });
    await workspace.initialize();
    await tester.pumpWidget(
      MaterialApp(
        home: WorkspaceScreen(auth: auth, workspace: workspace),
      ),
    );
    await tester.pumpAndSettle();
  }

  void dispose() {
    workspace.dispose();
    auth.dispose();
  }
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
  final BrainHierarchy hierarchy = BrainHierarchy.fromJson(<String, Object?>{
    'brain': <String, Object?>{
      'id': 'brain-a',
      'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
    },
    'layers': <Object?>[
      <String, Object?>{
        'id': 'layer-a',
        'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
        'name': 'Projetos',
        'version': 1,
      },
    ],
  });

  @override
  Future<BrainSummary> bootstrap() async =>
      BrainSummary(id: hierarchy.brainId, rootId: hierarchy.rootId);

  @override
  Future<BrainHierarchy> observe() async => hierarchy;

  @override
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  }) async => BrainLayer.fromEnvelope(<String, Object?>{
    'layer': <String, Object?>{
      'id': 'layer-new',
      'parent': <String, Object?>{'kind': parent.kindName, 'id': parent.id},
      'name': name,
      'version': 1,
    },
  });

  @override
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  }) async => BrainLayer.fromEnvelope(<String, Object?>{
    'layer': <String, Object?>{
      'id': layerId,
      'parent': <String, Object?>{'kind': 'root', 'id': hierarchy.rootId},
      'name': name,
      'version': expectedVersion + BigInt.one,
    },
  });
}
