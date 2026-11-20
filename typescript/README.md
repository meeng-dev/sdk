# @meeng-dev/sdk (TypeScript e JavaScript)

SDK do meeng para Node.js 20.3 ou mais novo (também funciona em Bun e Deno): e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Tem o mesmo formato do SDK do useSend: troque o import, o nome do cliente e a chave, e o resto do código continua igual.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão. Sem dependências em tempo de execução.

> O pacote ainda não foi publicado: o nome `@meeng-dev/sdk` é uma proposta aguardando confirmação.

## Começando

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng("mg_123");

const { data, error } = await meeng.emails.send({
  from: "hello@acme.com",
  to: "cliente@example.com",
  subject: "Olá",
  html: "<p>Olá!</p>",
});
```

Toda chamada devolve `{ data, error }`: `data` quando deu certo, `error` (`{ code, message, status }`) quando a API recusou. Uma falha de rede rejeita a promessa.

Sem argumentos, `new Meeng()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. O segundo argumento é o endereço do app (para instalações próprias), sem `/api/v1`; ele precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro.

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

`emails.create` é outro nome de `emails.send`, e `campaigns.getAll` de `campaigns.list`.

`emails.send` e `emails.batch` aceitam `{ idempotencyKey }` como último argumento: a mesma chave com o mesmo corpo devolve o primeiro resultado, sem enviar de novo.

`emails.batch` devolve na `data` a lista de ids (`[{ emailId }]`), como o useSend.

## Webhooks

```ts
const webhooks = meeng.webhooks("whsec_xxx");

// Express, Next.js, Hono: passe o corpo cru, antes de interpretá-lo.
const event = webhooks.constructEvent(rawBody, { headers: request.headers });

if (event.type === "email.delivered") {
  console.log(event.data.to);
}
```

`verify(body, { headers })` só confere a assinatura e devolve `true` ou `false`. `constructEvent` confere e devolve o evento tipado, ou lança `WebhookVerificationError` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH`, `INVALID_BODY` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance`, em milissegundos; um número negativo desliga a checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
npm ci
npm run typecheck
npm test
```
