import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/auth/auth_gateway.dart';
import 'package:mindforge_client/brain/brain_hierarchy.dart';
import 'package:mindforge_client/brain/brain_repository.dart';
import 'package:mindforge_client/brain/brain_workspace_controller.dart';
import 'package:mindforge_client/documents/document_models.dart';
import 'package:mindforge_client/documents/document_repository.dart';
import 'package:mindforge_client/documents/documents_controller.dart';
import 'package:mindforge_client/ui/workspace_screen.dart';
import 'package:mindforge_client/ui/workspace/widgets/document_title_dialog.dart';

void main() {
  testWidgets('empty document list fits while create dialog keyboard is open', (
    tester,
  ) async {
    final fixture = _DocumentWorkspaceFixture(empty: true);
    await fixture.initialize();
    await fixture.workspace.renameLayer(
      layerId: 'layer-a',
      draft: 'Projetos',
      expectedVersion: BigInt.one,
    );
    await fixture.mount(tester, const Size(400, 760));
    await tester.tap(find.text('Projetos'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-document')));
    await tester.pumpAndSettle();
    tester.view.viewInsets = const FakeViewPadding(bottom: 340);
    addTearDown(tester.view.resetViewInsets);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.text('Novo documento'), findsOneWidget);
    expect(find.text('A camada foi renomeada.'), findsOneWidget);
    await tester.tap(find.text('Cancelar'));
    await tester.pumpAndSettle();
    await tester.drag(
      find.byKey(const Key('document-list-scroll')),
      const Offset(0, -260),
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const Key('empty-create-document')).hitTestable(),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
    fixture.dispose();
  });

  testWidgets('empty title stays editable and creates only after correction', (
    tester,
  ) async {
    final fixture = _DocumentWorkspaceFixture(empty: true);
    await fixture.initialize();
    await fixture.mount(tester, const Size(400, 760));
    await tester.tap(find.text('Projetos'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-document')));
    await tester.pumpAndSettle();
    final input = find.descendant(
      of: find.byType(AlertDialog),
      matching: find.byType(TextField),
    );
    await tester.enterText(input, '   ');
    await tester.tap(find.widgetWithText(FilledButton, 'Criar'));
    await tester.pumpAndSettle();
    expect(find.text('Informe um título.'), findsOneWidget);
    expect(fixture.documents.state.items, isEmpty);
    await tester.enterText(input, 'Título válido');
    await tester.tap(find.widgetWithText(FilledButton, 'Criar'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(fixture.documents.state.items.single.title, 'Título válido');
    fixture.dispose();
  });

  testWidgets(
    'document dialog rejects repeated submit while pending and closing',
    (tester) async {
      final completion = Completer<void>();
      var submissions = 0;
      late BuildContext homeContext;
      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (context) {
              homeContext = context;
              return const Scaffold();
            },
          ),
        ),
      );
      unawaited(
        showDialog<bool>(
          context: homeContext,
          builder: (_) => WorkspaceDocumentTitleDialog(
            title: 'Novo documento',
            initialTitle: 'Teste',
            confirmLabel: 'Criar',
            failureMessage: 'Falha local',
            onSubmit: (_) {
              submissions++;
              return completion.future;
            },
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pump();
      expect(submissions, 1);
      expect(
        tester
            .widget<FilledButton>(find.widgetWithText(FilledButton, 'Criar'))
            .onPressed,
        isNull,
      );
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(find.byType(AlertDialog), findsOneWidget);
      completion.complete();
      await tester.pump();
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();
      expect(submissions, 1);
      expect(find.byType(AlertDialog), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('failed document submission preserves title for retry', (
    tester,
  ) async {
    var submissions = 0;
    late BuildContext homeContext;
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) {
            homeContext = context;
            return const Scaffold();
          },
        ),
      ),
    );
    unawaited(
      showDialog<bool>(
        context: homeContext,
        builder: (_) => WorkspaceDocumentTitleDialog(
          title: 'Novo documento',
          initialTitle: 'Meu rascunho',
          confirmLabel: 'Criar',
          failureMessage: 'Falha local',
          onSubmit: (_) async {
            submissions++;
            if (submissions == 1) throw StateError('fake failure');
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Criar'));
    await tester.pumpAndSettle();
    expect(find.text('Falha local'), findsOneWidget);
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      'Meu rascunho',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Criar'));
    await tester.pumpAndSettle();
    expect(submissions, 2);
    expect(find.byType(AlertDialog), findsNothing);
    expect(tester.takeException(), isNull);
  });

  for (final submitWithKeyboard in <bool>[false, true]) {
    testWidgets(
      'create document closes animated dialog safely (keyboard=$submitWithKeyboard)',
      (tester) async {
        final fixture = _DocumentWorkspaceFixture(empty: true);
        await fixture.initialize();
        await fixture.mount(tester, const Size(400, 760));
        await tester.tap(find.text('Projetos'));
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('create-document')));
        await tester.pumpAndSettle();
        final input = find.descendant(
          of: find.byType(AlertDialog),
          matching: find.byType(TextField),
        );
        await tester.enterText(input, 'Teste criado');
        if (submitWithKeyboard) {
          await tester.testTextInput.receiveAction(TextInputAction.done);
        } else {
          await tester.tap(find.widgetWithText(FilledButton, 'Criar'));
        }
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 80));
        expect(tester.takeException(), isNull);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        expect(find.byType(AlertDialog), findsNothing);
        expect(
          find.byKey(const Key('document-markdown-editor')),
          findsOneWidget,
        );
        expect(fixture.documents.state.items, hasLength(1));
        final id = fixture.documents.state.selected!.id;
        await tester.enterText(
          find.byKey(const Key('document-markdown-editor')),
          'texto preservado depois de criar',
        );
        await tester.pump(const Duration(seconds: 1));
        expect(fixture.store.saved.single.id, id);
        expect(
          fixture.store.saved.single.markdown,
          'texto preservado depois de criar',
        );
        expect(tester.takeException(), isNull);
        // Reopen through the same persistence boundary used at initialization.
        await tester.pumpWidget(const SizedBox.shrink());
        final reopened = DocumentsController(
          auth: fixture.auth,
          repository: _FakeDocumentRepository(empty: true),
          localStore: fixture.store,
        );
        await reopened.initialize();
        await reopened.loadLayer('layer-a');
        await reopened.openDocument(id);
        expect(reopened.state.selected!.id, id);
        expect(
          reopened.state.selected!.markdown,
          'texto preservado depois de criar',
        );
        reopened.dispose();
        fixture.dispose();
      },
    );
  }

  testWidgets('cancel document dialog completes its exit without creating', (
    tester,
  ) async {
    final fixture = _DocumentWorkspaceFixture(empty: true);
    await fixture.initialize();
    await fixture.mount(tester, const Size(400, 760));
    await tester.tap(find.text('Projetos'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-document')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Cancelar'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(tester.takeException(), isNull);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(fixture.documents.state.items, isEmpty);
    fixture.dispose();
  });

  testWidgets('rename document closes animated dialog and preserves body', (
    tester,
  ) async {
    final fixture = _DocumentWorkspaceFixture();
    await fixture.initialize();
    await fixture.mount(tester, const Size(400, 760));
    await tester.tap(find.text('Projetos'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Ações de Nota de teste'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Renomear'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(TextField),
      ),
      'Renomeado',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Salvar'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(tester.takeException(), isNull);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(fixture.documents.state.selected!.title, 'Renomeado');
    expect(fixture.documents.state.selected!.markdown, 'conteúdo inicial');
    fixture.dispose();
  });

  testWidgets(
    'compact navigation and editor draft survive back navigation and resize',
    (tester) async {
      final fixture = _DocumentWorkspaceFixture();
      await fixture.initialize();
      await fixture.mount(tester, const Size(400, 760));

      expect(find.byKey(const Key('workspace-compact-layout')), findsOneWidget);
      await tester.tap(find.text('Projetos'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('document-list-pane')), findsOneWidget);

      await tester.tap(find.text('Nota de teste'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('document-editor-pane')), findsOneWidget);

      await tester.enterText(
        find.byKey(const Key('document-markdown-editor')),
        '[[',
      );
      await tester.pumpAndSettle();
      expect(find.text('Escolha o destino do link'), findsOneWidget);
      const targetKey = Key('link-target-22222222-2222-4222-8222-222222222222');
      expect(find.byKey(targetKey), findsOneWidget);
      await tester.tap(find.byKey(targetKey));
      await tester.pump();
      expect(
        fixture.documents.state.selected?.markdown,
        '[[doc:22222222-2222-4222-8222-222222222222]]',
      );

      await tester.enterText(
        find.byKey(const Key('document-markdown-editor')),
        'rascunho que precisa sobreviver',
      );
      await tester.pump();

      tester.view.physicalSize = const Size(900, 760);
      await tester.pumpAndSettle();
      expect(
        find.byKey(const Key('workspace-expanded-layout')),
        findsOneWidget,
      );
      expect(find.text('rascunho que precisa sobreviver'), findsOneWidget);

      tester.view.physicalSize = const Size(400, 760);
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('workspace-compact-layout')), findsOneWidget);
      expect(find.text('rascunho que precisa sobreviver'), findsOneWidget);

      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('document-list-pane')), findsOneWidget);
      final editorField = find.byKey(
        const Key('document-markdown-editor'),
        skipOffstage: false,
      );
      expect(
        tester.widget<TextField>(editorField).focusNode?.hasFocus ?? false,
        isFalse,
      );
      expect(
        fixture.documents.state.selected?.markdown,
        'rascunho que precisa sobreviver',
      );
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(find.byKey(const Key('brain-layer-tree')), findsOneWidget);

      fixture.dispose();
    },
  );
}

class _DocumentWorkspaceFixture {
  _DocumentWorkspaceFixture({this.empty = false});

  final bool empty;
  final _FakeDocumentStore store = _FakeDocumentStore();
  final _FakeAuthGateway auth = _FakeAuthGateway();
  late final _FakeBrainRepository brainRepository = _FakeBrainRepository();
  late final BrainWorkspaceController workspace = BrainWorkspaceController(
    auth: auth,
    repository: brainRepository,
  );
  late final DocumentsController documents = DocumentsController(
    auth: auth,
    repository: _FakeDocumentRepository(empty: empty),
    localStore: store,
    autosaveInterval: const Duration(seconds: 1),
    syncInterval: const Duration(hours: 1),
  );

  Future<void> initialize() async {
    await workspace.initialize();
    await documents.initialize();
  }

  Future<void> mount(WidgetTester tester, Size size) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });
    await tester.pumpWidget(
      MaterialApp(
        home: WorkspaceScreen(
          auth: auth,
          workspace: workspace,
          documents: documents,
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  void dispose() {
    documents.dispose();
    workspace.dispose();
    auth.dispose();
  }
}

class _FakeAuthGateway extends AuthGateway {
  @override
  AuthViewState state = const AuthViewState(
    phase: AuthPhase.authenticated,
    generation: 1,
    userId: 'user-a',
  );

  @override
  Future<AuthCredential> credential() async => const AuthCredential(
    accessToken: 'test-token',
    userId: 'user-a',
    generation: 1,
  );

  @override
  Future<AuthCredential> refreshAfterUnauthorized(
    AuthCredential rejected,
  ) async => rejected;

  @override
  Future<void> expireAfterUnauthorized(AuthCredential rejected) async {}

  @override
  Future<void> initialize() async {}

  @override
  Future<void> beginGoogleSignIn() async {}

  @override
  Future<void> cancelGoogleSignIn() async {}

  @override
  Future<void> handleCallback(Uri uri) async {}

  @override
  Future<void> signOut() async {}
}

class _FakeBrainRepository implements BrainRepository {
  final BrainHierarchy hierarchy = BrainHierarchy.fromJson(<String, Object?>{
    'brain': <String, Object?>{
      'id': 'brain-a',
      'root': <String, Object?>{'kind': 'root', 'id': 'root-a'},
    },
    'layers': <Object?>[
      <String, Object?>{
        'id': 'layer-a',
        'parent': <String, Object?>{'kind': 'root', 'id': 'root-a'},
        'name': 'Projetos',
        'version': 1,
      },
    ],
  });

  @override
  Future<BrainSummary> bootstrap() async =>
      BrainSummary(id: hierarchy.brainId, rootId: hierarchy.rootId);

  @override
  Future<BrainHierarchy> observe() async => hierarchy;

  @override
  Future<BrainLayer> createLayer({
    required BrainParent parent,
    required String name,
  }) async => BrainLayer.fromEnvelope(<String, Object?>{
    'layer': <String, Object?>{
      'id': 'layer-new',
      'parent': <String, Object?>{'kind': parent.kindName, 'id': parent.id},
      'name': name,
      'version': 1,
    },
  });

  @override
  Future<BrainLayer> renameLayer({
    required String layerId,
    required String name,
    required BigInt expectedVersion,
  }) async => BrainLayer.fromEnvelope(<String, Object?>{
    'layer': <String, Object?>{
      'id': layerId,
      'parent': <String, Object?>{'kind': 'root', 'id': hierarchy.rootId},
      'name': name,
      'version': expectedVersion + BigInt.one,
    },
  });
}

class _FakeDocumentRepository implements DocumentRepository {
  _FakeDocumentRepository({this.empty = false});

  final bool empty;
  final _document = RemoteDocument(
    id: '11111111-1111-4111-8111-111111111111',
    layerId: 'layer-a',
    title: 'Nota de teste',
    version: BigInt.one,
    createdAt: DateTime.utc(2026, 10, 13),
    updatedAt: DateTime.utc(2026, 10, 13),
    markdown: 'conteúdo inicial',
    links: const <DocumentLink>[],
  );
  final _target = RemoteDocument(
    id: '22222222-2222-4222-8222-222222222222',
    layerId: 'layer-a',
    title: 'Destino',
    version: BigInt.one,
    createdAt: DateTime.utc(2026, 10, 13),
    updatedAt: DateTime.utc(2026, 10, 13),
    markdown: 'destino',
    links: const <DocumentLink>[],
  );

  @override
  Future<DocumentPage> list(
    String layerId, {
    bool subtree = false,
    bool trash = false,
    String? after,
  }) async => DocumentPage(
    documents: trash || empty
        ? const <DocumentMetadata>[]
        : <DocumentMetadata>[_document, _target],
  );

  @override
  Future<RemoteDocument> get(String id) async =>
      id == _target.id ? _target : _document;

  @override
  Future<RemoteDocument> create({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
  }) async => _document;

  @override
  Future<RemoteDocument> update({
    required String id,
    required String layerId,
    required String title,
    required String markdown,
    required BigInt expectedVersion,
  }) async => _document;

  @override
  Future<RemoteDocument> trash({
    required String id,
    required BigInt expectedVersion,
  }) async => _document;

  @override
  Future<RemoteDocument> restore({
    required String id,
    required BigInt expectedVersion,
    String? layerId,
  }) async => _document;
}

class _FakeDocumentStore implements DocumentLocalStore {
  List<LocalDocument> saved = <LocalDocument>[];

  @override
  Future<List<LocalDocument>> read(String userId) async => List.of(saved);

  @override
  Future<void> write(String userId, List<LocalDocument> documents) async {
    saved = List.of(documents);
  }
}
