# meeng_sdk (Elixir)

SDK do meeng para Elixir 1.18 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências além da biblioteca padrão (`:httpc` e o módulo `JSON`).

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado: o nome `meeng_sdk` (Hex) é uma proposta aguardando confirmação.

## Começando

```elixir
client = Meeng.new("mg_123")

{:ok, sent} =
  Meeng.Emails.send(client, %{
    from: "hello@acme.com",
    to: "cliente@example.com",
    subject: "Olá",
    html: "<p>Olá!</p>"
  })

sent["emailId"]
```

Toda chamada devolve `{:ok, dados}` ou `{:error, %Meeng.Error{}}` (com `status`, `code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `message` e `retry_after`). Os corpos são mapas (ou listas de palavras-chave) com as chaves da API, em camelCase (`contactBookId`), e os dados são o JSON da resposta decodificado em mapas com chaves de texto.

`Meeng.new/2` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL` quando não os recebe. A opção `:base_url` é o endereço do app (para instalações próprias), sem `/api/v1`; ele precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. O transporte padrão usa `:httpc` e não segue redirecionamentos.

## Recursos

### `Meeng.Emails`

- `send`: Sends an email, now or at scheduledAt.
- `batch`: Sends up to 100 emails in one request.
- `get`: Reads one email with its events.
- `list`: Lists emails, newest first, with an optional date range and domain filter.
- `update`: Reschedules an email that has not been sent.
- `cancel`: Cancels a scheduled email.

### `Meeng.Domains`

- `list`: Lists the sending domains of the team.
- `create`: Adds a sending domain and returns the DNS records to publish.
- `verify`: Asks for the domain's DNS records to be checked again.
- `get`: Reads one domain with its DNS records.
- `delete`: Removes a domain.

### `Meeng.Contacts`

- `create`: Adds a contact to a contact book.
- `list`: Lists the contacts of a contact book, optionally by email or id.
- `get`: Reads one contact.
- `update`: Changes the fields of a contact that are sent.
- `upsert`: Creates the contact, or replaces it when it exists.
- `delete`: Removes a contact.
- `bulk_create`: Adds many contacts to a contact book in one request.
- `bulk_delete`: Removes many contacts from a contact book in one request.

### `Meeng.ContactBooks`

- `list`: Lists the contact books of the team.
- `create`: Creates a contact book.
- `get`: Reads one contact book.
- `update`: Changes the fields of a contact book that are sent.
- `delete`: Removes a contact book and its contacts.

### `Meeng.Campaigns`

- `create`: Creates a campaign, and sends it now when sendNow is true.
- `list`: Lists campaigns, optionally by status or a search text.
- `get`: Reads one campaign with its counters.
- `delete`: Removes a campaign.
- `schedule`: Schedules a campaign, or sends it now when no time is given.
- `pause`: Pauses a running or scheduled campaign.
- `resume`: Resumes a paused campaign.

### `Meeng.Analytics`

- `email_time_series`: Counts emails by day over the last 7 or 30 days.
- `reputation_metrics`: Reads the bounce and complaint rates of the sending reputation.

`Meeng.Emails.create/3` é outro nome de `send/3`, e `Meeng.Campaigns.get_all/2` de `list/2`.

Os filtros de listagem são opções em snake_case (`start_date:`, `domain_id:`); uma lista em `domain_id` vira um `domainId` por valor. `Meeng.Emails.send/3` e `batch/3` aceitam `idempotency_key: "pedido-1"`: a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Webhooks

```elixir
# headers: uma lista ou mapa de pares; body: o corpo cru, antes de interpretá-lo.
case Meeng.Webhooks.construct_event("whsec_xxx", headers, body) do
  {:ok, %{"type" => "email.delivered", "data" => data}} -> data["to"]
  {:ok, _outro_evento} -> :ok
  {:error, %Meeng.WebhookVerificationError{wire_code: code}} -> {:refused, code}
end
```

`Meeng.Webhooks.verify/4` só confere a assinatura e devolve `true` ou `false`. `construct_event/4` confere e devolve o evento, ou `{:error, %Meeng.WebhookVerificationError{}}` com `wire_code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance:`, em milissegundos; um número negativo desliga a checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
mix compile --warnings-as-errors
mix test --warnings-as-errors
```
