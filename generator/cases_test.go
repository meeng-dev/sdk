package generator

import (
	"strings"
	"testing"
)

func TestTheContractTestCallsEveryMethodInEveryLanguage(t *testing.T) {
	ir := loadContract(t)
	calls := ir.ContractCalls()
	if len(calls) != 33 {
		t.Fatalf("%d calls, want one for each of the 33 methods", len(calls))
	}
	for language := range Languages {
		caller := callers[language]
		if caller == nil {
			t.Errorf("%s has no renderer for the calls of the contract test", language)
			continue
		}
		for _, call := range calls {
			text := caller(call)
			if text == "" || !strings.Contains(strings.ToLower(text), strings.ToLower(Snake(call.Method.Name))) && !strings.Contains(strings.ToLower(text), strings.ToLower(call.Method.Name)) {
				t.Errorf("%s renders %s as %q", language, call.Method.OperationID, text)
			}
		}
	}
}

func TestACallCarriesWhatTheContractRequires(t *testing.T) {
	ir := loadContract(t)
	byID := map[string]*ContractCall{}
	for _, call := range ir.ContractCalls() {
		byID[call.Method.OperationID] = call
	}
	send := byID["emails.send"]
	if send.Key == "" || len(send.Body) != 2 || send.Body[0].Field.Wire != "to" || send.Body[0].List == nil || send.Body[1].Field.Wire != "from" {
		t.Errorf("emails.send: %+v", send)
	}
	if list := byID["emails.list"]; len(list.Query) != 5 || list.Key != "" {
		t.Errorf("emails.list: %+v", list)
	}
	if update := byID["contacts.update"]; len(update.Path) != 2 || len(update.Body) != 0 {
		t.Errorf("contacts.update: %+v", update)
	}
	for _, id := range []string{"emails.send", "emails.batch"} {
		if byID[id].Key == "" {
			t.Errorf("%s carries no idempotency key", id)
		}
	}
	if n := len(byID["contacts.bulkCreate"].Body); n != 1 {
		t.Errorf("contacts.bulkCreate requires %d fields, want the email", n)
	}
}

func TestTheGoCallIsTheCallAMethodTakes(t *testing.T) {
	ir := loadContract(t)
	for _, call := range ir.ContractCalls() {
		if call.Method.OperationID == "emails.send" {
			want := `client.Emails.Send(ctx, EmailsSendParams{To: StringList{"hello@example.com"}, From: "hello@example.com"}, WithIdempotencyKey("contract-emails-send"))`
			if got := goCall(call); got != want {
				t.Errorf("got  %s\nwant %s", got, want)
			}
		}
	}
}
