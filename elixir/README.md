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

- `send`: envia um e-mail, agora ou em `scheduledAt`.
- `batch`: envia até 100 e-mails numa requisição.
- `get`: lê um e-mail com os seus eventos.
- `list`: lista e-mails, do mais novo ao mais velho, com período e domínio opcionais.
- `update`: reagenda um e-mail que ainda não saiu.
- `cancel`: cancela um e-mail agendado.

### `Meeng.Domains`

- `list`: lista os domínios de envio do time.
- `create`: adiciona um domínio de envio e devolve os registros DNS a publicar.
- `verify`: pede para conferir de novo os registros DNS do domínio.
- `get`: lê um domínio com os seus registros DNS.
- `delete`: remove um domínio.

### `Meeng.Contacts`

- `create`: adiciona um contato a uma lista de contatos.
- `list`: lista os contatos de uma lista, com filtro opcional por e-mail ou id.
- `get`: lê um contato.
- `update`: muda só os campos enviados de um contato.
- `upsert`: cria o contato, ou o substitui quando ele já existe.
- `delete`: remove um contato.
- `bulk_create`: adiciona vários contatos a uma lista numa requisição.
- `bulk_delete`: remove vários contatos de uma lista numa requisição.

### `Meeng.ContactBooks`

- `list`: lista as listas de contatos do time.
- `create`: cria uma lista de contatos.
- `get`: lê uma lista de contatos.
- `update`: muda só os campos enviados de uma lista de contatos.
- `delete`: remove uma lista de contatos e os seus contatos.

### `Meeng.Campaigns`

- `create`: cria uma campanha, e a envia agora quando `sendNow` é verdadeiro.
- `list`: lista campanhas, com filtro opcional por status ou texto.
- `get`: lê uma campanha com os seus contadores.
- `delete`: remove uma campanha.
- `schedule`: agenda uma campanha, ou a envia agora quando não há horário.
- `pause`: pausa uma campanha em andamento ou agendada.
- `resume`: retoma uma campanha pausada.

### `Meeng.Analytics`

- `email_time_series`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `reputation_metrics`: lê as taxas de rejeição e de reclamação da reputação de envio.

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
