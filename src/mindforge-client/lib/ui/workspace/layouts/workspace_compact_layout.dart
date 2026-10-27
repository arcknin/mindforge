import 'package:flutter/material.dart';

/// Presents the same workspace state as a touch-first, one-pane flow.
///
/// The children own their controllers and are kept mounted in an
/// [IndexedStack], so changing the compact pane does not recreate an editor or
/// discard its draft. The parent owns the selected pane and navigation events.
class WorkspaceCompactLayout extends StatelessWidget {
  const WorkspaceCompactLayout({
    required this.layers,
    required this.documents,
    required this.editor,
    required this.pane,
    this.onBack,
    this.paneTitle,
    super.key,
  });

  final Widget layers;
  final Widget documents;
  final Widget editor;
  final int pane;
  final VoidCallback? onBack;
  final String? paneTitle;

  @override
  Widget build(BuildContext context) {
    final safePane = pane.clamp(0, 2);
    return Container(
      key: const Key('workspace-compact-layout'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          if (safePane > 0)
            Material(
              color: Theme.of(context).colorScheme.surface,
              child: Row(
                children: <Widget>[
                  IconButton(
                    tooltip: 'Voltar',
                    onPressed: onBack,
                    icon: const Icon(Icons.arrow_back),
                  ),
                  Expanded(
                    child: Text(
                      paneTitle ?? (safePane == 1 ? 'Documentos' : 'Editor'),
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                  ),
                ],
              ),
            ),
          Expanded(
            child: IndexedStack(
              index: safePane,
              children: <Widget>[layers, documents, editor],
            ),
          ),
        ],
      ),
    );
  }
}
