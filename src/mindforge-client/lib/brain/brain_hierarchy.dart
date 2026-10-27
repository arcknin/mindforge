import 'brain_json.dart' show parseLayerVersion;

enum ParentKind { root, layer }

class BrainParent {
  const BrainParent({required this.kind, required this.id});

  final ParentKind kind;
  final String id;

  String get kindName => switch (kind) {
    ParentKind.root => 'root',
    ParentKind.layer => 'layer',
  };

  factory BrainParent.fromJson(Object? json, String path) {
    final object = _object(
      json,
      path,
      allowed: const <String>{'kind', 'id'},
      required: const <String>{'kind', 'id'},
    );
    final kindName = _string(object['kind'], '$path.kind');
    final kind = switch (kindName) {
      'root' => ParentKind.root,
      'layer' => ParentKind.layer,
      _ => throw FormatException('$path.kind is invalid.'),
    };
    return BrainParent(
      kind: kind,
      id: _nonEmptyString(object['id'], '$path.id'),
    );
  }
}

class BrainSummary {
  const BrainSummary({required this.id, required this.rootId});

  final String id;
  final String rootId;

  factory BrainSummary.fromEnvelope(Object? json) {
    final envelope = _object(
      json,
      'response',
      allowed: const <String>{'brain'},
      required: const <String>{'brain'},
    );
    final brain = _object(
      envelope['brain'],
      'response.brain',
      allowed: const <String>{'id', 'root'},
      required: const <String>{'id', 'root'},
    );
    final root = BrainParent.fromJson(brain['root'], 'response.brain.root');
    if (root.kind != ParentKind.root) {
      throw const FormatException('response.brain.root.kind must be root.');
    }
    return BrainSummary(
      id: _nonEmptyString(brain['id'], 'response.brain.id'),
      rootId: root.id,
    );
  }
}

class BrainLayer {
  BrainLayer._({
    required this.id,
    required this.parent,
    required this.name,
    required this.version,
    required List<BrainLayer> children,
  }) : children = List<BrainLayer>.unmodifiable(children);

  final String id;
  final BrainParent parent;
  final String name;
  final BigInt version;
  final List<BrainLayer> children;

  factory BrainLayer.fromEnvelope(Object? json) {
    final envelope = _object(
      json,
      'response',
      allowed: const <String>{'layer'},
      required: const <String>{'layer'},
    );
    return _parseLayer(envelope['layer'], 'response.layer');
  }
}

class BrainHierarchy {
  BrainHierarchy._({
    required this.brain,
    required List<BrainLayer> topLevelLayers,
    required Map<String, BrainLayer> layerById,
  }) : topLevelLayers = List<BrainLayer>.unmodifiable(topLevelLayers),
       _layerById = Map<String, BrainLayer>.unmodifiable(layerById);

  final BrainSummary brain;
  final List<BrainLayer> topLevelLayers;
  final Map<String, BrainLayer> _layerById;

  String get brainId => brain.id;
  String get rootId => brain.rootId;

  BrainLayer? layerById(String id) => _layerById[id];

