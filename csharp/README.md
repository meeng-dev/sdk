# MeengDev.Sdk (.NET)

SDK do meeng para .NET 8 ou mais novo (C#): e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências; o JSON é gerado em tempo de compilação, então o pacote funciona com trimming e AOT.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado: o nome `MeengDev.Sdk` (NuGet) é uma proposta aguardando confirmação.

## Começando

```csharp
using Meeng;

var meeng = new MeengClient("mg_123");

var sent = await meeng.Emails.SendAsync(new EmailsSendParams
{
    From = "hello@acme.com",
    To = ["cliente@example.com"],
    Subject = "Olá",
    Html = "<p>Olá!</p>",
});
Console.WriteLine(sent.EmailId);
```

Uma recusa da API lança `MeengException` (`Status`, `Code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `ApiMessage` e `RetryAfter`). O cliente pode ser compartilhado entre threads.

`new MeengClient()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. `MeengClientOptions.BaseUrl` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `HttpClient`, `Handler` e `Timeout` trocam o transporte e o prazo (o cliente padrão não segue redirecionamentos).

## Recursos

### `meeng.Emails`

- `SendAsync`: Sends an email, now or at scheduledAt.
- `BatchAsync`: Sends up to 100 emails in one request.
- `GetAsync`: Reads one email with its events.
- `ListAsync`: Lists emails, newest first, with an optional date range and domain filter.
- `UpdateAsync`: Reschedules an email that has not been sent.
- `CancelAsync`: Cancels a scheduled email.

### `meeng.Domains`

- `ListAsync`: Lists the sending domains of the team.
- `CreateAsync`: Adds a sending domain and returns the DNS records to publish.
- `VerifyAsync`: Asks for the domain's DNS records to be checked again.
- `GetAsync`: Reads one domain with its DNS records.
- `DeleteAsync`: Removes a domain.

### `meeng.Contacts`

- `CreateAsync`: Adds a contact to a contact book.
- `ListAsync`: Lists the contacts of a contact book, optionally by email or id.
- `GetAsync`: Reads one contact.
- `UpdateAsync`: Changes the fields of a contact that are sent.
- `UpsertAsync`: Creates the contact, or replaces it when it exists.
- `DeleteAsync`: Removes a contact.
- `BulkCreateAsync`: Adds many contacts to a contact book in one request.
- `BulkDeleteAsync`: Removes many contacts from a contact book in one request.

### `meeng.ContactBooks`

- `ListAsync`: Lists the contact books of the team.
- `CreateAsync`: Creates a contact book.
- `GetAsync`: Reads one contact book.
- `UpdateAsync`: Changes the fields of a contact book that are sent.
- `DeleteAsync`: Removes a contact book and its contacts.

### `meeng.Campaigns`

- `CreateAsync`: Creates a campaign, and sends it now when sendNow is true.
- `ListAsync`: Lists campaigns, optionally by status or a search text.
- `GetAsync`: Reads one campaign with its counters.
- `DeleteAsync`: Removes a campaign.
- `ScheduleAsync`: Schedules a campaign, or sends it now when no time is given.
- `PauseAsync`: Pauses a running or scheduled campaign.
- `ResumeAsync`: Resumes a paused campaign.

### `meeng.Analytics`

- `EmailTimeSeriesAsync`: Counts emails by day over the last 7 or 30 days.
- `ReputationMetricsAsync`: Reads the bounce and complaint rates of the sending reputation.

Os parâmetros de caminho são `string`; os filtros de listagem são o `<Recurso><Método>Query` (uma lista em `DomainId` vira um `domainId` por valor). `Emails.SendAsync` e `Emails.BatchAsync` aceitam `new RequestOptions { IdempotencyKey = "pedido-1" }`: a mesma chave com o mesmo corpo devolve o primeiro resultado. Os destinatários são `IReadOnlyList<string>`; a API aceita uma string solta, e o SDK a lê como lista de um item.

## Webhooks

```csharp
var webhooks = meeng.Webhooks("whsec_xxx");

// O corpo cru, antes de interpretá-lo.
var evento = webhooks.ConstructEvent(corpo, Request.Headers);

if (evento is EmailWebhookEvent email)
{
    Console.WriteLine(string.Join(",", email.Data.To));
}
```

`Verify(corpo, cabecalhos)` só confere a assinatura e devolve `true` ou `false`. `ConstructEvent` confere e devolve o evento, ou lança `WebhookVerificationException` com um `Code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`WebhookOptions.Tolerance`; negativa desliga a checagem; `WebhookOptions.Secret` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `UnknownWebhookEvent`.

## Desenvolvimento

```sh
dotnet test --nologo
```
