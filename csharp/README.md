# MeengDev.Sdk (.NET)

SDK do meeng para .NET 8 ou mais novo (C#): e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências; o JSON é gerado em tempo de compilação, então o pacote funciona com trimming e AOT.

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O pacote ainda não foi publicado: o nome `MeengDev.Sdk` (NuGet) é uma proposta aguardando confirmação.

## Começando

```csharp
using Meeng;

var meeng = new MeengClient("mg_123");

var sent = await meeng.Emails.SendAsync(new EmailsSendParams
{
    From = "hello@acme.com",
    To = ["cliente@example.com"],
    Subject = "Olá",
    Html = "<p>Olá!</p>",
});
Console.WriteLine(sent.EmailId);
```

Uma recusa da API lança `MeengException` (`Status`, `Code`, como `NOT_UNIQUE` ou `FORBIDDEN`, `ApiMessage` e `RetryAfter`). O cliente pode ser compartilhado entre threads.

`new MeengClient()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. `MeengClientOptions.BaseUrl` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `HttpClient`, `Handler` e `Timeout` trocam o transporte e o prazo (o cliente padrão não segue redirecionamentos).

## Recursos

### `meeng.Emails`

- `SendAsync`: envia um e-mail, agora ou em `scheduledAt`.
- `BatchAsync`: envia até 100 e-mails numa requisição.
- `GetAsync`: lê um e-mail com os seus eventos.
- `ListAsync`: lista e-mails, do mais novo ao mais velho, com período e domínio opcionais.
- `UpdateAsync`: reagenda um e-mail que ainda não saiu.
- `CancelAsync`: cancela um e-mail agendado.

### `meeng.Domains`

- `ListAsync`: lista os domínios de envio do time.
- `CreateAsync`: adiciona um domínio de envio e devolve os registros DNS a publicar.
- `VerifyAsync`: pede para conferir de novo os registros DNS do domínio.
- `GetAsync`: lê um domínio com os seus registros DNS.
- `DeleteAsync`: remove um domínio.

### `meeng.Contacts`

- `CreateAsync`: adiciona um contato a uma lista de contatos.
- `ListAsync`: lista os contatos de uma lista, com filtro opcional por e-mail ou id.
- `GetAsync`: lê um contato.
- `UpdateAsync`: muda só os campos enviados de um contato.
- `UpsertAsync`: cria o contato, ou o substitui quando ele já existe.
- `DeleteAsync`: remove um contato.
- `BulkCreateAsync`: adiciona vários contatos a uma lista numa requisição.
- `BulkDeleteAsync`: remove vários contatos de uma lista numa requisição.

### `meeng.ContactBooks`

- `ListAsync`: lista as listas de contatos do time.
- `CreateAsync`: cria uma lista de contatos.
- `GetAsync`: lê uma lista de contatos.
- `UpdateAsync`: muda só os campos enviados de uma lista de contatos.
- `DeleteAsync`: remove uma lista de contatos e os seus contatos.

### `meeng.Campaigns`

- `CreateAsync`: cria uma campanha, e a envia agora quando `sendNow` é verdadeiro.
- `ListAsync`: lista campanhas, com filtro opcional por status ou texto.
- `GetAsync`: lê uma campanha com os seus contadores.
- `DeleteAsync`: remove uma campanha.
- `ScheduleAsync`: agenda uma campanha, ou a envia agora quando não há horário.
- `PauseAsync`: pausa uma campanha em andamento ou agendada.
- `ResumeAsync`: retoma uma campanha pausada.

### `meeng.Analytics`

- `EmailTimeSeriesAsync`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `ReputationMetricsAsync`: lê as taxas de rejeição e de reclamação da reputação de envio.

Os parâmetros de caminho são `string`; os filtros de listagem são o `<Recurso><Método>Query` (uma lista em `DomainId` vira um `domainId` por valor). `Emails.SendAsync` e `Emails.BatchAsync` aceitam `new RequestOptions { IdempotencyKey = "pedido-1" }`: a mesma chave com o mesmo corpo devolve o primeiro resultado. Os destinatários são `IReadOnlyList<string>`; a API aceita uma string solta, e o SDK a lê como lista de um item.

## Webhooks

```csharp
var webhooks = meeng.Webhooks("whsec_xxx");

// O corpo cru, antes de interpretá-lo.
var evento = webhooks.ConstructEvent(corpo, Request.Headers);

if (evento is EmailWebhookEvent email)
{
    Console.WriteLine(string.Join(",", email.Data.To));
}
```

`Verify(corpo, cabecalhos)` só confere a assinatura e devolve `true` ou `false`. `ConstructEvent` confere e devolve o evento, ou lança `WebhookVerificationException` com um `Code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`WebhookOptions.Tolerance`; negativa desliga a checagem; `WebhookOptions.Secret` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `UnknownWebhookEvent`.

## Desenvolvimento

```sh
dotnet test --nologo
```
