// Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client
// at another app, the address from MEENG_BASE_URL.
//
//	MEENG_API_KEY=mg_xxx go run ./examples/send_email
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	subject, html, text := "Bem-vinda à Acme", "<p>Olá, Ana! Sua conta está pronta.</p>", "Olá, Ana! Sua conta está pronta."

	sent, err := client.Emails.Send(context.Background(), meeng.EmailsSendParams{
		From:    "Acme <contato@acme.com.br>",
		To:      meeng.StringList{"ana@example.com"},
		Subject: &subject,
		HTML:    &html,
		Text:    &text,
	},
		// A key makes the send safe to repeat: the same key with the same body
		// answers the first result again, and nothing is sent twice.
		meeng.WithIdempotencyKey("boas-vindas-ana-2026-11-20"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("e-mail na fila:", sent.EmailID)
}
