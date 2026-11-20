import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';

/// Creates a product and lists every product of the organization whose key
/// is in MEENG_API_KEY. With MEENG_PRODUCT_ID and MEENG_PRICE_ID set, it
/// also opens a checkout session, the server's half of the embedded checkout:
/// only the client secret goes to the buyer's browser.
Future<void> main() async {
  final meeng = Meeng(Platform.environment['MEENG_API_KEY']!);
  try {
    final product = await meeng.products.create(
      const ProductsCreateParams(name: 'Curso'),
      idempotencyKey: 'produto-curso',
    );
    stdout.writeln('created ${product.id}');

    await for (final product in meeng.products.listIter()) {
      stdout.writeln(product.name);
    }

    final productId = Platform.environment['MEENG_PRODUCT_ID'];
    final priceId = Platform.environment['MEENG_PRICE_ID'];
    if (productId != null && priceId != null) {
      final checkout = await meeng.checkouts.create(
        CheckoutsCreateParams(productId: productId, priceId: priceId),
      );
      stdout.writeln('hosted checkout ${checkout.url}');
      stdout.writeln('client secret for the browser ${checkout.clientSecret}');
      stdout.writeln('status key ${checkout.statusKey}');
    }
  } on MeengException catch (error) {
    stderr.writeln(error);
    exitCode = 1;
  } finally {
    meeng.close();
  }
}
