---
name: meeng-webhooks
description: Ensina a receber os avisos do Meeng.dev (webhooks), a plataforma brasileira de e-mail para desenvolvedores, com os SDKs oficiais (TypeScript, Go, Python, PHP e as demais linguagens) ou a API REST. Use quando o pedido for saber que um e-mail foi entregue, devolvido (bounce), aberto, clicado ou marcado como spam, receber o aviso de domínio verificado ou de contato criado, criar um endpoint de webhook no painel, conferir a assinatura X-Meeng-Signature, tratar o segredo whsec_, lidar com tentativas repetidas e duplicadas, ou depurar um aviso que não chega. Cobre criação, assinatura, eventos, idempotência, tentativas e testes.
---

# Webhooks do Meeng.dev

Um webhook é um aviso que o Meeng.dev manda ao servidor do dono, por um POST assinado, a cada coisa que acontece: e-mail entregue, devolvido, aberto, clicado, reclamação de spam, domínio verificado, contato criado. É por ele que o sistema do dono fica sabendo o que **aconteceu depois** do envio: um `200` no envio quer dizer só "aceito na fila".

Fale com o dono em português simples. Explique em uma frase o que é webhook (o Meeng.dev "liga" para o seu servidor quando algo acontece) e assinatura (o carimbo que prova que o aviso veio mesmo do Meeng.dev). Pergunte só o que falta: a linguagem e o framework do servidor, o endereço público onde o aviso vai chegar e quais eventos interessam.

## Quando usar

Use quando o trabalho é receber e tratar avisos do Meeng.dev. Para mandar e-mail use `meeng-enviar-email`; para o domínio, `meeng-dominios`.

Não use para receber e-mail (caixa de entrada). Para algo que não está aqui, a fonte da verdade é o contrato `https://meeng.dev/api/v1/doc`. Nunca invente evento ou campo.

## Como criar o endpoint

Os webhooks são criados **no painel**: Webhooks (`https://meeng.dev/painel/webhooks`) › criar um webhook. Não há rota pública da API para isso.

| Campo | Regra |
|---|---|
| Endereço | HTTPS público do servidor do dono. Endereços internos (localhost, redes privadas) são recusados |
| Eventos | os que o endpoint recebe; **vazio recebe todos**. Escolha só os que usa |
| Domínios | opcional: só os eventos dos domínios marcados; vazio recebe os de todos |
| Segredo | o painel gera um `whsec_...`; ou o dono informa o seu (mínimo 16 caracteres) |

O segredo aparece na criação; quem gerencia o webhook pode revelá-lo de novo depois, e há "Gerar novo segredo" (o anterior para de assinar na hora: atualize o servidor antes). Cada webhook tem o **seu** segredo. O botão "Testar" manda um evento `webhook.test`, e a tela de cada aviso mostra as tentativas, a resposta do servidor e o botão de tentar de novo.

## Regras que valem sempre

1. **Confira a assinatura** com o corpo **cru**, exatamente como chegou, antes de interpretar o JSON. Frameworks que já leram o JSON e o reescrevem mudam os bytes e a assinatura deixa de bater. Use o corpo em bytes ou texto cru.
2. Use o SDK para conferir (`constructEvent`, `ConstructEvent`, `construct_event`): ele confere a assinatura, o prazo (5 minutos de tolerância) e devolve o evento tipado. Não escreva a conferência à mão sem necessidade.
3. **Responda 2xx depressa.** Cada tentativa tem 10 segundos. Grave o aviso (ou ponha numa fila) e responda; o trabalho pesado vem depois. Qualquer outra resposta, ou demora, é uma falha e o aviso é repetido.
4. **Cada aviso pode chegar mais de uma vez.** O campo `id` do aviso é o da entrega e é o **mesmo em todas as tentativas**: grave os `id` já processados e ignore os repetidos.
5. **Não conte com a ordem.** Um `email.delivered` pode chegar antes de um `email.sent`, e um aviso atrasado depois de um mais novo. Use `data.occurredAt` e o status para decidir, e nunca "volte" um status mais avançado para um anterior.
6. O segredo `whsec_...` fica só no servidor, em variável de ambiente. Nunca em repositório, log ou chat. Vazou: gere um novo no painel.
7. Entregue só o que o dono precisa: assine os eventos que o sistema usa, não todos.
8. Não registre o corpo inteiro do aviso em log aberto: ele traz endereços de e-mail de clientes (dado pessoal pela LGPD).

## Roteiro essencial

1. **Escolher os eventos** ([`referencias/eventos.md`](referencias/eventos.md)): em geral `email.delivered`, `email.bounced`, `email.complained`, `email.failed` e `email.suppressed`; `domain.verified` se o sistema espera o domínio ficar pronto.
2. **Escrever o receptor**: um endereço público que lê o corpo cru, confere a assinatura, grava o `id`, responde 2xx. Exemplos abaixo e em [`referencias/receber-por-linguagem.md`](referencias/receber-por-linguagem.md).
3. **Criar o webhook no painel** com o endereço do receptor e copiar o segredo para a variável de ambiente `MEENG_WEBHOOK_SECRET`.
4. **Testar** com o botão "Testar" (evento `webhook.test`) e com um envio real para o próprio endereço do dono.
5. **Tratar o que o evento pede**: bounce duro e reclamação saem da lista de envio do sistema; falha de envio vira alerta ([`referencias/eventos.md`](referencias/eventos.md)).
6. **Conferir as entregas** quando um aviso não chega ([`referencias/tentativas-e-problemas.md`](referencias/tentativas-e-problemas.md)).

