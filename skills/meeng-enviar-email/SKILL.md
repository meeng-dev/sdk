---
name: meeng-enviar-email
description: Ensina a enviar e-mail pelo Meeng.dev, a plataforma brasileira de e-mail para desenvolvedores, com os SDKs oficiais (TypeScript, Go, Python, PHP, Ruby, Java, C#, Elixir, Dart, Rust e Swift) ou a API REST em https://meeng.dev/api/v1. Use quando o pedido for enviar e-mail transacional (confirmação de pedido, recibo, código de acesso, recuperação de senha, aviso), enviar vários e-mails de uma vez (lote), agendar ou cancelar um envio, anexar arquivo, usar template com variáveis, evitar envio duplicado com Idempotency-Key, ler o status de um e-mail, ou tratar erro 4xx e limite de requisições (429) de uma chave mg_. Cobre envio simples, lote, agendamento, anexos, idempotência, erros e limites.
---

# Enviar e-mail pelo Meeng.dev

Esta skill leva um agente do zero até um envio de e-mail funcionando e seguro. Este arquivo tem os passos essenciais e as regras que valem sempre; cada assunto tem o detalhe em `referencias/`.

Fale com o dono do projeto em português simples. Quando um termo técnico for inevitável (idempotência, DNS, webhook), explique em uma frase. Pergunte só o que falta: a linguagem do projeto, o endereço de quem envia (`from`) e se o domínio dele já está verificado no Meeng.dev.

## Quando usar

Use quando o trabalho é fazer o sistema do dono mandar e-mail por `POST /api/v1/emails` ou pelo SDK. Para cadastrar e verificar o domínio de envio, use a skill `meeng-dominios`; para saber que o e-mail foi entregue, `meeng-webhooks`; para e-mail em massa para uma lista de contatos, `meeng-contatos-e-campanhas`.

Não use para outro provedor de e-mail. Para algo que não está nesta skill, a fonte da verdade é o contrato da API: `https://meeng.dev/api/v1/doc`. Nunca invente campo, rota ou código de erro: se não estiver no contrato, diga ao dono que não existe e proponha o que fazer sem ele.

## O que é preciso ter

- **Chave de API** no formato `mg_<id>_<segredo>`, criada no painel em Configurações › Chaves (`https://meeng.dev/configuracoes/chaves`). O valor completo só aparece na criação. Vai no cabeçalho `Authorization: Bearer mg_...`, só no servidor.
- **Domínio verificado**: o `from` precisa ser de um domínio da conta com status `SUCCESS`. Sem isso o envio responde 400. Passo a passo em `meeng-dominios`.
- **Variáveis de ambiente**: `MEENG_API_KEY` (a chave) e, só se precisar mudar, `MEENG_BASE_URL` (o endereço do app, sem `/api/v1`: o SDK acrescenta o prefixo; o padrão já é `https://meeng.dev`). Os SDKs leem as duas sozinhos. Só `https` é aceito, e o cliente nunca segue redirecionamento.

## Regras que valem sempre

1. A chave de API fica só no servidor, em variável de ambiente. Nunca no navegador, no app de celular, no repositório, em log ou em chat. Se vazou, apague a chave no painel e crie outra.
2. `200` com `emailId` quer dizer **aceito na fila**, não entregue. A entrega só vale pelo evento `email.delivered` (webhook) ou pelo status lido em `GET /v1/emails/{id}`.
3. Todo envio que não pode sair duas vezes leva a sua `Idempotency-Key` (uma chave sua, estável para aquele envio, como `pedido-1042-confirmacao`). Repetir a chamada com a mesma chave e o mesmo corpo devolve o primeiro `emailId`, sem enviar de novo. Os SDKs não inventam chave. Detalhes em [`referencias/idempotencia-e-erros.md`](referencias/idempotencia-e-erros.md).
4. Mande sempre `html` **e** `text`: quem lê em cliente de e-mail simples recebe o texto, e o texto ajuda a não cair no spam.
5. `to`, `cc`, `bcc` e `replyTo` aceitam um endereço ou uma lista. A API não confere o formato do endereço: confira antes de chamar.
6. Destinatário na lista de supressão (já deu bounce duro, reclamou ou foi bloqueado à mão) **não recebe**, e a API responde `200` mesmo assim: o e-mail fica com o status `SUPPRESSED`. Quem precisa saber lê o status ou escuta `email.suppressed`.
7. O limite padrão é de **2 requisições por segundo por organização**. Para muitos e-mails use o lote (até 100 por chamada). Em `429`, espere o tempo do cabeçalho `Retry-After` e tente de novo; nunca em laço sem pausa.
8. Uma recusa `4xx` diz o motivo em `error.message`: leia e corrija o pedido, não tente de novo às cegas.
9. Nunca simule entrega nem trate `200` como "chegou". Teste com endereços reais do dono.

## Roteiro essencial

1. **Instalar o SDK da linguagem** (nomes por registro em [`referencias/linguagens.md`](referencias/linguagens.md)) e criar o cliente a partir do ambiente.
2. **Confirmar o domínio** do `from` (`meeng-dominios`).
3. **Enviar um e-mail de teste** para o próprio dono, com o exemplo abaixo.
4. **Escolher o formato**: envio simples ([`referencias/envio.md`](referencias/envio.md)), lote, agendado ou com anexo ([`referencias/lote-agendamento-anexos.md`](referencias/lote-agendamento-anexos.md)).
5. **Proteger contra repetição** com `Idempotency-Key` e tratar os erros ([`referencias/idempotencia-e-erros.md`](referencias/idempotencia-e-erros.md)).
6. **Acompanhar a entrega** por webhook (`meeng-webhooks`) em vez de consultar e-mail por e-mail.

Exemplo mínimo em TypeScript (o SDK devolve `{ data, error }`; uma falha de rede rejeita a promessa):

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

const { data, error } = await meeng.emails.send(
  {
    from: "Loja Exemplo <pedidos@envio.exemplo.com.br>",
    to: "cliente@example.com",
    subject: "Pedido 1042 confirmado",
    html: "<p>Recebemos o seu pedido <strong>1042</strong>. Obrigado!</p>",
    text: "Recebemos o seu pedido 1042. Obrigado!",
  },
  { idempotencyKey: "pedido-1042-confirmacao" },
);

if (error) {
  throw new Error(`o Meeng.dev recusou o envio: ${error.code} ${error.message}`);
}
console.log(data.emailId); // aceito na fila; a entrega chega por webhook
```

O mesmo envio em Go:

```go
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("") // vazio: a chave vem de MEENG_API_KEY; o endereço, de MEENG_BASE_URL
	subject := "Pedido 1042 confirmado"
	html := "<p>Recebemos o seu pedido <strong>1042</strong>. Obrigado!</p>"
	text := "Recebemos o seu pedido 1042. Obrigado!"

	sent, err := client.Emails.Send(context.Background(), meeng.EmailsSendParams{
		From:    "Loja Exemplo <pedidos@envio.exemplo.com.br>",
		To:      meeng.StringList{"cliente@example.com"},
		Subject: &subject,
		HTML:    &html,
		Text:    &text,
	}, meeng.WithIdempotencyKey("pedido-1042-confirmacao"))
	if err != nil {
		log.Fatal(err) // *meeng.Error: StatusCode, Code, Message, RetryAfter
	}
	fmt.Println(sent.EmailID)
}
```

Em Python e PHP:

```python
from meeng import Meeng

