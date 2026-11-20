// Sends up to 100 e-mails in one request.
//
//   MEENG_API_KEY=mg_xxx dart run examples/send_batch.dart
import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';

Future<void> main() async {
  final meeng = Meeng();
  const people = {
    'Ana': 'ana@example.com',
    'Bruno': 'bruno@example.com',
    'Carla': 'carla@example.com',
  };
  try {
    final sent = await meeng.emails.batch([
      for (final MapEntry(key: name, value: email) in people.entries)
        EmailsBatchParams(
          from: 'Acme <contato@acme.com.br>',
          to: [email],
          subject: 'Seu extrato de novembro',
          html: '<p>Olá, $name! Seu extrato está disponível.</p>',
        ),
    ], idempotencyKey: 'extrato-2026-11');
    for (final (index, email) in sent.data.indexed) {
      print('${index + 1}: ${email.emailId}');
    }
  } on MeengException catch (error) {
    stderr.writeln('${error.status} ${error.code}: ${error.message}');
    exitCode = 1;
  } finally {
    meeng.close();
  }
}
