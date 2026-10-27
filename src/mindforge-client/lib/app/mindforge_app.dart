import 'dart:async';

import 'package:flutter/material.dart';

import '../auth/auth_gateway.dart';
import '../brain/brain_workspace_controller.dart';
import '../documents/documents_controller.dart';
import '../ui/login_screen.dart';
import '../ui/workspace_screen.dart';

class MindforgeApp extends StatefulWidget {
  const MindforgeApp({
    required this.auth,
    required this.workspace,
    required this.incomingLinks,
    this.onDispose,
    this.documents,
    super.key,
  });

  final AuthGateway auth;
  final BrainWorkspaceController workspace;
  final Stream<Uri> incomingLinks;
  final VoidCallback? onDispose;
  final DocumentsController? documents;

  @override
  State<MindforgeApp> createState() => _MindforgeAppState();
}

class _MindforgeAppState extends State<MindforgeApp>
    with WidgetsBindingObserver {
  StreamSubscription<Uri>? _linkSubscription;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _linkSubscription = widget.incomingLinks.listen((uri) {
      unawaited(_handleIncomingLink(uri));
    }, onError: (Object _, StackTrace __) {});
    unawaited(_initialize());
  }

  Future<void> _handleIncomingLink(Uri uri) async {
    try {
      await widget.auth.handleCallback(uri);
    } on Object {
      // Ignore platform callback failures without exposing callback contents.
    }
  }

  Future<void> _initialize() async {
    await widget.workspace.initialize();
    if (mounted) await widget.documents?.initialize();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Mindforge',
      debugShowCheckedModeBanner: false,
      initialRoute: '/',
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF6255A5),
          brightness: Brightness.light,
        ),
        scaffoldBackgroundColor: const Color(0xFFF7F6FA),
        cardTheme: const CardThemeData(elevation: 0, margin: EdgeInsets.zero),
      ),
      home: _AuthRouter(
        auth: widget.auth,
        workspace: widget.workspace,
        documents: widget.documents,
      ),
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      unawaited(widget.documents?.onResume() ?? Future<void>.value());
    }
    if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.inactive) {
      unawaited(
        widget.documents?.flushLocal().catchError((Object _) {}) ??
            Future<void>.value(),
      );
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.documents?.dispose();
    unawaited(_linkSubscription?.cancel());
    widget.workspace.dispose();
    widget.auth.dispose();
    widget.onDispose?.call();
    super.dispose();
  }
}

class _AuthRouter extends StatelessWidget {
  const _AuthRouter({
    required this.auth,
    required this.workspace,
    this.documents,
  });

  final AuthGateway auth;
  final BrainWorkspaceController workspace;
  final DocumentsController? documents;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: auth,
      builder: (context, _) {
        final authState = auth.state;
        if (authState.phase == AuthPhase.authenticated) {
          return WorkspaceScreen(
            auth: auth,
            workspace: workspace,
            documents: documents,
          );
        }
        if (authState.phase == AuthPhase.initializing) {
          return const _LoadingScreen(label: 'Restaurando sua sessão…');
        }
        return LoginScreen(auth: auth);
      },
    );
  }
}

class _LoadingScreen extends StatelessWidget {
  const _LoadingScreen({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) => Scaffold(
    body: Center(
      child: Semantics(
        liveRegion: true,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            const CircularProgressIndicator(),
            const SizedBox(height: 18),
            Text(label),
          ],
        ),
      ),
    ),
  );
}

class ConfigurationErrorApp extends StatelessWidget {
  const ConfigurationErrorApp({super.key});

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Mindforge — configuração',
    home: Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 680),
          child: Padding(
            padding: const EdgeInsets.all(40),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  'Ambiente de desenvolvimento indisponível',
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 12),
                const Text(
                  'O Mindforge não conseguiu iniciar neste ambiente. '
                  'Confira a configuração de desenvolvimento no README do client.',
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}
