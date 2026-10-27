import 'dart:convert';

final BigInt maximumLayerVersion = BigInt.parse('9223372036854775807');
final BigInt minimumLayerVersion = BigInt.one;
final BigInt _maximumExactJavaScriptInteger = BigInt.parse('9007199254740991');

Object? decodeJsonPreservingLargeIntegers(String source) {
  final decoded = jsonDecode(source);
  final strings = <String>{};
  _collectStrings(decoded, strings);

  var markerPrefix = '__mindforge_integer_token__';
  while (strings.any((value) => value.contains(markerPrefix))) {
    markerPrefix = '_$markerPrefix';
  }

  final replacements = <String, BigInt>{};
  final transformed = _replaceLargeIntegerTokens(
    source,
    markerPrefix,
    replacements,
  );
  if (replacements.isEmpty) return decoded;

  return jsonDecode(
    transformed,
    reviver: (key, value) =>
        value is String ? replacements[value] ?? value : value,
  );
}

BigInt parseLayerVersion(Object? value, String path) {
  final version = switch (value) {
    BigInt number => number,
    int number => BigInt.from(number),
    _ => throw FormatException('$path must be an integer JSON number.'),
  };
  if (version < minimumLayerVersion || version > maximumLayerVersion) {
    throw FormatException('$path is outside the positive Int64 range.');
  }
  return version;
}

String encodeRenameRequest({
  required String name,
  required BigInt expectedVersion,
}) {
  if (expectedVersion < minimumLayerVersion ||
      expectedVersion > maximumLayerVersion) {
    throw ArgumentError.value(expectedVersion, 'expectedVersion');
  }
  return '{"name":${jsonEncode(name)},"expected_version":$expectedVersion}';
}

String _replaceLargeIntegerTokens(
  String source,
  String markerPrefix,
  Map<String, BigInt> replacements,
) {
  final output = StringBuffer();
  var index = 0;
  while (index < source.length) {
    final character = source[index];
    if (character == '"') {
      final end = _endOfJsonString(source, index);
      output.write(source.substring(index, end));
      index = end;
      continue;
    }
    if (character == '-' || _isDigit(character)) {
      final end = _endOfJsonNumber(source, index);
      final token = source.substring(index, end);
      if (_isIntegerToken(token)) {
        final integer = BigInt.parse(token);
        if (integer.abs() > _maximumExactJavaScriptInteger) {
          final marker = '$markerPrefix${replacements.length}__';
          replacements[marker] = integer;
          output.write(jsonEncode(marker));
        } else {
          output.write(token);
        }
      } else {
        output.write(token);
      }
      index = end;
      continue;
    }
    output.write(character);
    index++;
  }
  return output.toString();
}

int _endOfJsonString(String source, int start) {
  var index = start + 1;
  while (index < source.length) {
    if (source[index] == r'\') {
      index += 2;
    } else if (source[index] == '"') {
      return index + 1;
    } else {
      index++;
    }
  }
  throw const FormatException('JSON string was not closed.');
}

int _endOfJsonNumber(String source, int start) {
  var index = start;
  if (source[index] == '-') index++;
  if (source[index] == '0') {
    index++;
  } else {
    while (index < source.length && _isDigit(source[index])) {
      index++;
    }
  }
  if (index < source.length && source[index] == '.') {
    index++;
    while (index < source.length && _isDigit(source[index])) {
      index++;
    }
  }
  if (index < source.length && (source[index] == 'e' || source[index] == 'E')) {
    index++;
    if (index < source.length &&
        (source[index] == '+' || source[index] == '-')) {
      index++;
    }
    while (index < source.length && _isDigit(source[index])) {
      index++;
    }
  }
  return index;
}

bool _isIntegerToken(String token) =>
    !token.contains('.') && !token.contains('e') && !token.contains('E');

bool _isDigit(String value) =>
    value.codeUnitAt(0) >= 0x30 && value.codeUnitAt(0) <= 0x39;

void _collectStrings(Object? value, Set<String> strings) {
  if (value is String) {
    strings.add(value);
  } else if (value is List) {
    for (final item in value) {
      _collectStrings(item, strings);
    }
  } else if (value is Map) {
    for (final entry in value.entries) {
      _collectStrings(entry.key, strings);
      _collectStrings(entry.value, strings);
    }
  }
}