  factory BrainHierarchy.fromJson(Object? json) {
    final envelope = _object(
      json,
      'response',
      allowed: const <String>{'brain', 'layers'},
      required: const <String>{'brain', 'layers'},
    );
    final brain = BrainSummary.fromEnvelope(<String, Object?>{
      'brain': envelope['brain'],
    });
    final rawLayers = envelope['layers'];
    if (rawLayers is! List<Object?>) {
      throw const FormatException('response.layers must be an array.');
    }

    final records = <String, _LayerRecord>{};
    for (var index = 0; index < rawLayers.length; index++) {
      final layer = _parseLayerRecord(
        rawLayers[index],
        'response.layers[$index]',
      );
      if (layer.id == brain.rootId || records.containsKey(layer.id)) {
        throw const FormatException(
          'Layer IDs must be unique and separate from root.',
        );
      }
      records[layer.id] = layer;
    }

    final childIds = <String, List<String>>{brain.rootId: <String>[]};
    for (final layer in records.values) {
      if (layer.parent.kind == ParentKind.root) {
        if (layer.parent.id != brain.rootId) {
          throw const FormatException(
            'A layer references a different structural root.',
          );
        }
        childIds.putIfAbsent(brain.rootId, () => <String>[]).add(layer.id);
      } else {
        if (!records.containsKey(layer.parent.id)) {
          throw const FormatException(
            'A layer parent is missing from the hierarchy.',
          );
        }
        childIds.putIfAbsent(layer.parent.id, () => <String>[]).add(layer.id);
      }
    }
    for (final children in childIds.values) {
      children.sort((leftId, rightId) {
        final byName = records[leftId]!.name.compareTo(records[rightId]!.name);
        return byName == 0 ? leftId.compareTo(rightId) : byName;
      });
    }

    // Parent links form a graph with at most one parent per layer. Walking from
    // the hidden root proves reachability and detects cycles without recursion.
    final orderedIds = <String>[];
    final queue = List<String>.of(childIds[brain.rootId]!);
    for (var index = 0; index < queue.length; index++) {
      final id = queue[index];
      orderedIds.add(id);
      queue.addAll(childIds[id] ?? const <String>[]);
    }
    if (orderedIds.length != records.length) {
      throw const FormatException('The hierarchy contains a cycle.');
    }

    final builtLayers = <String, BrainLayer>{};
    for (final id in orderedIds.reversed) {
      final record = records[id]!;
      builtLayers[id] = BrainLayer._(
        id: record.id,
        parent: record.parent,
        name: record.name,
        version: record.version,
        children: (childIds[id] ?? const <String>[])
            .map((childId) => builtLayers[childId]!)
            .toList(growable: false),
      );
    }

    return BrainHierarchy._(
      brain: brain,
      topLevelLayers: childIds[brain.rootId]!
          .map((id) => builtLayers[id]!)
          .toList(growable: false),
      layerById: builtLayers,
    );
  }
}

class _LayerRecord {
  const _LayerRecord({
    required this.id,
    required this.parent,
    required this.name,
    required this.version,
  });

  final String id;
  final BrainParent parent;
  final String name;
  final BigInt version;
}

BrainLayer _parseLayer(Object? json, String path) {
  final record = _parseLayerRecord(json, path);
  return BrainLayer._(
    id: record.id,
    parent: record.parent,
    name: record.name,
    version: record.version,
    children: const <BrainLayer>[],
  );
}

_LayerRecord _parseLayerRecord(Object? json, String path) {
  final object = _object(
    json,
    path,
    allowed: const <String>{'id', 'parent', 'name', 'version'},
    required: const <String>{'id', 'parent', 'name', 'version'},
  );
  final name = _string(object['name'], '$path.name');
  if (name.trim().isEmpty) {
    throw FormatException('$path.name must contain a visible character.');
  }
  final version = parseLayerVersion(object['version'], '$path.version');
  return _LayerRecord(
    id: _nonEmptyString(object['id'], '$path.id'),
    parent: BrainParent.fromJson(object['parent'], '$path.parent'),
    name: name,
    version: version,
  );
}

Map<String, Object?> _object(
  Object? value,
  String path, {
  required Set<String> allowed,
  required Set<String> required,
}) {
  if (value is! Map) {
    throw FormatException('$path must be an object.');
  }
  final result = <String, Object?>{};
  for (final entry in value.entries) {
    if (entry.key is! String || !allowed.contains(entry.key)) {
      throw FormatException('$path contains an unknown field.');
    }
    result[entry.key as String] = entry.value;
  }
  if (!result.keys.toSet().containsAll(required)) {
    throw FormatException('$path is missing a required field.');
  }
  return result;
}

String _string(Object? value, String path) {
  if (value is! String) {
    throw FormatException('$path must be a string.');
  }
  return value;
}

String _nonEmptyString(Object? value, String path) {
  final result = _string(value, path);
  if (result.isEmpty) {
    throw FormatException('$path must not be empty.');
  }
  return result;
}
