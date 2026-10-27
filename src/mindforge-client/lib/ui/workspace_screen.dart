import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../auth/auth_gateway.dart';
import '../brain/brain_hierarchy.dart';
import '../brain/brain_workspace_controller.dart';
import '../documents/document_models.dart';
import '../documents/documents_controller.dart';
import 'workspace/layouts/workspace_compact_layout.dart';
import 'workspace/layouts/workspace_expanded_layout.dart';
import 'workspace/widgets/document_panes.dart';
import 'workspace/widgets/document_title_dialog.dart';
import 'workspace/widgets/layer_tree.dart';

class WorkspaceScreen extends StatefulWidget {
  const WorkspaceScreen({
    required this.auth,
    required this.workspace,
    this.documents,
    super.key,
  });

  final AuthGateway auth;
  final BrainWorkspaceController workspace;
  final DocumentsController? documents;

  @override
  State<WorkspaceScreen> createState() => _WorkspaceScreenState();
}

class _WorkspaceScreenState extends State<WorkspaceScreen> {
  final Set<String> _expandedLayerIds = <String>{};
  final Map<String, TextEditingController> _documentEditors =
      <String, TextEditingController>{};
  final Map<String, FocusNode> _documentFocusNodes = <String, FocusNode>{};
  final Map<String, int> _editorAppliedRevisions = <String, int>{};
  final Map<String, String> _editorAppliedValues = <String, String>{};
  final Map<String, bool> _editorAppliedContentLoaded = <String, bool>{};
  final Map<String, bool> _previewByDocumentId = <String, bool>{};
  int _compactPane = 0;
  bool _linkPickerOpen = false;

  @override
  Widget build(BuildContext context) {
    final displayName = widget.auth.state.displayName;
    return PopScope<void>(
      canPop: !_isCompactWidth(context) || _compactPane == 0,
      onPopInvokedWithResult: (didPop, result) {
        if (!didPop && _isCompactWidth(context) && _compactPane > 0) {
          _handleCompactBack();
        }
      },
      child: Scaffold(
        appBar: AppBar(
          title: Row(
            children: <Widget>[
              const Flexible(
                child: Text('Meu Cérebro', overflow: TextOverflow.ellipsis),
              ),
              if (MediaQuery.sizeOf(context).width >= 520 &&
                  displayName != null)
                Flexible(
                  child: Padding(
                    padding: const EdgeInsets.only(left: 12),
                    child: Text(
                      displayName,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.labelLarge,
                    ),
                  ),
                ),
            ],
          ),
          actions: <Widget>[
            IconButton(
              tooltip: 'Recarregar árvore',
              onPressed: () => unawaited(widget.workspace.reload()),
              icon: const Icon(Icons.refresh),
            ),
            IconButton(
              tooltip: 'Sair',
              onPressed: () => unawaited(_logout()),
              icon: const Icon(Icons.logout),
            ),
            const SizedBox(width: 8),
          ],
        ),
        body: _buildBody(),
      ),
    );
  }

  Widget _buildBody() {
    final documents = widget.documents;
    final listenable = documents == null
        ? widget.workspace
        : Listenable.merge(<Listenable>[widget.workspace, documents]);
    return AnimatedBuilder(
      animation: listenable,
      builder: (context, _) => _buildWorkspace(context, widget.workspace.state),
    );
  }

  @override
  void dispose() {
    for (final controller in _documentEditors.values) {
      controller.dispose();
    }
    for (final focusNode in _documentFocusNodes.values) {
      focusNode.dispose();
    }
    super.dispose();
  }

