import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for SettingsApi
void main() {
  final instance = Liquid2Api().getSettingsApi();

  group(SettingsApi, () {
    // Get app settings
    //
    //Future<AppSettings> getSettings() async
    test('test getSettings', () async {
      // TODO
    });

    // Update app settings
    //
    //Future<AppSettings> updateSettings(UpdateSettingsInput updateSettingsInput) async
    test('test updateSettings', () async {
      // TODO
    });

  });
}
