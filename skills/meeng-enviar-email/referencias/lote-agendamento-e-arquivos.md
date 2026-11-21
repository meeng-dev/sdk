# Lote, agendamento e links de arquivos

## Lote: até 100 e-mails numa chamada

`POST /api/v1/emails/batch` recebe uma **lista** de corpos iguais aos do envio simples (veja [`envio.md`](envio.md)) e responde `{ "data": [ { "emailId": "..." }, ... ] }`.

- No máximo **100** e-mails por chamada; mais que isso é recusado. Lista vazia dá `400` com `No emails provided for bulk send`.
- Aceita o cabeçalho `Idempotency-Key`, como o envio simples; o espaço de chaves é o mesmo dos dois.
- Cada e-mail do lote é validado como um envio simples (domínio verificado, template existente, supressão).
- **O lote é tudo ou nada**: todos os e-mails são validados antes de qualquer um ser gravado, e são gravados juntos. Se um deles é recusado, nenhum é aceito e a resposta diz qual e por quê; corrija e envie o lote de novo. A `Idempotency-Key` continua importando: repetir o mesmo lote com a mesma chave depois de um erro de rede devolve os mesmos ids, na mesma ordem, sem enviar de novo.
- A ordem da resposta é a ordem da entrada: o `emailId` da posição `n` é o do e-mail `n` do lote. Guarde mesmo assim o vínculo pelo seu próprio cabeçalho (`X-Pedido`) e confira com `GET /v1/emails/{id}`.
- Um lote conta como **uma** requisição no limite de requisições por segundo; cada e-mail dele conta no limite do plano.

Para mais de 100, divida em pedaços de 100 e dê uma chave diferente a cada pedaço, derivada do conteúdo (`campanha-natal-lote-0007`).

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

const destinatarios = ["a@example.com", "b@example.com", "c@example.com"];

const lote = destinatarios.map((to) => ({
  from: "Loja Exemplo <avisos@envio.exemplo.com.br>",
  to,
  subject: "Seu cupom chegou",
  html: "<p>Use <strong>NATAL10</strong> até domingo.</p>",
  text: "Use NATAL10 até domingo.",
}));

const { data, error } = await meeng.emails.batch(lote, { idempotencyKey: "cupom-natal-lote-0001" });

if (error) {
  throw new Error(`${error.code}: ${error.message}`);
}
console.log(`${data.length} e-mails aceitos`, data[0]?.emailId);
```

Em Go, o lote é uma fatia de `meeng.EmailsBatchParams`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	subject := "Seu cupom chegou"
	text := "Use NATAL10 até domingo."

	var lote []meeng.EmailsBatchParams
	for _, to := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		lote = append(lote, meeng.EmailsBatchParams{
			From:    "avisos@envio.exemplo.com.br",
			To:      meeng.StringList{to},
			Subject: &subject,
			Text:    &text,
		})
	}

	sent, err := client.Emails.Batch(context.Background(), lote, meeng.WithIdempotencyKey("cupom-natal-lote-0001"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(sent.Data), "e-mails aceitos")
}
```

## Agendar, reagendar e cancelar

- **Agendar:** mande `scheduledAt` no envio, em ISO 8601 com o fuso: `2026-12-01T09:00:00-03:00` ou `2026-12-01T12:00:00Z`. Só ISO: frases como "amanhã às 9h" não valem. Não há limite de distância; uma data no passado envia já.
- O e-mail agendado fica com o status `SCHEDULED` até sair.
- **Reagendar:** `PATCH /api/v1/emails/{id}` com `{ "scheduledAt": "..." }`. Aqui a data precisa ser em **UTC, terminando em `Z`**: não aceita deslocamento de fuso. Só vale enquanto o status é `SCHEDULED`.
- **Cancelar:** `POST /api/v1/emails/{id}/cancel`. Só vale para `SCHEDULED`; depois que o e-mail entrou na fila, `400` com `Email already processed`. Para saber que o cancelamento valeu, leia o status (`CANCELLED`); não dependa de um webhook para isso.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