O envelope de todo aviso:

```json
{
  "id": "cm3x...",
  "type": "email.delivered",
  "version": "2026-01-18",
  "createdAt": "2026-12-01T13:05:41.000Z",
  "teamId": 1,
  "attempt": 1,
  "data": { "id": "...", "status": "DELIVERED", "from": "...", "to": ["..."], "occurredAt": "..." }
}
```

Os cabeçalhos: `X-Meeng-Signature` (`v1=` mais o HMAC-SHA256 em hexadecimal de `{timestamp}.{corpo}`, com o segredo como chave), `X-Meeng-Timestamp` (milissegundos), `X-Meeng-Event` (o tipo) e `X-Meeng-Call` (o id da entrega). O Meeng.dev manda também um segundo conjunto de cabeçalhos legados com os mesmos valores; o SDK lê os dois.

Receptor em TypeScript, no formato de `Request` e `Response` que Next.js, Hono, Bun, Deno e Cloudflare Workers usam:

```ts
import { Meeng, WebhookVerificationError } from "@meeng-dev/sdk";

const meeng = new Meeng(); // lê MEENG_API_KEY; aqui o cliente só serve para criar o verificador
const webhooks = meeng.webhooks(process.env.MEENG_WEBHOOK_SECRET ?? "");

const processados = new Set<string>(); // em produção, uma tabela com o id como chave única

export async function POST(request: Request): Promise<Response> {
  const corpo = await request.text(); // o corpo cru, antes de qualquer JSON.parse

  let evento;
  try {
    evento = webhooks.constructEvent(corpo, { headers: request.headers });
  } catch (erro) {
    if (erro instanceof WebhookVerificationError) {
      return new Response(erro.code, { status: 401 }); // assinatura inválida ou fora do prazo
    }
    throw erro;
  }

  if (processados.has(evento.id)) {
    return new Response("ok", { status: 200 }); // repetido: já foi tratado
  }
  processados.add(evento.id);

  switch (evento.type) {
    case "email.delivered":
      console.log("entregue", evento.data.id, evento.data.to);
      break;
    case "email.bounced":
      console.log("devolvido", evento.data.to, evento.data.bounce.type, evento.data.bounce.subType);
      break;
    case "email.complained":
      console.log("marcado como spam", evento.data.to); // pare de enviar para este endereço
      break;
    case "domain.verified":
      console.log("domínio pronto", evento.data.name);
      break;
    default:
      break; // evento que o sistema não usa: responda 2xx mesmo assim
  }
  return new Response("ok", { status: 200 });
}
```

Receptor em Go, com `net/http` (o corpo cru vem de `io.ReadAll`):

```go
package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	webhooks := meeng.NewWebhooks(os.Getenv("MEENG_WEBHOOK_SECRET"))
	var seen sync.Map // em produção, uma tabela com o id como chave única

	http.HandleFunc("/webhooks/meeng", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20)) // o corpo cru
		if err != nil {
			http.Error(w, "corpo grande demais", http.StatusRequestEntityTooLarge)
			return
		}
		event, err := webhooks.ConstructEvent(body, r.Header)
		if err != nil {
			var refusal *meeng.WebhookVerificationError
			if errors.As(err, &refusal) {
				http.Error(w, string(refusal.Code), http.StatusUnauthorized)
				return
			}
			http.Error(w, "erro", http.StatusInternalServerError)
			return
		}
		if _, repeated := seen.LoadOrStore(event.Message().ID, true); repeated {
			w.WriteHeader(http.StatusOK) // repetido: já foi tratado
			return
		}

		switch event := event.(type) {
		case *meeng.EmailWebhookEvent:
			log.Println(event.Type, event.Data.ID, event.Data.To)
		case *meeng.EmailBouncedWebhookEvent:
			log.Println("devolvido", event.Data.To, event.Data.Bounce.Type)
		case *meeng.DomainWebhookEvent:
			log.Println("domínio", event.Data.Name, event.Type)
		}
		w.WriteHeader(http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Mapa das referências

| Quando você precisa de | Abra |
|---|---|
| Os 20 eventos e o de teste, os campos de cada um, o que fazer com cada evento | [`referencias/eventos.md`](referencias/eventos.md) |
| Receptor em Python, PHP e como ligar ao Flask, FastAPI, Express e Laravel; conferir a assinatura à mão | [`referencias/receber-por-linguagem.md`](referencias/receber-por-linguagem.md) |
| Tentativas, intervalos, endpoint desligado, aviso que não chega, erros de assinatura | [`referencias/tentativas-e-problemas.md`](referencias/tentativas-e-problemas.md) |

## Ao terminar

Entregue ao dono, em português: qual endereço recebe os avisos, quais eventos foram assinados, o que o sistema faz com cada um, o que foi testado (aviso de teste, e-mail real entregue, aviso repetido, assinatura errada) e o que falta (guardar o `id` numa tabela, alerta para falha).
