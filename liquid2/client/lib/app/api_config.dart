const configuredLiquid2ApiBaseUrl = String.fromEnvironment(
  'LIQUID2_API_BASE_URL',
);

const configuredLiquid2EnvironmentLabel = String.fromEnvironment(
  'LIQUID2_ENVIRONMENT_LABEL',
);

const fallbackLiquid2ApiBaseUrl = 'http://localhost:8080';

String resolveLiquid2ApiBaseUrl({Uri? pageUri}) {
  final configured = configuredLiquid2ApiBaseUrl.trim();
  if (configured.isNotEmpty) {
    return configured;
  }

  final uri = pageUri ?? Uri.base;
  if ((uri.scheme == 'http' || uri.scheme == 'https') && uri.host.isNotEmpty) {
    if (uri.hasPort && uri.port != 80 && uri.port != 443) {
      return Uri(
        scheme: uri.scheme,
        host: uri.host,
        port: 8080,
      ).toString().replaceFirst(RegExp(r'/$'), '');
    }
    return uri.origin;
  }

  return fallbackLiquid2ApiBaseUrl;
}
