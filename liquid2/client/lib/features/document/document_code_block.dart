import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:markdown/markdown.dart' as md;

import '../../app/app_theme.dart';

const documentCodeBlockScrollKey = Key('document-code-block-scroll');

final documentMarkdownBuilders = <String, MarkdownElementBuilder>{
  'pre': ScrollableCodeBlockBuilder(),
};

class ScrollableCodeBlockBuilder extends MarkdownElementBuilder {
  @override
  bool isBlockElement() => true;

  @override
  Widget visitElementAfterWithContext(
    BuildContext context,
    md.Element element,
    TextStyle? preferredStyle,
    TextStyle? parentStyle,
  ) {
    return _HorizontalCodeBlock(
      code: element.textContent.replaceFirst(RegExp(r'\n$'), ''),
      style: preferredStyle,
    );
  }
}

class _HorizontalCodeBlock extends StatefulWidget {
  const _HorizontalCodeBlock({required this.code, required this.style});

  final String code;
  final TextStyle? style;

  @override
  State<_HorizontalCodeBlock> createState() => _HorizontalCodeBlockState();
}

class _HorizontalCodeBlockState extends State<_HorizontalCodeBlock> {
  final _scrollController = ScrollController();

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final style = widget.style ?? DefaultTextStyle.of(context).style;
    return Container(
      width: double.infinity,
      clipBehavior: Clip.hardEdge,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(color: theme.colorScheme.outlineVariant),
        borderRadius: const BorderRadius.all(AppRadius.sm),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final textWidth = _codeTextWidth(context, widget.code, style);
          return Scrollbar(
            controller: _scrollController,
            thumbVisibility: true,
            notificationPredicate: (notification) =>
                notification.metrics.axis == Axis.horizontal,
            child: SingleChildScrollView(
              key: documentCodeBlockScrollKey,
              controller: _scrollController,
              scrollDirection: Axis.horizontal,
              primary: false,
              padding: const EdgeInsets.all(AppSpacing.md),
              child: SizedBox(
                width: math.max(constraints.maxWidth, textWidth),
                child: Text(
                  widget.code,
                  softWrap: false,
                  overflow: TextOverflow.visible,
                  style: style,
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

double _codeTextWidth(BuildContext context, String code, TextStyle style) {
  final painter = TextPainter(
    text: TextSpan(text: code, style: style),
    textDirection: Directionality.of(context),
    textScaler: MediaQuery.textScalerOf(context),
  )..layout();
  return painter.width.ceilToDouble();
}