  Widget _buildWorkspace(BuildContext context, WorkspaceViewState state) {
    final hierarchy = state.hierarchy;
    if (hierarchy == null) {
      if (state.phase == WorkspacePhase.loading) {
        return const Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              CircularProgressIndicator(),
              SizedBox(height: 16),
              Text('Abrindo seu Cérebro…'),
            ],
          ),
        );
      }
      return Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                const Icon(Icons.cloud_off_outlined, size: 42),
                const SizedBox(height: 16),
                Text(
                  state.message ?? 'O Cérebro não pôde ser carregado.',
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 16),
                FilledButton.icon(
                  onPressed: () => unawaited(widget.workspace.reload()),
                  icon: const Icon(Icons.refresh),
                  label: const Text('Tentar novamente'),
                ),
              ],
            ),
          ),
        ),
      );
    }

    final selectedLayer = state.selectedLayerId == null
        ? null
        : hierarchy.layerById(state.selectedLayerId!);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 1320),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
          child: LayoutBuilder(
            builder: (context, constraints) {
              final tree = WorkspaceLayerTree(
                hierarchy: hierarchy,
                selectedLayerId: state.selectedLayerId,
                expandedLayerIds: _expandedLayerIds,
                onSelect: _selectLayer,
                onToggleExpanded: _toggleExpanded,
                onCreateChild: (layer) => _showCreateDialog(
                  parent: BrainParent(kind: ParentKind.layer, id: layer.id),
                  parentName: layer.name,
                ),
                onRename: _showRenameDialog,
              );
              final details = WorkspaceLayerDetails(
                layer: selectedLayer,
                onCreateChild: selectedLayer == null
                    ? null
                    : () => _showCreateDialog(
                        parent: BrainParent(
                          kind: ParentKind.layer,
                          id: selectedLayer.id,
                        ),
                        parentName: selectedLayer.name,
                      ),
                onRename: selectedLayer == null
                    ? null
                    : () => _showRenameDialog(selectedLayer),
              );
              final empty = hierarchy.topLevelLayers.isEmpty
                  ? WorkspaceEmptyState(
                      onCreate: () => _showCreateDialog(
                        parent: BrainParent(
                          kind: ParentKind.root,
                          id: hierarchy.rootId,
                        ),
                        parentName: 'raiz',
                      ),
                    )
                  : null;
              final documents = widget.documents;
              final documentState = documents?.state;
              final documentsPane = documents == null
                  ? details
                  : WorkspaceDocumentListPane(
                      controller: documents,
                      hierarchy: hierarchy,
                      onCreate: () => unawaited(_showCreateDocumentDialog()),
                      onOpen: _openDocument,
                      onRename: _renameDocument,
                      onMove: _moveDocument,
                      onTrash: _trashDocument,
                      onRestore: _restoreDocument,
                      onReload: _reloadDocuments,
                      onSync: _syncDocuments,
                    );
              final selectedDocument = documentState?.selected;
              final editorPane = documents == null
                  ? details
                  : WorkspaceDocumentEditorPane(
                      controller: documents,
                      document: selectedDocument,
                      titleController: selectedDocument == null
                          ? null
                          : _editorController(selectedDocument, title: true),
                      markdownController: selectedDocument == null
                          ? null
                          : _editorController(selectedDocument),
                      markdownFocusNode: selectedDocument == null
                          ? null
                          : _editorFocusNode(selectedDocument),
                      preview: selectedDocument == null
                          ? false
                          : (_previewByDocumentId[selectedDocument.id] ??
                                false),
                      onPreviewChanged: selectedDocument == null
                          ? (_) {}
                          : (preview) => setState(
                              () => _previewByDocumentId[selectedDocument.id] =
                                  preview,
                            ),
                      onOpenDocument: _openDocument,
                      onLinkTrigger: _maybeShowLinkPicker,
                      onResolveConflict: selectedDocument == null
                          ? (_) {}
                          : (useLocal) => unawaited(
                              documents.resolveConflict(
                                selectedDocument.id,
                                useLocal: useLocal,
                              ),
                            ),
                    );
              final isCompact = _isCompactWidth(context);
              final content = Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  if (state.phase == WorkspacePhase.loading)
                    const LinearProgressIndicator(minHeight: 2),
                  if (state.message != null)
                    WorkspaceNoticeBanner(message: state.message!),
                  if (state.uncertainOperation != null)
                    WorkspaceNoticeBanner(
                      message: state.uncertainOperation!,
                      icon: Icons.help_outline,
                      action: TextButton(
                        onPressed: () => unawaited(widget.workspace.reload()),
                        child: const Text('Recarregar árvore'),
                      ),
                    ),
                  const SizedBox(height: 12),
                  _WorkspaceHeader(
                    state: state,
                    onCreate: () => _showCreateDialog(
                      parent: BrainParent(
                        kind: ParentKind.root,
                        id: hierarchy.rootId,
                      ),
                      parentName: 'raiz',
                    ),
                  ),
                  const SizedBox(height: 16),
                  Expanded(
                    child:
                        empty ??
                        (isCompact
                            ? WorkspaceCompactLayout(
                                pane: documents == null ? 0 : _compactPane,
                                layers: tree,
                                documents: documentsPane,
                                editor: editorPane,
                                onBack: _compactPane == 0
                                    ? null
                                    : _handleCompactBack,
                                paneTitle: _compactPane == 1
                                    ? 'Documentos'
                                    : _compactPane == 2
                                    ? documentState?.selected?.title ?? 'Editor'
                                    : null,
                              )
                            : WorkspaceExpandedLayout(
                                layers: tree,
                                documents: documentsPane,
                                editor: editorPane,
                                includeDocuments: documents != null,
                              )),
                  ),
                ],
              );
              return content;
            },
          ),
        ),
      ),
    );
  }

  void _selectLayer(String? layerId) {
    _clearEditorFocus();
    widget.workspace.selectLayer(layerId);
    final documents = widget.documents;
    if (documents == null || layerId == null) return;
    setState(() => _compactPane = 1);
    unawaited(documents.loadLayer(layerId));
  }

  TextEditingController _editorController(
    LocalDocument document, {
    bool title = false,
  }) {
    final key = '${document.id}:${title ? 'title' : 'markdown'}';
    final value = title ? document.title : document.markdown;
    final existing = _documentEditors[key];
    if (existing != null) {
      final previousValue = _editorAppliedValues[key] ?? existing.text;
      final previousRevision =
          _editorAppliedRevisions[key] ?? document.localRevision;
      final previousContentLoaded =
          _editorAppliedContentLoaded[key] ?? document.contentLoaded;
      final modelChanged =
          previousRevision != document.localRevision ||
          previousContentLoaded != document.contentLoaded ||
          previousValue != value;
      final controllerUntouched = existing.text == previousValue;
      final bodyLoaded = title || document.contentLoaded;
      if (modelChanged && controllerUntouched && bodyLoaded) {
        existing.value = existing.value.copyWith(
          text: value,
          selection: TextSelection.collapsed(offset: value.length),
          composing: TextRange.empty,
        );
      }
      _editorAppliedRevisions[key] = document.localRevision;
      _editorAppliedValues[key] = existing.text;
      _editorAppliedContentLoaded[key] = document.contentLoaded;
      return existing;
    }
    final controller = TextEditingController(text: value);
    _documentEditors[key] = controller;
    _editorAppliedRevisions[key] = document.localRevision;
    _editorAppliedValues[key] = value;
    _editorAppliedContentLoaded[key] = document.contentLoaded;
    return controller;
  }

  FocusNode _editorFocusNode(LocalDocument document) =>
      _documentFocusNodes.putIfAbsent(document.id, FocusNode.new);

  void _openDocument(String documentId) {
    final documents = widget.documents;
    if (documents == null) return;
    _clearEditorFocus();
    if (_isCompactWidth(context)) {
      setState(() => _compactPane = 2);
    }
    unawaited(documents.openDocument(documentId));
  }

  void _handleCompactBack() {
    if (!_isCompactWidth(context) || _compactPane == 0) return;
    _clearEditorFocus();
    setState(() => _compactPane--);
  }

  void _clearEditorFocus() {
    FocusManager.instance.primaryFocus?.unfocus();
    unawaited(SystemChannels.textInput.invokeMethod<void>('TextInput.hide'));
  }

  void _maybeShowLinkPicker(String markdown) {
    final documents = widget.documents;
    final selected = documents?.state.selected;
    if (documents == null || selected == null || _linkPickerOpen) return;
    final match = RegExp(r'(?<!\\)\[\[([^\]\n]*)$').firstMatch(markdown);
    if (match == null) return;
    final query = match.group(1)!.trim().toLowerCase();
    _linkPickerOpen = true;
    unawaited(
      _showLinkPicker(
        query: query,
        document: selected,
        controller: documents,
      ).whenComplete(() => _linkPickerOpen = false),
    );
  }

  Future<void> _showLinkPicker({
    required String query,
    required LocalDocument document,
    required DocumentsController controller,
  }) async {
    final hierarchy = widget.workspace.state.hierarchy;
    if (hierarchy == null || !mounted) return;
    final candidates = controller.state.items
        .where(
          (item) =>
              item.id != document.id &&
              (query.isEmpty || item.title.toLowerCase().contains(query)),
        )
        .toList(growable: false);
    final targetId = await showDialog<String>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Escolha o destino do link'),
        content: SizedBox(
          width: 420,
          child: candidates.isEmpty
              ? const Text('Nenhum documento corresponde ao título digitado.')
              : ListView.builder(
                  shrinkWrap: true,
                  itemCount: candidates.length,
                  itemBuilder: (context, index) {
                    final candidate = candidates[index];
                    final origin = hierarchy.layerById(candidate.layerId);
                    return ListTile(
                      key: ValueKey<String>('link-target-${candidate.id}'),
                      title: Text(candidate.title),
                      subtitle: Text(
                        origin == null
                            ? 'Camada indisponível'
                            : '${origin.name}${candidate.isTrash ? ' · na lixeira' : ''}',
                      ),
                      onTap: () => Navigator.pop(dialogContext, candidate.id),
                    );
                  },
                ),
        ),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Cancelar'),
          ),
        ],
      ),
    );
    if (!mounted || targetId == null) return;
    final markdown = _documentEditors['${document.id}:markdown'];
    if (markdown == null) return;
    final value = markdown.text;
    final match = RegExp(r'(?<!\\)\[\[([^\]\n]*)$').firstMatch(value);
    if (match == null) return;
    final replacement = '[[doc:$targetId]]';
    markdown.value = markdown.value.copyWith(
      text: value.replaceRange(match.start, value.length, replacement),
      selection: TextSelection.collapsed(
        offset: match.start + replacement.length,
      ),
      composing: TextRange.empty,
    );
    await controller.editDocument(markdown: markdown.text);
  }

  Future<void> _showCreateDocumentDialog() async {
    final documents = widget.documents;
    if (documents == null) return;
    final layerId =
        documents.state.layerId ?? widget.workspace.state.selectedLayerId;
    if (layerId == null) return;
    final created = await showDialog<bool>(
      context: context,
      builder: (_) => WorkspaceDocumentTitleDialog(
        title: 'Novo documento',
        initialTitle: 'Sem título',
        confirmLabel: 'Criar',
        failureMessage: 'Não foi possível criar o documento localmente.',
        onSubmit: (title) async {
          await documents.createDocument(layerId, title: title);
        },
      ),
    );
    if (created == true && mounted && _isCompactWidth(context)) {
      setState(() => _compactPane = 2);
    }
  }

  Future<void> _renameDocument(String documentId) async {
    final documents = widget.documents;
    if (documents == null) return;
    final document = documents.state.items.cast<LocalDocument?>().firstWhere(
      (item) => item?.id == documentId,
      orElse: () => null,
    );
    if (document == null) return;
    await showDialog<bool>(
      context: context,
      builder: (_) => WorkspaceDocumentTitleDialog(
        title: 'Renomear documento',
        initialTitle: document.title,
        confirmLabel: 'Salvar',
        failureMessage: 'Não foi possível renomear o documento localmente.',
        onSubmit: (title) async {
          await documents.openDocument(documentId);
          await documents.editDocument(title: title);
        },
      ),
    );
  }

  Future<void> _moveDocument(String documentId) async {
    final documents = widget.documents;
    final hierarchy = widget.workspace.state.hierarchy;
    if (documents == null || hierarchy == null) return;
    final targetLayerId = await showDialog<String>(
      context: context,
      builder: (dialogContext) => SimpleDialog(
        title: const Text('Mover para a camada'),
        children: <Widget>[
          for (final layer in _flattenLayers(hierarchy.topLevelLayers))
            SimpleDialogOption(
              onPressed: () => Navigator.pop(dialogContext, layer.id),
              child: Text(layer.name),
            ),
        ],
      ),
    );
    if (targetLayerId != null) {
      await documents.moveDocument(documentId, targetLayerId);
    }
  }

  void _trashDocument(String documentId) {
    final documents = widget.documents;
    if (documents != null) unawaited(documents.trashDocument(documentId));
  }

  void _restoreDocument(String documentId) {
    final documents = widget.documents;
    if (documents == null) return;
    final hierarchy = widget.workspace.state.hierarchy;
    final document = documents.state.items.cast<LocalDocument?>().firstWhere(
      (item) => item?.id == documentId,
      orElse: () => null,
    );
    if (document == null) return;
    if (hierarchy?.layerById(document.layerId) != null) {
      unawaited(documents.restoreDocument(documentId));
      return;
    }
    unawaited(_chooseRestoreLayer(documents, documentId));
  }

  Future<void> _chooseRestoreLayer(
    DocumentsController documents,
    String documentId,
  ) async {
    final hierarchy = widget.workspace.state.hierarchy;
    if (hierarchy == null || !mounted) return;
    final layerId = await showDialog<String>(
      context: context,
      builder: (dialogContext) => SimpleDialog(
        title: const Text('Escolha a camada para restaurar'),
        children: <Widget>[
          for (final layer in _flattenLayers(hierarchy.topLevelLayers))
            SimpleDialogOption(
              onPressed: () => Navigator.pop(dialogContext, layer.id),
              child: Text(layer.name),
            ),
        ],
      ),
    );
    if (layerId != null) {
      await documents.restoreDocument(documentId, layerId: layerId);
    }
  }

  void _reloadDocuments() {
    final documents = widget.documents;
    final layerId = documents?.state.layerId;
    if (documents == null || layerId == null) return;
    unawaited(
      documents.loadLayer(
        layerId,
        subtree: documents.state.subtree,
        trash: documents.state.trash,
      ),
    );
  }

  void _syncDocuments() {
    final documents = widget.documents;
    if (documents != null) unawaited(documents.sync());
  }

  Future<void> _logout() async {
    final documents = widget.documents;
    if (documents == null || await documents.prepareLogout()) {
      await widget.auth.signOut();
      return;
    }
    if (!mounted) return;
    final leave = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Sair com pendências'),
        content: Text(
          widget.documents?.state.message ??
              'O estado local permanece no dispositivo, mas o envio ao servidor ainda não foi confirmado. Você pode cancelar ou sair conscientemente.',
        ),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Cancelar'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text('Sair mesmo assim'),
          ),
        ],
      ),
    );
    if (leave == true && mounted) await widget.auth.signOut();
  }

  static Iterable<BrainLayer> _flattenLayers(
    Iterable<BrainLayer> layers,
  ) sync* {
    for (final layer in layers) {
      yield layer;
      yield* _flattenLayers(layer.children);
    }
  }

  static bool _isCompactWidth(BuildContext context) =>
      MediaQuery.sizeOf(context).width < 860;

  void _toggleExpanded(String layerId) {
    setState(() {
      if (!_expandedLayerIds.add(layerId)) _expandedLayerIds.remove(layerId);
    });
  }

  Future<void> _showCreateDialog({
    required BrainParent parent,
    required String parentName,
  }) async {
    final textController = TextEditingController();
    String? errorMessage;
    bool submitting = false;
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          Future<void> submit() async {
            if (textController.text.trim().isEmpty) {
              setDialogState(() => errorMessage = 'Informe um nome.');
              return;
            }
            setDialogState(() {
              submitting = true;
              errorMessage = null;
            });
            final result = await widget.workspace.createLayer(
              parent: parent,
              name: textController.text,
            );
            if (!dialogContext.mounted) return;
            if (result.status == MutationStatus.success ||
                result.status == MutationStatus.uncertain) {
              Navigator.pop(dialogContext);
              return;
            }
            setDialogState(() {
              submitting = false;
              errorMessage =
                  result.message ?? 'Não foi possível criar a camada.';
            });
          }

          return AlertDialog(
            title: Text(
              parent.kind == ParentKind.root
                  ? 'Criar camada'
                  : 'Criar subcamada',
            ),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text('Dentro de $parentName'),
                const SizedBox(height: 14),
                _DialogTextField(
                  controller: textController,
                  autofocus: true,
                  decoration: const InputDecoration(
                    labelText: 'Nome da camada',
                    border: OutlineInputBorder(),
                  ),
                  onSubmitted: (_) => unawaited(submit()),
                ),
                if (errorMessage != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      errorMessage!,
                      style: TextStyle(
                        color: Theme.of(dialogContext).colorScheme.error,
                      ),
                    ),
                  ),
                if (submitting)
                  const Padding(
                    padding: EdgeInsets.only(top: 12),
                    child: LinearProgressIndicator(),
                  ),
              ],
            ),
            actions: <Widget>[
              TextButton(
                onPressed: submitting
                    ? null
                    : () => Navigator.pop(dialogContext),
                child: const Text('Cancelar'),
              ),
              FilledButton(
                onPressed: submitting ? null : () => unawaited(submit()),
                child: const Text('Criar'),
              ),
            ],
          );
        },
      ),
    );
  }

  Future<void> _showRenameDialog(BrainLayer layer) async {
    final textController = TextEditingController(text: layer.name);
    String? errorMessage;
    bool submitting = false;
    bool conflict = false;
    bool uncertain = false;
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          final latestLayer =
              widget.workspace.state.renameConflict?.latestLayer;
          Future<void> save() async {
            if (textController.text.trim().isEmpty) {
              setDialogState(() => errorMessage = 'Informe um nome.');
              return;
            }
            setDialogState(() {
              submitting = true;
              errorMessage = null;
            });
            final result = await widget.workspace.renameLayer(
              layerId: layer.id,
              draft: textController.text,
              expectedVersion: layer.version,
            );
            if (!dialogContext.mounted) return;
            if (result.status == MutationStatus.success) {
              Navigator.pop(dialogContext);
              return;
            }
            setDialogState(() {
              submitting = false;
              conflict =
                  result.status == MutationStatus.conflict ||
                  result.status == MutationStatus.uncertain;
              uncertain = result.status == MutationStatus.uncertain;
              errorMessage = result.status == MutationStatus.uncertain
                  ? 'O resultado é incerto. Confira a árvore antes de tentar novamente.'
                  : (result.status == MutationStatus.conflict
                        ? null
                        : result.message ??
                              'Não foi possível renomear a camada.');
            });
          }

          Future<void> reapply() async {
            setDialogState(() {
              submitting = true;
              errorMessage = null;
            });
            final result = await widget.workspace.retryRenameAfterConflict();
            if (!dialogContext.mounted) return;
            if (result.status == MutationStatus.success) {
              Navigator.pop(dialogContext);
              return;
            }
            setDialogState(() {
              submitting = false;
              conflict =
                  result.status == MutationStatus.conflict ||
                  result.status == MutationStatus.uncertain;
              uncertain = result.status == MutationStatus.uncertain;
              errorMessage =
                  result.message ??
                  (result.status == MutationStatus.uncertain
                      ? 'O resultado é incerto. Confira a árvore antes de tentar novamente.'
                      : 'Não foi possível reaplicar o rascunho.');
            });
          }

          return AlertDialog(
            title: const Text('Renomear camada'),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                _DialogTextField(
                  controller: textController,
                  autofocus: true,
                  decoration: const InputDecoration(
                    labelText: 'Nome da camada',
                    border: OutlineInputBorder(),
                  ),
                  onSubmitted: (_) => unawaited(save()),
                ),
                if (conflict)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      latestLayer == null
                          ? (uncertain
                                ? 'Resultado incerto. Atualize a árvore antes de decidir se reaplica o rascunho.'
                                : 'A camada mudou. O rascunho foi preservado; atualize a árvore antes de reaplicar.')
                          : (uncertain
                                ? 'A camada está em “${latestLayer.name}” (versão ${latestLayer.version}). Confira antes de reaplicar.'
                                : 'A camada mudou para “${latestLayer.name}” (versão ${latestLayer.version}). O rascunho foi preservado.'),
                    ),
                  ),
                if (errorMessage != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      errorMessage!,
                      style: TextStyle(
                        color: Theme.of(dialogContext).colorScheme.error,
                      ),
                    ),
                  ),
                if (submitting)
                  const Padding(
                    padding: EdgeInsets.only(top: 12),
                    child: LinearProgressIndicator(),
                  ),
              ],
            ),
            actions: <Widget>[
              if (conflict && latestLayer != null)
                TextButton(
                  onPressed: submitting ? null : () => unawaited(reapply()),
                  child: const Text('Reaplicar rascunho'),
                ),
              if (conflict && latestLayer == null)
                TextButton(
                  onPressed: submitting
                      ? null
                      : () async {
                          await widget.workspace.reload();
                          if (!dialogContext.mounted) return;
                          setDialogState(() {});
                        },
                  child: const Text('Recarregar árvore'),
                ),
              TextButton(
                onPressed: submitting
                    ? null
                    : () => Navigator.pop(dialogContext),
                child: const Text('Cancelar'),
              ),
              FilledButton(
                onPressed: submitting ? null : () => unawaited(save()),
                child: const Text('Salvar'),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _WorkspaceHeader extends StatelessWidget {
  const _WorkspaceHeader({required this.state, required this.onCreate});

  final WorkspaceViewState state;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final compactAction = constraints.maxWidth < 520;
      return Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: <Widget>[
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  'Camadas',
                  style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 5),
                Text(
                  'Organize seu Cérebro em camadas. A raiz estrutural fica oculta.',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
              ],
            ),
          ),
          if (compactAction)
            IconButton.filled(
              key: const Key('create-layer-action'),
              tooltip: 'Nova camada',
              onPressed: state.createInFlight ? null : onCreate,
              icon: const Icon(Icons.add),
            )
          else
            FilledButton.icon(
              key: const Key('create-layer-action'),
              onPressed: state.createInFlight ? null : onCreate,
              icon: const Icon(Icons.add),
              label: const Text('Nova camada'),
            ),
        ],
      );
    },
  );
}

class _DialogTextField extends StatefulWidget {
  const _DialogTextField({
    required this.controller,
    required this.autofocus,
    required this.decoration,
    required this.onSubmitted,
  });

  final TextEditingController controller;
  final bool autofocus;
  final InputDecoration decoration;
  final ValueChanged<String> onSubmitted;

  @override
  State<_DialogTextField> createState() => _DialogTextFieldState();
}

class _DialogTextFieldState extends State<_DialogTextField> {
  @override
  Widget build(BuildContext context) => TextField(
    controller: widget.controller,
    autofocus: widget.autofocus,
    decoration: widget.decoration,
    onSubmitted: widget.onSubmitted,
  );

  @override
  void dispose() {
    widget.controller.dispose();
    super.dispose();
  }
}
