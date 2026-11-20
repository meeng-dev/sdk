package generator

import (
	"strings"
	"testing"
)

const rival = "use" + "Send"

func TestTheContractLosesTheSentenceThatNamesAnotherProduct(t *testing.T) {
	raw := `{"openapi": "3.1.0", "info": {"title": "meeng.dev API", "description": "The e-mail API. It is the ` + rival + ` v1 API: the same paths, so an integration written for ` + rival + ` works. Authenticate with ` + "`Bearer \\u003ctoken\\u003e`" + `.", "version": "1.0.0"}, "paths": {}}`
	got, err := SanitizeContract([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(got)), strings.ToLower(rival)) {
		t.Fatalf("still named: %s", got)
	}
	if !strings.Contains(string(got), "The e-mail API. Authenticate with") || !strings.Contains(string(got), `"version": "1.0.0"`) {
		t.Errorf("the rest changed: %s", got)
	}
}

func TestAContractThatNamesAnotherProductElsewhereIsRefused(t *testing.T) {
	raw := `{"info": {"description": "Fine."}, "paths": {"/x": {"get": {"summary": "Works like Re` + "send" + `"}}}}`
	if _, err := SanitizeContract([]byte(raw)); err == nil {
		t.Fatal("a contract naming another product was accepted")
	}
}

func TestAContractThatNamesNoOneIsLeftAlone(t *testing.T) {
	raw := `{"info": {"description": "Fine."}}`
	got, err := SanitizeContract([]byte(raw))
	if err != nil || string(got) != raw {
		t.Errorf("changed %q: %v", got, err)
	}
}
