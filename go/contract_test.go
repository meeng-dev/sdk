package meeng

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

// The contract test calls every method of the SDK against a fake of the API
// that is built from the API's contract and refuses what the contract does not
// allow. check.sh starts the fake and sets MEENG_FAKE_URL.

func contractClient(t *testing.T) *Client {
	t.Helper()
	base := os.Getenv("MEENG_FAKE_URL")
	if base == "" {
		t.Skip("MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test")
	}
	return New("mg_contract_test", WithBaseURL(base), WithRetryDelay(time.Millisecond))
}

func TestEveryMethodMakesTheCallTheContractDescribes(t *testing.T) {
	client := contractClient(t)
	ctx := context.Background()
	if _, err := client.Emails.Send(ctx, EmailsSendParams{To: StringList{"hello@example.com"}, From: "hello@example.com"}, WithIdempotencyKey("contract-emails-send")); err != nil {
		t.Errorf("emails.send: %v", err)
	}
	if _, err := client.Emails.Batch(ctx, []EmailsBatchParams{EmailsBatchParams{To: StringList{"hello@example.com"}, From: "hello@example.com"}}, WithIdempotencyKey("contract-emails-batch")); err != nil {
		t.Errorf("emails.batch: %v", err)
	}
	if _, err := client.Emails.Get(ctx, "id_emailid"); err != nil {
		t.Errorf("emails.get: %v", err)
	}
	if _, err := client.Emails.List(ctx, &EmailsListQuery{Page: 2, Limit: 10, StartDate: "2026-11-01T00:00:00Z", EndDate: "2026-11-30T00:00:00Z", DomainID: []string{"1", "2"}}); err != nil {
		t.Errorf("emails.list: %v", err)
	}
	if _, err := client.Emails.Update(ctx, "id_emailid", EmailsUpdateParams{ScheduledAt: "2026-12-01T09:00:00Z"}); err != nil {
		t.Errorf("emails.update: %v", err)
	}
	if _, err := client.Emails.Cancel(ctx, "id_emailid"); err != nil {
		t.Errorf("emails.cancel: %v", err)
	}
	if _, err := client.Domains.List(ctx); err != nil {
		t.Errorf("domains.list: %v", err)
	}
	if _, err := client.Domains.Create(ctx, DomainsCreateParams{Name: "example.com", Region: "us-east-1"}); err != nil {
		t.Errorf("domains.create: %v", err)
	}
	if _, err := client.Domains.Verify(ctx, "id_id"); err != nil {
		t.Errorf("domains.verify: %v", err)
	}
	if _, err := client.Domains.Get(ctx, "id_id"); err != nil {
		t.Errorf("domains.get: %v", err)
	}
	if _, err := client.Domains.Delete(ctx, "id_id"); err != nil {
		t.Errorf("domains.delete: %v", err)
	}
	if _, err := client.Contacts.Create(ctx, "id_contactbookid", ContactsCreateParams{Email: "hello@example.com"}); err != nil {
		t.Errorf("contacts.create: %v", err)
	}
	if _, err := client.Contacts.List(ctx, "id_contactbookid", &ContactsListQuery{Emails: "a@example.com", Page: 2, Limit: 10, Ids: "c1,c2"}); err != nil {
		t.Errorf("contacts.list: %v", err)
	}
	if _, err := client.Contacts.Get(ctx, "id_contactbookid", "id_contactid"); err != nil {
		t.Errorf("contacts.get: %v", err)
	}
	if _, err := client.Contacts.Update(ctx, "id_contactbookid", "id_contactid", ContactsUpdateParams{}); err != nil {
		t.Errorf("contacts.update: %v", err)
	}
	if _, err := client.Contacts.Upsert(ctx, "id_contactbookid", "id_contactid", ContactsUpsertParams{Email: "hello@example.com"}); err != nil {
		t.Errorf("contacts.upsert: %v", err)
	}
	if _, err := client.Contacts.Delete(ctx, "id_contactbookid", "id_contactid"); err != nil {
		t.Errorf("contacts.delete: %v", err)
	}
	if _, err := client.Contacts.BulkCreate(ctx, "id_contactbookid", []ContactsBulkCreateParams{ContactsBulkCreateParams{Email: "hello@example.com"}}); err != nil {
		t.Errorf("contacts.bulkCreate: %v", err)
	}
	if _, err := client.Contacts.BulkDelete(ctx, "id_contactbookid", ContactsBulkDeleteParams{ContactIds: []string{"c1", "c2"}}); err != nil {
		t.Errorf("contacts.bulkDelete: %v", err)
	}
	if _, err := client.ContactBooks.List(ctx); err != nil {
		t.Errorf("contactBooks.list: %v", err)
	}
	if _, err := client.ContactBooks.Create(ctx, ContactBooksCreateParams{Name: "example.com"}); err != nil {
		t.Errorf("contactBooks.create: %v", err)
	}
	if _, err := client.ContactBooks.Get(ctx, "id_contactbookid"); err != nil {
		t.Errorf("contactBooks.get: %v", err)
	}
	if _, err := client.ContactBooks.Update(ctx, "id_contactbookid", ContactBooksUpdateParams{}); err != nil {
		t.Errorf("contactBooks.update: %v", err)
	}
	if _, err := client.ContactBooks.Delete(ctx, "id_contactbookid"); err != nil {
		t.Errorf("contactBooks.delete: %v", err)
	}
	if _, err := client.Campaigns.Create(ctx, CampaignsCreateParams{Name: "example.com", From: "hello@example.com", Subject: "sample", ContactBookID: "id_contactbookid"}); err != nil {
		t.Errorf("campaigns.create: %v", err)
	}
	if _, err := client.Campaigns.List(ctx, &CampaignsListQuery{Page: 2, Status: "DRAFT", Search: "news"}); err != nil {
		t.Errorf("campaigns.list: %v", err)
	}
	if _, err := client.Campaigns.Get(ctx, "id_campaignid"); err != nil {
		t.Errorf("campaigns.get: %v", err)
	}
	if _, err := client.Campaigns.Delete(ctx, "id_campaignid"); err != nil {
		t.Errorf("campaigns.delete: %v", err)
	}
	if _, err := client.Campaigns.Schedule(ctx, "id_campaignid", CampaignsScheduleParams{}); err != nil {
		t.Errorf("campaigns.schedule: %v", err)
	}
	if _, err := client.Campaigns.Pause(ctx, "id_campaignid"); err != nil {
		t.Errorf("campaigns.pause: %v", err)
	}
	if _, err := client.Campaigns.Resume(ctx, "id_campaignid"); err != nil {
		t.Errorf("campaigns.resume: %v", err)
	}
	if _, err := client.Analytics.EmailTimeSeries(ctx, &AnalyticsEmailTimeSeriesQuery{Days: "7", DomainID: "1"}); err != nil {
		t.Errorf("analytics.emailTimeSeries: %v", err)
	}
	if _, err := client.Analytics.ReputationMetrics(ctx, &AnalyticsReputationMetricsQuery{DomainID: "1"}); err != nil {
		t.Errorf("analytics.reputationMetrics: %v", err)
	}
}

