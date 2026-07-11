import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for HealthApi
void main() {
  final instance = Liquid2Api().getHealthApi();

  group(HealthApi, () {
    // Check process health
    //
    //Future<Health> getHealth() async
    test('test getHealth', () async {
      // TODO
    });

  });
}
