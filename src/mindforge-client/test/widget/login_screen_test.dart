import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/ui/login_screen.dart';

void main() {
  testWidgets('offers Google-only entry with no password form', (tester) async {
    final auth = _FakeAuthGateway();
    await tester.pumpWidget(MaterialApp(home: LoginScreen(auth: auth)));

    expect(find.text('Entrar com Google'), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
    expect(find.text('Senha'), findsNothing);

    await tester.tap(find.byKey(const Key('google-sign-in')));
    await tester.pump();
    expect(auth.signInCalls, 1);
    auth.dispose();
  });

  testWidgets(
    'shows a cancel action while OAuth waits for the browser callback',
    (tester) async {
      final auth = _FakeAuthGateway()..setPhase(AuthPhase.signingIn);
      await tester.pumpWidget(MaterialApp(home: LoginScreen(auth: auth)));

      expect(find.text('Cancelar acesso'), findsOneWidget);
      expect(find.text('Entrar com Google'), findsNothing);
      auth.dispose();
    },
  );

  testWidgets('offers retry when secure session restoration is unavailable', (
    tester,
  ) async {
    final auth = _FakeAuthGateway()
      ..setState(
        const AuthViewState(
          phase: AuthPhase.unavailable,
          generation: 2,
          message: 'A autenticação está temporariamente indisponível.',
        ),
      );
    await tester.pumpWidget(MaterialApp(home: LoginScreen(auth: auth)));

    expect(find.byKey(const Key('retry-auth-restore')), findsOneWidget);
    await tester.tap(find.byKey(const Key('retry-auth-restore')));
    await tester.pump();
    expect(auth.initializeCalls, 1);
    auth.dispose();
  });

  testWidgets(
    'remains reachable with keyboard, narrow height, and large text',
    (tester) async {
      final auth = _FakeAuthGateway();
      tester.view.physicalSize = const Size(360, 420);
      tester.view.devicePixelRatio = 1;
      tester.view.viewInsets = const FakeViewPadding(bottom: 180);
      addTearDown(() {
        tester.view.resetPhysicalSize();
        tester.view.resetDevicePixelRatio();
        tester.view.resetViewInsets();
      });

      await tester.pumpWidget(
        MediaQuery(
          data: const MediaQueryData(textScaler: TextScaler.linear(1.8)),
          child: MaterialApp(home: LoginScreen(auth: auth)),
        ),
      );

      expect(find.byKey(const Key('google-sign-in')), findsOneWidget);
      expect(tester.takeException(), isNull);
      auth.dispose();
    },
  );
}

class _FakeAuthGateway extends AuthGateway {
  AuthViewState _state = const AuthViewState(
    phase: AuthPhase.signedOut,
    generation: 0,
  );
  int signInCalls = 0;
  int initializeCalls = 0;

  @override
  AuthViewState get state => _state;

  void setPhase(AuthPhase phase) {
    _state = AuthViewState(phase: phase, generation: _state.generation);
    notifyListeners();
  }

  void setState(AuthViewState state) {
    _state = state;
    notifyListeners();
  }

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
  Future<void> initialize() async {
    initializeCalls++;
  }

  @override
  Future<void> beginGoogleSignIn() async {
    signInCalls++;
  }

  @override
  Future<void> cancelGoogleSignIn() async {}

  @override
  Future<void> handleCallback(Uri uri) async {}

  @override
  Future<void> signOut() async {}
}
