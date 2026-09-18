# Liquid2 Flutter client

Web/macOS UI plus Android share-receiver support (v0.18, issue #504).

## Android tester build

Use the installed Android SDK and JDK 17. Do not commit SDK paths or server
hostnames. Set ANDROID_HOME/JAVA_HOME in your local shell as necessary.

From the parent `liquid2/` directory, use the existing Makefile:

```sh
make android-dev-apk       # Liquid2 DEV / kr.smpl.liquid2.dev
make android-release-apk   # Liquid2 / kr.smpl.liquid2
make android-apks          # both, sequentially
```

Flutter must be on PATH, or pass `FLUTTER=/path/to/flutter`. These targets inherit
SDK/JDK settings from your shell and never pass a server URL. Outputs are
`client/build/app/outputs/flutter-apk/app-dev-release.apk` and
`client/build/app/outputs/flutter-apk/app-prod-release.apk`. Both currently use
development signing, including the production flavor.

For a GitHub pre-release tester asset, build from the parent directory after the
release source commit is fixed:

```sh
make android-release-asset RELEASE_VERSION=0.18.0 RELEASE_BUILD_NUMBER=5
# After the matching GitHub pre-release exists:
make android-release-asset RELEASE_VERSION=0.18.0 RELEASE_BUILD_NUMBER=5 UPLOAD=1
```

This command always runs `flutter clean`, injects version name/code, verifies the
`kr.smpl.liquid2` package and Android debug certificate, and writes
`build/releases/vX.Y.Z/liquid2-android-vX.Y.Z.apk` plus its `.sha256` file. It
uses local Android SDK tools and `gh`; no GitHub workflow builds or uploads the
APK. Keep build numbers increasing across published APKs so Android accepts
updates.

Equivalent direct development build:

```sh
flutter pub get
flutter analyze
flutter test
flutter build apk --release --flavor dev \
  --dart-define=LIQUID2_ENVIRONMENT_LABEL=DEV
```

Android requires an explicit flavor: `dev` installs as `Liquid2 DEV` with ID
`kr.smpl.liquid2.dev`; `prod` installs as `Liquid2` with ID
`kr.smpl.liquid2`. Both can coexist with separate settings. Build the
production flavor with `--flavor prod` and without the DEV define. Older previews
used the production ID, so the new dev app requires server setup again.

This build currently uses the development signing key for sideload testing, not
Play Store distribution. Keep the key stable for updates. AGP 8.11.1 with the
explicit Kotlin plugin supports the existing file_picker/lifecycle plugins;
AGP 9 built-in Kotlin support is inconsistent across the current plugin set.

Android starts without a server default, including development builds. Do not
inject private server URLs into APK builds. On first launch, enter an HTTP or HTTPS origin;
the app validates a read-only Liquid2 documents API response before persisting it.
Failed checks keep the input editable and do not save the address. No library or
share-save request runs before setup. Previously saved user settings survive updates.
Enable Tailscale on the phone before checking a tailnet server. The server button
in the app changes the saved origin. HTTP supports tailnet IPs and explicit ports.
Cleartext traffic is enabled for user-configured HTTP servers; use a trusted network
such as Tailscale. HTTPS certificate verification remains enabled. The app does not configure Tailscale,
Serve, Funnel, certificates or server authentication.

Share text/plain containing one HTTP(S) URL to Liquid2: scrape is attempted first;
on failure the same URL is bookmarked. Success distinguishes scraped content from
link-only saving. If both operations fail, the URL remains available for retry.
Multiple distinct URLs require selection. Cold-start and warm ACTION_SEND intents
are supported; warm shares are queued in memory while another share is processed.
The queue is not a durable offline sync system. Process termination can lose an
in-progress share. Response-loss after a server commit can also produce duplicate
URLs on fallback/retry because the existing ingestion API is not idempotent.

## Android QR input (#500)

The library toolbar has an Android-only camera button next to +. It uses
mobile_scanner's bundled on-device Android barcode model, not a remote recognition
service. Scan one HTTP(S) URL to automatically scrape it, falling back to bookmark
saving through the same function as Android sharing. Success refreshes the library;
if both operations fail, the URL and a retry button remain in a dialog.
Invalid QR contents remain on the scanner with guidance. Close/back cancels; the
scanner owns camera permission, background pause/resume and disposal. If permission
is denied, enable camera access in Android app settings and reopen the scanner.
Mobile web scanning and HTTPS provisioning are not part of this feature.

Device acceptance: verify permission denial and grant, invalid QR, first valid
URL input, cancellation, background/resume, and explicit link/page saving. A built
APK and widget tests are not a substitute for real camera validation.

Existing Web/macOS entrypoints are retained. Android sharing is enabled explicitly
at application startup; desktop/web rendering bypasses the native bridge.
