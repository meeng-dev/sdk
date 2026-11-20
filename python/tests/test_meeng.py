import hashlib
import hmac
import json
import os
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from unittest import mock
from urllib.parse import parse_qs, urlparse

from meeng import VERSION, Meeng, MeengHTTPError, WebhookVerificationError


class Recorder:
    def __init__(self, answer):
        self.answer = answer
        self.seen = []

    def __call__(self, method, url, headers, body, timeout):
        request = {"method": method, "url": urlparse(url), "headers": headers, "body": body}
        self.seen.append(request)
        status, payload = self.answer(request)
        raw = payload if isinstance(payload, bytes) else json.dumps(payload).encode()
        return status, {}, raw


def client(answer, **options):
    recorder = Recorder(answer)
    return Meeng("mg_test_key", "https://meeng.test/", transport=recorder, **options), recorder.seen


class MeengTest(unittest.TestCase):
    def test_sending_an_email_carries_the_key_the_body_and_the_idempotency_key(self):
        meeng, seen = client(lambda r: (200, {"emailId": "em_1"}))
        data, error = meeng.emails.send(
            {"from": "hello@acme.com", "to": ["a@x.com", "b@x.com"], "subject": "Hi", "html": "<p>Hi</p>"},
            options={"idempotency_key": "order-1"},
        )
        self.assertIsNone(error)
        self.assertEqual(data["emailId"], "em_1")
        request = seen[0]
        self.assertEqual(request["method"], "POST")
        self.assertEqual(request["url"].path, "/api/v1/emails")
        self.assertEqual(request["headers"]["Authorization"], "Bearer mg_test_key")
        self.assertEqual(request["headers"]["Idempotency-Key"], "order-1")
        self.assertEqual(request["headers"]["Content-Type"], "application/json")
        self.assertEqual(request["headers"]["User-Agent"], f"meeng-python/{VERSION}")
        self.assertEqual(json.loads(request["body"]), {"from": "hello@acme.com", "to": ["a@x.com", "b@x.com"], "subject": "Hi", "html": "<p>Hi</p>"})

    def test_a_write_with_no_body_of_its_own_sends_an_empty_object(self):
        meeng, seen = client(lambda r: (200, {"emailId": "em_1"}))
        meeng.emails.cancel("em_1")
        self.assertEqual(seen[0]["method"], "POST")
        self.assertEqual(seen[0]["url"].path, "/api/v1/emails/em_1/cancel")
        self.assertEqual(seen[0]["body"], b"{}")

    def test_a_send_without_an_idempotency_key_sends_none(self):
        meeng, seen = client(lambda r: (200, {"emailId": "em_1"}))
        meeng.emails.create({"from": "hello@acme.com", "to": "a@x.com", "subject": "Hi", "text": "Hi"})
        self.assertNotIn("Idempotency-Key", seen[0]["headers"])

    def test_a_batch_answers_the_list_of_ids(self):
        meeng, seen = client(lambda r: (200, {"data": [{"emailId": "em_1"}, {"emailId": "em_2"}]}))
        data, _ = meeng.emails.batch([
            {"from": "hello@acme.com", "to": "a@x.com", "subject": "One", "text": "1"},
            {"from": "hello@acme.com", "to": "b@x.com", "subject": "Two", "text": "2"},
        ])
        self.assertEqual(data["data"][1]["emailId"], "em_2")
        self.assertEqual(seen[0]["url"].path, "/api/v1/emails/batch")
        self.assertEqual(len(json.loads(seen[0]["body"])), 2)

    def test_listing_emails_sends_the_filters_a_list_once_for_each_value(self):
        meeng, seen = client(lambda r: (200, {"data": [], "count": 0}))
        data, _ = meeng.emails.list(page=2, limit=10, start_date="2026-11-01T00:00:00Z", domain_id=["1", "2"])
        self.assertEqual(data["count"], 0)
        query = parse_qs(seen[0]["url"].query)
        self.assertEqual(seen[0]["method"], "GET")
        self.assertEqual(seen[0]["url"].path, "/api/v1/emails")
        self.assertEqual(query["domainId"], ["1", "2"])
        self.assertEqual(query["page"], ["2"])
        self.assertEqual(query["limit"], ["10"])
        self.assertEqual(query["startDate"], ["2026-11-01T00:00:00Z"])
        self.assertNotIn("endDate", query)

    def test_a_path_parameter_is_escaped_and_a_number_is_accepted(self):
        meeng, seen = client(lambda r: (200, {"id": 7}))
        meeng.domains.get(7)
        meeng.emails.get("a/b c")
        self.assertEqual(seen[0]["url"].path, "/api/v1/domains/7")
        self.assertEqual(seen[1]["url"].geturl(), "https://meeng.test/api/v1/emails/a%2Fb%20c")

    def test_a_list_the_api_answers_without_an_object_around_it_is_a_list(self):
        meeng, _ = client(lambda r: (200, [{"id": "cb_1", "name": "News"}]))
        data, _ = meeng.contact_books.list()
        self.assertEqual(data[0]["name"], "News")
        self.assertIs(meeng.contactBooks, meeng.contact_books)

    def test_a_delete_can_carry_a_body(self):
        meeng, seen = client(lambda r: (200, {"success": True, "count": 2}))
        data, _ = meeng.contacts.bulk_delete("cb_1", {"contactIds": ["c1", "c2"]})
        self.assertEqual(data["count"], 2)
        self.assertEqual(seen[0]["method"], "DELETE")
        self.assertEqual(seen[0]["url"].path, "/api/v1/contactBooks/cb_1/contacts/bulk")
        self.assertEqual(json.loads(seen[0]["body"]), {"contactIds": ["c1", "c2"]})

    def test_upsert_takes_the_contacts_id_in_the_path(self):
        meeng, seen = client(lambda r: (200, {"contactId": "c_1"}))
        data, _ = meeng.contacts.upsert("cb_1", "c_1", {"email": "a@x.com", "firstName": "Ana"})
        self.assertEqual(data["contactId"], "c_1")
        self.assertEqual(seen[0]["method"], "PUT")
        self.assertEqual(seen[0]["url"].path, "/api/v1/contactBooks/cb_1/contacts/c_1")

    def test_a_refusal_raises_by_default_in_every_shape_the_api_writes_it(self):
        cases = [
            (409, {"error": {"code": "NOT_UNIQUE", "message": "Idempotency-Key already used"}}, "NOT_UNIQUE", "Idempotency-Key already used"),
            (403, {"error": "Forbidden"}, "INTERNAL_SERVER_ERROR", "Forbidden"),
            (400, {"code": "BAD_REQUEST", "message": "bad"}, "BAD_REQUEST", "bad"),
            (400, {"success": False, "error": {"issues": [{"code": "invalid_type", "path": ["from"], "message": "Required"}], "name": "ZodError"}}, "ZodError", "from: Required"),
            (502, b"<html>", "INTERNAL_SERVER_ERROR", "Bad Gateway"),
        ]
        for status, body, code, message in cases:
            meeng, _ = client(lambda r, status=status, body=body: (status, body))
            with self.assertRaises(MeengHTTPError) as raised:
                meeng.emails.cancel("em_1")
            self.assertEqual(raised.exception.status_code, status)
            self.assertEqual(raised.exception.error, {"code": code, "message": message})
            quiet, _ = client(lambda r, status=status, body=body: (status, body), raise_on_error=False)
            data, error = quiet.emails.cancel("em_1")
            self.assertIsNone(data)
            self.assertEqual(error, {"code": code, "message": message})

    def test_the_key_and_the_address_come_from_the_environment(self):
        with mock.patch.dict(os.environ, {"MEENG_API_KEY": "mg_env", "MEENG_BASE_URL": "https://self.hosted.test"}):
            meeng = Meeng()
            self.assertEqual(meeng.key, "mg_env")
            self.assertEqual(meeng.url, "https://self.hosted.test/api")
        with mock.patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError):
                Meeng()

    def test_the_address_must_be_https_except_to_localhost(self):
        for base in ["http://app.meeng.test", "ftp://app.meeng.test", "app.meeng.test", "http://localhost.evil.test"]:
            with self.assertRaises(ValueError, msg=base):
                Meeng("k", base)
        for base in ["https://app.meeng.test", "http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"]:
            Meeng("k", base)

    def test_the_standard_library_transport_never_follows_a_redirect(self):
        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                self.send_response(302)
                self.send_header("Location", "https://elsewhere.test/")
                self.send_header("Content-Length", "0")
                self.end_headers()

            def log_message(self, *args):
                pass

        server = HTTPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        try:
            meeng = Meeng("k", f"http://127.0.0.1:{server.server_port}", raise_on_error=False)
            data, error = meeng.emails.cancel("em_1")
            self.assertIsNone(data)
            self.assertEqual(error["code"], "INTERNAL_SERVER_ERROR")
        finally:
            server.shutdown()
            server.server_close()


