# meeng-sdk (Python)

SDK do meeng para Python 3.11 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Tem o mesmo formato do SDK do useSend: troque o import, o nome do cliente e a chave, e o resto do código continua igual. Sem dependências além da biblioteca padrão.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado: o nome `meeng-sdk` é uma proposta aguardando confirmação.

## Começando

```python
from meeng import Meeng

meeng = Meeng("mg_123")

data, _ = meeng.emails.send({
    "from": "hello@acme.com",
    "to": "cliente@example.com",
    "subject": "Olá",
    "html": "<p>Olá!</p>",
})
print(data["emailId"])
```

Toda chamada devolve `(data, error)`, como no useSend. Por padrão uma recusa da API levanta `MeengHTTPError` (com `status_code` e `error`, que é `{"code", "message"}`) e `error` volta sempre `None`; com `Meeng("mg_123", raise_on_error=False)` a chamada devolve `(None, error)`.

Os corpos e as respostas são dicionários com as chaves da API, em camelCase (`contactBookId`, `scheduledAt`). Os tipos estão em `meeng.types` (`meeng.types.EmailsSendParams`, `meeng.types.Domain`…) e funcionam com o `mypy --strict`.

Sem argumentos, `Meeng()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. O segundo argumento é o endereço do app (para instalações próprias), sem `/api/v1`; ele precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. O transporte padrão usa `urllib` e não segue redirecionamentos.

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
- `bulk_create`: Adds many contacts to a contact book in one request.
- `bulk_delete`: Removes many contacts from a contact book in one request.

### `meeng.contact_books`

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

- `email_time_series`: Counts emails by day over the last 7 or 30 days.
- `reputation_metrics`: Reads the bounce and complaint rates of the sending reputation.

`emails.create` é outro nome de `emails.send`, `campaigns.get_all` de `campaigns.list`, e `meeng.contactBooks` de `meeng.contact_books`.

Os filtros de listagem são argumentos nomeados em snake_case (`start_date`, `domain_id`); uma lista em `domain_id` vira um `domainId` por valor. `emails.send` e `emails.batch` aceitam `options={"idempotency_key": "pedido-1"}`: a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Webhooks

```python
webhooks = meeng.webhooks("whsec_xxx")

# Flask: request.data; Django: request.body; FastAPI: await request.body().
event = webhooks.construct_event(request.data, headers=request.headers)

if event["type"] == "email.delivered":
    print(event["data"]["to"])
elif event["type"] == "email.bounced":
    print(event["data"]["bounce"]["type"])
```

`verify(body, headers=...)` só confere a assinatura e devolve `True` ou `False`. `construct_event` confere e devolve o evento, ou levanta `WebhookVerificationError` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH`, `INVALID_BODY` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance`, em milissegundos; `-1` desliga a checagem; `secret` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
PYTHONPATH=src python3 -m unittest discover -s tests
mypy --strict --python-version 3.11 src/meeng
```
