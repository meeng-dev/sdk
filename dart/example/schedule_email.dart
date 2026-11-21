// Schedules an e-mail, moves it to another time and cancels it.
//
//   MEENG_API_KEY=mg_xxx dart run example/schedule_email.dart
import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';

String inHours(int hours) =>
    DateTime.now().toUtc().add(Duration(hours: hours)).toIso8601String();

Future<void> main() async {
  final meeng = Meeng();
  try {
    final sent = await meeng.emails.send(
      EmailsSendParams(
        from: 'Acme <contato@acme.com.br>',
        to: const ['ana@example.com'],
        subject: 'Lembrete: sua reunião é amanhã',
        html: '<p>Nos vemos às 10h.</p>',
        scheduledAt: inHours(24),
      ),
    );
    print('agendado: ${sent.emailId}');

    await meeng.emails.update(
      sent.emailId,
      EmailsUpdateParams(scheduledAt: inHours(48)),
    );
    print('remarcado');

    await meeng.emails.cancel(sent.emailId);
    print('cancelado');
  } on MeengException catch (error) {
    stderr.writeln('${error.status} ${error.code}: ${error.message}');
    exitCode = 1;
  } finally {
    meeng.close();
  }
}
