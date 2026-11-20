// Schedules an e-mail, moves it to another time and cancels it.
//
//	MEENG_API_KEY=mg_xxx go run ./examples/schedule_email
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	ctx := context.Background()
	subject, html := "Lembrete: sua reunião é amanhã", "<p>Nos vemos às 10h.</p>"
	at := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	sent, err := client.Emails.Send(ctx, meeng.EmailsSendParams{
		From:        "Acme <contato@acme.com.br>",
		To:          meeng.StringList{"ana@example.com"},
		Subject:     &subject,
		HTML:        &html,
		ScheduledAt: &at,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agendado para", at, "-", sent.EmailID)

	later := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	if _, err := client.Emails.Update(ctx, sent.EmailID, meeng.EmailsUpdateParams{ScheduledAt: later}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("remarcado para", later)

	if _, err := client.Emails.Cancel(ctx, sent.EmailID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("cancelado")
}
