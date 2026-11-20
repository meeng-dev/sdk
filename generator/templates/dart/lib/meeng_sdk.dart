/// meeng's SDK for Dart and Flutter.
///
/// ```dart
/// final meeng = Meeng(apiKey);
/// final product = await meeng.products.create(
///   const ProductsCreateParams(name: 'Curso'),
/// );
/// await for (final product in meeng.products.listIter()) { … }
/// ```
library;

export 'src/client.dart' show Meeng;
export 'src/core.dart' show Requester;
export 'src/errors.dart' show MeengException;
export 'src/models.dart';
export 'src/pagination.dart' show Page;
export 'src/services.dart';
export 'src/version.dart' show apiVersion, defaultBaseUrl, sdkVersion;
export 'src/webhooks.dart';
