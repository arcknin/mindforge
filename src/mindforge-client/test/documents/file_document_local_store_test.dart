import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/documents/document_models.dart';
import 'package:mindforge_client/documents/document_repository.dart';
import 'package:mindforge_client/documents/file_document_local_store.dart';

void main() {
  late Directory root;
  late FileDocumentLocalStore store;

  setUp(() async {
    root = await Directory.systemTemp.createTemp('mindforge-documents-');
    store = FileDocumentLocalStore(
      namespace: 'mindforge-dev',
      directory: () async => root,
    );
  });

  tearDown(() async {
    if (root.existsSync()) await root.delete(recursive: true);
  });

  test('reads empty state and commits an atomic per-user snapshot', () async {
    expect(await store.read('user-a'), isEmpty);
    final document = LocalDocument(
      id: '11111111-1111-4111-8111-111111111111',
      layerId: '22222222-2222-4222-8222-222222222222',
      title: 'Título',
      markdown: 'conteúdo',
      remoteVersion: BigInt.parse('9223372036854775807'),
      localRevision: 2,
      dirty: true,
      contentLoaded: true,
    );

    await store.write('user-a', <LocalDocument>[document]);

    final restored = await store.read('user-a');
    expect(restored.single.remoteVersion, BigInt.parse('9223372036854775807'));
    expect(restored.single.markdown, 'conteúdo');
    expect(root.listSync().whereType<File>(), hasLength(1));
    expect(
      root.listSync().whereType<File>().single.path,
      isNot(contains('.tmp')),
    );
  });

  test('does not erase a valid snapshot when the file is corrupted', () async {
    await store.write('user-a', <LocalDocument>[]);
    final file = root.listSync().whereType<File>().single;
    await file.writeAsString('{broken');

    await expectLater(
      store.read('user-a'),
      throwsA(isA<DocumentLocalStoreException>()),
    );
    expect(file.existsSync(), isTrue);
  });

  test('keeps accounts with similar sanitized characters isolated', () async {
    final first = LocalDocument(
      id: '11111111-1111-4111-8111-111111111111',
      layerId: '22222222-2222-4222-8222-222222222222',
      title: 'A',
      markdown: 'A',
      remoteVersion: BigInt.one,
      localRevision: 0,
      dirty: false,
      contentLoaded: true,
    );
    final second = first.copyWith(markdown: 'B');
    await store.write('user@a', <LocalDocument>[first]);
    await store.write('user#a', <LocalDocument>[second]);

    expect((await store.read('user@a')).single.markdown, 'A');
    expect((await store.read('user#a')).single.markdown, 'B');
    expect(root.listSync().whereType<File>(), hasLength(2));
  });
}
