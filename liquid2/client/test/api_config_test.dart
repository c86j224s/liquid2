import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/api_config.dart';

void main() {
  test('infers local web API URL from the page host', () {
    final baseUrl = resolveLiquid2ApiBaseUrl(
      pageUri: Uri.parse('http://172.30.1.5:3000/'),
    );

    expect(baseUrl, 'http://172.30.1.5:8080');
  });

  test('uses same origin when the web app is on a standard port', () {
    final baseUrl = resolveLiquid2ApiBaseUrl(
      pageUri: Uri.parse('https://docs.example.com/library'),
    );

    expect(baseUrl, 'https://docs.example.com');
  });

  test('falls back to localhost outside http web contexts', () {
    final baseUrl = resolveLiquid2ApiBaseUrl(pageUri: Uri.parse('file:///app'));

    expect(baseUrl, fallbackLiquid2ApiBaseUrl);
  });
}
