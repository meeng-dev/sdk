# Idempotência, erros e limites

## Idempotência: não enviar duas vezes

Uma chamada pode ser repetida sem querer: a rede cai depois que o servidor aceitou, o job tenta de novo, o usuário clica duas vezes. Sem proteção, o cliente recebe dois e-mails iguais. A `Idempotency-Key` resolve isso.

Vale em `POST /api/v1/emails` e `POST /api/v1/emails/batch`. Nas outras rotas o cabeçalho não faz nada.

- É um texto seu, de **1 a 256 caracteres**, mandado no cabeçalho `Idempotency-Key`. Os SDKs mandam quando você passa a chave e nunca inventam uma.
- A chave é **por organização** e fica guardada por **24 horas**. O espaço de chaves é o mesmo para envio simples e lote.
- Mesma chave e **mesmo corpo**: a API devolve o resultado do primeiro envio (`{"emailId": "..."}`, ou a lista de ids no lote), sem enviar de novo. A ordem das chaves do JSON não importa.
- Mesma chave e **corpo diferente**: `409` com código `NOT_UNIQUE` e a mensagem `Idempotency-Key already used with a different payload`.
- Outra chamada com a mesma chave ainda em andamento: `409` com `Request with same Idempotency-Key is in progress. Retry later.` Espere um instante e repita.
- Se a primeira chamada **falhou**, nada fica guardado e repetir executa de novo.

**Como escolher a chave:** uma que descreva o envio, não uma aleatória. `pedido-1042-confirmacao`, `fatura-2026-12-aviso`, `usuario-77-recuperar-senha-2026-12-01T10`. Uma chave aleatória gerada a cada tentativa não protege nada, porque a repetição teria outra chave. Se o mesmo pedido pode mandar dois e-mails diferentes (confirmação e entrega), a chave inclui o motivo.

## Formato do erro

Uma recusa tem status `4xx` ou `5xx` e, no corpo:

```json
{ "error": { "code": "NOT_FOUND", "message": "Email not found" } }
```

- `code` é um destes: `BAD_REQUEST` (400), `UNAUTHORIZED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `METHOD_NOT_ALLOWED` (405), `NOT_UNIQUE` (409), `RATE_LIMITED` (429), `INTERNAL_SERVER_ERROR` (500).
- Os textos de `message` estão em inglês: identifique o erro pelo par status e `code`, e use o texto para mostrar ao desenvolvedor, nunca ao cliente final.
- Quando o corpo da chamada tem um campo faltando ou do tipo errado, a resposta é `400` no formato da validação: `{"success": false, "error": {"issues": [...], "name": "ZodError"}}`. Os SDKs leem isso como o código `ZodError` e a primeira falha como mensagem (`from: Required`). Cada item de `issues` tem o `path` do campo.
- Alguns erros (por exemplo ao verificar um domínio que não existe) trazem `{"error": "texto"}`, sem `code`; os SDKs entregam `INTERNAL_SERVER_ERROR` com o texto como mensagem.

Como cada SDK entrega o erro:

| Linguagem | Como o erro chega |
|---|---|
| TypeScript | `{ data: null, error: { code, message, status } }`; falha de rede rejeita a promessa |
| Go | `*meeng.Error` com `StatusCode`, `Code`, `Message` e `RetryAfter` |
| Python | levanta `MeengHTTPError` (`status_code`, `error`); com `raise_on_error=False` devolve `(None, error)` |
| PHP | lança `Meeng\MeengException` (`status`, `errorCode`, `retryAfter`) |
| Ruby, Java, C#, Dart, Elixir, Rust, Swift | exceção ou `Result` com os mesmos campos (`status`, `code`, `message`, `retry_after`) |

## Erros de envio que você vai ver

| Status e código | Mensagem | O que fazer |
|---|---|---|
| 401 `UNAUTHORIZED` | `No Authorization header provided` | mande `Authorization: Bearer mg_...` |
| 403 `FORBIDDEN` | `Invalid API token` | chave errada, apagada ou de outro ambiente; crie outra no painel |
| 400 `BAD_REQUEST` | `From email is invalid` | `from` precisa ser exatamente um endereço |
| 400 `BAD_REQUEST` | `Domain: x of from email is wrong` | o domínio não é da conta; cadastre-o (`meeng-dominios`) |
| 400 `BAD_REQUEST` | `Domain: x is not verified` | publique o DNS e verifique (`meeng-dominios`) |
| 403 `FORBIDDEN` | `API key does not have access to domain: x` | a chave é restrita a outro domínio |
| 403 `FORBIDDEN` | `O plano Grátis envia só para os endereços da própria organização. ...` | o destinatário não é da organização e ela está no plano Grátis: assine o plano Pro ou envie a um endereço da equipe |
| 403 `FORBIDDEN` | `O limite diário de envio foi atingido (N e-mails por dia). ...` ou `O limite mensal do plano Grátis foi atingido (N e-mails por mês). ...` | o limite do dia ou do mês acabou: espere o dia (a conta é no horário de Brasília) ou troque de plano |
| 403 `FORBIDDEN` | (sem `message` útil) com uma chave "Só envio" | a chave "Só envio" só envia, remarca e cancela; leitura, domínios, contatos e o resto pedem uma chave de acesso total |
| 404 `NOT_FOUND` | `Template not found` | `templateId` errado ou de outra organização |
| 400 `BAD_REQUEST` | `Either text or html is required` | falta `text` ou `html` |
| 400 `BAD_REQUEST` | `"inReplyTo" is invalid` | `inReplyToId` não é de um e-mail da organização |
| 400 `BAD_REQUEST` | `Email already processed` | só se reagenda ou cancela e-mail `SCHEDULED` |
| 409 `NOT_UNIQUE` | `Idempotency-Key already used with a different payload` | uma chave por envio diferente |
| 429 `RATE_LIMITED` | `Rate limit exceeded. Try again in N seconds.` | espere `Retry-After` |

## Limite de requisições (429)

- Cada organização tem um limite de requisições por segundo, **2 por padrão**. É por organização, não por chave: duas chaves dividem o mesmo limite.
- Toda resposta traz `X-RateLimit-Limit`, `X-RateLimit-Remaining` e `X-RateLimit-Reset` (o segundo, em tempo Unix, em que a janela acaba).
- Estourou: `429` com `RATE_LIMITED` e o cabeçalho `Retry-After` (segundos). Espere esse tempo e repita **com a mesma `Idempotency-Key`**.
- Para mandar muitos e-mails, prefira o lote (100 por chamada) a 100 chamadas.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

type Corpo = Parameters<typeof meeng.emails.send>[0];

const esperar = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

// Envia com a mesma Idempotency-Key a cada tentativa: repetir é seguro.
async function enviar(corpo: Corpo, chave: string, tentativas = 4): Promise<string> {
  for (let n = 1; ; n++) {
    const { data, error } = await meeng.emails.send(corpo, { idempotencyKey: chave });
    if (!error) {
      return data.emailId ?? "";
    }
    const repetivel = error.status === 429 || (error.status !== undefined && error.status >= 500);
    if (!repetivel || n >= tentativas) {
      throw new Error(`${error.status} ${error.code}: ${error.message}`);
    }
    await esperar(Math.min(1000 * 2 ** (n - 1), 8000)); // 1 s, 2 s, 4 s
  }
}

const id = await enviar(
  {
    from: "Loja Exemplo <pedidos@envio.exemplo.com.br>",
    to: "cliente@example.com",
    subject: "Pedido 1042 confirmado",
    text: "Recebemos o seu pedido 1042.",
  },
  "pedido-1042-confirmacao",
);
console.log(id);
```

