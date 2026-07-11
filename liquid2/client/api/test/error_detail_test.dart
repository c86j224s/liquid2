import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';

// tests for ErrorDetail
void main() {
  final instance = ErrorDetailBuilder();
  // TODO add properties to the builder and call build()

  group(ErrorDetail, () {
    // Where the error occurred, e.g. 'body.items[3].tags' or 'path.thing-id'
    // String location
    test('to test the property `location`', () async {
      // TODO
    });

    // Error message text
    // String message
    test('to test the property `message`', () async {
      // TODO
    });

    // The value at the given location
    // JsonObject value
    test('to test the property `value`', () async {
      // TODO
    });

  });
}
