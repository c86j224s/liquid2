import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../data/library_repository.dart';
import 'providers.dart';
import 'package:liquid2_api/liquid2_api.dart';

final androidServerCheckProvider = Provider<Future<void> Function(String)>((
  ref,
) {
  return (origin) async {
    final api = Liquid2Api(basePathOverride: origin);
    api.dio.options.connectTimeout = const Duration(seconds: 10);
    api.dio.options.receiveTimeout = const Duration(seconds: 10);
    try {
      final response = await api.getDocumentsApi().listDocuments();
      if (response.data?.items == null) {
        throw StateError('Invalid Liquid2 response');
      }
    } finally {
      api.dio.close(force: true);
    }
  };
});

List<String> sharedUrls(String text) =>
    RegExp(r'https?://[^\s<>"\x27]+', caseSensitive: false)
        .allMatches(text)
        .map((m) => m.group(0)!)
        .where((s) {
          final u = Uri.tryParse(s);
          return u != null && u.host.isNotEmpty && u.userInfo.isEmpty;
        })
        .toSet()
        .toList();

Future<bool> saveSharedUrl(LibraryRepository repository, String url) async {
  try {
    await repository.scrapeUrl(url: url);
    return true;
  } catch (_) {
    await repository.bookmarkUrl(url: url);
    return false;
  }
}

class AndroidShareShell extends ConsumerStatefulWidget {
  const AndroidShareShell({super.key, required this.child});
  final Widget child;
  @override
  ConsumerState<AndroidShareShell> createState() => _AndroidShareShellState();
}

class _AndroidShareShellState extends ConsumerState<AndroidShareShell> {
  static const channel = MethodChannel('liquid2/share');
  final server = TextEditingController();
  bool ready = false, busy = false, settings = false;
  bool receiving = false;
  String? url, message;
  List<String> choices = [];
  bool get supported =>
      !kIsWeb && defaultTargetPlatform == TargetPlatform.android;
  @override
  void initState() {
    super.initState();
    if (supported) {
      channel.setMethodCallHandler((call) async {
        if (call.method == 'shareAvailable') await receive();
      });
      initialize();
    }
  }

  Future<void> initialize() async {
    final saved = await channel.invokeMethod<String>('getServer') ?? '';
    if (!mounted) return;
    server.text = saved;
    if (server.text.isNotEmpty) {
      ref.read(androidServerUrlProvider.notifier).state = server.text;
    }
    if (!mounted) return;
    setState(() {
      ready = true;
      settings = server.text.isEmpty;
    });
    await receive();
  }

  Future<void> receive() async {
    if (!ready || receiving || busy || url != null || choices.isNotEmpty) {
      return;
    }
    receiving = true;
    Map<String, dynamic>? share;
    try {
      share = await channel.invokeMapMethod<String, dynamic>('takeShare');
    } finally {
      receiving = false;
    }
    if (share == null || !mounted) return;
    final urls = sharedUrls(share['text'] as String? ?? '');
    setState(() {
      choices = urls;
      url = urls.length == 1 ? urls.single : null;
      message = urls.isEmpty ? '공유한 내용에 HTTP(S) 링크가 없습니다.' : null;
    });
    if (url != null && !settings) await save();
  }

  Future<void> save() async {
    if (busy || url == null) return;
    setState(() {
      busy = true;
      message = '스크랩 중… 실패하면 링크만 저장합니다.';
    });
    try {
      final scraped = await saveSharedUrl(
        ref.read(libraryRepositoryProvider),
        url!,
      );
      if (!mounted) return;
      setState(() {
        message = scraped ? '스크랩 저장됨' : '링크만 저장됨 · 원문 스크랩 실패';
        url = null;
        choices = [];
      });
      ref.invalidate(librarySnapshotProvider);
    } catch (_) {
      if (mounted) {
        setState(
          () => message = '저장하지 못했습니다. Tailscale 연결과 서버 주소를 확인하고 재시도하세요.',
        );
      }
    } finally {
      if (mounted) {
        setState(() => busy = false);
        await receive();
      }
    }
  }

