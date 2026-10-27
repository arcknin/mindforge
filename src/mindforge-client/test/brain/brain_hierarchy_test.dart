import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';

void main() {
  group('BrainHierarchy', () {
    test('keeps the structural root hidden and builds nested layers by ID', () {
      final hierarchy = BrainHierarchy.fromJson(<String, Object?>{
        'brain': <String, Object?>{
          'id': 'brain-a',
          'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
        },
        'layers': <Object?>[
          <String, Object?>{
            'id': 'child',
            'parent': <String, Object?>{'kind': 'layer', 'id': 'parent'},
            'name': 'Child',
            'version': 1,
          },
          <String, Object?>{
            'id': 'top',
            'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
            'name': 'Top',
            'version': 1,
          },
          <String, Object?>{
            'id': 'parent',
            'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
            'name': 'Parent',
            'version': 2,
          },
        ],
      });

      expect(hierarchy.brainId, 'brain-a');
      expect(hierarchy.rootId, 'root-a');
      expect(hierarchy.topLevelLayers.map((layer) => layer.id), <String>[
        'parent',
        'top',
      ]);
      expect(hierarchy.layerById('parent')!.children.single.id, 'child');
    });

    test('rejects a layer whose parent is not in the same hierarchy', () {
      expect(
        () => BrainHierarchy.fromJson(<String, Object?>{
          'brain': <String, Object?>{
            'id': 'brain-a',
            'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
          },
          'layers': <Object?>[
            <String, Object?>{
              'id': 'orphan',
              'parent': <String, Object?>{'kind': 'layer', 'id': 'missing'},
              'name': 'Orphan',
              'version': 1,
            },
          ],
        }),
        throwsFormatException,
      );
    });

    test('rejects cycles without recursive traversal', () {
      expect(
        () => BrainHierarchy.fromJson(<String, Object?>{
          'brain': <String, Object?>{
            'id': 'brain-a',
            'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
          },
          'layers': <Object?>[
            <String, Object?>{
              'id': 'a',
              'parent': <String, Object?>{'kind': 'layer', 'id': 'b'},
              'name': 'A',
              'version': 1,
            },
            <String, Object?>{
              'id': 'b',
              'parent': <String, Object?>{'kind': 'layer', 'id': 'a'},
              'name': 'B',
              'version': 1,
            },
          ],
        }),
        throwsFormatException,
      );
    });

    test('rejects non-positive versions and duplicate IDs', () {
      expect(
        () => BrainHierarchy.fromJson(<String, Object?>{
          'brain': <String, Object?>{
            'id': 'brain-a',
            'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
          },
          'layers': <Object?>[
            <String, Object?>{
              'id': 'same',
              'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
              'name': 'One',
              'version': 0,
            },
          ],
        }),
        throwsFormatException,
      );

      expect(
        () => BrainHierarchy.fromJson(<String, Object?>{
          'brain': <String, Object?>{
            'id': 'brain-a',
            'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
          },
          'layers': <Object?>[
            <String, Object?>{
              'id': 'same',
              'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
              'name': 'One',
              'version': 1,
            },
            <String, Object?>{
              'id': 'same',
              'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
              'name': 'Two',
              'version': 1,
            },
          ],
        }),
        throwsFormatException,
      );
    });
  });
}
