import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:markdown/markdown.dart' as md;
import 'package:url_launcher/url_launcher.dart';

import '../../../brain/brain_hierarchy.dart';
import '../../../documents/document_models.dart';
import '../../../documents/documents_controller.dart';

class WorkspaceDocumentListPane extends StatelessWidget {
  const WorkspaceDocumentListPane({
    required this.controller,
    required this.hierarchy,
    required this.onCreate,
    required this.onOpen,
    required this.onRename,
    required this.onMove,
    required this.onTrash,
    required this.onRestore,
    required this.onReload,
    required this.onSync,
    super.key,
  });

  final DocumentsController controller;
  final BrainHierarchy hierarchy;
  final VoidCallback onCreate;
  final ValueChanged<String> onOpen;
  final ValueChanged<String> onRename;
  final ValueChanged<String> onMove;
  final ValueChanged<String> onTrash;
  final ValueChanged<String> onRestore;
  final VoidCallback onReload;
  final VoidCallback onSync;

  @override
  Widget build(BuildContext context) {
    final state = controller.state;
    final selectedLayer = state.layerId == null
        ? null
        : hierarchy.layerById(state.layerId!);
    final title = state.trash
        ? 'Lixeira'
        : selectedLayer == null
        ? 'Documentos'
        : 'Documentos · ${selectedLayer.name}';
    return Card(
      key: const Key('document-list-pane'),
      child: CustomScrollView(
        key: const Key('document-list-scroll'),
        slivers: <Widget>[
          SliverToBoxAdapter(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                Padding(
                  padding: const EdgeInsets.fromLTRB(14, 14, 8, 4),
                  child: Row(
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          title,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                      ),
                      IconButton(
                        key: const Key('document-sync'),
                        tooltip: 'Sincronizar agora',
                        onPressed: state.syncing ? null : onSync,
                        icon: const Icon(Icons.sync),
                      ),
                      IconButton(
                        tooltip: 'Recarregar documentos',
                        onPressed: onReload,
                        icon: const Icon(Icons.refresh),
                      ),
                      IconButton(
                        key: const Key('create-document'),
                        tooltip: 'Novo documento',
                        onPressed: state.layerId == null ? null : onCreate,
                        icon: const Icon(Icons.note_add_outlined),
                      ),
                    ],
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
                  child: Wrap(
                    spacing: 8,
                    runSpacing: 6,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: <Widget>[
                      ChoiceChip(
                        label: const Text('Diretos'),
                        selected: !state.subtree,
                        onSelected: state.layerId == null
                            ? null
                            : (_) => unawaited(
                                controller.loadLayer(
                                  state.layerId!,
                                  subtree: false,
                                  trash: state.trash,
                                ),
                              ),
                      ),
                      ChoiceChip(
                        label: const Text('Subárvore'),
                        selected: state.subtree,
                        onSelected: state.layerId == null
                            ? null
                            : (_) => unawaited(
                                controller.loadLayer(
                                  state.layerId!,
                                  subtree: true,
                                  trash: state.trash,
                                ),
                              ),
                      ),
                      FilterChip(
                        key: const Key('document-trash-filter'),
                        label: const Text('Lixeira'),
                        selected: state.trash,
                        onSelected: state.layerId == null
                            ? null
                            : (value) => unawaited(
                                controller.loadLayer(
                                  state.layerId!,
                                  subtree: state.subtree,
                                  trash: value,
                                ),
                              ),
                      ),
                    ],
                  ),
                ),
                if (state.syncing || state.saving)
                  const LinearProgressIndicator(minHeight: 2),
                if (state.message != null)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
                    child: Text(
                      state.message!,
                      key: const Key('document-status-message'),
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ),
              ],
            ),
          ),
          if (state.phase == DocumentsPhase.loading)
            const SliverFillRemaining(
              hasScrollBody: false,
              child: Center(child: CircularProgressIndicator()),
            )
          else if (state.items.isEmpty)
            SliverFillRemaining(
              hasScrollBody: false,
              child: _EmptyDocuments(trash: state.trash, onCreate: onCreate),
            )
          else
            SliverPadding(
              padding: const EdgeInsets.fromLTRB(8, 4, 8, 12),
              sliver: SliverList.separated(
                key: const Key('document-list'),
                itemCount: state.items.length,
                separatorBuilder: (_, _) => const SizedBox(height: 4),
                itemBuilder: (context, index) {
                  final document = state.items[index];
                  final origin = hierarchy.layerById(document.layerId);
                  return ListTile(
                    key: ValueKey<String>('document-${document.id}'),
                    selected: state.selected?.id == document.id,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                    leading: Icon(
                      document.isTrash
                          ? Icons.delete_outline
                          : Icons.description_outlined,
                    ),
                    title: Text(
                      document.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      _documentSubtitle(document, origin?.name),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    onTap: () => onOpen(document.id),
                    trailing: PopupMenuButton<_DocumentAction>(
                      tooltip: 'Ações de ${document.title}',
                      onSelected: (action) => switch (action) {
                        _DocumentAction.rename => onRename(document.id),
                        _DocumentAction.move => onMove(document.id),
                        _DocumentAction.trash => onTrash(document.id),
                        _DocumentAction.restore => onRestore(document.id),
                      },
                      itemBuilder: (context) =>
                          <PopupMenuEntry<_DocumentAction>>[
                            if (document.isTrash)
                              const PopupMenuItem(
                                value: _DocumentAction.restore,
                                child: Text('Restaurar'),
                              )
                            else ...<PopupMenuEntry<_DocumentAction>>[
                              const PopupMenuItem(
                                value: _DocumentAction.rename,
                                child: Text('Renomear'),
                              ),
                              const PopupMenuItem(
                                value: _DocumentAction.move,
                                child: Text('Mover'),
                              ),
                              const PopupMenuItem(
                                value: _DocumentAction.trash,
                                child: Text('Mover para a lixeira'),
                              ),
                            ],
                          ],
                      icon: const Icon(Icons.more_vert),
                    ),
                  );
                },
              ),
            ),
        ],
      ),
    );
  }

  String _documentSubtitle(LocalDocument document, String? origin) {
    final state = <String>[];
    if (origin != null) state.add('Origem: $origin');
    if (document.dirty) state.add('pendente local');
    if (document.conflict) state.add('conflito');
    if (document.uncertain) state.add('resultado incerto');
    return state.isEmpty ? 'Documento sincronizado' : state.join(' · ');
  }
}

