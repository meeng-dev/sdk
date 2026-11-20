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
