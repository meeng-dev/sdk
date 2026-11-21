# Lote, agendamento e anexos

## Lote: até 100 e-mails numa chamada

`POST /api/v1/emails/batch` recebe uma **lista** de corpos iguais aos do envio simples (veja [`envio.md`](envio.md)) e responde `{ "data": [ { "emailId": "..." }, ... ] }`.

- No máximo **100** e-mails por chamada; mais que isso é recusado. Lista vazia dá `400` com `No emails provided for bulk send`.
- Aceita o cabeçalho `Idempotency-Key`, como o envio simples; o espaço de chaves é o mesmo dos dois.
- Cada e-mail do lote é validado como um envio simples (domínio verificado, template existente, supressão).
- **O lote não é atômico**: um erro de validação no meio da lista pode deixar já aceitos os e-mails dos grupos anteriores. Por isso a `Idempotency-Key` importa ainda mais: repetir o mesmo lote com a mesma chave devolve os mesmos ids, na mesma ordem, sem enviar de novo.
- A ordem da resposta segue a ordem da entrada, mas o SDK não garante correspondência por posição quando algum item falha ao ser gravado. Guarde o vínculo pelo seu próprio cabeçalho (`X-Pedido`) e confira pelos eventos de webhook.
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

## Anexos

`attachments` é uma lista de `{ "filename": "...", "content": "..." }` com o conteúdo em **base64**.

- No máximo **10** anexos por e-mail (mais que isso é recusado na validação).
- `filename` e `content` não podem ser vazios. O tipo do arquivo é deduzido da extensão do nome (`.pdf`, `.png`); não existe campo de tipo, nem imagem embutida por `Content-ID`, nem anexo por link.
- A API não confere se o base64 é válido nem o tamanho de cada anexo. Os servidores de destino recusam mensagens grandes, então mantenha o e-mail inteiro pequeno (poucos megabytes) e, para arquivo grande, mande um link para o arquivo no seu armazenamento.

```go
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")

	pdf, err := os.ReadFile("fatura-1042.pdf")
	if err != nil {
		log.Fatal(err)
	}
	subject := "Sua fatura"
	text := "A fatura segue em anexo."

	sent, err := client.Emails.Send(context.Background(), meeng.EmailsSendParams{
		From:    "financeiro@envio.exemplo.com.br",
		To:      meeng.StringList{"cliente@example.com"},
		Subject: &subject,
		Text:    &text,
		Attachments: []meeng.EmailsSendAttachment{
			{Filename: "fatura-1042.pdf", Content: base64.StdEncoding.EncodeToString(pdf)},
		},
	}, meeng.WithIdempotencyKey("fatura-1042-envio"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.EmailID)
}
```

Em Python:

```python
import base64
from pathlib import Path

from meeng import Meeng

meeng = Meeng()

conteudo = base64.b64encode(Path("fatura-1042.pdf").read_bytes()).decode("ascii")

data, _ = meeng.emails.send(
    {
        "from": "financeiro@envio.exemplo.com.br",
        "to": "cliente@example.com",
        "subject": "Sua fatura",
        "text": "A fatura segue em anexo.",
        "attachments": [{"filename": "fatura-1042.pdf", "content": conteudo}],
    },
    options={"idempotency_key": "fatura-1042-envio"},
)
if data is None:  # só acontece com raise_on_error=False
    raise RuntimeError("o Meeng.dev não respondeu")
print(data["emailId"])
```
