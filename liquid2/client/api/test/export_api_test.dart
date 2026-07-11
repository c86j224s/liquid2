import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for ExportApi
void main() {
  final instance = Liquid2Api().getExportApi();

  group(ExportApi, () {
    // Create markdown export
    //
    //Future<ExportOutputBody> createExport(CreateExportInputBody createExportInputBody) async
    test('test createExport', () async {
      // TODO
    });

    // Get export metadata
    //
    //Future<ExportOutputBody> getExport(String id) async
    test('test getExport', () async {
      // TODO
    });

  });
}
