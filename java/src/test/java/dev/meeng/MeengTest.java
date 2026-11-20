package dev.meeng;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.security.InvalidKeyException;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.function.Function;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import org.junit.jupiter.api.Test;

class MeengTest {
    /** Answers every request with what answer says, recording what was sent. */
    private static final class Recorder implements Transport {
        final List<Request> seen = new ArrayList<>();
        private final Function<Request, Response> answer;

        Recorder(Function<Request, Response> answer) {
            this.answer = answer;
        }

        @Override
        public Response send(Request request) {
            seen.add(request);
            return answer.apply(request);
        }
    }

    private static Meeng client(Recorder recorder) {
        return Meeng.builder("mg_test_key").baseUrl("https://meeng.test/").transport(recorder).build();
    }

    private static Transport.Response json(int status, Object payload) {
        byte[] body = (payload instanceof String ? (String) payload : Json.write(payload)).getBytes(StandardCharsets.UTF_8);
        return new Transport.Response(status, Map.of("retry-after", "7"), body);
    }

    private static Map<String, Object> map(Object... pairs) {
        Map<String, Object> out = new LinkedHashMap<>();
        for (int i = 0; i < pairs.length; i += 2) {
            out.put((String) pairs[i], pairs[i + 1]);
        }
        return out;
    }

    private static Map<String, Object> body(Transport.Request request) {
        return Json.asObject(Json.parse(new String(request.getBody(), StandardCharsets.UTF_8)));
    }

    @Test
    void sendingAnEmailCarriesTheKeyTheBodyAndTheIdempotencyKey() {
        Recorder recorder = new Recorder(r -> json(200, map("emailId", "em_1")));
        EmailsSendResponse sent = client(recorder).emails().send(
                EmailsService.SendParams.builder().from("hello@acme.com").to(List.of("a@x.com", "b@x.com")).subject("Hi").html("<p>Hi</p>").build(),
                RequestOptions.builder().idempotencyKey("order-1").build());
        assertEquals("em_1", sent.getEmailId());
        Transport.Request request = recorder.seen.get(0);
        assertEquals("POST", request.getMethod());
        assertEquals("https://meeng.test/api/v1/emails", request.getUrl());
        assertEquals("Bearer mg_test_key", request.getHeaders().get("Authorization"));
        assertEquals("order-1", request.getHeaders().get("Idempotency-Key"));
        assertEquals("application/json", request.getHeaders().get("Content-Type"));
        assertEquals("meeng-java/" + Meeng.VERSION, request.getHeaders().get("User-Agent"));
        assertEquals(map("from", "hello@acme.com", "to", List.of("a@x.com", "b@x.com"), "subject", "Hi", "html", "<p>Hi</p>"), body(request));
    }

    @Test
    void aWriteWithNoBodyOfItsOwnSendsAnEmptyObject() {
        Recorder recorder = new Recorder(r -> json(200, map("emailId", "em_1")));
        client(recorder).emails().cancel("em_1");
        Transport.Request request = recorder.seen.get(0);
        assertEquals("POST", request.getMethod());
        assertEquals("https://meeng.test/api/v1/emails/em_1/cancel", request.getUrl());
        assertEquals("{}", new String(request.getBody(), StandardCharsets.UTF_8));
    }

    @Test
    void aSendWithoutAnIdempotencyKeySendsNone() {
        Recorder recorder = new Recorder(r -> json(200, map("emailId", "em_1")));
        client(recorder).emails().send(EmailsService.SendParams.builder().from("hello@acme.com").to(List.of("a@x.com")).text("Hi").build());
        assertFalse(recorder.seen.get(0).getHeaders().containsKey("Idempotency-Key"));
    }

