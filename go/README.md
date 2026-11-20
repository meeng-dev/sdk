# meeng para Go

SDK do meeng para Go 1.23 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências além da biblioteca padrão.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O módulo ainda não foi publicado: o caminho `github.com/meeng-dev/sdk/go` é uma proposta aguardando confirmação.

```sh
go get github.com/meeng-dev/sdk/go
```

## Começando

```go
package main

import (
	"context"
	"fmt"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("") // vazio: a chave vem de MEENG_API_KEY
	subject, html := "Olá", "<p>Olá!</p>"
	sent, err := client.Emails.Send(context.Background(), meeng.EmailsSendParams{
		From:    "hello@acme.com",
		To:      meeng.StringList{"cliente@example.com"},
		Subject: &subject,
		HTML:    &html,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(sent.EmailID)
}
```

Campos opcionais do corpo são ponteiros; campos que a API pode devolver como `null` também. `meeng.StringList` é a lista de destinatários: a API aceita uma string ou uma lista, o SDK envia sempre a lista e lê uma string solta como lista de um item.

`meeng.New(chave, opções...)` lê a chave de `MEENG_API_KEY` quando recebe `""` e o endereço do app de `MEENG_BASE_URL`. `meeng.WithBaseURL` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `meeng.WithHTTPClient` troca o cliente HTTP (o padrão não segue redirecionamentos, então a chave nunca vai para outro lugar).

## Recursos

### `client.Emails`

- `Send`: Sends an email, now or at scheduledAt.
- `Batch`: Sends up to 100 emails in one request.
- `Get`: Reads one email with its events.
- `List`: Lists emails, newest first, with an optional date range and domain filter.
- `Update`: Reschedules an email that has not been sent.
- `Cancel`: Cancels a scheduled email.

### `client.Domains`

- `List`: Lists the sending domains of the team.
- `Create`: Adds a sending domain and returns the DNS records to publish.
- `Verify`: Asks for the domain's DNS records to be checked again.
- `Get`: Reads one domain with its DNS records.
- `Delete`: Removes a domain.

### `client.Contacts`

- `Create`: Adds a contact to a contact book.
- `List`: Lists the contacts of a contact book, optionally by email or id.
- `Get`: Reads one contact.
- `Update`: Changes the fields of a contact that are sent.
- `Upsert`: Creates the contact, or replaces it when it exists.
- `Delete`: Removes a contact.
- `BulkCreate`: Adds many contacts to a contact book in one request.
- `BulkDelete`: Removes many contacts from a contact book in one request.

### `client.ContactBooks`

- `List`: Lists the contact books of the team.
- `Create`: Creates a contact book.
- `Get`: Reads one contact book.
- `Update`: Changes the fields of a contact book that are sent.
- `Delete`: Removes a contact book and its contacts.

### `client.Campaigns`

- `Create`: Creates a campaign, and sends it now when sendNow is true.
- `List`: Lists campaigns, optionally by status or a search text.
- `Get`: Reads one campaign with its counters.
- `Delete`: Removes a campaign.
- `Schedule`: Schedules a campaign, or sends it now when no time is given.
- `Pause`: Pauses a running or scheduled campaign.
- `Resume`: Resumes a paused campaign.

### `client.Analytics`

- `EmailTimeSeries`: Counts emails by day over the last 7 or 30 days.
- `ReputationMetrics`: Reads the bounce and complaint rates of the sending reputation.

Os parâmetros de caminho são strings; os filtros de listagem são a struct `<Recurso><Método>Query` (campo zero é omitido, e `DomainID` é uma lista que vira um `domainId` por valor).

`meeng.WithIdempotencyKey("pedido-1")` torna um envio seguro para repetir: vale em `Emails.Send` e `Emails.Batch`; a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Erros

Uma recusa da API é um `*meeng.Error` com `StatusCode`, `Code` (como `NOT_UNIQUE`, `FORBIDDEN`, `NOT_FOUND`; `INTERNAL_SERVER_ERROR` quando a resposta não trouxe um), `Message` e `RetryAfter`:

```go
var refusal *meeng.Error
if errors.As(err, &refusal) && refusal.StatusCode == http.StatusConflict {
	// a mesma Idempotency-Key com outro corpo
}
```

## Webhooks

```go
webhooks := client.Webhooks("whsec_xxx") // ou meeng.NewWebhooks("whsec_xxx")

body, _ := io.ReadAll(r.Body) // o corpo cru, antes de interpretá-lo
event, err := webhooks.ConstructEvent(body, r.Header)
if err != nil {
	http.Error(w, "assinatura inválida", http.StatusUnauthorized)
	return
}
switch event := event.(type) {
case *meeng.EmailWebhookEvent:
	fmt.Println(event.Type, event.Data.To)
}
```

`Verify(body, header)` só confere a assinatura e devolve `true` ou `false`. `ConstructEvent` confere e devolve o evento, ou um `*meeng.WebhookVerificationError` com `Code` (`MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`).

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`WithWebhookTolerance`; negativa desliga a checagem; `WithWebhookSecret` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `*meeng.UnknownWebhookEvent`.