class WorkspaceDocumentEditorPane extends StatelessWidget {
  const WorkspaceDocumentEditorPane({
    required this.controller,
    required this.document,
    required this.titleController,
    required this.markdownController,
    this.markdownFocusNode,
    required this.preview,
    required this.onPreviewChanged,
    required this.onOpenDocument,
    required this.onLinkTrigger,
    required this.onResolveConflict,
    super.key,
  });

  final DocumentsController controller;
  final LocalDocument? document;
  final TextEditingController? titleController;
  final TextEditingController? markdownController;
  final FocusNode? markdownFocusNode;
  final bool preview;
  final ValueChanged<bool> onPreviewChanged;
  final ValueChanged<String> onOpenDocument;
  final ValueChanged<String> onLinkTrigger;
  final ValueChanged<bool> onResolveConflict;

  @override
  Widget build(BuildContext context) {
    final selected = document;
    if (selected == null ||
        titleController == null ||
        markdownController == null) {
      return Card(
        key: const Key('document-editor-pane'),
        child: const Center(
          child: Text('Selecione um documento para abrir o editor.'),
        ),
      );
    }
    return Card(
      key: const Key('document-editor-pane'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 12, 14, 4),
            child: Row(
              children: <Widget>[
                Expanded(
                  child: TextField(
                    key: const Key('document-title-editor'),
                    controller: titleController,
                    readOnly: !selected.contentLoaded,
                    maxLines: 1,
                    textInputAction: TextInputAction.done,
                    decoration: const InputDecoration(
                      labelText: 'Título',
                      border: InputBorder.none,
                    ),
                    onChanged: (value) =>
                        unawaited(controller.editDocument(title: value)),
                  ),
                ),
                if (selected.dirty)
                  const Tooltip(
                    message: 'Rascunho local pendente',
                    child: Icon(Icons.cloud_upload_outlined),
                  ),
                if (selected.conflict)
                  PopupMenuButton<bool>(
                    key: const Key('resolve-document-conflict'),
                    tooltip: 'Resolver conflito',
                    onSelected: onResolveConflict,
                    itemBuilder: (context) => const <PopupMenuEntry<bool>>[
                      PopupMenuItem<bool>(
                        value: true,
                        child: Text('Manter versão local'),
                      ),
                      PopupMenuItem<bool>(
                        value: false,
                        child: Text('Usar versão do servidor'),
                      ),
                    ],
                    icon: const Icon(Icons.warning_amber_outlined),
                  ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: SegmentedButton<bool>(
              segments: const <ButtonSegment<bool>>[
                ButtonSegment<bool>(
                  value: false,
                  label: Text('Fonte'),
                  icon: Icon(Icons.code),
                ),
                ButtonSegment<bool>(
                  value: true,
                  label: Text('Prévia'),
                  icon: Icon(Icons.visibility_outlined),
                ),
              ],
              selected: <bool>{preview},
              onSelectionChanged: (values) => onPreviewChanged(values.first),
            ),
          ),
          const SizedBox(height: 8),
          Expanded(
            child: preview
                ? WorkspaceDocumentMarkdownPreview(
                    markdown: selected.markdown,
                    items: controller.state.items,
                    links: selected.links,
                    onOpenDocument: onOpenDocument,
                  )
                : TextField(
                    key: const Key('document-markdown-editor'),
                    controller: markdownController,
                    focusNode: markdownFocusNode,
                    readOnly: !selected.contentLoaded,
                    expands: true,
                    maxLines: null,
                    minLines: null,
                    textAlignVertical: TextAlignVertical.top,
                    keyboardType: TextInputType.multiline,
                    decoration: const InputDecoration(
                      hintText: 'Escreva em Markdown…',
                      border: InputBorder.none,
                      contentPadding: EdgeInsets.all(14),
                    ),
                    onChanged: (value) =>
                        unawaited(_editMarkdown(controller, value)),
                  ),
          ),
          _DocumentEditorStatus(document: selected, state: controller.state),
        ],
      ),
    );
  }

