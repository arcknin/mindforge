import 'dart:async';

/// Bridges app_links' cold-start and warm-link APIs into one guarded stream.
///
/// The native platform adapter owns the [getInitialLink] and [linkStream]
/// calls. Keeping them as injected ports makes callback filtering and
/// de-duplication testable without a device or the app_links plugin.
class NativeAuthCallbackAdapter {
  const NativeAuthCallbackAdapter({
    required this.callbackUri,
    required this.getInitialLink,
    required this.linkStream,
  });

  final Uri callbackUri;
  final Future<Uri?> Function() getInitialLink;
  final Stream<Uri> linkStream;

  /// Emits an expected callback from either cold start or warm delivery once.
  ///
  /// An initial-link lookup failure is treated as an absent cold callback so a
  /// warm callback can still complete the sign-in flow. Unexpected schemes,
  /// hosts, ports, paths, or user-info are ignored before reaching auth.
  Stream<Uri> get callbacks {
    late final StreamController<Uri> controller;
    StreamSubscription<Uri>? sourceSubscription;
    final seen = <String>{};
    final buffered = <Uri>[];
    final initialDone = Completer<void>();
    var initialComplete = false;

    bool isExpected(Uri uri) =>
        uri.scheme == callbackUri.scheme &&
        uri.userInfo.isEmpty &&
        uri.host == callbackUri.host &&
        uri.port == callbackUri.port &&
        uri.path == callbackUri.path &&
        !uri.hasFragment;

    void deliver(Uri uri) {
      if (!isExpected(uri) || !seen.add(uri.toString())) return;
      if (!initialComplete) {
        buffered.add(uri);
      } else if (!controller.isClosed) {
        controller.add(uri);
      }
    }

    Future<void> loadInitial() async {
      try {
        final initial = await getInitialLink();
        if (initial != null) deliver(initial);
      } on Object {
        // A missing/unavailable cold-start link must not disable warm links.
      }
      initialComplete = true;
      if (!controller.isClosed) {
        for (final uri in buffered) {
          controller.add(uri);
        }
      }
      buffered.clear();
      if (!initialDone.isCompleted) initialDone.complete();
    }

    controller = StreamController<Uri>(
      sync: true,
      onListen: () {
        sourceSubscription = linkStream.listen(
          deliver,
          onError: (Object error, StackTrace stackTrace) {
            if (!controller.isClosed) controller.addError(error, stackTrace);
          },
          onDone: () async {
            await initialDone.future;
            if (!controller.isClosed) await controller.close();
          },
        );
        unawaited(loadInitial());
      },
      onCancel: () async {
        await sourceSubscription?.cancel();
        buffered.clear();
        if (!initialDone.isCompleted) initialDone.complete();
        if (!controller.isClosed) await controller.close();
      },
    );
    return controller.stream;
  }
}
