# meeng-sdk (Rust)

SDK do meeng para Rust 1.88 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Chamadas bloqueantes (com `ureq` e `rustls`); o transporte é um trait, então dá para trocá-lo.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> A crate ainda não foi publicada (`publish = false`): o nome `meeng-sdk` (crates.io) é uma proposta aguardando confirmação. No código, `use meeng::…`.

## Começando

```rust
use meeng::{Client, EmailsSendParams};

fn main() -> Result<(), meeng::Error> {
    let client = Client::new("mg_123");

    let sent = client.emails().send(
        &EmailsSendParams::new(vec!["cliente@example.com".to_owned()], "hello@acme.com")
            .subject("Olá")
            .html("<p>Olá!</p>"),
    )?;
    println!("{:?}", sent.email_id);
    Ok(())
}
```

Os campos obrigatórios de um corpo vão em `new`; cada campo opcional tem um método com o seu nome. Uma recusa da API é `Error::Api(ApiError)` (`status`, `code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `message` e `retry_after`; `error.api()` a devolve). Os destinatários são `Vec<String>`; a API aceita uma string solta, e o SDK a lê como lista de um item.

`Client::from_env()` lê a chave de `MEENG_API_KEY`; o endereço do app vem de `MEENG_BASE_URL` quando definida. `Client::builder(chave).base_url(...)` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro (senão toda chamada responde `Error::InsecureBaseUrl`). `timeout(...)` e `transport(...)` trocam o prazo e o HTTP (o padrão não segue redirecionamentos).

## Recursos

### `client.emails()`

- `send`: Sends an email, now or at scheduledAt.
- `batch`: Sends up to 100 emails in one request.
- `get`: Reads one email with its events.
- `list`: Lists emails, newest first, with an optional date range and domain filter.
- `update`: Reschedules an email that has not been sent.
- `cancel`: Cancels a scheduled email.

### `client.domains()`

- `list`: Lists the sending domains of the team.
- `create`: Adds a sending domain and returns the DNS records to publish.
- `verify`: Asks for the domain's DNS records to be checked again.
- `get`: Reads one domain with its DNS records.
- `delete`: Removes a domain.

### `client.contacts()`

- `create`: Adds a contact to a contact book.
- `list`: Lists the contacts of a contact book, optionally by email or id.
- `get`: Reads one contact.
- `update`: Changes the fields of a contact that are sent.
- `upsert`: Creates the contact, or replaces it when it exists.
- `delete`: Removes a contact.
- `bulk_create`: Adds many contacts to a contact book in one request.
- `bulk_delete`: Removes many contacts from a contact book in one request.

### `client.contact_books()`

- `list`: Lists the contact books of the team.
- `create`: Creates a contact book.
- `get`: Reads one contact book.
- `update`: Changes the fields of a contact book that are sent.
- `delete`: Removes a contact book and its contacts.

### `client.campaigns()`

- `create`: Creates a campaign, and sends it now when sendNow is true.
- `list`: Lists campaigns, optionally by status or a search text.
- `get`: Reads one campaign with its counters.
- `delete`: Removes a campaign.
- `schedule`: Schedules a campaign, or sends it now when no time is given.
- `pause`: Pauses a running or scheduled campaign.
- `resume`: Resumes a paused campaign.

### `client.analytics()`

- `email_time_series`: Counts emails by day over the last 7 or 30 days.
- `reputation_metrics`: Reads the bounce and complaint rates of the sending reputation.

Os parâmetros de caminho são `&str`; os filtros de listagem são o `<Recurso><Método>Query` (uma lista em `domain_id` vira um `domainId` por valor). `client.emails().idempotency_key("pedido-1").send(...)` torna o envio seguro para repetir (vale em `send` e `batch`): a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Webhooks

```rust
let webhooks = client.webhooks("whsec_xxx");

// O corpo cru, antes de interpretá-lo; os cabeçalhos podem ser um HashMap, uma lista de pares ou um http::HeaderMap.
match webhooks.construct_event(body, headers)? {
    meeng::WebhookEvent::Email(event) => println!("{:?}", event.data.to),
    _ => {}
}
```

`verify(body, headers)` só confere a assinatura e devolve `true` ou `false`. `construct_event` confere e devolve o evento, ou um `WebhookError` com `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`WebhookOptions::tolerance_ms`; negativa desliga a checagem; `secret` troca o segredo de uma checagem; use `verify_with` e `construct_event_with`). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `WebhookEvent::Unknown`.

## Desenvolvimento

```sh
cargo fmt --check
cargo clippy --all-targets -- -D warnings
cargo test
```
