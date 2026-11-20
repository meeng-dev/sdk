import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';

/// Sends an email and lists the domains of the team whose key is in
/// MEENG_API_KEY.
Future<void> main() async {
  final meeng = Meeng();
  try {
    final sent = await meeng.emails.send(
      const EmailsSendParams(
        from: 'hello@acme.com',
        to: ['customer@example.com'],
        subject: 'Hello',
        html: '<p>Hello</p>',
      ),
      idempotencyKey: 'welcome-1',
    );
    stdout.writeln('sent ${sent.emailId}');

    for (final domain in await meeng.domains.list()) {
      stdout.writeln('${domain.name} ${domain.status}');
    }
  } on MeengException catch (error) {
    stderr.writeln(error);
    exitCode = 1;
  } finally {
    meeng.close();
  }
}