Em Go, o `*meeng.Error` traz o `RetryAfter` pronto:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	ctx := context.Background()
	subject, text := "Pedido 1042 confirmado", "Recebemos o seu pedido 1042."
	params := meeng.EmailsSendParams{
		From:    "pedidos@envio.exemplo.com.br",
		To:      meeng.StringList{"cliente@example.com"},
		Subject: &subject,
		Text:    &text,
	}

	for attempt := 1; ; attempt++ {
		sent, err := client.Emails.Send(ctx, params, meeng.WithIdempotencyKey("pedido-1042-confirmacao"))
		if err == nil {
			fmt.Println(sent.EmailID)
			return
		}
		var refusal *meeng.Error
		if !errors.As(err, &refusal) || attempt >= 4 {
			log.Fatal(err)
		}
		switch {
		case refusal.StatusCode == http.StatusConflict:
			log.Fatal("a mesma Idempotency-Key com outro corpo: ", refusal.Message)
		case refusal.StatusCode == http.StatusTooManyRequests:
			time.Sleep(max(refusal.RetryAfter, time.Second))
		case refusal.StatusCode >= 500:
			time.Sleep(time.Duration(attempt) * time.Second)
		default:
			log.Fatal(refusal) // 4xx: corrija o pedido, não repita
		}
	}
}
```

## Limites do plano

O plano da organização limita quantos e-mails saem por dia e por mês, quantos domínios, listas de contatos e webhooks existem, e **para quem** o plano Grátis envia. Como cada um chega:

- **Plano Grátis, destinatário de fora da equipe, ou o limite do dia ou do mês estourado: a API recusa na hora**, com `403 FORBIDDEN` e o motivo em português em `error.message`. Nada é aceito, então não há e-mail que falhe depois: trate o `403` no envio. O plano Grátis envia só para os endereços da própria organização.
- Um bloqueio posto pela plataforma na organização, ou um limite diário que a plataforma escreveu à mão para a conta, **não** é recusado na entrada: a API responde `200` e o e-mail termina em `FAILED`, com o motivo no evento (`Email sending limit reached`). Acompanhe os eventos `email.failed`.
- Criar domínio, lista de contatos ou webhook além do plano dá `403` com o nome do limite como mensagem (`DOMAIN`, `CONTACT_BOOK`). No plano Grátis a organização tem uma lista de contatos.

O uso e o limite aparecem no painel, em Configurações › Planos (`https://meeng.dev/configuracoes/planos`). Dúvida que o painel não resolve: `ajuda@meeng.dev`.

## Limites que não mudam

| O quê | Limite |
|---|---|
| E-mails por chamada de lote | 100 |
| Anexos por e-mail | 0: use links no corpo |
| Tamanho de `Idempotency-Key` | 1 a 256 caracteres; vale 24 horas |
| Itens por página em `GET /v1/emails` | 1 a 50 |
| Requisições por segundo | 2 por organização, padrão |
