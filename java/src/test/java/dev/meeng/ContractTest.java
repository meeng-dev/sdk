package dev.meeng;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;

/**
 * The contract test calls every method of the SDK against a fake of the API that is
 * built from the API's contract and refuses what the contract does not allow.
 * check.sh starts the fake and sets MEENG_FAKE_URL.
 */
class ContractTest {
    private static Meeng client() {
        String fake = System.getenv("MEENG_FAKE_URL");
        Assumptions.assumeTrue(fake != null && !fake.isEmpty(), "MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test");
        return Meeng.builder("mg_contract_test").baseUrl(fake).retryDelay(Duration.ofMillis(1)).build();
    }

    @Test
    void everyMethodMakesTheCallTheContractDescribes() {
        Meeng meeng = client();
        List<String> failures = new ArrayList<>();
        try {
            meeng.emails().send(EmailsService.SendParams.builder().to(List.of("hello@example.com")).from("hello@example.com").build(), RequestOptions.builder().idempotencyKey("contract-emails-send").build());
        } catch (RuntimeException e) {
            failures.add("emails.send: " + e.getMessage());
        }
        try {
            meeng.emails().batch(List.of(EmailsService.BatchParams.builder().to(List.of("hello@example.com")).from("hello@example.com").build()), RequestOptions.builder().idempotencyKey("contract-emails-batch").build());
        } catch (RuntimeException e) {
            failures.add("emails.batch: " + e.getMessage());
        }
        try {
            meeng.emails().get("id_emailid");
        } catch (RuntimeException e) {
            failures.add("emails.get: " + e.getMessage());
        }
        try {
            meeng.emails().list(EmailsService.ListParams.builder().page(2).limit(10).startDate("2026-11-01T00:00:00Z").endDate("2026-11-30T00:00:00Z").domainId(List.of("1", "2")).build());
        } catch (RuntimeException e) {
            failures.add("emails.list: " + e.getMessage());
        }
        try {
            meeng.emails().update("id_emailid", EmailsService.UpdateParams.builder().scheduledAt("2026-12-01T09:00:00Z").build());
        } catch (RuntimeException e) {
            failures.add("emails.update: " + e.getMessage());
        }
        try {
            meeng.emails().cancel("id_emailid");
        } catch (RuntimeException e) {
            failures.add("emails.cancel: " + e.getMessage());
        }
        try {
            meeng.domains().list();
        } catch (RuntimeException e) {
            failures.add("domains.list: " + e.getMessage());
        }
        try {
            meeng.domains().create(DomainsService.CreateParams.builder().name("example.com").region("us-east-1").build());
        } catch (RuntimeException e) {
            failures.add("domains.create: " + e.getMessage());
        }
        try {
            meeng.domains().verify("id_id");
        } catch (RuntimeException e) {
            failures.add("domains.verify: " + e.getMessage());
        }
        try {
            meeng.domains().get("id_id");
        } catch (RuntimeException e) {
            failures.add("domains.get: " + e.getMessage());
        }
        try {
            meeng.domains().delete("id_id");
        } catch (RuntimeException e) {
            failures.add("domains.delete: " + e.getMessage());
        }
        try {
            meeng.contacts().create("id_contactbookid", ContactsService.CreateParams.builder().email("hello@example.com").build());
        } catch (RuntimeException e) {
            failures.add("contacts.create: " + e.getMessage());
        }
        try {
            meeng.contacts().list("id_contactbookid", ContactsService.ListParams.builder().emails("a@example.com").page(2).limit(10).ids("c1,c2").build());
        } catch (RuntimeException e) {
            failures.add("contacts.list: " + e.getMessage());
        }
        try {
            meeng.contacts().get("id_contactbookid", "id_contactid");
        } catch (RuntimeException e) {
            failures.add("contacts.get: " + e.getMessage());
        }
        try {
            meeng.contacts().update("id_contactbookid", "id_contactid", ContactsService.UpdateParams.builder().build());
        } catch (RuntimeException e) {
            failures.add("contacts.update: " + e.getMessage());
        }
        try {
            meeng.contacts().upsert("id_contactbookid", "id_contactid", ContactsService.UpsertParams.builder().email("hello@example.com").build());
        } catch (RuntimeException e) {
            failures.add("contacts.upsert: " + e.getMessage());
        }
        try {
            meeng.contacts().delete("id_contactbookid", "id_contactid");
        } catch (RuntimeException e) {
            failures.add("contacts.delete: " + e.getMessage());
        }
        try {
            meeng.contacts().bulkCreate("id_contactbookid", List.of(ContactsService.BulkCreateParams.builder().email("hello@example.com").build()));
        } catch (RuntimeException e) {
            failures.add("contacts.bulkCreate: " + e.getMessage());
        }
        try {
            meeng.contacts().bulkDelete("id_contactbookid", ContactsService.BulkDeleteParams.builder().contactIds(List.of("c1", "c2")).build());
        } catch (RuntimeException e) {
            failures.add("contacts.bulkDelete: " + e.getMessage());
        }
        try {
            meeng.contactBooks().list();
        } catch (RuntimeException e) {
            failures.add("contactBooks.list: " + e.getMessage());
        }
        try {
            meeng.contactBooks().create(ContactBooksService.CreateParams.builder().name("example.com").build());
        } catch (RuntimeException e) {
            failures.add("contactBooks.create: " + e.getMessage());
        }
        try {
            meeng.contactBooks().get("id_contactbookid");
        } catch (RuntimeException e) {
            failures.add("contactBooks.get: " + e.getMessage());
        }
        try {
            meeng.contactBooks().update("id_contactbookid", ContactBooksService.UpdateParams.builder().build());
        } catch (RuntimeException e) {
            failures.add("contactBooks.update: " + e.getMessage());
        }
        try {
            meeng.contactBooks().delete("id_contactbookid");
        } catch (RuntimeException e) {
            failures.add("contactBooks.delete: " + e.getMessage());
        }
        try {
            meeng.campaigns().create(CampaignsService.CreateParams.builder().name("example.com").from("hello@example.com").subject("sample").contactBookId("id_contactbookid").build());
        } catch (RuntimeException e) {
            failures.add("campaigns.create: " + e.getMessage());
        }
        try {
            meeng.campaigns().list(CampaignsService.ListParams.builder().page(2).status("DRAFT").search("news").build());
        } catch (RuntimeException e) {
            failures.add("campaigns.list: " + e.getMessage());
        }
        try {
            meeng.campaigns().get("id_campaignid");
        } catch (RuntimeException e) {
            failures.add("campaigns.get: " + e.getMessage());
        }
        try {
            meeng.campaigns().delete("id_campaignid");
        } catch (RuntimeException e) {
            failures.add("campaigns.delete: " + e.getMessage());
        }
        try {
            meeng.campaigns().schedule("id_campaignid", CampaignsService.ScheduleParams.builder().build());
        } catch (RuntimeException e) {
            failures.add("campaigns.schedule: " + e.getMessage());
        }
        try {
            meeng.campaigns().pause("id_campaignid");
        } catch (RuntimeException e) {
            failures.add("campaigns.pause: " + e.getMessage());
        }
        try {
            meeng.campaigns().resume("id_campaignid");
        } catch (RuntimeException e) {
            failures.add("campaigns.resume: " + e.getMessage());
        }
        try {
            meeng.analytics().emailTimeSeries(AnalyticsService.EmailTimeSeriesParams.builder().days("7").domainId("1").build());
        } catch (RuntimeException e) {
            failures.add("analytics.emailTimeSeries: " + e.getMessage());
        }
        try {
            meeng.analytics().reputationMetrics(AnalyticsService.ReputationMetricsParams.builder().domainId("1").build());
        } catch (RuntimeException e) {
            failures.add("analytics.reputationMetrics: " + e.getMessage());
        }
        assertTrue(failures.isEmpty(), failures.toString());
    }

