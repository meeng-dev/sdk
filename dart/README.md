# meeng_sdk (Dart e Flutter)

SDK do meeng para Dart 3.8 ou mais novo e Flutter: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Depende só de `http` e `crypto`, ambos do dart.dev.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado (`publish_to: none`): o nome `meeng_sdk` (pub.dev) é uma proposta aguardando confirmação.

## Começando

```dart
import 'package:meeng_sdk/meeng_sdk.dart';

Future<void> main() async {
  final meeng = Meeng(apiKey: 'mg_123');
  try {
    final sent = await meeng.emails.send(
      const EmailsSendParams(
        from: 'hello@acme.com',
        to: ['cliente@example.com'],
        subject: 'Olá',
        html: '<p>Olá!</p>',
      ),
    );
    print(sent.emailId);
  } on MeengException catch (error) {
    print(error); // meeng: 403 FORBIDDEN: ...
  } finally {
    meeng.close();
  }
}
```

Uma recusa da API lança `MeengException` (`status`, `code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `message` e `retryAfter`).

`Meeng()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL` (fora do navegador, onde há variáveis de ambiente). `baseUrl` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `httpClient` e `timeout` trocam o transporte e o prazo; a chamada nunca segue redirecionamentos.

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

Os parâmetros de caminho são `String`; os filtros de listagem são argumentos nomeados (`startDate:`, `domainId:`; uma lista em `domainId` vira um `domainId` por valor). `emails.send` e `emails.batch` aceitam `idempotencyKey: 'pedido-1'`: a mesma chave com o mesmo corpo devolve o primeiro resultado. Os destinatários são `List<String>`; a API aceita uma string solta, e o SDK a lê como lista de um item.

## Webhooks

```dart
final webhooks = meeng.webhooks('whsec_xxx');

// O corpo cru, antes de interpretá-lo.
final event = webhooks.constructEvent(body, headers);

if (event case EmailWebhookEvent(:final data)) {
  print(data.to);
}
```

`verify(body, headers)` só confere a assinatura e devolve `true` ou `false`. `constructEvent` confere e devolve o evento, ou lança `WebhookVerificationException` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance:`; uma duração negativa desliga a checagem; `secret:` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `UnknownWebhookEvent`.

## Desenvolvimento

```sh
dart pub get
dart format --output=none --set-exit-if-changed .
dart analyze --fatal-infos
dart test
```
