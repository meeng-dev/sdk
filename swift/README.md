# Meeng (Swift)

SDK do meeng para Swift 6 (macOS 12, iOS 15, tvOS 15, watchOS 8 e visionOS 1 ou mais novos): e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Usa `async`/`await`, `Codable` e `URLSession`, sem dependências. A verificação de webhooks usa o CryptoKit, então só existe nas plataformas da Apple.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado: o nome (`github.com/meeng-dev/sdk`, produto `Meeng`) é uma proposta aguardando confirmação.

## Começando

```swift
import Meeng

let meeng = MeengClient(apiKey: "mg_123")

let sent = try await meeng.emails.send(
    EmailsSendParams(
        to: ["cliente@example.com"],
        from: "hello@acme.com",
        subject: "Olá",
        html: "<p>Olá!</p>"
    )
)
print(sent.emailID ?? "")
```

Os campos obrigatórios de um corpo vêm primeiro no inicializador; os opcionais, com valor `nil`, não são enviados. Uma recusa da API lança `MeengError` (`status`, `code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `message` e `retryAfter`). Os destinatários são um `StringList` (um array de `String`; a API aceita uma string solta, e o SDK a lê como lista de um item).

`MeengClient()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. `baseURL:` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `timeout:` e `transport:` trocam o prazo e o HTTP (o padrão não segue redirecionamentos). Sem chave ou com um endereço recusado, toda chamada lança `MeengError` antes de enviar qualquer coisa.

## Recursos

### `meeng.emails`

- `send`: Sends an email, now or at scheduledAt.
- `batch`: Sends up to 100 emails in one request.
- `get`: Reads one email with its events.
- `list`: Lists emails, newest first, with an optional date range and domain filter.
- `update`: Reschedules an email that has not been sent.
- `cancel`: Cancels a scheduled email.

### `meeng.domains`

- `list`: Lists the sending domains of the team.
- `create`: Adds a sending domain and returns the DNS records to publish.
- `verify`: Asks for the domain's DNS records to be checked again.
- `get`: Reads one domain with its DNS records.
- `delete`: Removes a domain.

### `meeng.contacts`

- `create`: Adds a contact to a contact book.
- `list`: Lists the contacts of a contact book, optionally by email or id.
- `get`: Reads one contact.
- `update`: Changes the fields of a contact that are sent.
- `upsert`: Creates the contact, or replaces it when it exists.
- `delete`: Removes a contact.
- `bulkCreate`: Adds many contacts to a contact book in one request.
- `bulkDelete`: Removes many contacts from a contact book in one request.

### `meeng.contactBooks`

- `list`: Lists the contact books of the team.
- `create`: Creates a contact book.
- `get`: Reads one contact book.
- `update`: Changes the fields of a contact book that are sent.
- `delete`: Removes a contact book and its contacts.

### `meeng.campaigns`

- `create`: Creates a campaign, and sends it now when sendNow is true.
- `list`: Lists campaigns, optionally by status or a search text.
- `get`: Reads one campaign with its counters.
- `delete`: Removes a campaign.
- `schedule`: Schedules a campaign, or sends it now when no time is given.
- `pause`: Pauses a running or scheduled campaign.
- `resume`: Resumes a paused campaign.

### `meeng.analytics`

- `emailTimeSeries`: Counts emails by day over the last 7 or 30 days.
- `reputationMetrics`: Reads the bounce and complaint rates of the sending reputation.

Os parâmetros de caminho são `String`; os filtros de listagem são argumentos nomeados (`startDate:`, `domainId:`; um array em `domainId` vira um `domainId` por valor). `emails.send` e `emails.batch` aceitam `idempotencyKey: "pedido-1"`: a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Webhooks

```swift
let webhooks = meeng.webhooks(secret: "whsec_xxx")

// O corpo cru (Data), antes de interpretá-lo.
let event = try webhooks.constructEvent(body, headers: headers)

if case let .email(email) = event {
    print(email.data.to)
}
```

`verify(_:headers:)` só confere a assinatura e devolve `true` ou `false`. `constructEvent` confere e devolve o evento, ou lança `WebhookVerificationError` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance:`, em segundos; negativa desliga a checagem; `secret:` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `.unknown`.

## Desenvolvimento

```sh
swift test
```
