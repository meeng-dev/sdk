// Adds a sending domain, prints the DNS records to publish and asks for them to
// be checked.
//
//   MEENG_API_KEY=mg_xxx dart run example/add_domain.dart
import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';

Future<void> main() async {
  final meeng = Meeng();
  try {
    final domain = await meeng.domains.create(
      const DomainsCreateParams(name: 'acme.com.br', region: 'us-east-1'),
    );
    print('publique estes registros no DNS de ${domain.name}');
    for (final record in domain.dnsRecords) {
      print(
        '  ${record.type.padRight(5)} ${record.name.padRight(40)} ${record.value}',
      );
    }

    await meeng.domains.verify('${domain.id}');
    final checked = await meeng.domains.get('${domain.id}');
    print('situação: ${checked.status}');
  } on MeengException catch (error) {
    stderr.writeln('${error.status} ${error.code}: ${error.message}');
    exitCode = 1;
  } finally {
    meeng.close();
  }
}
