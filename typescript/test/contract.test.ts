import assert from "node:assert/strict";
import { test } from "node:test";

import { Meeng } from "../src/index.ts";

// The contract test calls every method of the SDK against a fake of the API that
// is built from the API's contract and refuses what the contract does not allow.
// check.sh starts the fake and sets MEENG_FAKE_URL.

const base = process.env.MEENG_FAKE_URL;
const skip = base ? false : "MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test";

function client(): Meeng {
  return new Meeng("mg_contract_test", base, { retryDelayMs: 1 });
}

test("every method makes the call the contract describes", { skip }, async () => {
  const meeng = client();
  const answers = [
    ["emails.send", meeng.emails.send({ "to": ["hello@example.com"], "from": "hello@example.com" }, { idempotencyKey: "contract-emails-send" })],
    ["emails.batch", meeng.emails.batch([{ "to": ["hello@example.com"], "from": "hello@example.com" }], { idempotencyKey: "contract-emails-batch" })],
    ["emails.get", meeng.emails.get("id_emailid")],
    ["emails.list", meeng.emails.list({ "page": 2, "limit": 10, "startDate": "2026-11-01T00:00:00Z", "endDate": "2026-11-30T00:00:00Z", "domainId": ["1", "2"] })],
    ["emails.update", meeng.emails.update("id_emailid", { "scheduledAt": "2026-12-01T09:00:00Z" })],
    ["emails.cancel", meeng.emails.cancel("id_emailid")],
    ["domains.list", meeng.domains.list()],
    ["domains.create", meeng.domains.create({ "name": "example.com", "region": "us-east-1" })],
    ["domains.verify", meeng.domains.verify("id_id")],
    ["domains.get", meeng.domains.get("id_id")],
    ["domains.delete", meeng.domains.delete("id_id")],
    ["contacts.create", meeng.contacts.create("id_contactbookid", { "email": "hello@example.com" })],
    ["contacts.list", meeng.contacts.list("id_contactbookid", { "emails": "a@example.com", "page": 2, "limit": 10, "ids": "c1,c2" })],
    ["contacts.get", meeng.contacts.get("id_contactbookid", "id_contactid")],
    ["contacts.update", meeng.contacts.update("id_contactbookid", "id_contactid", {})],
    ["contacts.upsert", meeng.contacts.upsert("id_contactbookid", "id_contactid", { "email": "hello@example.com" })],
    ["contacts.delete", meeng.contacts.delete("id_contactbookid", "id_contactid")],
    ["contacts.bulkCreate", meeng.contacts.bulkCreate("id_contactbookid", [{ "email": "hello@example.com" }])],
    ["contacts.bulkDelete", meeng.contacts.bulkDelete("id_contactbookid", { "contactIds": ["c1", "c2"] })],
    ["contactBooks.list", meeng.contactBooks.list()],
    ["contactBooks.create", meeng.contactBooks.create({ "name": "example.com" })],
    ["contactBooks.get", meeng.contactBooks.get("id_contactbookid")],
    ["contactBooks.update", meeng.contactBooks.update("id_contactbookid", {})],
    ["contactBooks.delete", meeng.contactBooks.delete("id_contactbookid")],
    ["campaigns.create", meeng.campaigns.create({ "name": "example.com", "from": "hello@example.com", "subject": "sample", "contactBookId": "id_contactbookid" })],
    ["campaigns.list", meeng.campaigns.list({ "page": 2, "status": "DRAFT", "search": "news" })],
    ["campaigns.get", meeng.campaigns.get("id_campaignid")],
    ["campaigns.delete", meeng.campaigns.delete("id_campaignid")],
    ["campaigns.schedule", meeng.campaigns.schedule("id_campaignid", {})],
    ["campaigns.pause", meeng.campaigns.pause("id_campaignid")],
    ["campaigns.resume", meeng.campaigns.resume("id_campaignid")],
    ["analytics.emailTimeSeries", meeng.analytics.emailTimeSeries({ "days": "7", "domainId": "1" })],
    ["analytics.reputationMetrics", meeng.analytics.reputationMetrics({ "domainId": "1" })],
  ] as const;
  for (const [name, answer] of answers) {
    const { error } = await answer;
    assert.equal(error, null, `${name}: ${JSON.stringify(error)}`);
  }
});

test("a refusal is a typed error", { skip }, async () => {
  const { data, error } = await client().emails.get("fake-404");
  assert.equal(data, null);
  assert.equal(error?.status, 404);
  assert.equal(error?.code, "NOT_FOUND");
});

test("a too many requests answer is repeated until it passes", { skip }, async () => {
  const meeng = client();
  assert.equal((await meeng.domains.get("flaky-429-2")).error, null);
  const { error } = await meeng.domains.get("fake-429");
  assert.equal(error?.status, 429);
  assert.equal(error?.code, "RATE_LIMITED");
});

test("a server failure is repeated for a read", { skip }, async () => {
  assert.equal((await client().emails.get("flaky-503-2")).error, null);
});

test("the same idempotency key answers the same email", { skip }, async () => {
  const meeng = client();
  const key = `order-${Date.now()}`;
  const email = { from: "hello@example.com", to: ["a@example.com"] };
  const first = await meeng.emails.send(email, { idempotencyKey: key });
  const again = await meeng.emails.send(email, { idempotencyKey: key });
  assert.equal(first.error, null);
  assert.equal(again.data?.emailId, first.data?.emailId);
  const other = await meeng.emails.send({ ...email, to: ["b@example.com"] }, { idempotencyKey: key });
  assert.equal(other.error?.status, 409);
  assert.equal(other.error?.code, "NOT_UNIQUE");
});

test("an email needs its required fields", { skip }, async () => {
  const { error } = await client().emails.send({ to: "a@example.com" } as never);
  assert.equal(error?.status, 400);
  assert.equal(error?.code, "ZodError");
});
