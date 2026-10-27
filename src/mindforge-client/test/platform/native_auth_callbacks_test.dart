import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/platform/native_auth_callbacks.dart';

void main() {
  const callbackUri = 'mindforge-dev://auth/callback';

  group('NativeAuthCallbackAdapter', () {
    test(
      'emits an accepted cold callback once when warm delivery repeats it',
      () async {
        final warmLinks = StreamController<Uri>();
        final cold = Uri.parse('$callbackUri?code=cold');
        final warm = Uri.parse('$callbackUri?code=warm');
        final received = <Uri>[];
        final adapter = NativeAuthCallbackAdapter(
          callbackUri: Uri.parse(callbackUri),
          getInitialLink: () async => cold,
          linkStream: warmLinks.stream,
        );

        final subscription = adapter.callbacks.listen(received.add);
        warmLinks
          ..add(cold)
          ..add(Uri.parse('mindforge-dev://other/callback?code=wrong'))
          ..add(warm);
        await pumpEventQueue();

        expect(received, <Uri>[cold, warm]);

        await subscription.cancel();
        await warmLinks.close();
      },
    );

    test(
      'emits a valid warm callback when the initial link is absent',
      () async {
        final warmLinks = StreamController<Uri>();
        final received = <Uri>[];
        final adapter = NativeAuthCallbackAdapter(
          callbackUri: Uri.parse(callbackUri),
          getInitialLink: () async => null,
          linkStream: warmLinks.stream,
        );

        final subscription = adapter.callbacks.listen(received.add);
        warmLinks
          ..add(Uri.parse('mindforge-dev://auth/not-callback?code=wrong'))
          ..add(Uri.parse('$callbackUri?code=wrong#fragment'))
          ..add(Uri.parse('$callbackUri?error=access_denied'));
        await pumpEventQueue();

        expect(received, <Uri>[Uri.parse('$callbackUri?error=access_denied')]);

        await subscription.cancel();
        await warmLinks.close();
      },
    );

    test(
      'continues with warm callbacks when initial-link lookup fails',
      () async {
        final warmLinks = StreamController<Uri>();
        final expected = Uri.parse('$callbackUri?code=warm');
        final received = <Uri>[];
        final adapter = NativeAuthCallbackAdapter(
          callbackUri: Uri.parse(callbackUri),
          getInitialLink: () async => throw StateError('platform unavailable'),
          linkStream: warmLinks.stream,
        );

        final subscription = adapter.callbacks.listen(received.add);
        warmLinks.add(expected);
        await pumpEventQueue();

        expect(received, <Uri>[expected]);

        await subscription.cancel();
        await warmLinks.close();
      },
    );
  });
}
