import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/app/mindforge_app.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_repository.dart';
import 'package:mindforge_client/brain/brain_workspace_controller.dart';
import 'package:mindforge_client/documents/documents_controller.dart';

void main() {
  testWidgets('app owns document initialization, resume and disposal', (
    tester,
  ) async {
    final auth = _SignedOutAuth();
    final workspace = BrainWorkspaceController(
      auth: auth,
      repository: _UnusedBrain(),
    );
    final documents = _TrackingDocuments();
    await tester.pumpWidget(
      MindforgeApp(
        auth: auth,
        workspace: workspace,
        documents: documents,
        incomingLinks: const Stream<Uri>.empty(),
      ),
    );
    await tester.pumpAndSettle();
    expect(documents.initializations, 1);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();
    expect(documents.resumptions, 1);
    expect(documents.flushes, 1);
    await tester.pumpWidget(const SizedBox());
    expect(documents.disposals, 1);
  });
}

class _TrackingDocuments extends ChangeNotifier implements DocumentsController {
  int initializations = 0, resumptions = 0, disposals = 0, flushes = 0;
  @override
  Future<void> initialize() async {
    initializations++;
  }

  @override
  Future<void> onResume() async {
    resumptions++;
  }

  @override
  Future<void> flushLocal() async {
    flushes++;
  }

  @override
  void dispose() {
    disposals++;
    super.dispose();
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _SignedOutAuth extends AuthGateway {
  @override
  AuthViewState get state =>
      const AuthViewState(phase: AuthPhase.signedOut, generation: 0);
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
  @override
  Future<AuthCredential> credential() async =>
      throw const UnauthenticatedException();
  @override
  Future<AuthCredential> refreshAfterUnauthorized(
    AuthCredential rejected,
  ) async => throw const UnauthenticatedException();
  @override
  Future<void> expireAfterUnauthorized(AuthCredential rejected) async {}
}

class _UnusedBrain implements BrainRepository {
  @override
  Future<BrainSummary> bootstrap() async => throw UnimplementedError();
  @override
  Future<BrainHierarchy> observe() async => throw UnimplementedError();
  @override
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  }) async => throw UnimplementedError();
  @override
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  }) async => throw UnimplementedError();
}
