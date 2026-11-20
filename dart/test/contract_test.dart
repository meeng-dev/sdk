import 'dart:io';

import 'package:meeng_sdk/meeng_sdk.dart';
import 'package:test/test.dart';

/// The contract test calls every method of the SDK against a fake of the API that
/// is built from the API's contract and refuses what the contract does not allow.
/// check.sh starts the fake and sets MEENG_FAKE_URL.
void main() {
  final base = Platform.environment['MEENG_FAKE_URL'];
  final skip = base == null || base.isEmpty
      ? 'MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test'
      : false;

  Meeng client() => Meeng(
    apiKey: 'mg_contract_test',
    baseUrl: base,
    retryDelay: const Duration(milliseconds: 1),
  );

  test('every method makes the call the contract describes', () async {
    final meeng = client();
    final failures = <String>[];
    try {
      await meeng.emails.send(
        const EmailsSendParams(
          to: ['hello@example.com'],
          from: 'hello@example.com',
        ),
        idempotencyKey: 'contract-emails-send',
      );
    } on Object catch (error) {
      failures.add('emails.send: $error');
    }
    try {
      await meeng.emails.batch(const [
        EmailsBatchParams(to: ['hello@example.com'], from: 'hello@example.com'),
      ], idempotencyKey: 'contract-emails-batch');
    } on Object catch (error) {
      failures.add('emails.batch: $error');
    }
    try {
      await meeng.emails.get('id_emailid');
    } on Object catch (error) {
      failures.add('emails.get: $error');
    }
    try {
      await meeng.emails.list(
        page: 2,
        limit: 10,
        startDate: '2026-11-01T00:00:00Z',
        endDate: '2026-11-30T00:00:00Z',
        domainId: ['1', '2'],
      );
    } on Object catch (error) {
      failures.add('emails.list: $error');
    }
    try {
      await meeng.emails.update(
        'id_emailid',
        const EmailsUpdateParams(scheduledAt: '2026-12-01T09:00:00Z'),
      );
    } on Object catch (error) {
      failures.add('emails.update: $error');
    }
    try {
      await meeng.emails.cancel('id_emailid');
    } on Object catch (error) {
      failures.add('emails.cancel: $error');
    }
    try {
      await meeng.domains.list();
    } on Object catch (error) {
      failures.add('domains.list: $error');
    }
    try {
      await meeng.domains.create(
        const DomainsCreateParams(name: 'example.com', region: 'us-east-1'),
      );
    } on Object catch (error) {
      failures.add('domains.create: $error');
    }
    try {
      await meeng.domains.verify('id_id');
    } on Object catch (error) {
      failures.add('domains.verify: $error');
    }
    try {
      await meeng.domains.get('id_id');
    } on Object catch (error) {
      failures.add('domains.get: $error');
    }
    try {
      await meeng.domains.delete('id_id');
    } on Object catch (error) {
      failures.add('domains.delete: $error');
    }
    try {
      await meeng.contacts.create(
        'id_contactbookid',
        const ContactsCreateParams(email: 'hello@example.com'),
      );
    } on Object catch (error) {
      failures.add('contacts.create: $error');
    }
    try {
      await meeng.contacts.list(
        'id_contactbookid',
        emails: 'a@example.com',
        page: 2,
        limit: 10,
        ids: 'c1,c2',
      );
    } on Object catch (error) {
      failures.add('contacts.list: $error');
    }
    try {
      await meeng.contacts.get('id_contactbookid', 'id_contactid');
    } on Object catch (error) {
      failures.add('contacts.get: $error');
    }
    try {
      await meeng.contacts.update(
        'id_contactbookid',
        'id_contactid',
        const ContactsUpdateParams(),
      );
    } on Object catch (error) {
      failures.add('contacts.update: $error');
    }
    try {
      await meeng.contacts.upsert(
        'id_contactbookid',
        'id_contactid',
        const ContactsUpsertParams(email: 'hello@example.com'),
      );
    } on Object catch (error) {
      failures.add('contacts.upsert: $error');
    }
    try {
      await meeng.contacts.delete('id_contactbookid', 'id_contactid');
    } on Object catch (error) {
      failures.add('contacts.delete: $error');
    }
    try {
      await meeng.contacts.bulkCreate('id_contactbookid', const [
        ContactsBulkCreateParams(email: 'hello@example.com'),
      ]);
    } on Object catch (error) {
      failures.add('contacts.bulkCreate: $error');
    }
    try {
      await meeng.contacts.bulkDelete(
        'id_contactbookid',
        const ContactsBulkDeleteParams(contactIds: ['c1', 'c2']),
      );
    } on Object catch (error) {
      failures.add('contacts.bulkDelete: $error');
    }
    try {
      await meeng.contactBooks.list();
    } on Object catch (error) {
      failures.add('contactBooks.list: $error');
    }
    try {
      await meeng.contactBooks.create(
        const ContactBooksCreateParams(name: 'example.com'),
      );
    } on Object catch (error) {
      failures.add('contactBooks.create: $error');
    }
    try {
      await meeng.contactBooks.get('id_contactbookid');
    } on Object catch (error) {
      failures.add('contactBooks.get: $error');
    }
    try {
      await meeng.contactBooks.update(
        'id_contactbookid',
        const ContactBooksUpdateParams(),
      );
    } on Object catch (error) {
      failures.add('contactBooks.update: $error');
    }
    try {
      await meeng.contactBooks.delete('id_contactbookid');
    } on Object catch (error) {
      failures.add('contactBooks.delete: $error');
    }
    try {
      await meeng.campaigns.create(
        const CampaignsCreateParams(
          name: 'example.com',
          from: 'hello@example.com',
          subject: 'sample',
          contactBookId: 'id_contactbookid',
        ),
      );
    } on Object catch (error) {
      failures.add('campaigns.create: $error');
    }
    try {
      await meeng.campaigns.list(page: 2, status: 'DRAFT', search: 'news');
    } on Object catch (error) {
      failures.add('campaigns.list: $error');
    }
    try {
      await meeng.campaigns.get('id_campaignid');
    } on Object catch (error) {
      failures.add('campaigns.get: $error');
    }
    try {
      await meeng.campaigns.delete('id_campaignid');
    } on Object catch (error) {
      failures.add('campaigns.delete: $error');
    }
    try {
      await meeng.campaigns.schedule(
        'id_campaignid',
        const CampaignsScheduleParams(),
      );
    } on Object catch (error) {
      failures.add('campaigns.schedule: $error');
    }
    try {
      await meeng.campaigns.pause('id_campaignid');
    } on Object catch (error) {
      failures.add('campaigns.pause: $error');
    }
    try {
      await meeng.campaigns.resume('id_campaignid');
    } on Object catch (error) {
      failures.add('campaigns.resume: $error');
    }
    try {
      await meeng.analytics.emailTimeSeries(days: '7', domainId: '1');
    } on Object catch (error) {
      failures.add('analytics.emailTimeSeries: $error');
    }
    try {
      await meeng.analytics.reputationMetrics(domainId: '1');
    } on Object catch (error) {
      failures.add('analytics.reputationMetrics: $error');
    }
    meeng.close();
    expect(failures, isEmpty);
  }, skip: skip);

  test('a refusal is a typed error', () async {
    final meeng = client();
    await expectLater(
      meeng.emails.get('fake-404'),
      throwsA(
        isA<MeengException>()
            .having((e) => e.status, 'status', 404)
            .having((e) => e.code, 'code', 'NOT_FOUND'),
      ),
    );
    meeng.close();
  }, skip: skip);

  test('a too many requests answer is repeated until it passes', () async {
    final meeng = client();
    await meeng.domains.get('flaky-429-2');
    await expectLater(
      meeng.domains.get('fake-429'),
      throwsA(
        isA<MeengException>()
            .having((e) => e.status, 'status', 429)
            .having((e) => e.code, 'code', 'RATE_LIMITED'),
      ),
    );
    meeng.close();
  }, skip: skip);

  test('a server failure is repeated for a read', () async {
    final meeng = client();
    await meeng.emails.get('flaky-503-2');
    meeng.close();
  }, skip: skip);

  test('the same idempotency key answers the same email', () async {
    final meeng = client();
    final key = 'order-${DateTime.now().microsecondsSinceEpoch}';
    const email = EmailsSendParams(
      from: 'hello@example.com',
      to: ['a@example.com'],
    );
    final first = await meeng.emails.send(email, idempotencyKey: key);
    final again = await meeng.emails.send(email, idempotencyKey: key);
    expect(again.emailId, first.emailId);
    await expectLater(
      meeng.emails.send(
        const EmailsSendParams(
          from: 'hello@example.com',
          to: ['b@example.com'],
        ),
        idempotencyKey: key,
      ),
      throwsA(
        isA<MeengException>()
            .having((e) => e.status, 'status', 409)
            .having((e) => e.code, 'code', 'NOT_UNIQUE'),
      ),
    );
    meeng.close();
  }, skip: skip);
}
