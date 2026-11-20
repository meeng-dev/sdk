package generator

import (
	"sort"
	"testing"
)

func loadContract(t *testing.T) *IR {
	t.Helper()
	ir, err := Load("../openapi.json", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	return ir
}

func method(ir *IR, id string) *Method {
	for _, service := range ir.Services {
		for _, m := range service.Methods {
			if m.OperationID == id {
				return m
			}
		}
	}
	return nil
}

// The surface is the contract's 33 routes, under the names useSend's SDKs use.
func TestTheSurfaceIsEveryRouteOfTheContract(t *testing.T) {
	ir := loadContract(t)
	var got []string
	for _, service := range ir.Services {
		for _, m := range service.Methods {
			got = append(got, m.OperationID)
		}
	}
	if len(got) != 33 {
		t.Fatalf("%d methods: %v", len(got), got)
	}
	for _, want := range []string{
		"emails.send", "emails.batch", "emails.get", "emails.list", "emails.update", "emails.cancel",
		"domains.list", "domains.create", "domains.verify", "domains.get", "domains.delete",
		"contacts.create", "contacts.list", "contacts.get", "contacts.update", "contacts.upsert", "contacts.delete", "contacts.bulkCreate", "contacts.bulkDelete",
		"contactBooks.list", "contactBooks.create", "contactBooks.get", "contactBooks.update", "contactBooks.delete",
		"campaigns.create", "campaigns.list", "campaigns.get", "campaigns.delete", "campaigns.schedule", "campaigns.pause", "campaigns.resume",
		"analytics.emailTimeSeries", "analytics.reputationMetrics",
	} {
		if method(ir, want) == nil {
			t.Errorf("%s is missing", want)
		}
	}
	var services []string
	for _, service := range ir.Services {
		services = append(services, service.Name)
	}
	want := []string{"emails", "domains", "contacts", "contactBooks", "campaigns", "analytics"}
	if !equal(services, want) {
		t.Errorf("services %v, want %v", services, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTheServerIsSplitInTheAppAndThePrefix(t *testing.T) {
	ir := loadContract(t)
	if ir.BaseURL != "https://app.meeng.dev" || ir.APIPrefix != "/api" {
		t.Errorf("base %q, prefix %q", ir.BaseURL, ir.APIPrefix)
	}
}

func TestOnlyTheTwoSendsTakeAnIdempotencyKey(t *testing.T) {
	ir := loadContract(t)
	var got []string
	for _, service := range ir.Services {
		for _, m := range service.Methods {
			if m.Idempotent {
				got = append(got, m.OperationID)
			}
		}
	}
	sort.Strings(got)
	if !equal(got, []string{"emails.batch", "emails.send"}) {
		t.Errorf("idempotent: %v", got)
	}
}

func TestAPostOrPutWithNoBodyOfItsOwnSendsAnEmptyObject(t *testing.T) {
	ir := loadContract(t)
	var got []string
	for _, service := range ir.Services {
		for _, m := range service.Methods {
			if m.EmptyBody {
				got = append(got, m.OperationID)
			}
		}
	}
	sort.Strings(got)
	want := []string{"campaigns.pause", "campaigns.resume", "domains.verify", "emails.cancel"}
	if !equal(got, want) {
		t.Errorf("empty body: %v, want %v", got, want)
	}
}

func TestListsAndBatchesAreFirstClass(t *testing.T) {
	ir := loadContract(t)
	for _, id := range []string{"domains.list", "contacts.list", "contactBooks.list"} {
		if m := method(ir, id); m == nil || !m.ResponseList {
			t.Errorf("%s does not answer a list", id)
		}
	}
	batch := method(ir, "emails.batch")
	if !batch.BodyList || batch.Body.Name != "EmailsBatchParams" || batch.Response != "EmailsBatchResponse" {
		t.Errorf("batch: body list %v %s, response %s", batch.BodyList, batch.Body.Name, batch.Response)
	}
	bulk := method(ir, "contacts.bulkCreate")
	if !bulk.BodyList {
		t.Error("bulkCreate takes a list of contacts")
	}
}

func TestPathParametersFollowThePathEvenWhenTheContractForgetsOne(t *testing.T) {
	ir := loadContract(t)
	upsert := method(ir, "contacts.upsert")
	if !equal(upsert.PathParams, []string{"contactBookId", "contactId"}) {
		t.Errorf("upsert path parameters %v", upsert.PathParams)
	}
}

func TestQueryFiltersKeepTheirKinds(t *testing.T) {
	ir := loadContract(t)
	kinds := map[string]string{}
	for _, p := range method(ir, "emails.list").Query {
		kinds[p.Name] = p.Kind
	}
	want := map[string]string{"page": "integer", "limit": "integer", "startDate": "string", "endDate": "string", "domainId": "stringlist"}
	for name, kind := range want {
		if kinds[name] != kind {
			t.Errorf("emails.list %s is %q, want %q", name, kinds[name], kind)
		}
	}
	for _, p := range method(ir, "contacts.list").Query {
		if (p.Name == "page" || p.Name == "limit") && p.Kind != "integer" {
			t.Errorf("contacts.list %s is %q", p.Name, p.Kind)
		}
	}
}

func TestRecipientsAreAStringOrAListOfStrings(t *testing.T) {
	ir := loadContract(t)
	body := method(ir, "emails.send").Body
	for _, f := range body.Fields {
		switch f.Wire {
		case "to", "replyTo", "cc", "bcc":
			if f.Type.Kind != "array" || !f.Type.OrScalar {
				t.Errorf("%s is %+v", f.Wire, f.Type)
			}
		}
	}
	list := ir.model("EmailsListItem")
	if list == nil {
		t.Fatal("the list's item has no model")
	}
	for _, f := range list.Fields {
		if f.Wire == "cc" && (!f.Nullable || f.Required) {
			t.Errorf("cc of a listed email is %+v", f)
		}
	}
}

func (ir *IR) model(name string) *Model {
	for _, m := range ir.Models {
		if m.Name == name {
			return m
		}
	}
	return nil
}

func TestTheSameShapeIsOneModel(t *testing.T) {
	ir := loadContract(t)
	for _, name := range []string{"Domain", "Contact", "ContactBook", "Campaign", "Email"} {
		if ir.model(name) == nil {
			t.Errorf("%s has no model", name)
		}
	}
	count := map[string]int{}
	for _, m := range ir.Models {
		count[m.Name]++
	}
	for name, n := range count {
		if n != 1 {
			t.Errorf("%s is %d models", name, n)
		}
	}
	for _, id := range []string{"domains.create", "domains.get", "domains.list"} {
		if method(ir, id).Response != "Domain" {
			t.Errorf("%s answers %s", id, method(ir, id).Response)
		}
	}
}

func TestAnIDIsAnIntegerAndARateIsNot(t *testing.T) {
	ir := loadContract(t)
	kinds := map[string]string{}
	for _, f := range ir.model("Domain").Fields {
		kinds[f.Wire] = f.Type.Kind
	}
	if kinds["id"] != "integer" || kinds["teamId"] != "integer" {
		t.Errorf("domain id kinds %v", kinds)
	}
	for _, f := range ir.model("AnalyticsReputationMetricsResponse").Fields {
		if f.Wire == "bounceRate" && f.Type.Kind != "number" {
			t.Errorf("bounceRate is %s", f.Type.Kind)
		}
	}
}

func TestWebhookEventsAreGroupedByTheirPayload(t *testing.T) {
	ir := loadContract(t)
	types := 0
	for _, group := range ir.Webhooks {
		types += len(group.Types)
		if ir.model(group.Data.Model) == nil {
			t.Errorf("%s carries %s, which has no model", group.Name, group.Data.Model)
		}
	}
	if types != 21 {
		t.Errorf("%d event types, want 21", types)
	}
}

func TestPascalCamelAndSnake(t *testing.T) {
	cases := []struct{ in, pascal, camel, snake string }{
		{"contactBooks", "ContactBooks", "contactBooks", "contact_books"},
		{"emailTimeSeries", "EmailTimeSeries", "emailTimeSeries", "email_time_series"},
		{"firstName", "FirstName", "firstName", "first_name"},
		{"_count", "Count", "count", "_count"},
		{"id", "ID", "id", "id"},
		{"bulk-create", "BulkCreate", "bulkCreate", "bulk_create"},
	}
	for _, c := range cases {
		if got := Pascal(c.in); got != c.pascal {
			t.Errorf("Pascal(%q) = %q, want %q", c.in, got, c.pascal)
		}
		if got := Camel(c.in); got != c.camel {
			t.Errorf("Camel(%q) = %q, want %q", c.in, got, c.camel)
		}
		if got := Snake(c.in); got != c.snake {
			t.Errorf("Snake(%q) = %q, want %q", c.in, got, c.snake)
		}
	}
}
