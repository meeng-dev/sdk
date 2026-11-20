# @meeng-dev/sdk (TypeScript e JavaScript)

SDK do meeng para Node.js 20.3 ou mais novo (também funciona em Bun e Deno): e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Tem o mesmo formato do SDK do useSend: troque o import, o nome do cliente e a chave, e o resto do código continua igual.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão. Sem dependências em tempo de execução.

> O pacote ainda não foi publicado: o nome `@meeng-dev/sdk` é uma proposta aguardando confirmação.

## Começando

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng("mg_123");

const { data, error } = await meeng.emails.send({
  from: "hello@acme.com",
  to: "cliente@example.com",
  subject: "Olá",
  html: "<p>Olá!</p>",
});
```

Toda chamada devolve `{ data, error }`: `data` quando deu certo, `error` (`{ code, message, status }`) quando a API recusou. Uma falha de rede rejeita a promessa.

Sem argumentos, `new Meeng()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. O segundo argumento é o endereço do app (para instalações próprias), sem `/api/v1`; ele precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro.

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
- `bulkCreate`: adiciona vários contatos a uma lista numa requisição.
- `bulkDelete`: remove vários contatos de uma lista numa requisição.

### `meeng.contactBooks`

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

- `emailTimeSeries`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `reputationMetrics`: lê as taxas de rejeição e de reclamação da reputação de envio.

`emails.create` é outro nome de `emails.send`, e `campaigns.getAll` de `campaigns.list`.

`emails.send` e `emails.batch` aceitam `{ idempotencyKey }` como último argumento: a mesma chave com o mesmo corpo devolve o primeiro resultado, sem enviar de novo.

`emails.batch` devolve na `data` a lista de ids (`[{ emailId }]`), como o useSend.

## Webhooks

```ts
const webhooks = meeng.webhooks("whsec_xxx");

// Express, Next.js, Hono: passe o corpo cru, antes de interpretá-lo.
const event = webhooks.constructEvent(rawBody, { headers: request.headers });

if (event.type === "email.delivered") {
  console.log(event.data.to);
}
```

`verify(body, { headers })` só confere a assinatura e devolve `true` ou `false`. `constructEvent` confere e devolve o evento tipado, ou lança `WebhookVerificationError` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH`, `INVALID_BODY` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance`, em milissegundos; um número negativo desliga a checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
npm ci
npm run typecheck
npm test
```
