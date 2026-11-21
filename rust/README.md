# meeng-sdk (Rust)

SDK do Meeng.dev para Rust 1.88 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Chamadas bloqueantes (com `ureq` e `rustls`); o transporte é um trait, então dá para trocá-lo.

Gerado pelo `sdk/generator` a partir do contrato da API do Meeng.dev do Meeng.dev (versão 1.0.0); não edite à mão.

> A crate ainda não foi publicada (`publish = false`): o nome `meeng-sdk` (crates.io) é uma proposta aguardando confirmação. No código, `use meeng::…`.

## Começando

```rust
use meeng::{Client, EmailsSendParams};

fn main() -> Result<(), meeng::Error> {
    let client = Client::new("mg_123");

    let sent = client.emails().send(
        &EmailsSendParams::new(vec!["ana@example.com".to_owned()], "contato@acme.com.br")
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

- `send`: envia um e-mail, agora ou em `scheduledAt`.
- `batch`: envia até 100 e-mails numa requisição.
- `get`: lê um e-mail com os seus eventos.
- `list`: lista e-mails, do mais novo ao mais velho, com período e domínio opcionais.
- `update`: reagenda um e-mail que ainda não saiu.
- `cancel`: cancela um e-mail agendado.

### `client.domains()`

- `list`: lista os domínios de envio do time.
- `create`: adiciona um domínio de envio e devolve os registros DNS a publicar.
- `verify`: pede para conferir de novo os registros DNS do domínio.
- `get`: lê um domínio com os seus registros DNS.
- `delete`: remove um domínio.

### `client.contacts()`

- `create`: adiciona um contato a uma lista de contatos.
- `list`: lista os contatos de uma lista, com filtro opcional por e-mail ou id.
- `get`: lê um contato.
- `update`: muda só os campos enviados de um contato.
- `upsert`: cria o contato, ou o substitui quando ele já existe.
- `delete`: remove um contato.
- `bulk_create`: adiciona vários contatos a uma lista numa requisição.
- `bulk_delete`: remove vários contatos de uma lista numa requisição.

### `client.contact_books()`

- `list`: lista as listas de contatos do time.
- `create`: cria uma lista de contatos.
- `get`: lê uma lista de contatos.
- `update`: muda só os campos enviados de uma lista de contatos.
- `delete`: remove uma lista de contatos e os seus contatos.

### `client.campaigns()`

- `create`: cria uma campanha, e a envia agora quando `sendNow` é verdadeiro.
- `list`: lista campanhas, com filtro opcional por status ou texto.
- `get`: lê uma campanha com os seus contadores.
- `delete`: remove uma campanha.
- `schedule`: agenda uma campanha, ou a envia agora quando não há horário.
- `pause`: pausa uma campanha em andamento ou agendada.
- `resume`: retoma uma campanha pausada.

### `client.analytics()`

- `email_time_series`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `reputation_metrics`: lê as taxas de rejeição e de reclamação da reputação de envio.

Os parâmetros de caminho são `&str`; os filtros de listagem são o `<Recurso><Método>Query` (uma lista em `domain_id` vira um `domainId` por valor). `client.emails().idempotency_key("pedido-1").send(...)` torna o envio seguro para repetir (vale em `send` e `batch`): a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Arquivos: use links, não anexos

O Meeng.dev não anexa, não guarda e não repassa arquivos, e por isso nenhum SDK tem o campo `attachments`. Hospede o arquivo onde preferir (seu armazenamento, um link assinado com prazo) e ponha o link no corpo do e-mail:

```html
<p>Sua fatura está pronta: <a href="https://exemplo.com.br/faturas/1042?token=...">baixar a fatura (PDF)</a></p>
```

Se um pedido chegar à API com `attachments` e itens, ela responde `400` (`ZodError`) com a mensagem `Attachments are not supported. Include links to your files in the email body.`; uma lista vazia passa. Para um documento sensível, use um link que expira.

## Configuração

A API do Meeng.dev fica em `https://meeng.dev/api`. O cliente recebe o endereço do app, `https://meeng.dev` (o padrão), e acrescenta `/api/v1` a cada chamada. Crie uma chave em [Configurações > Chaves](https://meeng.dev/configuracoes/chaves); ela tem a forma `mg_<id>_<segredo>`.

- A chave vem do argumento ou de `MEENG_API_KEY`; o endereço, do argumento ou de `MEENG_BASE_URL`.
- Só `https` é aceito. `http` vale para `localhost`, `127.0.0.0/8` e `::1` (um servidor local), para a chave nunca viajar em texto puro. Nenhum cliente segue redirecionamento.

