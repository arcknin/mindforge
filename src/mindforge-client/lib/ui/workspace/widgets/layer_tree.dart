import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../brain/brain_hierarchy.dart';

class WorkspaceEmptyState extends StatelessWidget {
  const WorkspaceEmptyState({required this.onCreate, super.key});

  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) => Card(
    child: Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 440),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 22),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              Icon(
                Icons.account_tree_outlined,
                size: 40,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(height: 12),
              Text(
                'Seu Cérebro está pronto',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 8),
              const Text(
                'Crie a primeira camada para começar a organizar suas informações.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 14),
              FilledButton.icon(
                key: const Key('create-first-layer'),
                onPressed: onCreate,
                icon: const Icon(Icons.add),
                label: const Text('Criar primeira camada'),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

class WorkspaceLayerTree extends StatelessWidget {
  const WorkspaceLayerTree({
    required this.hierarchy,
    required this.selectedLayerId,
    required this.expandedLayerIds,
    required this.onSelect,
    required this.onToggleExpanded,
    required this.onCreateChild,
    required this.onRename,
    super.key,
  });

  final BrainHierarchy hierarchy;
  final String? selectedLayerId;
  final Set<String> expandedLayerIds;
  final ValueChanged<String> onSelect;
  final ValueChanged<String> onToggleExpanded;
  final ValueChanged<BrainLayer> onCreateChild;
  final ValueChanged<BrainLayer> onRename;

  @override
  Widget build(BuildContext context) {
    final rows = _visibleRows(hierarchy, expandedLayerIds);
    return Card(
      child: ListView.separated(
        key: const Key('brain-layer-tree'),
        padding: const EdgeInsets.symmetric(vertical: 10),
        itemCount: rows.length,
        separatorBuilder: (context, index) =>
            const Divider(height: 1, indent: 18),
        itemBuilder: (context, index) {
          final row = rows[index];
          final layer = row.layer;
          final selected = layer.id == selectedLayerId;
          final hasChildren = layer.children.isNotEmpty;
          final expanded = expandedLayerIds.contains(layer.id);
          return ListTile(
            key: ValueKey<String>('layer-${layer.id}'),
            selected: selected,
            contentPadding: EdgeInsets.only(
              left: 12 + math.min(row.depth, 6) * 20,
              right: 12,
            ),
            leading: Icon(
              hasChildren ? Icons.folder_copy_outlined : Icons.folder_outlined,
              color: selected ? Theme.of(context).colorScheme.primary : null,
            ),
            title: Text(
              layer.name,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            subtitle: Text(
              'Nível ${row.depth + 1} · ${layer.children.length} subcamadas',
            ),
            onTap: () => onSelect(layer.id),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                if (hasChildren)
                  IconButton(
                    tooltip: expanded
                        ? 'Recolher ${layer.name}'
                        : 'Expandir ${layer.name}',
                    onPressed: () => onToggleExpanded(layer.id),
                    icon: Icon(
                      expanded ? Icons.expand_less : Icons.expand_more,
                    ),
                  ),
                IconButton(
                  tooltip: 'Criar subcamada em ${layer.name}',
                  onPressed: () => onCreateChild(layer),
                  icon: const Icon(Icons.add),
                ),
                IconButton(
                  tooltip: 'Renomear ${layer.name}',
                  onPressed: () => onRename(layer),
                  icon: const Icon(Icons.edit_outlined),
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  static List<_LayerRow> _visibleRows(
    BrainHierarchy hierarchy,
    Set<String> expandedIds,
  ) {
    final rows = <_LayerRow>[];
    final stack = <_LayerRow>[];
    for (final layer in hierarchy.topLevelLayers.reversed) {
      stack.add(_LayerRow(layer: layer, depth: 0));
    }
    while (stack.isNotEmpty) {
      final row = stack.removeLast();
      rows.add(row);
      if (expandedIds.contains(row.layer.id)) {
        for (final child in row.layer.children.reversed) {
          stack.add(_LayerRow(layer: child, depth: row.depth + 1));
        }
      }
    }
    return rows;
  }
}

class _LayerRow {
  const _LayerRow({required this.layer, required this.depth});

  final BrainLayer layer;
  final int depth;
}

class WorkspaceLayerDetails extends StatelessWidget {
  const WorkspaceLayerDetails({
    required this.layer,
    required this.onCreateChild,
    required this.onRename,
    super.key,
  });

  final BrainLayer? layer;
  final VoidCallback? onCreateChild;
  final VoidCallback? onRename;

  @override
  Widget build(BuildContext context) => Card(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: layer == null
          ? const Center(
              child: Text('Selecione uma camada para ver os detalhes.'),
            )
          : Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Icon(
                  Icons.folder_open_outlined,
                  color: Theme.of(context).colorScheme.primary,
                  size: 30,
                ),
                const SizedBox(height: 18),
                Text(
                  layer!.name,
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 6),
                Text('Versão ${layer!.version}'),
                const SizedBox(height: 20),
                Text('${layer!.children.length} subcamadas'),
                const Spacer(),
                OutlinedButton.icon(
                  onPressed: onCreateChild,
                  icon: const Icon(Icons.add),
                  label: const Text('Criar subcamada'),
                ),
                const SizedBox(height: 8),
                OutlinedButton.icon(
                  onPressed: onRename,
                  icon: const Icon(Icons.edit_outlined),
                  label: const Text('Renomear'),
                ),
              ],
            ),
    ),
  );
}

class WorkspaceNoticeBanner extends StatelessWidget {
  const WorkspaceNoticeBanner({
    required this.message,
    this.icon = Icons.info_outline,
    this.action,
    super.key,
  });

  final String message;
  final IconData icon;
  final Widget? action;

  @override
  Widget build(BuildContext context) => Container(
    margin: const EdgeInsets.only(bottom: 8),
    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
    decoration: BoxDecoration(
      color: Theme.of(context).colorScheme.secondaryContainer,
      borderRadius: BorderRadius.circular(12),
    ),
    child: Row(
      children: <Widget>[
        Icon(icon, size: 20),
        const SizedBox(width: 10),
        Expanded(child: Text(message)),
        if (action != null) action!,
      ],
    ),
  );
}
