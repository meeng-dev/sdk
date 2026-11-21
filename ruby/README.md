# meeng-sdk (Ruby)

SDK do Meeng.dev para Ruby 3.1 ou mais novo: e-mails, domínios, contatos, listas de contatos, campanhas, analytics e webhooks. Não depende de nenhuma gem.

Gerado pelo `sdk/generator` a partir do contrato da API do Meeng.dev do Meeng.dev (versão 1.0.0); não edite à mão.

> A gem ainda não foi publicada: o nome `meeng-sdk` é uma proposta aguardando confirmação.

## Começando

```ruby
require "meeng"

meeng = Meeng::Client.new("mg_123")

sent = meeng.emails.send(
  from: "contato@acme.com.br",
  to: "ana@example.com",
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

```ruby
meeng = Meeng::Client.new("mg_...", "https://meeng.dev", timeout: 10, max_retries: 3, retry_delay: 1.0)
```

## Exemplos

A pasta [`examples/`](examples) tem programas que rodam: enviar um e-mail, enviar em lote, agendar (remarcar e cancelar), adicionar um domínio (com os registros DNS) e verificar um webhook.

| Exemplo | Arquivo | Rode com |
|---|---|---|
| Enviar um e-mail | `examples/send_email.rb` | `ruby examples/send_email.rb` |
| Enviar em lote | `examples/send_batch.rb` | `ruby examples/send_batch.rb` |
| Agendar, remarcar e cancelar | `examples/schedule_email.rb` | `ruby examples/schedule_email.rb` |
| Adicionar um domínio | `examples/add_domain.rb` | `ruby examples/add_domain.rb` |
| Verificar um webhook | `examples/verify_webhook.rb` | `ruby examples/verify_webhook.rb` |

Todos leem `MEENG_API_KEY` e, se quiser apontar para outro app, `MEENG_BASE_URL`; o de webhook roda sem chave, com uma requisição montada no próprio programa.

## Paginação

As listagens de e-mails, contatos e campanhas são paginadas por `page` (a partir de 1) e, nas duas primeiras, `limit`. `emails.list` devolve `count`, o total de e-mails; `campaigns.list` devolve `totalPage`, o total de páginas; `contacts.list` devolve a lista sem total, e a última página é a que vem com menos de `limit` contatos.

```ruby
todos = []
page = 1
loop do
  lista = meeng.emails.list(page: page, limit: 100)
  todos += lista[:data]
  break if todos.size >= lista[:count] || lista[:data].empty?

  page += 1
end
```

## Erros

Uma recusa da API levanta `Meeng::Error`, com `status`, `code`, `message` e `retry_after`:

```ruby
begin
  meeng.emails.get("em_123")
rescue Meeng::Error => erro
  # o e-mail não existe, ou é de outro time
  raise unless erro.code == "NOT_FOUND"
end
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

```ruby
webhooks = meeng.webhooks("whsec_xxx")

# O corpo cru, antes de interpretá-lo.
event = webhooks.construct_event(request.body.read, headers: request.headers)

puts event[:data][:to] if event[:type] == "email.delivered"
```

`verify(body, headers:)` só confere a assinatura e devolve `true` ou `false`. `construct_event` confere e devolve o evento, ou levanta `Meeng::WebhookVerificationError` com um `code`: `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH` ou `INVALID_JSON`.

A assinatura é `v1=` mais o HMAC-SHA256, em hexadecimal, de `{timestamp}.{corpo}` com o segredo do webhook; o timestamp vem em milissegundos. A tolerância padrão é de 5 minutos (`tolerance:`, em milissegundos; um número negativo desliga a checagem; `secret:` troca o segredo de uma checagem). Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`.

Eventos: `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`, `email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.cancelled`, `email.bounced`, `email.clicked`, `email.failed`, `email.opened`, `email.suppressed`, `webhook.test`.

## Desenvolvimento

```sh
ruby -w -Ilib -Itest -e 'Dir["test/*_test.rb"].sort.each { |file| require "./#{file}" }'
```

O `./check.sh` da raiz do repositório sobe uma API de mentira gerada do contrato (`go run ./cmd/fakeapi`), que recusa o que o contrato não permite, e roda contra ela o teste de contrato (cada método do SDK é chamado, e nenhuma operação pode ficar de fora) e os exemplos.
