# Envio simples

`POST /api/v1/emails` aceita um e-mail e o coloca na fila de envio. A resposta `200` traz só o `emailId`; o resto da vida do e-mail (enviado, entregue, aberto, devolvido) se lê depois.

## Campos do corpo

| Campo | Tipo | Regra |
|---|---|---|
| `from` | texto | **obrigatório**. Um único endereço, com ou sem nome: `pedidos@envio.exemplo.com.br` ou `Loja Exemplo <pedidos@envio.exemplo.com.br>`. O domínio precisa ser da conta e estar verificado |
| `to` | texto ou lista | **obrigatório**. Um endereço ou uma lista. A API não confere o formato |
| `subject` | texto | obrigatório quando não há `templateId` |
| `html` | texto | o corpo em HTML. `html` ou `text` é obrigatório |
| `text` | texto | o corpo em texto puro. Mande os dois |
| `cc`, `bcc` | texto ou lista | cópia e cópia oculta |
| `replyTo` | texto ou lista | para onde vai a resposta de quem lê |
| `templateId` | texto | o id de um template criado no painel |
| `variables` | objeto texto para texto | os valores do template; só texto |
| `headers` | objeto texto para texto | cabeçalhos extras (veja abaixo) |
| `attachments` | lista | até 10 anexos; veja [`lote-agendamento-anexos.md`](lote-agendamento-anexos.md) |
| `scheduledAt` | texto | data e hora em ISO 8601 com fuso; veja o mesmo arquivo |
| `inReplyToId` | texto | o `emailId` de um e-mail da mesma organização, para responder na mesma conversa |

Os nomes são em `camelCase` (`replyTo`, `scheduledAt`, `templateId`). Os SDKs usam o mesmo nome em todas as linguagens, ajustado ao jeito de cada uma (em Go `ReplyTo` e `ScheduledAt`).

A resposta é `{"emailId": "..."}`.

## Regras do remetente

- `from` precisa ser **exatamente um** endereço. Lista, grupo ou dois endereços dão `400` com `From email is invalid`.
- Domínio que a conta não tem: `400` com `Domain: <domínio> of from email is wrong`. Domínio que existe mas não está verificado: `400` com `Domain: <domínio> is not verified`. Resolva com a skill `meeng-dominios`.
- Chave de API restrita a um domínio só envia por ele: de outro, `403` com `API key does not have access to domain`.
- Use um subdomínio de envio (`envio.exemplo.com.br`) em vez do domínio principal: a reputação de quem envia e-mail em massa não contamina o e-mail da empresa.

## Template com variáveis

Templates são criados e editados **só no painel** (`https://meeng.dev/painel/modelos`); a API apenas os usa. Copie o id do template no painel e mande `templateId` e `variables`.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

const { data, error } = await meeng.emails.send(
  {
    from: "Loja Exemplo <pedidos@envio.exemplo.com.br>",
    to: ["cliente@example.com"],
    templateId: "COLE_O_ID_DO_TEMPLATE_DO_PAINEL",
    variables: { nome: "Ana", pedido: "1042", total: "R$ 189,90" },
  },
  { idempotencyKey: "pedido-1042-confirmacao" },
);

if (error) {
  console.error(error.code, error.message); // por exemplo 404 NOT_FOUND "Template not found"
} else {
  console.log(data.emailId);
}
```

Como as variáveis valem:

- O **assunto** e o **HTML** vêm do template; o `subject` e o `html` do corpo são ignorados quando há `templateId`.
- No assunto, `{{chave}}` (chaves coladas, sem espaço) vira o valor. No corpo do template, a variável `{{nome}}` vira o valor, ou o texto padrão quando o template define um (`{{nome,fallback=cliente}}`), ou fica como está quando não há valor nem texto padrão.
- O campo `text` do corpo **não** passa por variáveis: ele vale como vem. Monte o texto você mesmo.
- Um `templateId` que não é da organização da chave dá `404` com `Template not found`, mesmo que o id exista em outra conta.

## Cabeçalhos extras

`headers` aceita pares de texto. Ficam de fora, em silêncio e sem erro, os que a plataforma controla: `from`, `to`, `cc`, `bcc`, `subject`, `reply-to`, `date`, `message-id`, `return-path`, `received`, `content-type`, `content-transfer-encoding`, `mime-version`, `dkim-signature`, os que começam com `arc-`, `resent-` e `x-meeng-`, e qualquer nome ou valor com quebra de linha. Para marcar um e-mail do seu lado, use um cabeçalho `X-` seu (`X-Pedido: 1042`) e guarde o `emailId` junto do pedido.

A plataforma acrescenta `X-Meeng-Email-ID` com o id do e-mail.

## Responder na mesma conversa

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
	subject := "Re: Pedido 1042"
	text := "Seu pedido saiu para entrega."
	original := "ID_DO_EMAIL_ORIGINAL" // o emailId guardado no primeiro envio

	sent, err := client.Emails.Send(context.Background(), meeng.EmailsSendParams{
		From:        "pedidos@envio.exemplo.com.br",
		To:          meeng.StringList{"cliente@example.com"},
		Subject:     &subject,
		Text:        &text,
		InReplyToID: &original,
	}, meeng.WithIdempotencyKey("pedido-1042-entrega"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.EmailID)
}
```

Um `inReplyToId` que não é de um e-mail da organização dá `400` com `"inReplyTo" is invalid`.

## Ler o status de um e-mail

`GET /api/v1/emails/{emailId}` devolve o e-mail e a lista `emailEvents`, um evento por mudança. `GET /api/v1/emails` lista do mais novo ao mais velho, com `page`, `limit` (de 1 a 50, padrão 50), `startDate`, `endDate` (UTC, terminando em `Z`) e `domainId`; a resposta é `{ data: [...], count }`, e `count` é o total do filtro.

Os status, do primeiro ao último na ordem de precedência: `SCHEDULED`, `QUEUED`, `SENT`, `DELIVERY_DELAYED`, `BOUNCED`, `REJECTED`, `RENDERING_FAILURE`, `DELIVERED`, `OPENED`, `CLICKED`, `COMPLAINED`, `FAILED`, `CANCELLED`, `SUPPRESSED`.

- `DELIVERED` só aparece quando o servidor do destinatário aceitou a mensagem (resposta 250). Nenhum e-mail é marcado como entregue sem isso.
- `BOUNCED` é a devolução: o endereço não existe, a caixa está cheia ou o servidor recusou. Um bounce duro põe o endereço na lista de supressão da conta, e os próximos envios para ele ficam `SUPPRESSED`.
- `FAILED` é uma falha do lado do envio (por exemplo o limite do plano); o `data` do evento diz o motivo.

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
	ctx := context.Background()

	email, err := client.Emails.Get(ctx, "ID_DO_EMAIL")
	if err != nil {
		log.Fatal(err)
	}
	for _, event := range email.EmailEvents {
		fmt.Println(event.CreatedAt, event.Status)
	}

	page, err := client.Emails.List(ctx, &meeng.EmailsListQuery{Page: 1, Limit: 20, StartDate: "2026-12-01T00:00:00Z"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(page.Count, "e-mails no período")
}
```

Para acompanhar muitos e-mails, não consulte um por um: receba os eventos por webhook (`meeng-webhooks`).