  Future<void> _editMarkdown(
    DocumentsController controller,
    String value,
  ) async {
    await controller.editDocument(markdown: value);
    onLinkTrigger(value);
  }
}

class WorkspaceDocumentMarkdownPreview extends StatelessWidget {
  const WorkspaceDocumentMarkdownPreview({
    required this.markdown,
    required this.items,
    required this.links,
    required this.onOpenDocument,
    this.onUnavailableLink,
    super.key,
  });

  final String markdown;
  final List<LocalDocument> items;
  final List<DocumentLink> links;
  final ValueChanged<String> onOpenDocument;
  final ValueChanged<DocumentLinkStatus>? onUnavailableLink;

  @override
  Widget build(BuildContext context) {
    final labels = <String, String>{
      for (final item in items)
        item.id.toLowerCase(): item.isTrash
            ? '${item.title} (na lixeira)'
            : item.title,
    };
    final statuses = <String, DocumentLinkStatus>{
      for (final item in items)
        item.id.toLowerCase(): item.isTrash
            ? DocumentLinkStatus.trash
            : DocumentLinkStatus.available,
    };
    for (final link in links) {
      final id = link.targetId.toLowerCase();
      labels.putIfAbsent(id, () => _linkLabel(link));
      statuses.putIfAbsent(id, () => link.status);
    }
    return SingleChildScrollView(
      key: const Key('document-markdown-preview'),
      padding: const EdgeInsets.all(16),
      child: MarkdownBody(
        data: _renderMarkdown(markdown, labels),
        extensionSet: _safeMarkdownExtensions,
        imageBuilder: (_, _, _) => const SizedBox.shrink(),
        onTapLink: (text, href, title) {
          final destination = href;
          if (destination == null) return;
          if (destination.startsWith('mindforge-doc://')) {
            final targetId = destination.substring('mindforge-doc://'.length);
            final status =
                statuses[targetId.toLowerCase()] ?? DocumentLinkStatus.missing;
            if (status != DocumentLinkStatus.available) {
              onUnavailableLink?.call(status);
              final message = status == DocumentLinkStatus.trash
                  ? 'O documento de destino está na lixeira.'
                  : 'O documento de destino não está disponível.';
              ScaffoldMessenger.of(
                context,
              ).showSnackBar(SnackBar(content: Text(message)));
              return;
            }
            onOpenDocument(targetId);
            return;
          }
          final uri = Uri.tryParse(destination);
          if (uri == null || (uri.scheme != 'http' && uri.scheme != 'https')) {
            return;
          }
          unawaited(launchUrl(uri, mode: LaunchMode.externalApplication));
        },
      ),
    );
  }

  String _linkLabel(DocumentLink link) => switch (link.status) {
    DocumentLinkStatus.available => link.title,
    DocumentLinkStatus.trash => '${link.title} (na lixeira)',
    DocumentLinkStatus.missing =>
      link.title.isEmpty
          ? 'Documento indisponível'
          : '${link.title} (indisponível)',
  };
}

final _safeMarkdownExtensions = md.ExtensionSet(
  <md.BlockSyntax>[md.FencedCodeBlockSyntax(), md.TableSyntax()],
  <md.InlineSyntax>[md.StrikethroughSyntax(), md.AutolinkExtensionSyntax()],
);

