// Checks a webhook the way an endpoint does: the signature and the time are
// verified against the raw body, and then the event is read as a typed value.
// The request here is made in the program; a real endpoint reads the body and
// the headers of the request it receives, and the secret from its settings.
//
//	go run ./examples/verify_webhook
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	secret := "whsec_exemplo"
	body := []byte(`{"id":"call_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":1,"data":{"id":"em_1","status":"DELIVERED","from":"contato@acme.com.br","to":["ana@example.com"],"occurredAt":"2026-11-20T12:00:00Z"}}`)

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(body)))
	header := http.Header{}
	header.Set(meeng.WebhookSignatureHeader, "v1="+hex.EncodeToString(mac.Sum(nil)))
	header.Set(meeng.WebhookTimestampHeader, timestamp)

	event, err := meeng.NewWebhooks(secret).ConstructEvent(body, header)
	if err != nil {
		log.Fatal(err)
	}
	switch e := event.(type) {
	case *meeng.EmailWebhookEvent:
		fmt.Println("e-mail", e.Data.ID, "-", e.Type, "para", e.Data.To)
	default:
		fmt.Printf("evento %T\n", e)
	}
}
