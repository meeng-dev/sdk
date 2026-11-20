package meeng

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func str(s string) *string { return &s }

func server(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(handle))
	t.Cleanup(srv.Close)
	return New("mg_test_key", WithBaseURL(srv.URL))
}

func TestSendingAnEmailCarriesTheKeyTheBodyAndTheIdempotencyKey(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		to, _ := body["to"].([]any)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/emails" || r.Header.Get("Authorization") != "Bearer mg_test_key" ||
			r.Header.Get("Idempotency-Key") != "order-1" || r.Header.Get("Content-Type") != "application/json" ||
			r.Header.Get("User-Agent") != "meeng-go/"+Version || len(to) != 2 || body["subject"] != "Hi" {
			t.Errorf("request %s %s %v %v", r.Method, r.URL.Path, r.Header, body)
		}
		if _, sent := body["cc"]; sent {
			t.Error("an optional field left out was sent")
		}
		fmt.Fprint(w, `{"emailId":"em_1"}`)
	})
	sent, err := client.Emails.Send(context.Background(), EmailsSendParams{From: "hello@acme.com", To: StringList{"a@x.com", "b@x.com"}, Subject: str("Hi"), HTML: str("<p>Hi</p>")}, WithIdempotencyKey("order-1"))
	if err != nil || sent.EmailID != "em_1" {
		t.Fatalf("sent %+v, %v", sent, err)
	}
}

func TestASendWithoutAnIdempotencyKeySendsNone(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		if _, sent := r.Header["Idempotency-Key"]; sent {
			t.Error("an Idempotency-Key was sent that nobody asked for")
		}
		fmt.Fprint(w, `{"emailId":"em_1"}`)
	})
	if _, err := client.Emails.Send(context.Background(), EmailsSendParams{From: "hello@acme.com", To: StringList{"a@x.com"}, Text: str("Hi")}); err != nil {
		t.Fatal(err)
	}
}

func TestABatchAnswersTheListOfIDs(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		var body []map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/api/v1/emails/batch" || len(body) != 2 {
			t.Errorf("request %s %v", r.URL.Path, body)
		}
		fmt.Fprint(w, `{"data":[{"emailId":"em_1"},{"emailId":"em_2"}]}`)
	})
	got, err := client.Emails.Batch(context.Background(), []EmailsBatchParams{
		{From: "hello@acme.com", To: StringList{"a@x.com"}, Subject: str("One"), Text: str("1")},
		{From: "hello@acme.com", To: StringList{"b@x.com"}, Subject: str("Two"), Text: str("2")},
	})
	if err != nil || len(got.Data) != 2 || got.Data[1].EmailID != "em_2" {
		t.Fatalf("batch %+v, %v", got, err)
	}
}

func TestListingEmailsSendsTheFiltersAListOnceForEachValue(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/emails" || q.Get("page") != "2" || q.Get("limit") != "10" ||
			q.Get("startDate") != "2026-11-01T00:00:00Z" || len(q["domainId"]) != 2 || q["domainId"][1] != "2" || q.Has("endDate") {
			t.Errorf("request %s %s", r.Method, r.URL.RawQuery)
		}
		fmt.Fprint(w, `{"data":[{"id":"em_1","to":"a@x.com","from":"h@acme.com","subject":"Hi","html":null,"text":"Hi","createdAt":"2026-11-20T00:00:00Z","updatedAt":"2026-11-20T00:00:00Z","latestStatus":"DELIVERED","scheduledAt":null,"domainId":3}],"count":1}`)
	})
	list, err := client.Emails.List(context.Background(), &EmailsListQuery{Page: 2, Limit: 10, StartDate: "2026-11-01T00:00:00Z", DomainID: []string{"1", "2"}})
	if err != nil || list.Count != 1 || len(list.Data) != 1 || list.Data[0].To[0] != "a@x.com" || list.Data[0].HTML != nil ||
		list.Data[0].LatestStatus == nil || *list.Data[0].LatestStatus != EmailsListItemLatestStatusDelivered || *list.Data[0].DomainID != 3 {
		t.Fatalf("list %+v, %v", list, err)
	}
}

func TestAPathParameterIsEscaped(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v1/emails/a%2Fb%20c" {
			t.Errorf("path %s", r.URL.EscapedPath())
		}
		fmt.Fprint(w, `{"id":"a/b c","teamId":1,"to":["a@x.com"],"from":"h@acme.com","subject":"Hi","html":null,"text":null,"createdAt":"x","updatedAt":"x","emailEvents":[{"emailId":"a/b c","status":"SENT","createdAt":"x","data":{"ip":"1.2.3.4"}}]}`)
	})
	email, err := client.Emails.Get(context.Background(), "a/b c")
	if err != nil || email.ID != "a/b c" || len(email.EmailEvents) != 1 || email.EmailEvents[0].Status != EmailEventStatusSent {
		t.Fatalf("email %+v, %v", email, err)
	}
}

