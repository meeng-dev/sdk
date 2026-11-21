# @meeng-dev/sdk (TypeScript e JavaScript)

SDK do Meeng.dev para Node.js 20.3 ou mais novo (também funciona em Bun e Deno): e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks.

Gerado pelo `sdk/generator` a partir do contrato da API do Meeng.dev do Meeng.dev (versão 1.0.0); não edite à mão. Sem dependências em tempo de execução.

> O pacote ainda não foi publicado: o nome `@meeng-dev/sdk` é uma proposta aguardando confirmação.

## Começando

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng("mg_123");

const { data, error } = await meeng.emails.send({
  from: "contato@acme.com.br",
  to: "ana@example.com",
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

`emails.batch` devolve na `data` a lista de ids (`[{ emailId }]`).

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

```ts
const meeng = new Meeng("mg_...", "https://meeng.dev", {
  timeoutMs: 10_000,
  maxRetries: 3,
  retryDelayMs: 1_000,
});
```

## Exemplos

A pasta [`examples/`](examples) tem programas que rodam: enviar um e-mail, enviar em lote, agendar (remarcar e cancelar), adicionar um domínio (com os registros DNS) e verificar um webhook.

| Exemplo | Arquivo | Rode com |
|---|---|---|
| Enviar um e-mail | `examples/send-email.ts` | `node examples/send-email.ts` |
| Enviar em lote | `examples/send-batch.ts` | `node examples/send-batch.ts` |
| Agendar, remarcar e cancelar | `examples/schedule-email.ts` | `node examples/schedule-email.ts` |
| Adicionar um domínio | `examples/add-domain.ts` | `node examples/add-domain.ts` |
| Verificar um webhook | `examples/verify-webhook.ts` | `node examples/verify-webhook.ts` |

Todos leem `MEENG_API_KEY` e, se quiser apontar para outro app, `MEENG_BASE_URL`; o de webhook roda sem chave, com uma requisição montada no próprio programa.

## Paginação

As listagens de e-mails, contatos e campanhas são paginadas por `page` (a partir de 1) e, nas duas primeiras, `limit`. `emails.list` devolve `count`, o total de e-mails; `campaigns.list` devolve `totalPage`, o total de páginas; `contacts.list` devolve a lista sem total, e a última página é a que vem com menos de `limit` contatos.

```ts
const todos = [];
for (let page = 1; ; page++) {
  const { data, error } = await meeng.emails.list({ page, limit: 100 });
  if (error) throw new Error(error.message);
  todos.push(...data.data);
  if (todos.length >= data.count || data.data.length === 0) break;
}
```

## Erros

Uma recusa da API é o `error` do resultado, com `status`, `code` e `message`:

```ts
const { data, error } = await meeng.emails.get("em_123");
if (error?.code === "NOT_FOUND") {
  // o e-mail não existe, ou é de outro time
}
```

Os códigos que a API devolve: `UNAUTHORIZED` (sem chave), `FORBIDDEN` (chave inválida ou sem permissão), `NOT_FOUND`, `NOT_UNIQUE` (a mesma chave de idempotência com outro corpo, ou ainda em andamento), `RATE_LIMITED`, `BAD_REQUEST`, `METHOD_NOT_ALLOWED`, `NOT_IMPLEMENTED` e `INTERNAL_SERVER_ERROR`. Uma validação recusada vira o código `ZodError`, e a mensagem é a primeira falha (`from: Required`).

## Tentativas automáticas

O cliente repete sozinho a chamada que pode ser repetida, até `maxRetries` vezes depois da primeira tentativa (2 por padrão; 0 envia cada chamada uma vez só):

| Resposta | Repete? |
|---|---|
| 429 (limite de requisições por segundo) | Sempre: a chamada não foi processada, então qualquer uma pode ser repetida. |
| 500, 502, 503, 504 ou falha de conexão | Só quando repetir não faz o trabalho duas vezes: uma leitura (`GET`), um `PUT`, um `DELETE`, ou uma chamada com chave de idempotência. Um envio sem chave não é repetido. |
| 409 e qualquer outra recusa | Nunca. |

A primeira repetição espera `retryDelayMs` (500 ms por padrão), cada uma seguinte espera o dobro, até 30 segundos, e o `Retry-After` que a API devolve vale no lugar dessa espera. Esgotadas as tentativas, a última resposta é o erro.

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

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance`, em milissegundos; um número negativo desliga a checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
npm ci
npm run typecheck
npm test
```

O `./check.sh` da raiz do repositório sobe uma API de mentira gerada do contrato (`go run ./cmd/fakeapi`), que recusa o que o contrato não permite, e roda contra ela o teste de contrato (cada método do SDK é chamado, e nenhuma operação pode ficar de fora) e os exemplos.
