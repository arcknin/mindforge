import 'dart:async';

import 'package:flutter/material.dart';

import '../auth/auth_gateway.dart';

class LoginScreen extends StatelessWidget {
  const LoginScreen({required this.auth, super.key});

  final AuthGateway auth;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) => SingleChildScrollView(
            key: const Key('login-scroll'),
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: BoxConstraints(
                minHeight: constraints.maxHeight - 48,
              ),
              child: Align(
                alignment: Alignment.topCenter,
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 480),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    child: AnimatedBuilder(
                      animation: auth,
                      builder: (context, _) {
                        final state = auth.state;
                        final waiting = state.phase == AuthPhase.signingIn;
                        final storageUnavailable =
                            state.phase == AuthPhase.unavailable;
                        return Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: <Widget>[
                            const Icon(Icons.psychology_alt_outlined, size: 64),
                            const SizedBox(height: 18),
                            Text(
                              'Mindforge',
                              textAlign: TextAlign.center,
                              style: Theme.of(context).textTheme.headlineLarge
                                  ?.copyWith(fontWeight: FontWeight.w700),
                            ),
                            const SizedBox(height: 8),
                            Text(
                              'Seu Cérebro, organizado por você.',
                              textAlign: TextAlign.center,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            const SizedBox(height: 28),
                            if (waiting) ...<Widget>[
                              const Center(child: CircularProgressIndicator()),
                              const SizedBox(height: 16),
                              Text(
                                state.message ??
                                    'Conclua o acesso no navegador.',
                                textAlign: TextAlign.center,
                                semanticsLabel: state.message,
                              ),
                              const SizedBox(height: 16),
                              OutlinedButton(
                                onPressed: () =>
                                    unawaited(auth.cancelGoogleSignIn()),
                                child: const Text('Cancelar acesso'),
                              ),
                            ] else ...<Widget>[
                              if (storageUnavailable)
                                OutlinedButton.icon(
                                  key: const Key('retry-auth-restore'),
                                  onPressed: () => unawaited(auth.initialize()),
                                  icon: const Icon(Icons.refresh),
                                  label: const Text('Tentar restaurar'),
                                )
                              else
                                FilledButton.icon(
                                  key: const Key('google-sign-in'),
                                  onPressed: () =>
                                      unawaited(auth.beginGoogleSignIn()),
                                  icon: const Icon(Icons.login),
                                  label: const Text('Entrar com Google'),
                                ),
                              const SizedBox(height: 12),
                              Text(
                                'O Google é o único método de entrada disponível.',
                                textAlign: TextAlign.center,
                                style: Theme.of(context).textTheme.bodySmall,
                              ),
                              if (state.message != null) ...<Widget>[
                                const SizedBox(height: 18),
                                _MessageBanner(message: state.message!),
                              ],
                            ],
                          ],
                        );
                      },
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _MessageBanner extends StatelessWidget {
  const _MessageBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) => Semantics(
    liveRegion: true,
    child: Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.errorContainer,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(message),
    ),
  );
}