func TestADomainIDIsAnInteger(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/domains/7" {
			t.Errorf("path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":7,"name":"acme.com","teamId":1,"status":"SUCCESS","publicKey":"k","createdAt":"x","updatedAt":"x","dnsRecords":[{"type":"TXT","name":"a","value":"b","ttl":"Auto","status":"PENDING"}]}`)
	})
	domain, err := client.Domains.Get(context.Background(), strconv.Itoa(7))
	if err != nil || domain.ID != 7 || domain.Status != DomainStatusSuccess || domain.DNSRecords[0].Type != DomainDnsRecordTypeTxt {
		t.Fatalf("domain %+v, %v", domain, err)
	}
}

func TestAListWithNoObjectAroundItIsAList(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"cb_1","name":"News","teamId":1,"properties":{"plan":"string"},"variables":["plan"],"emoji":"x","createdAt":"x","updatedAt":"x","_count":{"contacts":4}}]`)
	})
	books, err := client.ContactBooks.List(context.Background())
	if err != nil || len(books) != 1 || books[0].Name != "News" || books[0].Count == nil || books[0].Count.Contacts != 4 || books[0].Properties["plan"] != "string" {
		t.Fatalf("books %+v, %v", books, err)
	}
}

func TestADeleteCanCarryABody(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/contactBooks/cb_1/contacts/bulk" || string(raw) != `{"contactIds":["c1","c2"]}` {
			t.Errorf("request %s %s %s", r.Method, r.URL.Path, raw)
		}
		fmt.Fprint(w, `{"success":true,"count":2}`)
	})
	got, err := client.Contacts.BulkDelete(context.Background(), "cb_1", ContactsBulkDeleteParams{ContactIds: []string{"c1", "c2"}})
	if err != nil || !got.Success || got.Count != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestUpsertTakesTheContactsIDInThePath(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/contactBooks/cb_1/contacts/c_1" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"contactId":"c_1"}`)
	})
	got, err := client.Contacts.Upsert(context.Background(), "cb_1", "c_1", ContactsUpsertParams{Email: "a@x.com", FirstName: str("Ana")})
	if err != nil || got.ContactID != "c_1" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestARefusalIsAnErrorInEveryShapeTheAPIWritesIt(t *testing.T) {
	cases := []struct {
		status        int
		body          string
		code, message string
	}{
		{409, `{"error":{"code":"NOT_UNIQUE","message":"Idempotency-Key already used"}}`, "NOT_UNIQUE", "Idempotency-Key already used"},
		{403, `{"error":"Forbidden"}`, "INTERNAL_SERVER_ERROR", "Forbidden"},
		{400, `{"code":"BAD_REQUEST","message":"bad"}`, "BAD_REQUEST", "bad"},
		{502, `<html>`, "INTERNAL_SERVER_ERROR", "Bad Gateway"},
	}
	for _, c := range cases {
		client := server(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(c.status)
			fmt.Fprint(w, c.body)
		})
		_, err := client.Emails.Cancel(context.Background(), "em_1")
		var refusal *Error
		if !errors.As(err, &refusal) || refusal.StatusCode != c.status || refusal.Code != c.code || refusal.Message != c.message || refusal.RetryAfter != 7*time.Second {
			t.Errorf("status %d: error %#v", c.status, err)
		}
	}
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		fmt.Fprint(w, cases[0].body)
	})
	_, err := client.Emails.Cancel(context.Background(), "em_1")
	if err == nil || err.Error() != "meeng: 409 NOT_UNIQUE: Idempotency-Key already used" {
		t.Errorf("message %v", err)
	}
}

