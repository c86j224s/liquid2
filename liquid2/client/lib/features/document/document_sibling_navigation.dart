import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../app/providers.dart';
import '../library/library_keyboard_cursor.dart';

class DocumentNextIntent extends Intent {
  const DocumentNextIntent();
}

class DocumentPreviousIntent extends Intent {
  const DocumentPreviousIntent();
}

class DocumentNextAction extends _DocumentSiblingAction<DocumentNextIntent> {
  DocumentNextAction(super.onTrigger);
}

class DocumentPreviousAction
    extends _DocumentSiblingAction<DocumentPreviousIntent> {
  DocumentPreviousAction(super.onTrigger);
}

class _DocumentSiblingAction<T extends Intent> extends Action<T> {
  _DocumentSiblingAction(this.onTrigger);

  final VoidCallback onTrigger;

  @override
  bool isEnabled(T intent) => !editableHasFocus();

  @override
  void invoke(T intent) => onTrigger();
}

const _shortcuts = <ShortcutActivator, Intent>{
  SingleActivator(LogicalKeyboardKey.keyJ, shift: true): DocumentNextIntent(),
  SingleActivator(LogicalKeyboardKey.keyK, shift: true):
      DocumentPreviousIntent(),
};

class DocumentSiblingNavigation extends ConsumerStatefulWidget {
  const DocumentSiblingNavigation({
    required this.documentId,
    required this.child,
    super.key,
  });

  final String documentId;
  final Widget child;

  @override
  ConsumerState<DocumentSiblingNavigation> createState() =>
      _DocumentSiblingNavigationState();
}

class _DocumentSiblingNavigationState
    extends ConsumerState<DocumentSiblingNavigation> {
  void _move(int step) {
    final documents = ref.read(librarySnapshotProvider).value?.documents;
    if (documents == null) return;
    final current = documents.indexWhere(
      (document) => document.id == widget.documentId,
    );
    if (current < 0) return;
    final target = current + step;
    if (target < 0 || target >= documents.length) return;
    ref.read(libraryCursorProvider.notifier).jumpTo(target, documents.length);
    // replace() drops the last match before pushing an imperative one, which
    // leaves the reported uri at '/' and breaks reload and link sharing.
    context.go('/documents/${documents[target].id}');
  }

  @override
  Widget build(BuildContext context) {
    return Shortcuts(
      shortcuts: _shortcuts,
      child: Actions(
        actions: <Type, Action<Intent>>{
          DocumentNextIntent: DocumentNextAction(() => _move(1)),
          DocumentPreviousIntent: DocumentPreviousAction(() => _move(-1)),
        },
        child: Focus(autofocus: true, child: widget.child),
      ),
    );
  }
}
