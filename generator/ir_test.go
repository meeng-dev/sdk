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

// The surface is the contract's 33 routes, under the names the SDKs give them.
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
	if ir.BaseURL != "https://meeng.dev" || ir.APIPrefix != "/api" {
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
		if f.Wire == "cc" && (f.Type.Kind != "array" || !f.Required) {
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

// The contract writes a list as a named schema that refers to itself; the SDKs
// read it as the list of its item.
func TestAListThatRefersToItselfIsTheListOfItsItem(t *testing.T) {
	ir := loadContract(t)
	for id, item := range map[string]string{"domains.list": "Domain", "contactBooks.list": "ContactBook", "contacts.list": "Contact"} {
		m := method(ir, id)
		if m == nil || !m.ResponseList || m.Response != item {
			t.Errorf("%s answers %+v, want a list of %s", id, m, item)
		}
	}
}

func TestTheAPIPrefixIsReadFromThePathsAndTheBaseURLIsTheApp(t *testing.T) {
	ir := loadContract(t)
	if ir.BaseURL != "https://meeng.dev" || ir.APIPrefix != "/api" {
		t.Fatalf("base %q, prefix %q", ir.BaseURL, ir.APIPrefix)
	}
	if send := method(ir, "emails.send"); send.Path != "/v1/emails" {
		t.Errorf("emails.send is at %s", send.Path)
	}
}

// OpenAPI 3.1 writes a nullable value as a list of types, or as a schema or null.
func TestNullableValuesAreReadInBothFormsOf31(t *testing.T) {
	ir := loadContract(t)
	fields := map[string]*Field{}
	for _, f := range ir.model("Campaign").Fields {
		fields[f.Wire] = f
	}
	if f := fields["previewText"]; f == nil || !f.Nullable || f.Type.Kind != "string" {
		t.Errorf("previewText is %+v", f)
	}
	if f := fields["name"]; f == nil || f.Nullable {
		t.Errorf("name is %+v", f)
	}
	var count *Field
	for _, f := range ir.model("ContactBook").Fields {
		if f.Wire == "_count" {
			count = f
		}
	}
	if count == nil || !count.Nullable || count.Required || count.Type.Kind != "model" {
		t.Errorf("_count of a contact book is %+v", count)
	}
}

func TestTheDeletedCampaignHasItsOwnModel(t *testing.T) {
	ir := loadContract(t)
	if got := method(ir, "campaigns.delete").Response; got != "CampaignRow" {
		t.Errorf("campaigns.delete answers %s", got)
	}
	if ir.model("CampaignRow") == nil {
		t.Error("CampaignRow has no model")
	}
}

// The API takes no attachments (ADR-0008) and the contract lists the field as a
// list of at most zero items: no SDK has it, and the model says it left it out.
func TestNoSendModelHasAnAttachmentsField(t *testing.T) {
	ir := loadContract(t)
	for _, name := range []string{"EmailsSendParams", "EmailsBatchParams"} {
		model := ir.model(name)
		if model == nil {
			t.Fatalf("%s has no model", name)
		}
		for _, f := range model.Fields {
			if f.Wire == "attachments" {
				t.Errorf("%s offers attachments", name)
			}
		}
		if len(model.Refused) != 1 || model.Refused[0] != "attachments" {
			t.Errorf("%s did not record the field it left out: %v", name, model.Refused)
		}
	}
	if model := ir.model("Email"); model != nil {
		for _, f := range model.Fields {
			if f.Wire == "attachments" {
				t.Error("an e-mail read back carries attachments")
			}
		}
	}
}

// A list with no items that is not the refused field is still a stop: only the
// contract's own zero-length list is left out.
func TestAnArrayWithNoItemsIsStillAStopUnlessTheAPIRefusesIt(t *testing.T) {
	zero, one := 0, 1
	if !(&schema{Type: "array", MaxItems: &zero}).refused() {
		t.Error("a list of at most zero items is the refused field")
	}
	if (&schema{Type: "array", MaxItems: &one}).refused() || (&schema{Type: "array"}).refused() {
		t.Error("only a list capped at zero is refused")
	}
}

// A domain's records carry any type the platform publishes, a CNAME for the return
// path included, and the priority only where the record has one.
func TestADomainRecordTypeIsAFreeStringAndThePriorityIsOptional(t *testing.T) {
	ir := loadContract(t)
	model := ir.model("DomainDnsRecord")
	if model == nil {
		t.Fatal("DomainDnsRecord has no model")
	}
	for _, f := range model.Fields {
		switch f.Wire {
		case "type":
			if len(f.Type.Enum) != 0 {
				t.Errorf("the record type is an enum %v: a CNAME would be refused", f.Type.Enum)
			}
		case "priority", "recommended":
			if f.Required {
				t.Errorf("%s is required, and a CNAME record has none", f.Wire)
			}
		}
	}
}
