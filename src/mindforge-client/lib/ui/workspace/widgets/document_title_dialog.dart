import 'dart:async';

import 'package:flutter/material.dart';

/// Owns the input for the entire dialog route, including its exit animation.
class WorkspaceDocumentTitleDialog extends StatefulWidget {
  const WorkspaceDocumentTitleDialog({
    required this.title,
    required this.initialTitle,
    required this.confirmLabel,
    required this.failureMessage,
    required this.onSubmit,
    super.key,
  });

  final String title;
  final String initialTitle;
  final String confirmLabel;
  final String failureMessage;
  final Future<void> Function(String title) onSubmit;

  @override
  State<WorkspaceDocumentTitleDialog> createState() =>
      _WorkspaceDocumentTitleDialogState();
}

class _WorkspaceDocumentTitleDialogState
    extends State<WorkspaceDocumentTitleDialog> {
  late final TextEditingController _titleController = TextEditingController(
    text: widget.initialTitle,
  );
  String? _errorMessage;
  bool _submitting = false;
  bool _closing = false;

  @override
  void dispose() {
    _titleController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_submitting || _closing) return;
    final title = _titleController.text.trim();
    if (title.isEmpty) {
      setState(() => _errorMessage = 'Informe um título.');
      return;
    }
    setState(() {
      _submitting = true;
      _errorMessage = null;
    });
    try {
      await widget.onSubmit(title);
    } on Object {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _errorMessage = widget.failureMessage;
      });
      return;
    }
    if (!mounted) return;
    _closing = true;
    Navigator.pop(context, true);
  }

  @override
  Widget build(BuildContext context) => PopScope(
    canPop: !_submitting || _closing,
    child: AlertDialog(
      title: Text(widget.title),
      content: TextField(
        controller: _titleController,
        autofocus: true,
        readOnly: _submitting,
        textInputAction: TextInputAction.done,
        decoration: InputDecoration(
          labelText: 'Título',
          errorText: _errorMessage,
        ),
        onSubmitted: (_) => unawaited(_submit()),
      ),
      actions: <Widget>[
        TextButton(
          onPressed: _submitting
              ? null
              : () {
                  if (_closing) return;
                  _closing = true;
                  Navigator.pop(context, false);
                },
          child: const Text('Cancelar'),
        ),
        FilledButton(
          onPressed: _submitting ? null : () => unawaited(_submit()),
          child: Text(widget.confirmLabel),
        ),
      ],
    ),
  );
}