    @Test
    void aBatchAnswersTheListOfIds() {
        Recorder recorder = new Recorder(r -> json(200, map("data", List.of(map("emailId", "em_1"), map("emailId", "em_2")))));
        EmailsBatchResponse got = client(recorder).emails().batch(List.of(
                EmailsService.BatchParams.builder().from("hello@acme.com").to(List.of("a@x.com")).subject("One").text("1").build(),
                EmailsService.BatchParams.builder().from("hello@acme.com").to(List.of("b@x.com")).subject("Two").text("2").build()));
        assertEquals("em_2", got.getData().get(1).getEmailId());
        assertEquals("https://meeng.test/api/v1/emails/batch", recorder.seen.get(0).getUrl());
        assertEquals(2, ((List<?>) Json.parse(new String(recorder.seen.get(0).getBody(), StandardCharsets.UTF_8))).size());
    }

    @Test
    void listingEmailsSendsTheFiltersAListOnceForEachValue() {
        Recorder recorder = new Recorder(r -> json(200, map("data", List.of(), "count", 0)));
        EmailsListResponse got = client(recorder).emails().list(
                EmailsService.ListParams.builder().page(2).limit(10).startDate("2026-11-01T00:00:00Z").domainId(List.of("1", "2")).build());
        assertEquals(0L, got.getCount());
        Transport.Request request = recorder.seen.get(0);
        assertEquals("GET", request.getMethod());
        assertEquals("https://meeng.test/api/v1/emails?page=2&limit=10&startDate=2026-11-01T00%3A00%3A00Z&domainId=1&domainId=2", request.getUrl());
    }

    @Test
    void aPathParameterIsEscaped() {
        Recorder recorder = new Recorder(r -> json(200, map("id", "a/b c", "teamId", 1, "to", "a@x.com", "from", "h@acme.com", "subject", "Hi", "html", null, "text", null,
                "createdAt", "x", "updatedAt", "x", "emailEvents", List.of(map("emailId", "a/b c", "status", "SENT", "createdAt", "x", "data", map("ip", "1.2.3.4"))))));
        Email email = client(recorder).emails().get("a/b c");
        assertEquals("https://meeng.test/api/v1/emails/a%2Fb%20c", recorder.seen.get(0).getUrl());
        assertEquals(List.of("a@x.com"), email.getTo());
        assertEquals(EmailEvent.STATUS_SENT, email.getEmailEvents().get(0).getStatus());
    }

    @Test
    void aListTheApiAnswersWithoutAnObjectAroundItIsAList() {
        Recorder recorder = new Recorder(r -> json(200, List.of(map("id", "cb_1", "name", "News", "teamId", 1, "properties", map("plan", "string"),
                "variables", List.of("plan"), "emoji", "x", "createdAt", "x", "updatedAt", "x", "_count", map("contacts", 4)))));
        List<ContactBook> books = client(recorder).contactBooks().list();
        assertEquals("News", books.get(0).getName());
        assertEquals(4L, books.get(0).getCount().getContacts());
    }

    @Test
    void aDeleteCanCarryABody() {
        Recorder recorder = new Recorder(r -> json(200, map("success", true, "count", 2)));
        ContactsBulkDeleteResponse got = client(recorder).contacts().bulkDelete("cb_1", ContactsService.BulkDeleteParams.builder().contactIds(List.of("c1", "c2")).build());
        assertEquals(2L, got.getCount());
        Transport.Request request = recorder.seen.get(0);
        assertEquals("DELETE", request.getMethod());
        assertEquals("https://meeng.test/api/v1/contactBooks/cb_1/contacts/bulk", request.getUrl());
        assertEquals("{\"contactIds\":[\"c1\",\"c2\"]}", new String(request.getBody(), StandardCharsets.UTF_8));
    }

    @Test
    void upsertTakesTheContactsIdInThePath() {
        Recorder recorder = new Recorder(r -> json(200, map("contactId", "c_1")));
        ContactsUpsertResponse got = client(recorder).contacts().upsert("cb_1", "c_1", ContactsService.UpsertParams.builder().email("a@x.com").firstName("Ana").build());
        assertEquals("c_1", got.getContactId());
        assertEquals("PUT", recorder.seen.get(0).getMethod());
        assertEquals("https://meeng.test/api/v1/contactBooks/cb_1/contacts/c_1", recorder.seen.get(0).getUrl());
    }

