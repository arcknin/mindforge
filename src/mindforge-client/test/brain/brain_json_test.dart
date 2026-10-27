import 'package:flutter_test/flutter_test.dart';
import 'package:mindforge_client/brain/brain_json.dart';

void main() {
  group('Brain JSON Int64 handling', () {
    test('preserves integers above the JavaScript exact range', () {
      final decoded =
          decodeJsonPreservingLargeIntegers(
                '{"version":9007199254740993,"maximum":9223372036854775807,'
                '"text":"9007199254740993"}',
              )!
              as Map<String, dynamic>;

      expect(decoded['version'], BigInt.parse('9007199254740993'));
      expect(decoded['maximum'], BigInt.parse('9223372036854775807'));
      expect(decoded['text'], '9007199254740993');
    });

    test(
      'does not collide with existing strings or rewrite escaped content',
      () {
        const source =
            '{"text":"__mindforge_integer_token__0__ and \\"quoted\\"",'
            '"version":9007199254740993}';
        final decoded = decodeJsonPreservingLargeIntegers(source)! as Map;

        expect(decoded['text'], '__mindforge_integer_token__0__ and "quoted"');
        expect(decoded['version'], BigInt.parse('9007199254740993'));
      },
    );

    test('accepts only JSON integer values inside the signed Int64 range', () {
      expect(
        parseLayerVersion(BigInt.parse('9223372036854775807'), 'layer.version'),
        BigInt.parse('9223372036854775807'),
      );
      expect(parseLayerVersion(3, 'layer.version'), BigInt.from(3));
      for (final invalid in <Object?>[
        0,
        -1,
        1.0,
        '1',
        BigInt.parse('9223372036854775808'),
      ]) {
        expect(
          () => parseLayerVersion(invalid, 'layer.version'),
          throwsFormatException,
          reason: 'Expected $invalid to be rejected',
        );
      }
    });

    test(
      'encodes expected_version as an exact JSON number and escapes names',
      () {
        final request = encodeRenameRequest(
          name: 'layer", "expected_version":0, "extra":"value',
          expectedVersion: BigInt.parse('9223372036854775807'),
        );
        final decoded = decodeJsonPreservingLargeIntegers(request)! as Map;

        expect(request, contains('"expected_version":9223372036854775807'));
        expect(request, isNot(contains('"expected_version":"')));
        expect(decoded['name'], 'layer", "expected_version":0, "extra":"value');
        expect(
          decoded['expected_version'],
          BigInt.parse('9223372036854775807'),
        );
        expect(
          () => encodeRenameRequest(
            name: 'invalid',
            expectedVersion: BigInt.parse('9223372036854775808'),
          ),
          throwsArgumentError,
        );
      },
    );

    test('rejects malformed JSON before replacing integer tokens', () {
      expect(
        () => decodeJsonPreservingLargeIntegers('{"version":9007199254740993'),
        throwsFormatException,
      );
    });
  });
}