func TestAnEmailsRecipientsMayComeAsOneStringOrAList(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"em_1","teamId":1,"to":"a@x.com","cc":null,"from":"h@acme.com","subject":"Hi","html":null,"text":null,"createdAt":"x","updatedAt":"x","emailEvents":[]}`)
	})
	email, err := client.Emails.Get(context.Background(), "em_1")
	if err != nil || len(email.To) != 1 || email.To[0] != "a@x.com" || email.Cc != nil {
		t.Fatalf("email %+v, %v", email, err)
	}
}

func TestTheKeyAndTheAddressComeFromTheEnvironment(t *testing.T) {
	t.Setenv("MEENG_API_KEY", "mg_env")
	t.Setenv("MEENG_BASE_URL", "http://127.0.0.1:1")
	client := New("")
	if client.apiKey != "mg_env" || client.baseURL != "http://127.0.0.1:1" || client.setupErr != nil {
		t.Errorf("client %+v", client)
	}
	t.Setenv("MEENG_API_KEY", "")
	if _, err := New("").Emails.Get(context.Background(), "x"); !errors.Is(err, ErrMissingAPIKey) {
		t.Errorf("a missing key answered %v", err)
	}
}

func TestABaseURLMustBeHTTPSUnlessItIsLoopback(t *testing.T) {
	for _, base := range []string{"http://app.meeng.test", "ftp://app.meeng.test", "app.meeng.test", "http://localhost.evil.test"} {
		client := New("mg_test_key", WithBaseURL(base))
		if _, err := client.Emails.Get(context.Background(), "em_1"); !errors.Is(err, ErrInsecureBaseURL) {
			t.Errorf("%s answered %v", base, err)
		}
	}
	for _, base := range []string{"https://app.meeng.test", "http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"} {
		if err := checkBaseURL(base); err != nil {
			t.Errorf("%s refused: %v", base, err)
		}
	}
}

func TestAnAnswerIsNeverFollowedToARedirect(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://elsewhere.test/", http.StatusFound)
	})
	_, err := client.Emails.Cancel(context.Background(), "em_1")
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.StatusCode != http.StatusFound {
		t.Errorf("a redirect answered %v", err)
	}
}

const webhookSecret = "whsec_test"
const webhookBody = `{"id":"evt_1","type":"email.delivered","createdAt":"2026-11-20T12:00:00Z","data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}`

func sign(timestamp, body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + body))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestASignedWebhookIsVerifiedAndReadAsATypedEvent(t *testing.T) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	header := http.Header{}
	header.Set("X-Meeng-Signature", sign(timestamp, webhookBody, webhookSecret))
	header.Set("X-Meeng-Timestamp", timestamp)
	webhooks := New("k").Webhooks(webhookSecret)
	if !webhooks.Verify([]byte(webhookBody), header) {
		t.Fatal("a signed webhook was refused")
	}
	event, err := webhooks.ConstructEvent([]byte(webhookBody), header)
	delivered, ok := event.(*EmailWebhookEvent)
	if err != nil || !ok || delivered.Type != "email.delivered" || delivered.ID != "evt_1" || delivered.Data.To[0] != "a@x.com" || delivered.Data.Status != EmailBasePayloadStatusDelivered {
		t.Fatalf("event %#v, %v", event, err)
	}
}

func TestTheUseSendHeadersAreAcceptedAsWell(t *testing.T) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	header := http.Header{}
	header.Set("x-usesend-signature", sign(timestamp, webhookBody, webhookSecret))
	header.Set("x-usesend-timestamp", timestamp)
	if !NewWebhooks(webhookSecret).Verify([]byte(webhookBody), header) {
		t.Fatal("a webhook signed with the useSend headers was refused")
	}
}

func TestAWebhookIsRefusedWithTheCodeThatSaysWhy(t *testing.T) {
	now := time.UnixMilli(1_800_000_000_000)
	clock := WithWebhookClock(func() time.Time { return now })
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	old := strconv.FormatInt(now.Add(-6*time.Minute).UnixMilli(), 10)
	webhooks := NewWebhooks(webhookSecret)
	headers := func(signature, stamp string) http.Header {
		h := http.Header{}
		if signature != "" {
			h.Set("X-Meeng-Signature", signature)
		}
		if stamp != "" {
			h.Set("X-Meeng-Timestamp", stamp)
		}
		return h
	}
	good := sign(timestamp, webhookBody, webhookSecret)
	cases := []struct {
		name   string
		body   string
		header http.Header
		opts   []WebhookOption
		want   WebhookErrorCode
	}{
		{"good", webhookBody, headers(good, timestamp), nil, ""},
		{"no signature", webhookBody, headers("", timestamp), nil, WebhookMissingSignature},
		{"no timestamp", webhookBody, headers(good, ""), nil, WebhookMissingTimestamp},
		{"no prefix", webhookBody, headers(good[3:], timestamp), nil, WebhookInvalidSignatureForm},
		{"bad timestamp", webhookBody, headers(good, "yesterday"), nil, WebhookInvalidTimestamp},
		{"too old", webhookBody, headers(sign(old, webhookBody, webhookSecret), old), nil, WebhookTimestampOutOfRange},
		{"old but unchecked", webhookBody, headers(sign(old, webhookBody, webhookSecret), old), []WebhookOption{WithWebhookTolerance(-1)}, ""},
		{"old but tolerated", webhookBody, headers(sign(old, webhookBody, webhookSecret), old), []WebhookOption{WithWebhookTolerance(10 * time.Minute)}, ""},
		{"another secret", webhookBody, headers(good, timestamp), []WebhookOption{WithWebhookSecret("another")}, WebhookSignatureMismatch},
		{"another body", `{"id":"evt_2"}`, headers(good, timestamp), nil, WebhookSignatureMismatch},
		{"not json", "not json", headers(sign(timestamp, "not json", webhookSecret), timestamp), nil, WebhookInvalidJSON},
	}
	for _, c := range cases {
		_, err := webhooks.ConstructEvent([]byte(c.body), c.header, append([]WebhookOption{clock}, c.opts...)...)
		var refusal *WebhookVerificationError
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s: refused: %v", c.name, err)
		case c.want != "" && (!errors.As(err, &refusal) || refusal.Code != c.want):
			t.Errorf("%s: answered %v, want %s", c.name, err, c.want)
		}
	}
}