SECRET = "whsec_test"
BODY = json.dumps({"id": "evt_1", "type": "email.delivered", "version": "2026-01-18", "createdAt": "2026-11-20T12:00:00Z", "teamId": 7, "attempt": 2, "data": {"id": "em_1", "status": "DELIVERED", "from": "h@acme.com", "to": ["a@x.com"], "occurredAt": "2026-11-20T12:00:00Z"}})


def sign(timestamp, text=BODY, secret=SECRET):
    return "v1=" + hmac.new(secret.encode(), f"{timestamp}.{text}".encode(), hashlib.sha256).hexdigest()


class WebhookTest(unittest.TestCase):
    def test_a_signed_webhook_is_verified_and_read_as_an_event(self):
        timestamp = str(int(time.time() * 1000))
        webhooks = Meeng("k", "https://meeng.test").webhooks(SECRET)
        headers = {"X-Meeng-Signature": sign(timestamp), "X-Meeng-Timestamp": timestamp}
        self.assertTrue(webhooks.verify(BODY, headers=headers))
        self.assertTrue(webhooks.verify(BODY.encode(), headers=headers))
        event = webhooks.construct_event(BODY, headers=headers)
        self.assertEqual(event["type"], "email.delivered")
        self.assertEqual((event["version"], event["teamId"], event["attempt"]), ("2026-01-18", 7, 2))
        self.assertEqual(event["data"]["to"], ["a@x.com"])

    def test_the_headers_are_accepted_in_any_case(self):
        timestamp = str(int(time.time() * 1000))
        webhooks = Meeng("k", "https://meeng.test").webhooks(SECRET)
        self.assertTrue(webhooks.verify(BODY, headers={"x-meeng-signature": sign(timestamp), "x-meeng-timestamp": timestamp}))

    def test_a_webhook_is_refused_with_the_code_that_says_why(self):
        now = 1_800_000_000_000
        timestamp = str(now)
        old = str(now - 6 * 60 * 1000)
        webhooks = Meeng("k", "https://meeng.test").webhooks(SECRET)
        good = {"X-Meeng-Signature": sign(timestamp), "X-Meeng-Timestamp": timestamp}
        stale = {"X-Meeng-Signature": sign(old), "X-Meeng-Timestamp": old}

        def refused(headers, text=BODY, **options):
            try:
                webhooks.construct_event(text, headers=headers, now=now, **options)
            except WebhookVerificationError as error:
                return error.code
            return "ACCEPTED"

        self.assertEqual(refused(good), "ACCEPTED")
        self.assertEqual(refused({"X-Meeng-Timestamp": timestamp}), "MISSING_SIGNATURE")
        self.assertEqual(refused({"X-Meeng-Signature": sign(timestamp)}), "MISSING_TIMESTAMP")
        self.assertEqual(refused({**good, "X-Meeng-Signature": sign(timestamp)[3:]}), "INVALID_SIGNATURE_FORMAT")
        self.assertEqual(refused({**good, "X-Meeng-Timestamp": "yesterday"}), "INVALID_TIMESTAMP")
        self.assertEqual(refused(stale), "TIMESTAMP_OUT_OF_RANGE")
        self.assertEqual(refused(stale, tolerance=-1), "ACCEPTED")
        self.assertEqual(refused(stale, tolerance=10 * 60 * 1000), "ACCEPTED")
        self.assertEqual(refused(good, secret="another"), "SIGNATURE_MISMATCH")
        self.assertEqual(refused(good, text=BODY.replace("em_1", "em_2")), "SIGNATURE_MISMATCH")
        self.assertEqual(refused({"X-Meeng-Signature": sign(timestamp, "not json"), "X-Meeng-Timestamp": timestamp}, text="not json"), "INVALID_JSON")
        self.assertEqual(refused(good, text=b"\xff"), "INVALID_BODY")


