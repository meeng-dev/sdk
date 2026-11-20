import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import { test } from "node:test";

import { Meeng, VERSION, WebhookVerificationError } from "../src/index.ts";

interface Seen {
  method: string;
  url: URL;
  headers: Headers;
  body: string | undefined;
}

function client(answer: (seen: Seen) => Response): { meeng: Meeng; seen: Seen[] } {
  const seen: Seen[] = [];
  const fake: typeof fetch = async (input, init) => {
    const request = { method: init?.method ?? "GET", url: new URL(String(input)), headers: new Headers(init?.headers), body: init?.body as string | undefined };
    seen.push(request);
    return answer(request);
  };
  return { meeng: new Meeng("mg_test_key", "https://meeng.test/", { fetch: fake }), seen };
}

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

test("sending an email carries the key, the body and the idempotency key", async () => {
  const { meeng, seen } = client(() => json(200, { emailId: "em_1" }));
  const { data, error } = await meeng.emails.send(
    { from: "hello@acme.com", to: ["a@x.com", "b@x.com"], subject: "Hi", html: "<p>Hi</p>", headers: { "X-Order": "1" } },
    { idempotencyKey: "order-1" },
  );
  assert.equal(error, null);
  assert.equal(data?.emailId, "em_1");
  assert.equal(seen[0]?.method, "POST");
  assert.equal(seen[0]?.url.pathname, "/api/v1/emails");
  assert.equal(seen[0]?.headers.get("authorization"), "Bearer mg_test_key");
  assert.equal(seen[0]?.headers.get("idempotency-key"), "order-1");
  assert.equal(seen[0]?.headers.get("content-type"), "application/json");
  assert.equal(seen[0]?.headers.get("user-agent"), `meeng-typescript/${VERSION}`);
  assert.deepEqual(JSON.parse(seen[0]?.body ?? ""), { from: "hello@acme.com", to: ["a@x.com", "b@x.com"], subject: "Hi", html: "<p>Hi</p>", headers: { "X-Order": "1" } });
});

test("a send without an idempotency key sends none", async () => {
  const { meeng, seen } = client(() => json(200, { emailId: "em_1" }));
  await meeng.emails.create({ from: "hello@acme.com", to: "a@x.com", subject: "Hi", text: "Hi" });
  assert.equal(seen[0]?.headers.has("idempotency-key"), false);
});

test("a batch answers the list of ids", async () => {
  const { meeng, seen } = client(() => json(200, { data: [{ emailId: "em_1" }, { emailId: "em_2" }] }));
  const { data, error } = await meeng.emails.batch([
    { from: "hello@acme.com", to: "a@x.com", subject: "One", text: "1" },
    { from: "hello@acme.com", to: "b@x.com", subject: "Two", text: "2" },
  ]);
  assert.equal(error, null);
  assert.deepEqual(data, [{ emailId: "em_1" }, { emailId: "em_2" }]);
  assert.equal(seen[0]?.url.pathname, "/api/v1/emails/batch");
  assert.equal((JSON.parse(seen[0]?.body ?? "") as unknown[]).length, 2);
});

test("listing emails sends the filters, a list once for each value", async () => {
  const { meeng, seen } = client(() => json(200, { data: [{ id: "em_1", to: "a@x.com", from: "h@acme.com", subject: "Hi", html: null, text: "Hi", createdAt: "2026-11-20T00:00:00Z", updatedAt: "2026-11-20T00:00:00Z", latestStatus: "DELIVERED", scheduledAt: null, domainId: 3 }], count: 1 }));
  const { data } = await meeng.emails.list({ page: 2, limit: 10, startDate: "2026-11-01T00:00:00Z", domainId: ["1", "2"] });
  assert.equal(data?.count, 1);
  assert.equal(data?.data[0]?.latestStatus, "DELIVERED");
  assert.equal(seen[0]?.method, "GET");
  assert.equal(seen[0]?.url.pathname, "/api/v1/emails");
  assert.deepEqual(seen[0]?.url.searchParams.getAll("domainId"), ["1", "2"]);
  assert.equal(seen[0]?.url.searchParams.get("page"), "2");
  assert.equal(seen[0]?.url.searchParams.get("limit"), "10");
  assert.equal(seen[0]?.url.searchParams.get("startDate"), "2026-11-01T00:00:00Z");
  assert.equal(seen[0]?.url.searchParams.has("endDate"), false);
});

