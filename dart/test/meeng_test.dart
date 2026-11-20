import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:meeng_sdk/meeng_sdk.dart';
import 'package:test/test.dart';

typedef Answer = http.Response Function(http.Request request);

(Meeng, List<http.Request>) client(Answer answer) {
  final seen = <http.Request>[];
  final fake = MockClient((request) async {
    seen.add(request);
    return answer(request);
  });
  final meeng = Meeng(
    apiKey: 'mg_test_key',
    baseUrl: 'https://meeng.test/',
    httpClient: fake,
  );
  return (meeng, seen);
}

http.Response json(int status, Object? body) => http.Response(
  body is String ? body : jsonEncode(body),
  status,
  headers: {'content-type': 'application/json', 'retry-after': '7'},
);

void main() {
  test(
    'sending an email carries the key, the body and the idempotency key',
    () async {
      final (meeng, seen) = client((_) => json(200, {'emailId': 'em_1'}));
      final sent = await meeng.emails.send(
        const EmailsSendParams(
          from: 'hello@acme.com',
          to: ['a@x.com', 'b@x.com'],
          subject: 'Hi',
          html: '<p>Hi</p>',
        ),
        idempotencyKey: 'order-1',
      );
      expect(sent.emailId, 'em_1');
      final request = seen.single;
      expect(request.method, 'POST');
      expect(request.url.toString(), 'https://meeng.test/api/v1/emails');
      expect(request.headers['Authorization'], 'Bearer mg_test_key');
      expect(request.headers['Idempotency-Key'], 'order-1');
      expect(request.headers['Content-Type'], 'application/json');
      expect(request.headers['User-Agent'], 'meeng-dart/$sdkVersion');
      expect(jsonDecode(request.body), {
        'from': 'hello@acme.com',
        'to': ['a@x.com', 'b@x.com'],
        'subject': 'Hi',
        'html': '<p>Hi</p>',
      });
    },
  );

  test('a write with no body of its own sends an empty object', () async {
    final (meeng, seen) = client((_) => json(200, {'emailId': 'em_1'}));
    await meeng.emails.cancel('em_1');
    expect(seen.single.method, 'POST');
    expect(seen.single.url.path, '/api/v1/emails/em_1/cancel');
    expect(seen.single.body, '{}');
  });

  test('a send without an idempotency key sends none', () async {
    final (meeng, seen) = client((_) => json(200, {'emailId': 'em_1'}));
    await meeng.emails.send(
      const EmailsSendParams(
        from: 'hello@acme.com',
        to: ['a@x.com'],
        text: 'Hi',
      ),
    );
    expect(seen.single.headers.containsKey('Idempotency-Key'), isFalse);
  });

  test('a batch answers the list of ids', () async {
    final (meeng, seen) = client(
      (_) => json(200, {
        'data': [
          {'emailId': 'em_1'},
          {'emailId': 'em_2'},
        ],
      }),
    );
    final got = await meeng.emails.batch([
      const EmailsBatchParams(
        from: 'hello@acme.com',
        to: ['a@x.com'],
        subject: 'One',
        text: '1',
      ),
      const EmailsBatchParams(
        from: 'hello@acme.com',
        to: ['b@x.com'],
        subject: 'Two',
        text: '2',
      ),
    ]);
    expect(got.data[1].emailId, 'em_2');
    expect(seen.single.url.path, '/api/v1/emails/batch');
    expect((jsonDecode(seen.single.body) as List<Object?>).length, 2);
  });

  test('listing emails sends the filters, a list once for each value', () async {
    final (meeng, seen) = client(
      (_) => json(200, {
        'data': [
          {
            'id': 'em_1',
            'to': 'a@x.com',
            'from': 'h@acme.com',
            'subject': 'Hi',
            'html': null,
            'text': 'Hi',
            'createdAt': 'x',
            'updatedAt': 'x',
            'latestStatus': 'DELIVERED',
            'scheduledAt': null,
            'domainId': 3,
          },
        ],
        'count': 1,
      }),
    );
    final got = await meeng.emails.list(
      page: 2,
      limit: 10,
      startDate: '2026-11-01T00:00:00Z',
      domainId: ['1', '2'],
    );
    expect(got.count, 1);
    expect(got.data.single.to, ['a@x.com']);
    expect(got.data.single.latestStatus, 'DELIVERED');
    expect(
      seen.single.url.toString(),
      'https://meeng.test/api/v1/emails?page=2&limit=10&startDate=2026-11-01T00%3A00%3A00Z&domainId=1&domainId=2',
    );
  });

  test('a path parameter is escaped', () async {
    final (meeng, seen) = client(
      (_) => json(200, {
        'id': 'a/b c',
        'teamId': 1,
        'to': ['a@x.com'],
        'from': 'h@acme.com',
        'subject': 'Hi',
        'html': null,
        'text': null,
        'createdAt': 'x',
        'updatedAt': 'x',
        'emailEvents': [
          {
            'emailId': 'a/b c',
            'status': 'SENT',
            'createdAt': 'x',
            'data': {'ip': '1.2.3.4'},
          },
        ],
      }),
    );
    final email = await meeng.emails.get('a/b c');
    expect(
      seen.single.url.toString(),
      'https://meeng.test/api/v1/emails/a%2Fb%20c',
    );
    expect(email.emailEvents.single.status, 'SENT');
  });

  test(
    'a list the API answers without an object around it is a list',
    () async {
      final (meeng, _) = client(
        (_) => json(200, [
          {
            'id': 'cb_1',
            'name': 'News',
            'teamId': 1,
            'properties': {'plan': 'string'},
            'variables': ['plan'],
            'emoji': 'x',
            'createdAt': 'x',
            'updatedAt': 'x',
            '_count': {'contacts': 4},
          },
        ]),
      );
      final books = await meeng.contactBooks.list();
      expect(books.single.name, 'News');
      expect(books.single.count!.contacts, 4);
    },
  );

  test('a delete can carry a body', () async {
    final (meeng, seen) = client(
      (_) => json(200, {'success': true, 'count': 2}),
    );
    final got = await meeng.contacts.bulkDelete(
      'cb_1',
      const ContactsBulkDeleteParams(contactIds: ['c1', 'c2']),
    );
    expect(got.count, 2);
    expect(seen.single.method, 'DELETE');
    expect(seen.single.url.path, '/api/v1/contactBooks/cb_1/contacts/bulk');
    expect(seen.single.body, '{"contactIds":["c1","c2"]}');
  });

  test("upsert takes the contact's id in the path", () async {
    final (meeng, seen) = client((_) => json(200, {'contactId': 'c_1'}));
    final got = await meeng.contacts.upsert(
      'cb_1',
      'c_1',
      const ContactsUpsertParams(email: 'a@x.com', firstName: 'Ana'),
    );
    expect(got.contactId, 'c_1');
    expect(seen.single.method, 'PUT');
    expect(seen.single.url.path, '/api/v1/contactBooks/cb_1/contacts/c_1');
  });

  test('a refusal throws, in every shape the API writes it', () async {
    final cases = <(int, Object, String, String)>[
      (
        409,
        {
          'error': {
            'code': 'NOT_UNIQUE',
            'message': 'Idempotency-Key already used',
          },
        },
        'NOT_UNIQUE',
        'Idempotency-Key already used',
      ),
      (403, {'error': 'Forbidden'}, 'INTERNAL_SERVER_ERROR', 'Forbidden'),
      (400, {'code': 'BAD_REQUEST', 'message': 'bad'}, 'BAD_REQUEST', 'bad'),
      (502, '<html>', 'INTERNAL_SERVER_ERROR', 'Bad Gateway'),
    ];
    for (final (status, body, code, message) in cases) {
      final (meeng, _) = client((_) => json(status, body));
      await expectLater(
        meeng.emails.cancel('em_1'),
        throwsA(
          isA<MeengException>()
              .having((e) => e.status, 'status', status)
              .having((e) => e.code, 'code', code)
              .having((e) => e.retryAfter, 'retryAfter', 7)
              .having(
                (e) => e.toString(),
                'message',
                'meeng: $status $code: $message',
              ),
        ),
      );
    }
  });

  test("an email's recipients may come as one string or a list", () async {
    final (meeng, _) = client(
      (_) => json(200, {
        'id': 'em_1',
        'teamId': 1,
        'to': 'a@x.com',
        'from': 'h@acme.com',
        'subject': 'Hi',
        'html': null,
        'text': null,
        'createdAt': 'x',
        'updatedAt': 'x',
        'emailEvents': <Object?>[],
      }),
    );
    expect((await meeng.emails.get('em_1')).to, ['a@x.com']);
  });

  test('the key and the address come from the environment', () {
    // The test runs with MEENG_API_KEY and MEENG_BASE_URL unset.
    if (Platform.environment['MEENG_API_KEY'] == null) {
      expect(() => Meeng(), throwsArgumentError);
    }
    expect(
      Meeng(apiKey: 'k', baseUrl: 'https://self.hosted.test').url,
      'https://self.hosted.test/api',
    );
  });

  test('the address must be https, except to localhost', () {
    for (final base in [
      'http://app.meeng.test',
      'ftp://app.meeng.test',
      'app.meeng.test',
      'http://localhost.evil.test',
    ]) {
      expect(
        () => Meeng(apiKey: 'k', baseUrl: base),
        throwsArgumentError,
        reason: base,
      );
    }
    for (final base in [
      'https://app.meeng.test',
      'http://localhost:3000',
      'http://127.0.0.1:3000',
    ]) {
      Meeng(apiKey: 'k', baseUrl: base).close();
    }
  });

  test('the default client never follows a redirect', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((request) {
      request.response.statusCode = 302;
      request.response.headers.set('location', 'https://elsewhere.test/');
      request.response.close();
    });
    final meeng = Meeng(
      apiKey: 'k',
      baseUrl: 'http://127.0.0.1:${server.port}',
    );
    await expectLater(
      meeng.emails.cancel('em_1'),
      throwsA(isA<MeengException>().having((e) => e.status, 'status', 302)),
    );
    meeng.close();
    await server.close();
  });

  const secret = 'whsec_test';
  const body =
      '{"id":"evt_1","type":"email.delivered","createdAt":"2026-11-20T12:00:00Z","data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}';
  String sign(String timestamp, [String text = body, String key = secret]) =>
      'v1=${Hmac(sha256, utf8.encode(key)).convert(utf8.encode('$timestamp.$text'))}';

  test('a signed webhook is verified and read as a typed event', () {
    final timestamp = '${DateTime.now().millisecondsSinceEpoch}';
    final webhooks = Meeng(apiKey: 'k').webhooks(secret);
    final headers = {
      'X-Meeng-Signature': sign(timestamp),
      'X-Meeng-Timestamp': timestamp,
    };
    expect(webhooks.verify(body, headers), isTrue);
    final event = webhooks.constructEvent(body, headers);
    expect(event, isA<EmailWebhookEvent>());
    expect(event.type, 'email.delivered');
    expect((event as EmailWebhookEvent).data.to, ['a@x.com']);
    expect(event.data.status, 'DELIVERED');
  });

  test('the useSend headers are accepted as well, in any case', () {
    final timestamp = '${DateTime.now().millisecondsSinceEpoch}';
    final headers = {
      'x-usesend-signature': sign(timestamp),
      'X-UseSend-Timestamp': timestamp,
    };
    expect(Webhooks(secret).verify(body, headers), isTrue);
  });

  test('a webhook is refused with the code that says why', () {
    final now = DateTime.fromMillisecondsSinceEpoch(1800000000000);
    final timestamp = '${now.millisecondsSinceEpoch}';
    final old = '${now.millisecondsSinceEpoch - 6 * 60 * 1000}';
    final webhooks = Webhooks(secret);
    final good = {
      'X-Meeng-Signature': sign(timestamp),
      'X-Meeng-Timestamp': timestamp,
    };
    final stale = {'X-Meeng-Signature': sign(old), 'X-Meeng-Timestamp': old};
    String refused(
      Map<String, String> headers, {
      String text = body,
      String? secret,
      Duration? tolerance,
    }) {
      try {
        webhooks.constructEvent(
          text,
          headers,
          secret: secret,
          tolerance: tolerance,
          now: now,
        );
        return 'ACCEPTED';
      } on WebhookVerificationException catch (error) {
        return error.code;
      }
    }

    expect(refused(good), 'ACCEPTED');
    expect(refused({'X-Meeng-Timestamp': timestamp}), 'MISSING_SIGNATURE');
    expect(
      refused({'X-Meeng-Signature': sign(timestamp)}),
      'MISSING_TIMESTAMP',
    );
    expect(
      refused({
        'X-Meeng-Signature': sign(timestamp).substring(3),
        'X-Meeng-Timestamp': timestamp,
      }),
      'INVALID_SIGNATURE_FORMAT',
    );
    expect(
      refused({
        'X-Meeng-Signature': sign(timestamp),
        'X-Meeng-Timestamp': 'yesterday',
      }),
      'INVALID_TIMESTAMP',
    );
    expect(refused(stale), 'TIMESTAMP_OUT_OF_RANGE');
    expect(
      refused(stale, tolerance: const Duration(milliseconds: -1)),
      'ACCEPTED',
    );
    expect(refused(stale, tolerance: const Duration(minutes: 10)), 'ACCEPTED');
    expect(refused(good, secret: 'another'), 'SIGNATURE_MISMATCH');
    expect(
      refused(good, text: body.replaceFirst('em_1', 'em_2')),
      'SIGNATURE_MISMATCH',
    );
    expect(
      refused({
        'X-Meeng-Signature': sign(timestamp, 'not json'),
        'X-Meeng-Timestamp': timestamp,
      }, text: 'not json'),
      'INVALID_JSON',
    );
  });
}
