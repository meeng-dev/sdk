# dev.meeng:sdk (Java)

SDK do meeng para Java 11 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Sem dependências em tempo de execução (usa o `java.net.http` do JDK e um leitor de JSON próprio).

Gerado pelo `sdk/generator` a partir do contrato da API (versão 1.0.0); não edite à mão.

> O artefato ainda não foi publicado: as coordenadas `dev.meeng:sdk` (Maven) são uma proposta aguardando confirmação.

## Começando

```java
import dev.meeng.*;
import java.util.List;

Meeng meeng = new Meeng("mg_123");

EmailsSendResponse sent = meeng.emails().send(
    EmailsService.SendParams.builder()
        .from("hello@acme.com")
        .to(List.of("cliente@example.com"))
        .subject("Olá")
        .html("<p>Olá!</p>")
        .build());
System.out.println(sent.getEmailId());
```

Uma recusa da API lança `MeengException` (`getStatus()`, `getCode()`, como `NOT_UNIQUE` ou `FORBIDDEN`, `getApiMessage()` e `getRetryAfter()`); uma falha de rede, `UncheckedIOException`. O cliente pode ser compartilhado entre threads.

`new Meeng()` lê a chave de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL`. `Meeng.builder(chave).baseUrl(...)` aponta para outro app (instalação própria), sem `/api/v1`; o endereço precisa ser https, e http só vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. `timeout(...)` e `transport(...)` trocam o prazo e o transporte HTTP (o padrão não segue redirecionamentos).

## Recursos

### `meeng.emails()`

- `send`: envia um e-mail, agora ou em `scheduledAt`.
- `batch`: envia até 100 e-mails numa requisição.
- `get`: lê um e-mail com os seus eventos.
- `list`: lista e-mails, do mais novo ao mais velho, com período e domínio opcionais.
- `update`: reagenda um e-mail que ainda não saiu.
- `cancel`: cancela um e-mail agendado.

### `meeng.domains()`

- `list`: lista os domínios de envio do time.
- `create`: adiciona um domínio de envio e devolve os registros DNS a publicar.
- `verify`: pede para conferir de novo os registros DNS do domínio.
- `get`: lê um domínio com os seus registros DNS.
- `delete`: remove um domínio.

### `meeng.contacts()`

- `create`: adiciona um contato a uma lista de contatos.
- `list`: lista os contatos de uma lista, com filtro opcional por e-mail ou id.
- `get`: lê um contato.
- `update`: muda só os campos enviados de um contato.
- `upsert`: cria o contato, ou o substitui quando ele já existe.
- `delete`: remove um contato.
- `bulkCreate`: adiciona vários contatos a uma lista numa requisição.
- `bulkDelete`: remove vários contatos de uma lista numa requisição.

### `meeng.contactBooks()`

- `list`: lista as listas de contatos do time.
- `create`: cria uma lista de contatos.
- `get`: lê uma lista de contatos.
- `update`: muda só os campos enviados de uma lista de contatos.
- `delete`: remove uma lista de contatos e os seus contatos.

### `meeng.campaigns()`

- `create`: cria uma campanha, e a envia agora quando `sendNow` é verdadeiro.
- `list`: lista campanhas, com filtro opcional por status ou texto.
- `get`: lê uma campanha com os seus contadores.
- `delete`: remove uma campanha.
- `schedule`: agenda uma campanha, ou a envia agora quando não há horário.
- `pause`: pausa uma campanha em andamento ou agendada.
- `resume`: retoma uma campanha pausada.

### `meeng.analytics()`

- `emailTimeSeries`: conta e-mails por dia nos últimos 7 ou 30 dias.
- `reputationMetrics`: lê as taxas de rejeição e de reclamação da reputação de envio.

Os parâmetros de caminho são `String`; os filtros de listagem são o `ListParams` do serviço (uma lista em `domainId` vira um `domainId` por valor). `emails().send` e `emails().batch` aceitam `RequestOptions.builder().idempotencyKey("pedido-1").build()`: a mesma chave com o mesmo corpo devolve o primeiro resultado. Os destinatários são `List<String>`; a API aceita uma string solta, e o SDK a lê como lista de um item.

## Webhooks

```java
Webhooks webhooks = meeng.webhooks("whsec_xxx");

// O corpo cru, antes de interpretá-lo.
WebhookEvent event = webhooks.constructEvent(body, headers);

if (event instanceof EmailWebhookEvent) {
    System.out.println(((EmailWebhookEvent) event).getData().getTo());
}
```

`verify(body, headers)` só confere a assinatura e devolve `true` ou `false`. `constructEvent` confere e devolve o evento, ou lança `WebhookVerificationException` com um `getCode()`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`Webhooks.Options.defaults().tolerance(...)`; negativa desliga a checagem; `.secret(...)` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`; os `X-UseSend-*` também são aceitos.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`. Um tipo que o SDK não conhece vira `UnknownWebhookEvent`.

## Desenvolvimento

```sh
mvn -q -B verify
```