test("a path parameter is escaped and a number is accepted", async () => {
  const { meeng, seen } = client(() => json(200, { id: 7, name: "acme.com", teamId: 1, status: "SUCCESS", publicKey: "k", createdAt: "x", updatedAt: "x", dnsRecords: [] }));
  const { data } = await meeng.domains.get(7);
  assert.equal(data?.id, 7);
  assert.equal(seen[0]?.url.pathname, "/api/v1/domains/7");
  await meeng.emails.get("a/b c");
  assert.equal(seen[1]?.url.pathname, "/api/v1/emails/a%2Fb%20c");
});

test("a list the API answers without an object around it is a list", async () => {
  const { meeng } = client(() => json(200, [{ id: "cb_1", name: "News", teamId: 1, properties: {}, variables: [], emoji: "📘", createdAt: "x", updatedAt: "x" }]));
  const { data } = await meeng.contactBooks.list();
  assert.equal(data?.length, 1);
  assert.equal(data?.[0]?.name, "News");
});

test("a delete can carry a body", async () => {
  const { meeng, seen } = client(() => json(200, { success: true, count: 2 }));
  const { data } = await meeng.contacts.bulkDelete("cb_1", { contactIds: ["c1", "c2"] });
  assert.equal(data?.count, 2);
  assert.equal(seen[0]?.method, "DELETE");
  assert.equal(seen[0]?.url.pathname, "/api/v1/contactBooks/cb_1/contacts/bulk");
  assert.deepEqual(JSON.parse(seen[0]?.body ?? ""), { contactIds: ["c1", "c2"] });
});

test("upsert takes the contact's id in the path", async () => {
  const { meeng, seen } = client(() => json(200, { contactId: "c_1" }));
  const { data } = await meeng.contacts.upsert("cb_1", "c_1", { email: "a@x.com", firstName: "Ana" });
  assert.equal(data?.contactId, "c_1");
  assert.equal(seen[0]?.method, "PUT");
  assert.equal(seen[0]?.url.pathname, "/api/v1/contactBooks/cb_1/contacts/c_1");
});

test("a refusal is the error, in every shape the API writes it", async () => {
  const cases: [number, unknown, string, string][] = [
    [409, { error: { code: "NOT_UNIQUE", message: "Idempotency-Key already used" } }, "NOT_UNIQUE", "Idempotency-Key already used"],
    [403, { error: "Forbidden" }, "INTERNAL_SERVER_ERROR", "Forbidden"],
    [400, { code: "BAD_REQUEST", message: "bad" }, "BAD_REQUEST", "bad"],
  ];
  for (const [status, body, code, message] of cases) {
    const { meeng } = client(() => json(status, body));
    const { data, error } = await meeng.emails.cancel("em_1");
    assert.equal(data, null);
    assert.equal(error?.code, code);
    assert.equal(error?.message, message);
    assert.equal(error?.status, status);
  }
  const { meeng } = client(() => new Response("<html>", { status: 502, statusText: "Bad Gateway" }));
  const { error } = await meeng.emails.cancel("em_1");
  assert.equal(error?.code, "INTERNAL_SERVER_ERROR");
  assert.equal(error?.message, "Bad Gateway");
});

test("an email's recipients may come as one string or a list", async () => {
  const { meeng } = client(() => json(200, { id: "em_1", teamId: 1, to: "a@x.com", from: "h@acme.com", subject: "Hi", html: null, text: null, createdAt: "x", updatedAt: "x", emailEvents: [] }));
  const { data } = await meeng.emails.get("em_1");
  assert.equal(data?.to, "a@x.com");
});

test("the key and the address come from the environment", () => {
  const saved = { key: process.env.MEENG_API_KEY, url: process.env.MEENG_BASE_URL };
  try {
    process.env.MEENG_API_KEY = "mg_env";
    process.env.MEENG_BASE_URL = "https://self.hosted.test";
    const meeng = new Meeng();
    assert.equal(meeng.key, "mg_env");
    assert.equal(meeng.url, "https://self.hosted.test/api");
    delete process.env.MEENG_API_KEY;
    assert.throws(() => new Meeng(), /Missing API key/);
  } finally {
    for (const [name, value] of [["MEENG_API_KEY", saved.key], ["MEENG_BASE_URL", saved.url]] as const) {
      if (value === undefined) delete process.env[name];
      else process.env[name] = value;
    }
  }
});