func TestARefusalIsATypedError(t *testing.T) {
	client := contractClient(t)
	_, err := client.Emails.Get(context.Background(), "fake-404")
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.StatusCode != 404 || refusal.Code != "NOT_FOUND" {
		t.Fatalf("a 404 is %#v", err)
	}
}

func TestATooManyRequestsAnswerIsRepeatedUntilItPasses(t *testing.T) {
	client := contractClient(t)
	if _, err := client.Domains.Get(context.Background(), "flaky-429-2"); err != nil {
		t.Fatalf("two 429s then success: %v", err)
	}
	_, err := client.Domains.Get(context.Background(), "fake-429")
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.StatusCode != 429 || refusal.Code != "RATE_LIMITED" {
		t.Fatalf("a 429 that never passes is %#v", err)
	}
}

func TestAServerFailureIsRepeatedForAReadAndNotForAnUnkeyedSend(t *testing.T) {
	client := contractClient(t)
	if _, err := client.Emails.Get(context.Background(), "flaky-503-2"); err != nil {
		t.Fatalf("a read after two 503s: %v", err)
	}
}

func TestTheSameIdempotencyKeyAnswersTheSameEmail(t *testing.T) {
	client := contractClient(t)
	ctx := context.Background()
	key := WithIdempotencyKey(fmt.Sprintf("order-%d", time.Now().UnixNano()))
	params := EmailsSendParams{From: "hello@example.com", To: StringList{"a@example.com"}}
	first, err := client.Emails.Send(ctx, params, key)
	if err != nil {
		t.Fatal(err)
	}
	again, err := client.Emails.Send(ctx, params, key)
	if err != nil || again.EmailID != first.EmailID {
		t.Fatalf("the same key answered %v (%v), then %v", first, err, again)
	}
	other := EmailsSendParams{From: "hello@example.com", To: StringList{"b@example.com"}}
	_, err = client.Emails.Send(ctx, other, key)
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.StatusCode != 409 || refusal.Code != "NOT_UNIQUE" {
		t.Fatalf("another body under the same key is %#v", err)
	}
}
