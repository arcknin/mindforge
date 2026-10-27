import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/app/mindforge_app.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_repository.dart';
import 'package:mindforge_client/brain/brain_workspace_controller.dart';

void main() {
  testWidgets(
    'contains link stream and callback failures without changing auth state',
    (tester) async {
      final links = StreamController<Uri>();
      final auth = _FakeAuthGateway()
        ..callbackError = StateError('private callback data');
      final workspace = BrainWorkspaceController(
        auth: auth,
        repository: _UnusedBrainRepository(),
      );

      await tester.pumpWidget(
        MindforgeApp(
          auth: auth,
          workspace: workspace,
          incomingLinks: links.stream,
        ),
      );
      await tester.pumpAndSettle();
      expect(
        tester.widget<MaterialApp>(find.byType(MaterialApp).first).initialRoute,
        '/',
      );
      links.add(Uri.parse('mindforge-dev://auth/callback?code=one-time-code'));
      links.addError(StateError('platform callback failure'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(auth.callbackCalls, 1);
      expect(auth.state.phase, AuthPhase.signedOut);
      expect(find.text('Entrar com Google'), findsOneWidget);
      expect(find.textContaining('private callback data'), findsNothing);

      await links.close();
      await tester.pumpWidget(const SizedBox());
    },
  );
}

class _FakeAuthGateway extends AuthGateway {
  @override
  AuthViewState state = const AuthViewState(
    phase: AuthPhase.signedOut,
    generation: 0,
  );
  Object? callbackError;
  int callbackCalls = 0;

  @override
  Future<AuthCredential> credential() async =>
      throw const UnauthenticatedException();

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
  Future<void> handleCallback(Uri uri) async {
    callbackCalls++;
    final error = callbackError;
    if (error != null) throw error;
  }

  @override
  Future<void> signOut() async {}
}

class _UnusedBrainRepository implements BrainRepository {
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
