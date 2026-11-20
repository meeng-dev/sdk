package generator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// otherBrands are the names of the products this one is compared with. They do
// not appear anywhere a developer reads: not in the contract the SDKs carry,
// and so not in the code or the READMEs written from it.
var otherBrands = regexp.MustCompile(`(?i)use-?send|unsend|resend`)

// infoDescription finds the description of the contract's info object in the
// raw document: the raw JSON string, without its quotes.
var infoDescription = regexp.MustCompile(`(?s)("info":\s*\{.*?"description":\s*")((?:[^"\\]|\\.)*)(")`)

// SanitizeContract is the contract as the SDKs keep it: the one the app
// exports, with a sentence of the API's description that names another product
// left out. Everything else is byte for byte the app's. A contract that names
// another product anywhere else is refused, because that is the app's to fix,
// not something to edit in a copy.
func SanitizeContract(raw []byte) ([]byte, error) {
	match := infoDescription.FindSubmatchIndex(raw)
	if match != nil {
		var text string
		if err := json.Unmarshal([]byte(`"`+string(raw[match[4]:match[5]])+`"`), &text); err != nil {
			return nil, fmt.Errorf("generator: read the contract's description: %w", err)
		}
		if otherBrands.MatchString(text) {
			var kept []string
			for _, sentence := range strings.SplitAfter(text, ". ") {
				if !otherBrands.MatchString(sentence) {
					kept = append(kept, sentence)
				}
			}
			text = strings.TrimSpace(strings.Join(kept, ""))
			encoded, err := json.Marshal(text)
			if err != nil {
				return nil, err
			}
			out := append([]byte(nil), raw[:match[4]]...)
			out = append(out, encoded[1:len(encoded)-1]...)
			out = append(out, raw[match[5]:]...)
			raw = out
		}
	}
	if found := otherBrands.Find(raw); found != nil {
		return nil, fmt.Errorf("generator: the contract names %q: take it out of the app's contract, the SDKs do not name other products", found)
	}
	return raw, nil
}