```rust
let client = Client::builder("mg_...")
    .base_url("https://meeng.dev")
    .timeout(Duration::from_secs(10))
    .max_retries(3)
    .retry_delay(Duration::from_secs(1))
    .build();
```

## Exemplos

A pasta [`examples/`](examples) tem programas que rodam: enviar um e-mail, enviar em lote, agendar (remarcar e cancelar), adicionar um domínio (com os registros DNS) e verificar um webhook.

| Exemplo | Arquivo | Rode com |
|---|---|---|
| Enviar um e-mail | `examples/send_email.rs` | `cargo run --example send_email` |
| Enviar em lote | `examples/send_batch.rs` | `cargo run --example send_batch` |
| Agendar, remarcar e cancelar | `examples/schedule_email.rs` | `cargo run --example schedule_email` |
| Adicionar um domínio | `examples/add_domain.rs` | `cargo run --example add_domain` |
| Verificar um webhook | `examples/verify_webhook.rs` | `cargo run --example verify_webhook` |

Todos leem `MEENG_API_KEY` e, se quiser apontar para outro app, `MEENG_BASE_URL`; o de webhook roda sem chave, com uma requisição montada no próprio programa.

## Paginação

As listagens de e-mails, contatos e campanhas são paginadas por `page` (a partir de 1) e, nas duas primeiras, `limit`. `emails.list` devolve `count`, o total de e-mails; `campaigns.list` devolve `totalPage`, o total de páginas; `contacts.list` devolve a lista sem total, e a última página é a que vem com menos de `limit` contatos.

```rust
let mut todos = Vec::new();
for page in 1.. {
    let lista = client.emails().list(&EmailsListQuery::new().page(page).limit(100))?;
    let vazia = lista.data.is_empty();
    todos.extend(lista.data);
    if todos.len() as i64 >= lista.count || vazia {
        break;
    }
}
```

## Erros

Uma recusa da API é `Error::Api(ApiError)`, com `status`, `code`, `message` e `retry_after`; sem chave é `Error::MissingApiKey`, e uma falha de rede é `Error::Transport`:

```rust
match client.emails().get("em_123") {
    Err(Error::Api(erro)) if erro.code == "NOT_FOUND" => {
        // o e-mail não existe, ou é de outro time
    }
    outro => {
        outro?;
    }
}
```

Os códigos que a API devolve: `UNAUTHORIZED` (sem chave), `FORBIDDEN` (chave inválida ou sem permissão), `NOT_FOUND`, `NOT_UNIQUE` (a mesma chave de idempotência com outro corpo, ou ainda em andamento), `RATE_LIMITED`, `BAD_REQUEST`, `METHOD_NOT_ALLOWED`, `NOT_IMPLEMENTED` e `INTERNAL_SERVER_ERROR`. Uma validação recusada vira o código `ZodError`, e a mensagem é a primeira falha (`from: Required`).

## Tentativas automáticas

O cliente repete sozinho a chamada que pode ser repetida, até `max_retries` vezes depois da primeira tentativa (2 por padrão; 0 envia cada chamada uma vez só):

| Resposta | Repete? |
|---|---|
| 429 (limite de requisições por segundo) | Sempre: a chamada não foi processada, então qualquer uma pode ser repetida. |
| 500, 502, 503, 504 ou falha de conexão | Só quando repetir não faz o trabalho duas vezes: uma leitura (`GET`), um `PUT`, um `DELETE`, ou uma chamada com chave de idempotência. Um envio sem chave não é repetido. |
| 409 e qualquer outra recusa | Nunca. |

A primeira repetição espera `retry_delay` (500 ms por padrão), cada uma seguinte espera o dobro, até 30 segundos, e o `Retry-After` que a API devolve vale no lugar dessa espera. Esgotadas as tentativas, a última resposta é o erro.

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

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`WebhookOptions::tolerance_ms`; negativa desliga a checagem; `secret` troca o segredo de uma checagem; use `verify_with` e `construct_event_with`). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `WebhookEvent::Unknown`.

## Desenvolvimento

```sh
cargo fmt --check
cargo clippy --all-targets -- -D warnings
cargo test
```

O `./check.sh` da raiz do repositório sobe uma API de mentira gerada do contrato (`go run ./cmd/fakeapi`), que recusa o que o contrato não permite, e roda contra ela o teste de contrato (cada método do SDK é chamado, e nenhuma operação pode ficar de fora) e os exemplos.
