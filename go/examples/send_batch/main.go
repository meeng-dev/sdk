// Sends up to 100 e-mails in one request.
//
//	MEENG_API_KEY=mg_xxx go run ./examples/send_batch
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	var batch []meeng.EmailsBatchParams
	people := []struct{ name, email string }{
		{"Ana", "ana@example.com"},
		{"Bruno", "bruno@example.com"},
		{"Carla", "carla@example.com"},
	}
	for _, person := range people {
		subject := "Seu extrato de novembro"
		html := fmt.Sprintf("<p>Olá, %s! Seu extrato está disponível.</p>", person.name)
		batch = append(batch, meeng.EmailsBatchParams{
			From:    "Acme <contato@acme.com.br>",
			To:      meeng.StringList{person.email},
			Subject: &subject,
			HTML:    &html,
		})
	}

	sent, err := client.Emails.Batch(context.Background(), batch, meeng.WithIdempotencyKey("extrato-2026-11"))
	if err != nil {
		log.Fatal(err)
	}
	for i, email := range sent.Data {
		fmt.Printf("%d: %s\n", i+1, email.EmailID)
	}
}
