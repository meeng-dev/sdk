"""The contract test calls every method of the SDK against a fake of the API that
is built from the API's contract and refuses what the contract does not allow.
check.sh starts the fake and sets MEENG_FAKE_URL."""

import os
import time
import unittest

from meeng import Meeng, MeengHTTPError

BASE = os.environ.get("MEENG_FAKE_URL")


@unittest.skipUnless(BASE, "MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test")
class ContractTest(unittest.TestCase):
    def client(self, **options):
        return Meeng("mg_contract_test", BASE, retry_delay=0.001, **options)

    def test_every_method_makes_the_call_the_contract_describes(self):
        meeng = self.client()
        with self.subTest("emails.send"):
            meeng.emails.send({"to": ["hello@example.com"], "from": "hello@example.com"}, options={"idempotency_key": "contract-emails-send"})
        with self.subTest("emails.batch"):
            meeng.emails.batch([{"to": ["hello@example.com"], "from": "hello@example.com"}], options={"idempotency_key": "contract-emails-batch"})
        with self.subTest("emails.get"):
            meeng.emails.get("id_emailid")
        with self.subTest("emails.list"):
            meeng.emails.list(page=2, limit=10, start_date="2026-11-01T00:00:00Z", end_date="2026-11-30T00:00:00Z", domain_id=["1", "2"])
        with self.subTest("emails.update"):
            meeng.emails.update("id_emailid", {"scheduledAt": "2026-12-01T09:00:00Z"})
        with self.subTest("emails.cancel"):
            meeng.emails.cancel("id_emailid")
        with self.subTest("domains.list"):
            meeng.domains.list()
        with self.subTest("domains.create"):
            meeng.domains.create({"name": "example.com", "region": "us-east-1"})
        with self.subTest("domains.verify"):
            meeng.domains.verify("id_id")
        with self.subTest("domains.get"):
            meeng.domains.get("id_id")
        with self.subTest("domains.delete"):
            meeng.domains.delete("id_id")
        with self.subTest("contacts.create"):
            meeng.contacts.create("id_contactbookid", {"email": "hello@example.com"})
        with self.subTest("contacts.list"):
            meeng.contacts.list("id_contactbookid", emails="a@example.com", page=2, limit=10, ids="c1,c2")
        with self.subTest("contacts.get"):
            meeng.contacts.get("id_contactbookid", "id_contactid")
        with self.subTest("contacts.update"):
            meeng.contacts.update("id_contactbookid", "id_contactid", {})
        with self.subTest("contacts.upsert"):
            meeng.contacts.upsert("id_contactbookid", "id_contactid", {"email": "hello@example.com"})
        with self.subTest("contacts.delete"):
            meeng.contacts.delete("id_contactbookid", "id_contactid")
        with self.subTest("contacts.bulkCreate"):
            meeng.contacts.bulk_create("id_contactbookid", [{"email": "hello@example.com"}])
        with self.subTest("contacts.bulkDelete"):
            meeng.contacts.bulk_delete("id_contactbookid", {"contactIds": ["c1", "c2"]})
        with self.subTest("contactBooks.list"):
            meeng.contact_books.list()
        with self.subTest("contactBooks.create"):
            meeng.contact_books.create({"name": "example.com"})
        with self.subTest("contactBooks.get"):
            meeng.contact_books.get("id_contactbookid")
        with self.subTest("contactBooks.update"):
            meeng.contact_books.update("id_contactbookid", {})
        with self.subTest("contactBooks.delete"):
            meeng.contact_books.delete("id_contactbookid")
        with self.subTest("campaigns.create"):
            meeng.campaigns.create({"name": "example.com", "from": "hello@example.com", "subject": "sample", "contactBookId": "id_contactbookid"})
        with self.subTest("campaigns.list"):
            meeng.campaigns.list(page=2, status="DRAFT", search="news")
        with self.subTest("campaigns.get"):
            meeng.campaigns.get("id_campaignid")
        with self.subTest("campaigns.delete"):
            meeng.campaigns.delete("id_campaignid")
        with self.subTest("campaigns.schedule"):
            meeng.campaigns.schedule("id_campaignid", {})
        with self.subTest("campaigns.pause"):
            meeng.campaigns.pause("id_campaignid")
        with self.subTest("campaigns.resume"):
            meeng.campaigns.resume("id_campaignid")
        with self.subTest("analytics.emailTimeSeries"):
            meeng.analytics.email_time_series(days="7", domain_id="1")
        with self.subTest("analytics.reputationMetrics"):
            meeng.analytics.reputation_metrics(domain_id="1")

    def test_a_refusal_is_a_typed_error(self):
        with self.assertRaises(MeengHTTPError) as raised:
            self.client().emails.get("fake-404")
        self.assertEqual(raised.exception.status_code, 404)
        self.assertEqual(raised.exception.error["code"], "NOT_FOUND")
        data, error = self.client(raise_on_error=False).emails.get("fake-404")
        self.assertIsNone(data)
        self.assertEqual(error["code"], "NOT_FOUND")

    def test_a_too_many_requests_answer_is_repeated_until_it_passes(self):
        meeng = self.client()
        meeng.domains.get("flaky-429-2")
        with self.assertRaises(MeengHTTPError) as raised:
            meeng.domains.get("fake-429")
        self.assertEqual(raised.exception.status_code, 429)
        self.assertEqual(raised.exception.error["code"], "RATE_LIMITED")

    def test_a_server_failure_is_repeated_for_a_read(self):
        self.client().emails.get("flaky-503-2")

    def test_the_same_idempotency_key_answers_the_same_email(self):
        meeng = self.client()
        key = f"order-{time.time_ns()}"
        email = {"from": "hello@example.com", "to": ["a@example.com"]}
        first, _ = meeng.emails.send(email, options={"idempotency_key": key})
        again, _ = meeng.emails.send(email, options={"idempotency_key": key})
        self.assertEqual(first["emailId"], again["emailId"])
        with self.assertRaises(MeengHTTPError) as raised:
            meeng.emails.send({**email, "to": ["b@example.com"]}, options={"idempotency_key": key})
        self.assertEqual(raised.exception.status_code, 409)
        self.assertEqual(raised.exception.error["code"], "NOT_UNIQUE")

    def test_an_email_needs_its_required_fields(self):
        with self.assertRaises(MeengHTTPError) as raised:
            self.client().emails.send({"to": "a@example.com"})  # type: ignore[typeddict-item]
        self.assertEqual(raised.exception.status_code, 400)
        self.assertEqual(raised.exception.error["code"], "ZodError")


if __name__ == "__main__":
    unittest.main()