    @Test
    void aRefusalThrowsInEveryShapeTheApiWritesIt() {
        Object[][] cases = {
            {409, map("error", map("code", "NOT_UNIQUE", "message", "Idempotency-Key already used")), "NOT_UNIQUE", "Idempotency-Key already used"},
            {403, map("error", "Forbidden"), "INTERNAL_SERVER_ERROR", "Forbidden"},
            {400, map("code", "BAD_REQUEST", "message", "bad"), "BAD_REQUEST", "bad"},
            {400, map("success", false, "error", map("issues", List.of(map("code", "invalid_type", "path", List.of("from"), "message", "Required")), "name", "ZodError")), "ZodError", "from: Required"},
            {502, "<html>", "INTERNAL_SERVER_ERROR", "Bad Gateway"},
        };
        for (Object[] each : cases) {
            Recorder recorder = new Recorder(r -> json((Integer) each[0], each[1]));
            MeengException refusal = assertThrows(MeengException.class, () -> client(recorder).emails().cancel("em_1"));
            assertEquals((Integer) each[0], refusal.getStatus());
            assertEquals(each[2], refusal.getCode());
            assertEquals(7, refusal.getRetryAfter());
            assertEquals("meeng: " + each[0] + " " + each[2] + ": " + each[3], refusal.getMessage());
        }
    }

    @Test
    void anEmailsRecipientsMayComeAsOneStringOrAList() {
        Recorder recorder = new Recorder(r -> json(200, map("id", "em_1", "teamId", 1, "to", "a@x.com", "from", "h@acme.com", "subject", "Hi", "html", null, "text", null,
                "createdAt", "x", "updatedAt", "x", "emailEvents", List.of())));
        assertEquals(List.of("a@x.com"), client(recorder).emails().get("em_1").getTo());
    }

    @Test
    void theAddressMustBeHttpsExceptToLocalhost() {
        for (String base : new String[] {"http://app.meeng.test", "ftp://app.meeng.test", "app.meeng.test", "http://localhost.evil.test"}) {
            assertThrows(IllegalArgumentException.class, () -> Meeng.builder("k").baseUrl(base).build(), base);
        }
        for (String base : new String[] {"https://app.meeng.test", "http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"}) {
            Meeng.builder("k").baseUrl(base).build();
        }
        assertEquals("https://app.meeng.test/api", Meeng.builder("k").baseUrl("https://app.meeng.test/").build().getUrl());
    }

