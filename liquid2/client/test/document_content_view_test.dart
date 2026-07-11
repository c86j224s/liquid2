import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/features/document/document_content_view.dart';

void main() {
  testWidgets('renders markdown content through MarkdownBody', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(
            format: 'markdown',
            content: '# Heading\n\n[Link](https://example.com)',
          ),
        ],
      ),
    );

    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));
    expect(markdown.data, '# Heading\n\n[Link](https://example.com)');
    expect(markdown.selectable, isTrue);
    expect(markdown.softLineBreak, isTrue);
  });

  testWidgets('treats text markdown content type as markdown', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(format: 'text/markdown', content: 'First line\nSecond line'),
        ],
      ),
    );

    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));
    expect(markdown.data, 'First line\nSecond line');
    expect(markdown.softLineBreak, isTrue);
  });

  testWidgets('renders text content through selectable text', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [_content(format: 'text', content: 'Plain body')],
      ),
    );

    expect(find.byType(MarkdownBody), findsNothing);
    expect(find.widgetWithText(SelectableText, 'Plain body'), findsOneWidget);
  });

  testWidgets('preserves text content line breaks', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(format: 'text', content: 'First paragraph\n\nSecond line'),
        ],
      ),
    );

    expect(
      find.widgetWithText(SelectableText, 'First paragraph\n\nSecond line'),
      findsOneWidget,
    );
  });

  testWidgets('keeps html content on the readable text path', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(
            format: 'html',
            content:
                '<ul><li>Build <strong>small</strong> tools &amp; ship</li></ul>',
          ),
        ],
      ),
    );

    expect(find.text('- Build small tools & ship'), findsOneWidget);
    expect(find.textContaining('<strong>'), findsNothing);
    expect(find.textContaining('&amp;'), findsNothing);
  });

  testWidgets('preserves html paragraph breaks', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(format: 'html', content: '<p>First</p><p>Second</p>'),
        ],
      ),
    );

    expect(find.text('First\n\nSecond'), findsOneWidget);
  });

  testWidgets('shows empty content state', (tester) async {
    await tester.pumpWidget(const _TestHost(contents: []));

    expect(find.text('No content captured yet.'), findsOneWidget);
  });

  testWidgets('reports failed markdown link launches', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(format: 'markdown', content: '[Link](https://example.com)'),
        ],
        launchLink: (_) async => false,
      ),
    );

    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));
    markdown.onTapLink?.call('Link', 'https://example.com', '');
    await tester.pump();

    expect(find.textContaining('Unable to open Markdown link'), findsOneWidget);
  });

  testWidgets('ignores non-http markdown link schemes', (tester) async {
    var launched = false;
    await tester.pumpWidget(
      _TestHost(
        contents: [
          _content(format: 'markdown', content: '[Bad](javascript:alert(1))'),
        ],
        launchLink: (_) async {
          launched = true;
          return true;
        },
      ),
    );

    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));
    markdown.onTapLink?.call('Bad', 'javascript:alert(1)', '');
    await tester.pump();

    expect(launched, isFalse);
    expect(find.byType(SnackBar), findsNothing);
  });
}

class _TestHost extends StatelessWidget {
  const _TestHost({required this.contents, this.launchLink});

  final List<DocumentContent> contents;
  final DocumentLinkLauncher? launchLink;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        body: DocumentContentView(
          contents: contents,
          launchLink: launchLink ?? _defaultLaunchLink,
        ),
      ),
    );
  }
}

Future<bool> _defaultLaunchLink(Uri url) async => true;

DocumentContent _content({required String format, required String content}) {
  return DocumentContent(
    (b) => b
      ..id = 'content_1'
      ..role = 'extracted'
      ..format = format
      ..content = content,
  );
}