const enviado = await meeng.emails.send(
  {
    from: "Loja Exemplo <avisos@envio.exemplo.com.br>",
    to: "cliente@example.com",
    subject: "Sua fatura vence amanhã",
    text: "A fatura de dezembro vence amanhã.",
    scheduledAt: "2026-12-04T09:00:00-03:00",
  },
  { idempotencyKey: "fatura-2026-12-aviso" },
);
if (enviado.error) {
  throw new Error(enviado.error.message);
}
const emailId = enviado.data.emailId;
if (!emailId) {
  throw new Error("a API não devolveu o emailId");
}

// Mudou de ideia: outro horário, em UTC com Z.
const reagendado = await meeng.emails.update(emailId, { scheduledAt: "2026-12-05T12:00:00Z" });
if (reagendado.error) {
  console.error(reagendado.error.code, reagendado.error.message);
}

// Ou cancele de vez.
const cancelado = await meeng.emails.cancel(emailId);
if (cancelado.error) {
  console.error(cancelado.error.code, cancelado.error.message); // 400 "Email already processed" se já saiu
}
```

## Arquivos: mande um link, não um anexo

O Meeng.dev **não anexa, não guarda e não repassa arquivos** ([decisão do produto](../SKILL.md#regras-que-valem-sempre)). Os SDKs não têm o campo `attachments`; quem envia hospeda o arquivo (no armazenamento da própria aplicação) e põe o link no corpo do e-mail.

- A API recusa `attachments` com itens: `400`, `ZodError`, com a mensagem `Attachments are not supported. Include links to your files in the email body.` Uma lista vazia passa, e o campo ausente também. O relay SMTP faz o mesmo (`550 5.7.1`), inclusive para imagem embutida por `cid:`: para uma imagem no corpo, use uma URL `https`.
- Para um documento sensível (fatura, contrato, laudo), o link tem de **expirar** ou exigir login; um link público fica válido para quem receber o e-mail encaminhado. Gere o link assinado no seu sistema na hora do envio.
- O link precisa continuar vivo enquanto a pessoa puder abrir o e-mail. Decida o prazo (30 dias, por exemplo) e diga-o no texto: "o link vale até 20/12".

```go
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")

	subject := "Sua fatura"
	html := `<p>Sua fatura de novembro está pronta.</p>` +
		`<p><a href="https://exemplo.com.br/faturas/1042?assinatura=abc123">Baixar a fatura (PDF)</a>, disponível até 20/12.</p>`
	text := "Sua fatura de novembro está pronta: https://exemplo.com.br/faturas/1042?assinatura=abc123 (disponível até 20/12)."

	sent, err := client.Emails.Send(context.Background(), meeng.EmailsSendParams{
		From:    "financeiro@envio.exemplo.com.br",
		To:      meeng.StringList{"cliente@example.com"},
		Subject: &subject,
		HTML:    &html,
		Text:    &text,
	}, meeng.WithIdempotencyKey("fatura-1042-envio"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.EmailID)
}
```

Em Python:

```python
from meeng import Meeng

meeng = Meeng()

link = "https://exemplo.com.br/faturas/1042?assinatura=abc123"

data, _ = meeng.emails.send(
    {
        "from": "financeiro@envio.exemplo.com.br",
        "to": "cliente@example.com",
        "subject": "Sua fatura",
        "html": f'<p>Sua fatura está pronta.</p><p><a href="{link}">Baixar a fatura (PDF)</a>, disponível até 20/12.</p>',
        "text": f"Sua fatura está pronta: {link} (disponível até 20/12).",
    },
    options={"idempotency_key": "fatura-1042-envio"},
)
if data is None:  # só acontece com raise_on_error=False
    raise RuntimeError("o Meeng.dev não respondeu")
print(data["emailId"])
```