test("the address must be https, except to localhost", () => {
  assert.throws(() => new Meeng("k", "http://app.meeng.test"), /must be https/);
  assert.throws(() => new Meeng("k", "not a url"), /must be https/);
  assert.doesNotThrow(() => new Meeng("k", "http://localhost:3000"));
  assert.doesNotThrow(() => new Meeng("k", "http://127.0.0.1:3000"));
  assert.doesNotThrow(() => new Meeng("k", "http://[::1]:3000"));
});

const secret = "whsec_test";
const body = JSON.stringify({ id: "evt_1", type: "email.delivered", createdAt: "2026-11-20T12:00:00Z", data: { id: "em_1", status: "DELIVERED", from: "h@acme.com", to: ["a@x.com"], occurredAt: "2026-11-20T12:00:00Z" } });
const sign = (timestamp: string, text = body, key = secret) => `v1=${createHmac("sha256", key).update(`${timestamp}.${text}`).digest("hex")}`;

test("a signed webhook is verified and read as a typed event", () => {
  const now = Date.now();
  const timestamp = String(now);
  const webhooks = new Meeng("k", "https://meeng.test").webhooks(secret);
  const headers = { "X-Meeng-Signature": sign(timestamp), "X-Meeng-Timestamp": timestamp };
  assert.equal(webhooks.verify(body, { headers }), true);
  assert.equal(webhooks.verify(Buffer.from(body), { headers: new Headers(headers) }), true);
  const event = webhooks.constructEvent(body, { headers });
  assert.equal(event.type, "email.delivered");
  if (event.type === "email.delivered") {
    assert.deepEqual(event.data.to, ["a@x.com"]);
  }
});

test("the useSend headers are accepted as well, in any case", () => {
  const timestamp = String(Date.now());
  const webhooks = new Meeng("k", "https://meeng.test").webhooks(secret);
  const headers = { "x-usesend-signature": sign(timestamp), "x-usesend-timestamp": timestamp };
  assert.equal(webhooks.verify(body, { headers }), true);
});

test("a webhook is refused with the code that says why", () => {
  const now = 1_800_000_000_000;
  const timestamp = String(now);
  const webhooks = new Meeng("k", "https://meeng.test").webhooks(secret);
  const refused = (headers: Record<string, string>, options: { secret?: string; tolerance?: number; text?: string } = {}) => {
    try {
      webhooks.constructEvent(options.text ?? body, { headers, now, ...options });
    } catch (error) {
      assert.ok(error instanceof WebhookVerificationError);
      return error.code;
    }
    return "ACCEPTED";
  };
  const good = { "X-Meeng-Signature": sign(timestamp), "X-Meeng-Timestamp": timestamp };
  assert.equal(refused(good), "ACCEPTED");
  assert.equal(refused({ "X-Meeng-Timestamp": timestamp }), "MISSING_SIGNATURE");
  assert.equal(refused({ "X-Meeng-Signature": sign(timestamp) }), "MISSING_TIMESTAMP");
  assert.equal(refused({ ...good, "X-Meeng-Signature": sign(timestamp).slice(3) }), "INVALID_SIGNATURE_FORMAT");
  assert.equal(refused({ ...good, "X-Meeng-Timestamp": "yesterday" }), "INVALID_TIMESTAMP");
  const old = String(now - 6 * 60 * 1000);
  assert.equal(refused({ "X-Meeng-Signature": sign(old), "X-Meeng-Timestamp": old }), "TIMESTAMP_OUT_OF_RANGE");
  assert.equal(refused({ "X-Meeng-Signature": sign(old), "X-Meeng-Timestamp": old }, { tolerance: -1 }), "ACCEPTED");
  assert.equal(refused({ "X-Meeng-Signature": sign(old), "X-Meeng-Timestamp": old }, { tolerance: 10 * 60 * 1000 }), "ACCEPTED");
  assert.equal(refused(good, { secret: "another" }), "SIGNATURE_MISMATCH");
  assert.equal(refused(good, { text: body.replace("em_1", "em_2") }), "SIGNATURE_MISMATCH");
  const notJson = "not json";
  assert.equal(refused({ "X-Meeng-Signature": sign(timestamp, notJson), "X-Meeng-Timestamp": timestamp }, { text: notJson }), "INVALID_JSON");
  assert.equal(webhooks.verify(123 as unknown as string, { headers: good, now }), false);
});
