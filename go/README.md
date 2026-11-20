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

- `Send`: envia um e-mail, agora ou em `scheduledAt`.
- `Batch`: envia até 100 e-mails numa requisição.
- `Get`: lê um e-mail com os seus eventos.
- `List`: lista e-mails, do mais novo ao mais velho, com período e domínio opcionais.
- `Update`: reagenda um e-mail que ainda não saiu.
- `Cancel`: cancela um e-mail agendado.

### `client.Domains`

- `List`: lista os domínios de envio do time.
- `Create`: adiciona um domínio de envio e devolve os registros DNS a publicar.
- `Verify`: pede para conferir de novo os registros DNS do domínio.
- `Get`: lê um domínio com os seus registros DNS.
- `Delete`: remove um domínio.

### `client.Contacts`

- `Create`: adiciona um contato a uma lista de contatos.
- `List`: lista os contatos de uma lista, com filtro opcional por e-mail ou id.
- `Get`: lê um contato.
- `Update`: muda só os campos enviados de um contato.
- `Upsert`: cria o contato, ou o substitui quando ele já existe.
- `Delete`: remove um contato.
- `BulkCreate`: adiciona vários contatos a uma lista numa requisição.
- `BulkDelete`: remove vários contatos de uma lista numa requisição.

### `client.ContactBooks`

- `List`: lista as listas de contatos do time.
- `Create`: cria uma lista de contatos.
- `Get`: lê uma lista de contatos.
- `Update`: muda só os campos enviados de uma lista de contatos.
- `Delete`: remove uma lista de contatos e os seus contatos.

### `client.Campaigns`

- `Create`: cria uma campanha, e a envia agora quando `sendNow` é verdadeiro.
- `List`: lista campanhas, com filtro opcional por status ou texto.
- `Get`: lê uma campanha com os seus contadores.
- `Delete`: remove uma campanha.
- `Schedule`: agenda uma campanha, ou a envia agora quando não há horário.
- `Pause`: pausa uma campanha em andamento ou agendada.
- `Resume`: retoma uma campanha pausada.

### `client.Analytics`

- `EmailTimeSeries`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `ReputationMetrics`: lê as taxas de rejeição e de reclamação da reputação de envio.

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
