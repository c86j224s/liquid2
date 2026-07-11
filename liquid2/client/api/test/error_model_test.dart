import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';

// tests for ErrorModel
void main() {
  final instance = ErrorModelBuilder();
  // TODO add properties to the builder and call build()

  group(ErrorModel, () {
    // A human-readable explanation specific to this occurrence of the problem.
    // String detail
    test('to test the property `detail`', () async {
      // TODO
    });

    // Optional list of individual error details
    // BuiltList<ErrorDetail> errors
    test('to test the property `errors`', () async {
      // TODO
    });

    // A URI reference that identifies the specific occurrence of the problem.
    // String instance
    test('to test the property `instance`', () async {
      // TODO
    });

    // HTTP status code
    // int status
    test('to test the property `status`', () async {
      // TODO
    });

    // A short, human-readable summary of the problem type. This value should not change between occurrences of the error.
    // String title
    test('to test the property `title`', () async {
      // TODO
    });

    // A URI reference to human-readable documentation for the error.
    // String type (default value: 'about:blank')
    test('to test the property `type`', () async {
      // TODO
    });

  });
}