  Future<void> configure() async {
    if (busy) return;
    final u = Uri.tryParse(server.text.trim());
    if (u == null ||
        (u.scheme != 'https' && u.scheme != 'http') ||
        u.host.isEmpty ||
        u.userInfo.isNotEmpty ||
        u.hasQuery ||
        u.hasFragment ||
        !(u.path.isEmpty || u.path == '/')) {
      setState(() => message = 'HTTP 또는 HTTPS 서버 주소를 입력하세요.');
      return;
    }
    final value = u.origin;
    setState(() {
      busy = true;
      message = '서버 연결 확인 중…';
    });
    try {
      await ref.read(androidServerCheckProvider)(value);
      if (!mounted) return;
      await channel.invokeMethod('setServer', value);
      if (!mounted) return;
      ref.read(androidServerUrlProvider.notifier).state = value;
      setState(() {
        settings = false;
        message = null;
      });
    } catch (_) {
      if (mounted) {
        setState(
          () => message =
              '연결을 확인하지 못했습니다. 주소와 Tailscale 연결을 확인하고 다시 입력하세요. 설정은 저장되지 않았습니다.',
        );
      }
    } finally {
      if (mounted) setState(() => busy = false);
    }
    if (mounted && !settings) {
      if (url != null) {
        await save();
      } else {
        await receive();
      }
    }
  }

  @override
  void dispose() {
    server.dispose();
    if (supported) channel.setMethodCallHandler(null);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!supported) return widget.child;
    return Overlay.wrap(
      child: Stack(
        children: [
          if (ready && ref.watch(androidServerUrlProvider) != null)
            widget.child
          else
            const ColoredBox(
              color: Color(0xff101820),
              child: SizedBox.expand(),
            ),
          Positioned(
            right: 12,
            bottom: 12,
            child: SafeArea(
              child: FloatingActionButton.small(
                heroTag: 'server-settings',
                onPressed: busy ? null : () => setState(() => settings = true),

                child: const Icon(Icons.dns_outlined),
              ),
            ),
          ),
          if (!ready ||
              settings ||
              url != null ||
              choices.isNotEmpty ||
              message != null)
            Positioned.fill(
              child: ColoredBox(
                color: Colors.black54,
                child: Center(
                  child: SingleChildScrollView(
                    child: Card(
                      margin: const EdgeInsets.all(24),
                      child: Padding(
                        padding: const EdgeInsets.all(24),
                        child: ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 440),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              const Text(
                                'Liquid2 링크 저장',
                                style: TextStyle(
                                  fontSize: 22,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                              const SizedBox(height: 16),
                              if (!ready || busy)
                                const LinearProgressIndicator(),
                              if (settings) ...[
                                const Text(
                                  '서버 주소를 입력하세요. HTTP는 자체 암호화가 없으므로 신뢰할 수 있는 네트워크에서 사용하세요.',
                                ),
                                TextField(
                                  controller: server,
                                  enabled: !busy,
                                  keyboardType: TextInputType.url,
                                  decoration: const InputDecoration(
                                    labelText: '서버 주소',
                                  ),
                                ),
                                FilledButton(
                                  onPressed: busy ? null : configure,
                                  child: const Text('연결 설정 저장'),
                                ),
                              ],
                              if (url != null) SelectableText(url!),
                              if (choices.length > 1 && url == null) ...[
                                const Text('저장할 링크를 선택하세요.'),
                                ...choices.map(
                                  (s) => TextButton(
                                    onPressed: busy
                                        ? null
                                        : () {
                                            setState(() => url = s);
                                            if (!settings) save();
                                          },
                                    child: Text(s),
                                  ),
                                ),
                              ],
                              if (message != null)
                                Padding(
                                  padding: const EdgeInsets.symmetric(
                                    vertical: 12,
                                  ),
                                  child: Text(message!),
                                ),
                              if (!busy && url != null && !settings)
                                FilledButton(
                                  onPressed: save,
                                  child: const Text('다시 시도'),
                                ),
                              if (!busy && !settings && url != null)
                                TextButton(
                                  onPressed: () =>
                                      setState(() => settings = true),
                                  child: const Text('서버 주소 수정'),
                                ),
                              if (!busy &&
                                  ready &&
                                  ref.watch(androidServerUrlProvider) != null)
                                TextButton(
                                  onPressed: () {
                                    setState(() {
                                      settings = false;
                                      url = null;
                                      choices = [];
                                      message = null;
                                    });
                                  },
                                  child: const Text('닫기'),
                                ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
