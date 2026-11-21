// Checks a webhook the way an endpoint does: the signature and the time are
// verified against the raw body, and then the event is read as a typed value.
// The request here is made in the program; a real endpoint reads the body and the
// headers of the request it receives, and the secret from its settings.
//
//   dart run example/verify_webhook.dart
import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:meeng_sdk/meeng_sdk.dart';

void main() {
  const secret = 'whsec_exemplo';
  final body = jsonEncode({
    'id': 'call_1',
    'type': 'email.delivered',
    'version': '2026-01-18',
    'createdAt': '2026-11-20T12:00:00Z',
    'teamId': 7,
    'attempt': 1,
    'data': {
      'id': 'em_1',
      'status': 'DELIVERED',
      'from': 'contato@acme.com.br',
      'to': ['ana@example.com'],
      'occurredAt': '2026-11-20T12:00:00Z',
    },
  });
  final timestamp = '${DateTime.now().millisecondsSinceEpoch}';
  final signature = Hmac(
    sha256,
    utf8.encode(secret),
  ).convert(utf8.encode('$timestamp.$body'));
  final headers = {
    'x-meeng-signature': 'v1=$signature',
    'x-meeng-timestamp': timestamp,
  };

  try {
    final event = const Webhooks(secret).constructEvent(body, headers);
    if (event is EmailWebhookEvent) {
      print('e-mail ${event.data.id} entregue a ${event.data.to.join(', ')}');
    } else {
      print('evento ${event.type}');
    }
  } on WebhookVerificationException catch (error) {
    stderr.writeln('${error.code}: ${error.message}');
    exitCode = 1;
  }
}
