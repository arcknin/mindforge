import 'package:flutter/material.dart';

/// Presents the workspace as simultaneous tree, document list, and editor
/// panes. The parent supplies shared state and callbacks; this widget only
/// composes them for the available wide space.
class WorkspaceExpandedLayout extends StatelessWidget {
  const WorkspaceExpandedLayout({
    required this.layers,
    required this.documents,
    required this.editor,
    this.includeDocuments = true,
    super.key,
  });

  final Widget layers;
  final Widget documents;
  final Widget editor;
  final bool includeDocuments;

  @override
  Widget build(BuildContext context) => Container(
    key: const Key('workspace-expanded-layout'),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: includeDocuments
          ? <Widget>[
              Expanded(flex: 3, child: layers),
              const SizedBox(width: 12),
              Expanded(flex: 3, child: documents),
              const SizedBox(width: 12),
              Expanded(flex: 5, child: editor),
            ]
          : <Widget>[
              Expanded(flex: 3, child: layers),
              const SizedBox(width: 18),
              Expanded(flex: 2, child: editor),
            ],
    ),
  );
}
