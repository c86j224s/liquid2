import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for BackupApi
void main() {
  final instance = Liquid2Api().getBackupApi();

  group(BackupApi, () {
    // Create SQLite backup
    //
    //Future<BackupOutputBody> createBackup(JsonObject body) async
    test('test createBackup', () async {
      // TODO
    });

  });
}
