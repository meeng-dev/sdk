# meeng-sdk (Ruby)

SDK do meeng para Ruby 3.1 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Tem o mesmo formato do SDK do useSend, e não depende de nenhuma gem.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> A gem ainda não foi publicada: o nome `meeng-sdk` é uma proposta aguardando confirmação.

## Começando

```ruby
require "meeng"

meeng = Meeng::Client.new("mg_123")

sent = meeng.emails.send(
  from: "hello@acme.com",
  to: "cliente@example.com",
  subject: "Olá",
  html: "<p>Olá!</p>"
)
puts sent[:emailId]
```

Os corpos são hashes com as chaves da API, em camelCase (`contactBookId`, `scheduledAt`), e as respostas são hashes com essas mesmas chaves como símbolos. Uma recusa da API levanta `Meeng::Error` (`status`, `code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `retry_after` e a mensagem).

Sem argumentos, `Meeng::Client.new` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. O segundo argumento é o endereço do app (para instalações próprias), sem `/api/v1`; ele precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. O transporte padrão usa `Net::HTTP` e não segue redirecionamentos.

## Recursos

### `meeng.emails`

- `send`: envia um e-mail, agora ou em `scheduledAt`.
- `batch`: envia até 100 e-mails numa requisição.
- `get`: lê um e-mail com os seus eventos.
- `list`: lista e-mails, do mais novo ao mais velho, com período e domínio opcionais.
- `update`: reagenda um e-mail que ainda não saiu.
- `cancel`: cancela um e-mail agendado.

### `meeng.domains`

- `list`: lista os domínios de envio do time.
- `create`: adiciona um domínio de envio e devolve os registros DNS a publicar.
- `verify`: pede para conferir de novo os registros DNS do domínio.
- `get`: lê um domínio com os seus registros DNS.
- `delete`: remove um domínio.

### `meeng.contacts`

- `create`: adiciona um contato a uma lista de contatos.
- `list`: lista os contatos de uma lista, com filtro opcional por e-mail ou id.
- `get`: lê um contato.
- `update`: muda só os campos enviados de um contato.
- `upsert`: cria o contato, ou o substitui quando ele já existe.
- `delete`: remove um contato.
- `bulk_create`: adiciona vários contatos a uma lista numa requisição.
- `bulk_delete`: remove vários contatos de uma lista numa requisição.

### `meeng.contact_books`

- `list`: lista as listas de contatos do time.
- `create`: cria uma lista de contatos.
- `get`: lê uma lista de contatos.
- `update`: muda só os campos enviados de uma lista de contatos.
- `delete`: remove uma lista de contatos e os seus contatos.

### `meeng.campaigns`

- `create`: cria uma campanha, e a envia agora quando `sendNow` é verdadeiro.
- `list`: lista campanhas, com filtro opcional por status ou texto.
- `get`: lê uma campanha com os seus contadores.
- `delete`: remove uma campanha.
- `schedule`: agenda uma campanha, ou a envia agora quando não há horário.
- `pause`: pausa uma campanha em andamento ou agendada.
- `resume`: retoma uma campanha pausada.

### `meeng.analytics`

- `email_time_series`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `reputation_metrics`: lê as taxas de rejeição e de reclamação da reputação de envio.

`emails.create` é outro nome de `emails.send`, e `campaigns.get_all` de `campaigns.list`.

Os filtros de listagem são argumentos nomeados em snake_case (`start_date`, `domain_id`); uma lista em `domain_id` vira um `domainId` por valor. `emails.send` e `emails.batch` aceitam `idempotency_key: "pedido-1"`: a mesma chave com o mesmo corpo devolve o primeiro resultado.

## Webhooks

```ruby
webhooks = meeng.webhooks("whsec_xxx")

# O corpo cru, antes de interpretá-lo.
event = webhooks.construct_event(request.body.read, headers: request.headers)

puts event[:data][:to] if event[:type] == "email.delivered"
```

`verify(body, headers:)` só confere a assinatura e devolve `true` ou `false`. `construct_event` confere e devolve o evento, ou levanta `Meeng::WebhookVerificationError` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance:`, em milissegundos; um número negativo desliga a checagem; `secret:` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
ruby -w -Ilib -Itest test/meeng_test.rb
```
