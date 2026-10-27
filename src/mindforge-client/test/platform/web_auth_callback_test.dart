import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/platform/web_auth_callback.dart';

void main() {
  group('WebAuthCallbackCapture', () {
    const callbackUri = 'http://web-mind.home.arpa/login/callback';

    test(
      'captures only the configured callback and returns a clean history URL',
      () {
        final capture = WebAuthCallbackCapture.capture(
          Uri.parse('$callbackUri?code=one-time-code&state=opaque#fragment'),
          Uri.parse(callbackUri),
        );

        expect(capture, isNotNull);
        expect(capture!.callbackUri.queryParameters['code'], 'one-time-code');
        expect(capture.cleanedUrl, Uri.parse(callbackUri));
      },
    );

    test('ignores another origin or callback path', () {
      expect(
        WebAuthCallbackCapture.capture(
          Uri.parse('http://other.home.arpa/login/callback?code=secret'),
          Uri.parse(callbackUri),
        ),
        isNull,
      );
      expect(
        WebAuthCallbackCapture.capture(
          Uri.parse('http://web-mind.home.arpa/other?code=secret'),
          Uri.parse(callbackUri),
        ),
        isNull,
      );
    });
  });
}