meeng = Meeng()  # lê MEENG_API_KEY e MEENG_BASE_URL

data, _ = meeng.emails.send(
    {
        "from": "Loja Exemplo <pedidos@envio.exemplo.com.br>",
        "to": "cliente@example.com",
        "subject": "Pedido 1042 confirmado",
        "html": "<p>Recebemos o seu pedido <strong>1042</strong>. Obrigado!</p>",
        "text": "Recebemos o seu pedido 1042. Obrigado!",
    },
    options={"idempotency_key": "pedido-1042-confirmacao"},
)
if data is None:  # só acontece com raise_on_error=False
    raise RuntimeError("o Meeng.dev não respondeu")
print(data["emailId"])
```

```php
<?php

use Meeng\Meeng;

$meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

$sent = $meeng->emails->send([
    'from' => 'Loja Exemplo <pedidos@envio.exemplo.com.br>',
    'to' => 'cliente@example.com',
    'subject' => 'Pedido 1042 confirmado',
    'html' => '<p>Recebemos o seu pedido <strong>1042</strong>. Obrigado!</p>',
    'text' => 'Recebemos o seu pedido 1042. Obrigado!',
], 'pedido-1042-confirmacao');
echo $sent['emailId'];
```

## Mapa das referências

| Quando você precisa de | Abra |
|---|---|
| Todos os campos do envio, template com variáveis, cabeçalhos, responder a um e-mail, ler o status | [`referencias/envio.md`](referencias/envio.md) |
| Lote de até 100, agendar, reagendar, cancelar, anexos | [`referencias/lote-agendamento-anexos.md`](referencias/lote-agendamento-anexos.md) |
| `Idempotency-Key`, formato de erro, tabela de erros, limite de requisições, limite do plano | [`referencias/idempotencia-e-erros.md`](referencias/idempotencia-e-erros.md) |
| Instalar o SDK, nome do pacote por registro, o mesmo envio nas 11 linguagens | [`referencias/linguagens.md`](referencias/linguagens.md) |

## Ao terminar

Entregue ao dono, em português: o que foi feito, o que foi testado (um envio real para o próprio endereço, um repetido com a mesma `Idempotency-Key`, um para destinatário suprimido, um erro de propósito), o que falta (domínio verificado, webhook de entrega, limite do plano) e quem faz cada passo.
