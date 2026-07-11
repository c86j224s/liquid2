import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> setDesktopViewport(WidgetTester tester) async {
  tester.view.physicalSize = const Size(1200, 900);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(() async {
    tester.view.resetPhysicalSize();
    tester.view.resetDevicePixelRatio();
  });
}