    @Test
    void theJdkTransportNeverFollowsARedirect() throws IOException {
        HttpServer server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/", exchange -> {
            exchange.getResponseHeaders().add("Location", "https://elsewhere.test/");
            exchange.sendResponseHeaders(302, -1);
            exchange.close();
        });
        server.start();
        try {
            Meeng meeng = Meeng.builder("k").baseUrl("http://127.0.0.1:" + server.getAddress().getPort()).build();
            MeengException refusal = assertThrows(MeengException.class, () -> meeng.emails().cancel("em_1"));
            assertEquals(302, refusal.getStatus());
        } finally {
            server.stop(0);
        }
    }

    private static final String SECRET = "whsec_test";
    private static final String BODY = "{\"id\":\"evt_1\",\"type\":\"email.delivered\",\"createdAt\":\"2026-11-20T12:00:00Z\",\"data\":{\"id\":\"em_1\",\"status\":\"DELIVERED\",\"from\":\"h@acme.com\",\"to\":[\"a@x.com\"],\"occurredAt\":\"2026-11-20T12:00:00Z\"}}";

    private static String sign(String timestamp, String text, String secret) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            StringBuilder out = new StringBuilder("v1=");
            for (byte each : mac.doFinal((timestamp + "." + text).getBytes(StandardCharsets.UTF_8))) {
                out.append(String.format("%02x", each));
            }
            return out.toString();
        } catch (NoSuchAlgorithmException | InvalidKeyException e) {
            throw new IllegalStateException(e);
        }
    }

    @Test
    void aSignedWebhookIsVerifiedAndReadAsATypedEvent() {
        String timestamp = Long.toString(System.currentTimeMillis());
        Webhooks webhooks = new Meeng("k").webhooks(SECRET);
        Map<String, String> headers = Map.of("X-Meeng-Signature", sign(timestamp, BODY, SECRET), "X-Meeng-Timestamp", timestamp);
        assertTrue(webhooks.verify(BODY, headers));
        WebhookEvent event = webhooks.constructEvent(BODY, headers);
        EmailWebhookEvent delivered = assertInstanceOf(EmailWebhookEvent.class, event);
        assertEquals("email.delivered", delivered.getType());
        assertEquals(List.of("a@x.com"), delivered.getData().getTo());
    }

    @Test
    void theUseSendHeadersAreAcceptedAsWellInAnyCase() {
        String timestamp = Long.toString(System.currentTimeMillis());
        Map<String, String> headers = Map.of("x-usesend-signature", sign(timestamp, BODY, SECRET), "X-UseSend-Timestamp", timestamp);
        assertTrue(new Meeng("k").webhooks(SECRET).verify(BODY, headers));
    }

    private static String refused(Webhooks webhooks, Map<String, String> headers, String text, Webhooks.Options options) {
        try {
            webhooks.constructEvent(text, headers, options);
            return "ACCEPTED";
        } catch (WebhookVerificationException e) {
            return e.getCode();
        }
    }

    @Test
    void aWebhookIsRefusedWithTheCodeThatSaysWhy() {
        Instant now = Instant.ofEpochMilli(1_800_000_000_000L);
        String timestamp = Long.toString(now.toEpochMilli());
        String old = Long.toString(now.toEpochMilli() - 6 * 60 * 1000);
        Webhooks webhooks = new Meeng("k").webhooks(SECRET);
        Webhooks.Options at = Webhooks.Options.defaults().now(now);
        Map<String, String> good = Map.of("X-Meeng-Signature", sign(timestamp, BODY, SECRET), "X-Meeng-Timestamp", timestamp);
        Map<String, String> stale = Map.of("X-Meeng-Signature", sign(old, BODY, SECRET), "X-Meeng-Timestamp", old);
        assertEquals("ACCEPTED", refused(webhooks, good, BODY, at));
        assertEquals("MISSING_SIGNATURE", refused(webhooks, Map.of("X-Meeng-Timestamp", timestamp), BODY, at));
        assertEquals("MISSING_TIMESTAMP", refused(webhooks, Map.of("X-Meeng-Signature", sign(timestamp, BODY, SECRET)), BODY, at));
        assertEquals("INVALID_SIGNATURE_FORMAT", refused(webhooks, Map.of("X-Meeng-Signature", sign(timestamp, BODY, SECRET).substring(3), "X-Meeng-Timestamp", timestamp), BODY, at));
        assertEquals("INVALID_TIMESTAMP", refused(webhooks, Map.of("X-Meeng-Signature", sign(timestamp, BODY, SECRET), "X-Meeng-Timestamp", "yesterday"), BODY, at));
        assertEquals("TIMESTAMP_OUT_OF_RANGE", refused(webhooks, stale, BODY, at));
        assertEquals("ACCEPTED", refused(webhooks, stale, BODY, Webhooks.Options.defaults().now(now).tolerance(Duration.ofMillis(-1))));
        assertEquals("ACCEPTED", refused(webhooks, stale, BODY, Webhooks.Options.defaults().now(now).tolerance(Duration.ofMinutes(10))));
        assertEquals("SIGNATURE_MISMATCH", refused(webhooks, good, BODY, Webhooks.Options.defaults().now(now).secret("another")));
        assertEquals("SIGNATURE_MISMATCH", refused(webhooks, good, BODY.replace("em_1", "em_2"), at));
        assertEquals("INVALID_JSON", refused(webhooks, Map.of("X-Meeng-Signature", sign(timestamp, "not json", SECRET), "X-Meeng-Timestamp", timestamp), "not json", at));
        assertNull(URI.create("https://meeng.test").getQuery());
    }
}
