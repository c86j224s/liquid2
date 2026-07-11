import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../app/providers.dart';

class LibrarySearchField extends ConsumerStatefulWidget {
  const LibrarySearchField({super.key});

  @override
  ConsumerState<LibrarySearchField> createState() => _LibrarySearchFieldState();
}

class _LibrarySearchFieldState extends ConsumerState<LibrarySearchField> {
  final _controller = TextEditingController();
  final _focusNode = FocusNode();
  Timer? _debounce;

  @override
  void dispose() {
    _debounce?.cancel();
    _focusNode.dispose();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final query = ref.watch(
      libraryFiltersProvider.select((filters) => filters.query ?? ''),
    );
    if (!_focusNode.hasFocus && _controller.text != query) {
      _controller.value = TextEditingValue(
        text: query,
        selection: TextSelection.collapsed(offset: query.length),
      );
    }
    return TextField(
      key: const Key('library-search-field'),
      controller: _controller,
      focusNode: _focusNode,
      maxLength: 256,
      buildCounter:
          (context, {required currentLength, required isFocused, maxLength}) {
            return null;
          },
      decoration: InputDecoration(
        hintText: 'Search documents',
        prefixIcon: const Icon(Icons.search),
        suffixIcon: _controller.text.isEmpty
            ? null
            : IconButton(
                tooltip: 'Clear search',
                onPressed: _clear,
                icon: const Icon(Icons.close),
              ),
        border: const OutlineInputBorder(),
      ),
      textInputAction: TextInputAction.search,
      onChanged: (value) {
        setState(() {});
        _scheduleSearch(value);
      },
      onSubmitted: _applySearch,
    );
  }

  void _scheduleSearch(String value) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () {
      _applySearch(value);
    });
  }

  void _clear() {
    _debounce?.cancel();
    _controller.clear();
    _applySearch('');
  }

  void _applySearch(String value) {
    ref.read(libraryFiltersProvider.notifier).setQuery(value);
  }
}
