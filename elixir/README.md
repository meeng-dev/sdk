# meeng_sdk (Elixir)

SDK do Meeng.dev para Elixir 1.18 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências além da biblioteca padrão (`:httpc` e o módulo `JSON`).

Gerado pelo `sdk/generator` a partir do contrato da API do Meeng.dev do Meeng.dev (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado: o nome `meeng_sdk` (Hex) é uma proposta aguardando confirmação.

## Começando

```elixir
client = Meeng.new("mg_123")

{:ok, sent} =
  Meeng.Emails.send(client, %{
    from: "contato@acme.com.br",
    to: "ana@example.com",
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

## Configuração

A API do Meeng.dev fica em `https://meeng.dev/api`. O cliente recebe o endereço do app, `https://meeng.dev` (o padrão), e acrescenta `/api/v1` a cada chamada. Crie uma chave em [Configurações > Chaves](https://meeng.dev/configuracoes/chaves); ela tem a forma `mg_<id>_<segredo>`.

- A chave vem do argumento ou de `MEENG_API_KEY`; o endereço, do argumento ou de `MEENG_BASE_URL`.
- Só `https` é aceito. `http` vale para `localhost`, `127.0.0.0/8` e `::1` (um servidor local), para a chave nunca viajar em texto puro. Nenhum cliente segue redirecionamento.

```elixir
meeng =
  Meeng.new("mg_...",
    base_url: "https://meeng.dev",
    timeout: 10_000,
    max_retries: 3,
    retry_delay: 1_000
  )
```

## Exemplos

A pasta [`examples/`](examples) tem programas que rodam: enviar um e-mail, enviar em lote, agendar (remarcar e cancelar), adicionar um domínio (com os registros DNS) e verificar um webhook.

| Exemplo | Arquivo | Rode com |
|---|---|---|
| Enviar um e-mail | `examples/send_email.exs` | `mix run examples/send_email.exs` |
| Enviar em lote | `examples/send_batch.exs` | `mix run examples/send_batch.exs` |
| Agendar, remarcar e cancelar | `examples/schedule_email.exs` | `mix run examples/schedule_email.exs` |
| Adicionar um domínio | `examples/add_domain.exs` | `mix run examples/add_domain.exs` |
| Verificar um webhook | `examples/verify_webhook.exs` | `mix run examples/verify_webhook.exs` |

Todos leem `MEENG_API_KEY` e, se quiser apontar para outro app, `MEENG_BASE_URL`; o de webhook roda sem chave, com uma requisição montada no próprio programa.

## Paginação

As listagens de e-mails, contatos e campanhas são paginadas por `page` (a partir de 1) e, nas duas primeiras, `limit`. `emails.list` devolve `count`, o total de e-mails; `campaigns.list` devolve `totalPage`, o total de páginas; `contacts.list` devolve a lista sem total, e a última página é a que vem com menos de `limit` contatos.

```elixir
defmodule Emails do
  # Todas as páginas de e-mails, de 100 em 100.
  def todos(meeng, page \\ 1, acumulado \\ []) do
    {:ok, %{"data" => data, "count" => count}} = Meeng.Emails.list(meeng, page: page, limit: 100)
    acumulado = acumulado ++ data

    if length(acumulado) >= count or data == [],
      do: acumulado,
      else: todos(meeng, page + 1, acumulado)
  end
end
```

## Erros

Uma recusa da API é `{:error, %Meeng.Error{}}`, com `status`, `code`, `message` e `retry_after`; uma falha de rede tem o código `"transport"`:

```elixir
case Meeng.Emails.get(meeng, "em_123") do
  {:ok, email} -> email
  {:error, %Meeng.Error{code: "NOT_FOUND"}} -> nil
  {:error, error} -> raise error
end
```

Os códigos que a API devolve: `UNAUTHORIZED` (sem chave), `FORBIDDEN` (chave inválida ou sem permissão), `NOT_FOUND`, `NOT_UNIQUE` (a mesma chave de idempotência com outro corpo, ou ainda em andamento), `RATE_LIMITED`, `BAD_REQUEST`, `METHOD_NOT_ALLOWED`, `NOT_IMPLEMENTED` e `INTERNAL_SERVER_ERROR`. Uma validação recusada vira o código `ZodError`, e a mensagem é a primeira falha (`from: Required`).

## Tentativas automáticas

O cliente repete sozinho a chamada que pode ser repetida, até `:max_retries` vezes depois da primeira tentativa (2 por padrão; 0 envia cada chamada uma vez só):

| Resposta | Repete? |
|---|---|
| 429 (limite de requisições por segundo) | Sempre: a chamada não foi processada, então qualquer uma pode ser repetida. |
| 500, 502, 503, 504 ou falha de conexão | Só quando repetir não faz o trabalho duas vezes: uma leitura (`GET`), um `PUT`, um `DELETE`, ou uma chamada com chave de idempotência. Um envio sem chave não é repetido. |
| 409 e qualquer outra recusa | Nunca. |

A primeira repetição espera `:retry_delay` (em milissegundos) (500 ms por padrão), cada uma seguinte espera o dobro, até 30 segundos, e o `Retry-After` que a API devolve vale no lugar dessa espera. Esgotadas as tentativas, a última resposta é o erro.

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

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance:`, em milissegundos; um número negativo desliga a checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
mix compile --warnings-as-errors
mix test --warnings-as-errors
```

O `./check.sh` da raiz do repositório sobe uma API de mentira gerada do contrato (`go run ./cmd/fakeapi`), que recusa o que o contrato não permite, e roda contra ela o teste de contrato (cada método do SDK é chamado, e nenhuma operação pode ficar de fora) e os exemplos.
