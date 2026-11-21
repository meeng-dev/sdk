// Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
// another app, the address from MEENG_BASE_URL.
//
//   MEENG_API_KEY=mg_xxx dart run example/send_email.dart
import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';

Future<void> main() async {
  final meeng = Meeng();
  try {
    final sent = await meeng.emails.send(
      const EmailsSendParams(
        from: 'Acme <contato@acme.com.br>',
        to: ['ana@example.com'],
        subject: 'Bem-vinda à Acme',
        html: '<p>Olá, Ana! Sua conta está pronta.</p>',
        text: 'Olá, Ana! Sua conta está pronta.',
      ),
      // A key makes the send safe to repeat: the same key with the same body
      // answers the first result again, and nothing is sent twice.
      idempotencyKey: 'boas-vindas-ana-2026-11-20',
    );
    print('e-mail na fila: ${sent.emailId}');
  } on MeengException catch (error) {
    stderr.writeln('${error.status} ${error.code}: ${error.message}');
    exitCode = 1;
  } finally {
    meeng.close();
  }
}
