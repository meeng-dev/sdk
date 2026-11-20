# dev.meeng:sdk (Java)

SDK do meeng para Java 11 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências em tempo de execução (usa o `java.net.http` do JDK e um leitor de JSON próprio).

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O artefato ainda não foi publicado: as coordenadas `dev.meeng:sdk` (Maven) são uma proposta aguardando confirmação.

## Começando

```java
import dev.meeng.*;
import java.util.List;

Meeng meeng = new Meeng("mg_123");

EmailsSendResponse sent = meeng.emails().send(
    EmailsService.SendParams.builder()
        .from("hello@acme.com")
        .to(List.of("cliente@example.com"))
        .subject("Olá")
        .html("<p>Olá!</p>")
        .build());
System.out.println(sent.getEmailId());
```

Uma recusa da API lança `MeengException` (`getStatus()`, `getCode()`, como `NOT_UNIQUE` ou `FORBIDDEN`, `getApiMessage()` e `getRetryAfter()`); uma falha de rede, `UncheckedIOException`. O cliente pode ser compartilhado entre threads.

`new Meeng()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. `Meeng.builder(chave).baseUrl(...)` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `timeout(...)` e `transport(...)` trocam o prazo e o transporte HTTP (o padrão não segue redirecionamentos).

## Recursos

### `meeng.emails()`

- `send`: Sends an email, now or at scheduledAt.
- `batch`: Sends up to 100 emails in one request.
- `get`: Reads one email with its events.
- `list`: Lists emails, newest first, with an optional date range and domain filter.
- `update`: Reschedules an email that has not been sent.
- `cancel`: Cancels a scheduled email.

### `meeng.domains()`

- `list`: Lists the sending domains of the team.
- `create`: Adds a sending domain and returns the DNS records to publish.
- `verify`: Asks for the domain's DNS records to be checked again.
- `get`: Reads one domain with its DNS records.
- `delete`: Removes a domain.

### `meeng.contacts()`

- `create`: Adds a contact to a contact book.
- `list`: Lists the contacts of a contact book, optionally by email or id.
- `get`: Reads one contact.
- `update`: Changes the fields of a contact that are sent.
- `upsert`: Creates the contact, or replaces it when it exists.
- `delete`: Removes a contact.
- `bulkCreate`: Adds many contacts to a contact book in one request.
- `bulkDelete`: Removes many contacts from a contact book in one request.

### `meeng.contactBooks()`

- `list`: Lists the contact books of the team.
- `create`: Creates a contact book.
- `get`: Reads one contact book.
- `update`: Changes the fields of a contact book that are sent.
- `delete`: Removes a contact book and its contacts.

### `meeng.campaigns()`

- `create`: Creates a campaign, and sends it now when sendNow is true.
- `list`: Lists campaigns, optionally by status or a search text.
- `get`: Reads one campaign with its counters.
- `delete`: Removes a campaign.
- `schedule`: Schedules a campaign, or sends it now when no time is given.
- `pause`: Pauses a running or scheduled campaign.
- `resume`: Resumes a paused campaign.

### `meeng.analytics()`

- `emailTimeSeries`: Counts emails by day over the last 7 or 30 days.
- `reputationMetrics`: Reads the bounce and complaint rates of the sending reputation.

Os parâmetros de caminho são `String`; os filtros de listagem são o `ListParams` do serviço (uma lista em `domainId` vira um `domainId` por valor). `emails().send` e `emails().batch` aceitam `RequestOptions.builder().idempotencyKey("pedido-1").build()`: a mesma chave com o mesmo corpo devolve o primeiro resultado. Os destinatários são `List<String>`; a API aceita uma string solta, e o SDK a lê como lista de um item.

## Webhooks

```java
Webhooks webhooks = meeng.webhooks("whsec_xxx");

// O corpo cru, antes de interpretá-lo.
WebhookEvent event = webhooks.constructEvent(body, headers);

if (event instanceof EmailWebhookEvent) {
    System.out.println(((EmailWebhookEvent) event).getData().getTo());
}
```

`verify(body, headers)` só confere a assinatura e devolve `true` ou `false`. `constructEvent` confere e devolve o evento, ou lança `WebhookVerificationException` com um `getCode()`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`Webhooks.Options.defaults().tolerance(...)`; negativa desliga a checagem; `.secret(...)` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `UnknownWebhookEvent`.

## Desenvolvimento

```sh
mvn -q -B verify
```