def retrying(answer, **options):
    """A client that waits a millisecond between attempts; answer gets the number of the attempt."""
    seen = []

    def transport(method, url, headers, body, timeout):
        seen.append({"method": method, "headers": headers})
        result = answer(len(seen))
        if isinstance(result, Exception):
            raise result
        status, payload = result
        return status, {"retry-after": "0"}, json.dumps(payload).encode()

    options.setdefault("retry_delay", 0.001)
    return Meeng("mg_test_key", "https://meeng.test/", transport=transport, **options), seen


LIMITED = (429, {"error": {"code": "RATE_LIMITED", "message": "slow down"}})
BUSY = (503, {"error": {"code": "INTERNAL_SERVER_ERROR", "message": "busy"}})


class RetryTest(unittest.TestCase):
    def test_a_too_many_requests_answer_is_repeated_for_any_call(self):
        meeng, seen = retrying(lambda n: LIMITED if n < 3 else (200, {"emailId": "em_1"}))
        data, error = meeng.emails.send({"from": "h@acme.com", "to": "a@x.com"})
        self.assertIsNone(error)
        self.assertEqual(data["emailId"], "em_1")
        self.assertEqual(len(seen), 3)

    def test_a_server_failure_is_repeated_only_when_repeating_cannot_do_the_work_twice(self):
        answer = lambda n: BUSY if n == 1 else (200, [])
        meeng, seen = retrying(answer)
        meeng.domains.list()
        self.assertEqual(len(seen), 2)

        meeng, seen = retrying(answer)
        with self.assertRaises(MeengHTTPError):
            meeng.emails.send({"from": "h@acme.com", "to": "a@x.com"})
        self.assertEqual(len(seen), 1)

        meeng, seen = retrying(answer)
        meeng.emails.send({"from": "h@acme.com", "to": "a@x.com"}, options={"idempotency_key": "order-1"})
        self.assertEqual(len(seen), 2)
        self.assertEqual(seen[1]["headers"]["Idempotency-Key"], "order-1")

    def test_other_refusals_are_never_repeated_retries_can_be_turned_off_and_the_last_answer_is_the_error(self):
        meeng, seen = retrying(lambda n: (409, {"error": {"code": "NOT_UNIQUE", "message": "in progress"}}))
        with self.assertRaises(MeengHTTPError):
            meeng.domains.list()
        self.assertEqual(len(seen), 1)

        meeng, seen = retrying(lambda n: LIMITED, max_retries=0)
        with self.assertRaises(MeengHTTPError):
            meeng.domains.list()
        self.assertEqual(len(seen), 1)

        meeng, seen = retrying(lambda n: LIMITED, max_retries=2)
        with self.assertRaises(MeengHTTPError) as raised:
            meeng.domains.list()
        self.assertEqual(raised.exception.error["code"], "RATE_LIMITED")
        self.assertEqual(len(seen), 3)

    def test_a_connection_that_fails_is_repeated_for_a_read_only(self):
        meeng, seen = retrying(lambda n: ConnectionResetError("reset") if n == 1 else (200, []))
        meeng.domains.list()
        self.assertEqual(len(seen), 2)

        meeng, seen = retrying(lambda n: ConnectionResetError("reset"))
        with self.assertRaises(ConnectionResetError):
            meeng.emails.cancel("em_1")
        self.assertEqual(len(seen), 1)


if __name__ == "__main__":
    unittest.main()
