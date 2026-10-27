import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

String _readAndroidFile(String relativePath) =>
    File(relativePath).readAsStringSync();

void main() {
  test('Android shell keeps callback and network boundaries explicit', () {
    final manifest = _readAndroidFile(
      'android/app/src/main/AndroidManifest.xml',
    );
    final gradle = _readAndroidFile('android/app/build.gradle.kts');
    final network = _readAndroidFile(
      'android/app/src/main/res/xml/network_security_config.xml',
    );

    expect(
      manifest,
      contains('<uses-permission android:name="android.permission.INTERNET"'),
    );
    expect(manifest, contains('android:allowBackup="false"'));
    expect(
      manifest,
      contains('android:networkSecurityConfig="@xml/network_security_config"'),
    );
    expect(manifest, contains('android:name="flutter_deeplinking_enabled"'));
    expect(manifest, contains('android:value="false"'));
    expect(manifest, contains('android:scheme="mindforge-dev"'));
    expect(manifest, contains('android:host="auth"'));
    expect(manifest, contains('android:path="/callback"'));
    expect(manifest, isNot(contains('android:usesCleartextTraffic="true"')));

    expect(gradle, contains('applicationId = "app.mindforge.client"'));
    expect(gradle, contains('minSdk = 24'));
    expect(network, contains('<base-config cleartextTrafficPermitted="false"'));
    expect(
      network,
      contains('<domain includeSubdomains="false">127.0.0.1</domain>'),
    );
    expect(network, isNot(contains('10.0.2.2')));
  });
}
