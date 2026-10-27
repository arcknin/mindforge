import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/documents/document_models.dart';

void main() {
  test(
    'round-trips local documents without losing an Int64 remote version',
    () {
      final document = LocalDocument(
        id: '11111111-1111-4111-8111-111111111111',
        layerId: '22222222-2222-4222-8222-222222222222',
        title: 'Arquitetura',
        markdown: '[[doc:33333333-3333-4333-8333-333333333333]]',
        remoteVersion: BigInt.parse('9223372036854775807'),
        localRevision: 7,
        dirty: true,
        contentLoaded: true,
      );

      final decoded = LocalDocument.fromJson(document.toJson());

      expect(decoded.id, document.id);
      expect(decoded.markdown, document.markdown);
      expect(decoded.remoteVersion, BigInt.parse('9223372036854775807'));
      expect(decoded.localRevision, 7);
      expect(decoded.dirty, isTrue);
    },
  );

  test('parses document pages and preserves cursor order', () {
    final page = DocumentPage.fromJson(<String, Object?>{
      'documents': <Object?>[
        <String, Object?>{
          'id': '11111111-1111-4111-8111-111111111111',
          'layer_id': '22222222-2222-4222-8222-222222222222',
          'title': 'A',
          'version': 9007199254740993,
          'created_at': '2026-10-13T00:00:00Z',
          'updated_at': '2026-10-13T00:00:00Z',
          'deleted_at': null,
          'expires_at': null,
        },
      ],
      'next_cursor': '33333333-3333-4333-8333-333333333333',
    });

    expect(page.documents.single.version, BigInt.parse('9007199254740993'));
    expect(page.nextCursor, '33333333-3333-4333-8333-333333333333');
  });
}
