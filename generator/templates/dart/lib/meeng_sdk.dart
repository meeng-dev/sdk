/// meeng's SDK for Dart and Flutter.
///
/// ```dart
/// final meeng = Meeng(apiKey: 'mg_123');
/// final sent = await meeng.emails.send(
///   const EmailsSendParams(
///     from: 'hello@acme.com',
///     to: ['customer@example.com'],
///     subject: 'Hello',
///   ),
/// );
/// ```
library;

export 'src/client.dart' show Meeng;
export 'src/core.dart' show Requester;
export 'src/errors.dart' show MeengException;
export 'src/models.dart';
export 'src/services.dart';
export 'src/version.dart' show apiPrefix, apiVersion, defaultBaseUrl, sdkVersion;
export 'src/webhooks.dart';