String _renderMarkdown(String source, Map<String, String> labels) {
  final output = StringBuffer();
  String? fenceMarker;
  for (final line in source.split('\n')) {
    final fence = RegExp(r'^\s*(`{3,}|~{3,})').firstMatch(line);
    if (fenceMarker != null) {
      output.writeln(line);
      if (fence != null &&
          fence.group(1)!.codeUnitAt(0) == fenceMarker.codeUnitAt(0) &&
          fence.group(1)!.length >= fenceMarker.length) {
        fenceMarker = null;
      }
      continue;
    }
    if (fence != null) {
      fenceMarker = fence.group(1);
      output.writeln(line);
      continue;
    }
    output.writeln(_replaceDocumentLinks(_escapeHtmlOutsideCode(line), labels));
  }
  return output.toString();
}

String _escapeHtmlOutsideCode(String line) {
  final output = StringBuffer();
  String? codeDelimiter;
  var index = 0;
  while (index < line.length) {
    if (line[index] == '`') {
      var end = index + 1;
      while (end < line.length && line[end] == '`') end++;
      final delimiter = line.substring(index, end);
      if (codeDelimiter == null) {
        codeDelimiter = delimiter;
      } else if (line.startsWith(codeDelimiter, index)) {
        end = index + codeDelimiter.length;
        codeDelimiter = null;
      }
      output.write(line.substring(index, end));
      index = end;
      continue;
    }
    final character = line[index];
    if (codeDelimiter == null && character == '<') {
      output.write('&lt;');
    } else if (codeDelimiter == null && character == '>') {
      output.write('&gt;');
    } else {
      output.write(character);
    }
    index++;
  }
  return output.toString();
}

String _replaceDocumentLinks(String line, Map<String, String> labels) {
  final result = StringBuffer();
  String? codeDelimiter;
  var index = 0;
  while (index < line.length) {
    if (line[index] == '`') {
      var end = index + 1;
      while (end < line.length && line[end] == '`') end++;
      final delimiter = line.substring(index, end);
      if (codeDelimiter == null) {
        codeDelimiter = delimiter;
      } else if (line.startsWith(codeDelimiter, index)) {
        end = index + codeDelimiter.length;
        codeDelimiter = null;
      }
      result.write(line.substring(index, end));
      index = end;
      continue;
    }
    const prefix = '[[doc:';
    final idStart = index + prefix.length;
    final idEnd = idStart + 36;
    if (codeDelimiter == null &&
        line.startsWith(prefix, index) &&
        idEnd + 2 <= line.length &&
        line.startsWith(']]', idEnd) &&
        (index == 0 || line[index - 1] != '\\')) {
      final id = line.substring(idStart, idEnd);
      if (RegExp(r'^[0-9a-fA-F-]{36}$').hasMatch(id)) {
        final label = labels[id.toLowerCase()] ?? 'Documento indisponível';
        result
          ..write('[')
          ..write(label.replaceAll(']', '\\]'))
          ..write('](mindforge-doc://')
          ..write(id)
          ..write(')');
        index = idEnd + 2;
        continue;
      }
    }
    result.write(line[index]);
    index++;
  }
  return result.toString();
}

class _DocumentEditorStatus extends StatelessWidget {
  const _DocumentEditorStatus({required this.document, required this.state});

  final LocalDocument document;
  final DocumentsState state;

  @override
  Widget build(BuildContext context) {
    var label = 'Sincronizado';
    if (state.syncing) {
      label = 'Sincronizando…';
    } else if (state.saving) {
      label = 'Gravando local…';
    } else if (document.conflict) {
      label = 'Conflito: escolha como continuar';
    } else if (document.uncertain) {
      label = 'Resultado incerto: confira antes de tentar novamente';
    } else if (document.dirty) {
      label = state.localPending
          ? 'Rascunho em memória · pendente de gravação local'
          : 'Salvo localmente · pendente de sincronização';
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 8, 14, 12),
      child: Text(
        label,
        key: const Key('document-save-status'),
        style: Theme.of(context).textTheme.bodySmall,
      ),
    );
  }
}

class _EmptyDocuments extends StatelessWidget {
  const _EmptyDocuments({required this.trash, required this.onCreate});

  final bool trash;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(
            trash ? Icons.delete_outline : Icons.note_add_outlined,
            size: 36,
          ),
          const SizedBox(height: 8),
          Text(
            trash ? 'A lixeira está vazia.' : 'Nenhum documento nesta camada.',
          ),
          if (!trash) ...<Widget>[
            const SizedBox(height: 10),
            OutlinedButton.icon(
              key: const Key('empty-create-document'),
              onPressed: onCreate,
              icon: const Icon(Icons.add),
              label: const Text('Criar documento'),
            ),
          ],
        ],
      ),
    ),
  );
}

enum _DocumentAction { rename, move, trash, restore }