    @Test
    void aRefusalIsATypedError() {
        Meeng meeng = client();
        MeengException refusal = assertThrows(MeengException.class, () -> meeng.emails().get("fake-404"));
        assertEquals(404, refusal.getStatus());
        assertEquals("NOT_FOUND", refusal.getCode());
    }

    @Test
    void aTooManyRequestsAnswerIsRepeatedUntilItPasses() {
        Meeng meeng = client();
        meeng.domains().get("flaky-429-2");
        MeengException refusal = assertThrows(MeengException.class, () -> meeng.domains().get("fake-429"));
        assertEquals(429, refusal.getStatus());
        assertEquals("RATE_LIMITED", refusal.getCode());
    }

    @Test
    void aServerFailureIsRepeatedForARead() {
        client().emails().get("flaky-503-2");
    }

    @Test
    void theSameIdempotencyKeyAnswersTheSameEmail() {
        Meeng meeng = client();
        RequestOptions options = RequestOptions.builder().idempotencyKey("order-" + System.nanoTime()).build();
        EmailsService.SendParams email = EmailsService.SendParams.builder().from("hello@example.com").to(List.of("a@example.com")).build();
        EmailsSendResponse first = meeng.emails().send(email, options);
        EmailsSendResponse again = meeng.emails().send(email, options);
        assertEquals(first.getEmailId(), again.getEmailId());
        EmailsService.SendParams other = EmailsService.SendParams.builder().from("hello@example.com").to(List.of("b@example.com")).build();
        MeengException refusal = assertThrows(MeengException.class, () -> meeng.emails().send(other, options));
        assertEquals(409, refusal.getStatus());
        assertEquals("NOT_UNIQUE", refusal.getCode());
    }
}
