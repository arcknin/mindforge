import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/documents/document_models.dart';
import 'package:mindforge_client/ui/workspace/widgets/document_panes.dart';

void main() {
  testWidgets('projects an uncached remote link and keeps its tap in-app', (
    tester,
  ) async {
    const targetId = '22222222-2222-4222-8222-222222222222';
    final document = _document(
      markdown: '[[doc:$targetId]]',
      links: const <DocumentLink>[
        DocumentLink(
          targetId: targetId,
          title: 'Destino remoto',
          layerId: 'other-layer',
          status: DocumentLinkStatus.available,
        ),
      ],
    );
    String? opened;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: WorkspaceDocumentMarkdownPreview(
            markdown: document.markdown,
            items: <LocalDocument>[document],
            links: document.links,
            onOpenDocument: (id) => opened = id,
          ),
        ),
      ),
    );

    expect(find.text('Destino remoto'), findsOneWidget);
    await tester.tap(find.text('Destino remoto'));
    await tester.pump();
    expect(opened, targetId);
  });

  testWidgets('missing and trash targets show an in-app message', (
    tester,
  ) async {
    const missingId = '33333333-3333-4333-8333-333333333333';
    const trashId = '44444444-4444-4444-8444-444444444444';
    final missingDocument = _document(
      markdown: '[[doc:$missingId]]',
      links: const <DocumentLink>[
        DocumentLink(
          targetId: missingId,
          title: 'Ausente',
          layerId: 'other-layer',
          status: DocumentLinkStatus.missing,
        ),
      ],
    );
    final trashDocument = _document(
      markdown: '[[doc:$trashId]]',
      links: const <DocumentLink>[
        DocumentLink(
          targetId: trashId,
          title: 'Na lixeira',
          layerId: 'other-layer',
          status: DocumentLinkStatus.trash,
        ),
      ],
    );
    var opened = false;
    DocumentLinkStatus? blockedStatus;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: WorkspaceDocumentMarkdownPreview(
            markdown: missingDocument.markdown,
            items: <LocalDocument>[missingDocument],
            links: missingDocument.links,
            onOpenDocument: (_) => opened = true,
            onUnavailableLink: (status) => blockedStatus = status,
          ),
        ),
      ),
    );

    // The renderer may split link spans, so inspect the semantic text rather
    // than requiring a particular RichText span boundary.
    expect(find.textContaining('Ausente'), findsOneWidget);
    await tester.tap(find.textContaining('Ausente'));
    await tester.pumpAndSettle();
    expect(blockedStatus, DocumentLinkStatus.missing);
    expect(opened, isFalse);

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: WorkspaceDocumentMarkdownPreview(
            markdown: trashDocument.markdown,
            items: <LocalDocument>[trashDocument],
            links: trashDocument.links,
            onOpenDocument: (_) => opened = true,
            onUnavailableLink: (status) => blockedStatus = status,
          ),
        ),
      ),
    );
    blockedStatus = null;
    await tester.pumpAndSettle();
    await tester.tap(find.textContaining('Na lixeira'));
    await tester.pumpAndSettle();
    expect(blockedStatus, DocumentLinkStatus.trash);
    expect(opened, isFalse);
  });
}

LocalDocument _document({
  required String markdown,
  required List<DocumentLink> links,
}) => LocalDocument(
  id: '11111111-1111-4111-8111-111111111111',
  layerId: 'layer-a',
  title: 'Origem',
  markdown: markdown,
  remoteVersion: BigInt.one,
  localRevision: 0,
  dirty: false,
  contentLoaded: true,
  links: links,
);
