---
name: meeng-enviar-email
description: Ensina a enviar e-mail pelo Meeng.dev, a plataforma brasileira de e-mail para desenvolvedores, pela API, o caminho principal (o relay SMTP é só reserva), com os SDKs oficiais publicados (TypeScript e Go primeiro; Python, PHP, Ruby, C#, Elixir, Dart e Rust também) ou a API REST em https://meeng.dev/api/v1. A mesma chave mg_ serve na API, no formato compatível com bibliotecas de outros provedores (https://api.meeng.dev) e no MAIL_URL=meeng:// de apps Arandu. Use quando o pedido for enviar e-mail transacional (confirmação de pedido, recibo, código de acesso, aviso), enviar em lote, agendar ou cancelar, mandar arquivo por link (sem anexos), usar template com variáveis, evitar envio duplicado com Idempotency-Key, ler o status de um e-mail, ligar um app Arandu pelo MAIL_URL, usar a chave "Só envio", ou tratar erro 4xx, a recusa do plano Grátis (403) e o limite de requisições (429).
---

# Enviar e-mail pelo Meeng.dev

Esta skill leva um agente do zero até um envio de e-mail funcionando e seguro. Este arquivo tem os passos essenciais e as regras que valem sempre; cada assunto tem o detalhe em `referencias/`.

Fale com o dono do projeto em português simples. Quando um termo técnico for inevitável (idempotência, DNS, webhook), explique em uma frase. Pergunte só o que falta: a linguagem do projeto, o endereço de quem envia (`from`) e se o domínio dele já está verificado no Meeng.dev.

## Quando usar

Use quando o trabalho é fazer o sistema do dono mandar e-mail por `POST /api/v1/emails` ou pelo SDK. Para cadastrar e verificar o domínio de envio, use a skill `meeng-dominios`; para saber que o e-mail foi entregue, `meeng-webhooks`; para e-mail em massa para uma lista de contatos, `meeng-contatos-e-campanhas`.

Não use para outro provedor de e-mail. Para algo que não está nesta skill, a fonte da verdade é o contrato da API: `https://meeng.dev/api/v1/doc`. Nunca invente campo, rota ou código de erro: se não estiver no contrato, diga ao dono que não existe e proponha o que fazer sem ele.

## O caminho principal e as alternativas

O envio de um sistema é **pela API**: uma chamada `POST /v1/emails` (direto ou por um SDK) que devolve o `emailId`, aceita `Idempotency-Key`, agendamento e modelos, e responde com erro estruturado. O relay SMTP (`meeng-smtp`) é só reserva, para o programa que não sabe chamar API. Todas as portas abaixo usam a **mesma chave** `mg_...` e as mesmas regras (domínio verificado, supressão, plano):

| Quando | Como |
|---|---|
| Sistema próprio, em qualquer linguagem | SDK (`@meeng-dev/sdk`, `github.com/meeng-dev/sdk/go` e os outros de [`referencias/linguagens.md`](referencias/linguagens.md)) ou `POST https://meeng.dev/api/v1/emails` |
| Já usa a biblioteca de envio de outro provedor, que deixa trocar o endereço base | troque só o endereço base para `https://api.meeng.dev` e a chave pela `mg_...`; o resto do código fica ([`referencias/linguagens.md`](referencias/linguagens.md), "Bibliotecas de outros provedores") |
| App Arandu (framework v0.72.0 ou mais novo) | `MAIL_URL=meeng://<chave>@meeng.dev` no `.env`, sem código ([`referencias/linguagens.md`](referencias/linguagens.md), "App Arandu") |
| Programa que só fala SMTP | relay SMTP, `meeng-smtp` (reserva) |

Uma chave **"Só envio"** (a opção de criar a chave só para enviar) envia, remarca e cancela e-mail; qualquer outra chamada com ela responde `403`. É a chave certa para o servidor que só manda e-mail: se vazar, não lê contatos nem domínios.

## O que é preciso ter

- **Chave de API** no formato `mg_<id>_<segredo>`, criada no painel em Configurações › Chaves (`https://meeng.dev/configuracoes/chaves`). O valor completo só aparece na criação. Vai no cabeçalho `Authorization: Bearer mg_...`, só no servidor. Para um serviço que só envia, crie a chave "Só envio".
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
10. **No plano Grátis a conta envia só para os endereços da própria organização.** Outro destinatário, o limite do dia ou o do mês é recusado **na hora**, com `403 FORBIDDEN` e o motivo em português em `error.message`; não é e-mail aceito que falha depois. Para enviar a outras pessoas, a organização assina o plano Pro (Configurações › Planos).
11. **Sem anexos.** O Meeng.dev não anexa nem guarda arquivos: o e-mail leva o link do arquivo no corpo (um link que expira, se o documento é sensível). `attachments` com itens dá `400`; os SDKs nem têm o campo.

## Roteiro essencial

1. **Instalar o SDK da linguagem** (nomes por registro em [`referencias/linguagens.md`](referencias/linguagens.md)) e criar o cliente a partir do ambiente.
2. **Confirmar o domínio** do `from` (`meeng-dominios`).
3. **Enviar um e-mail de teste** para o próprio dono, com o exemplo abaixo.
4. **Escolher o formato**: envio simples ([`referencias/envio.md`](referencias/envio.md)), lote, agendado ou com link de arquivo ([`referencias/lote-agendamento-e-arquivos.md`](referencias/lote-agendamento-e-arquivos.md)).
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
| Lote de até 100, agendar, reagendar, cancelar, arquivos por link (sem anexos) | [`referencias/lote-agendamento-e-arquivos.md`](referencias/lote-agendamento-e-arquivos.md) |
| `Idempotency-Key`, formato de erro, tabela de erros, limite de requisições, limite do plano | [`referencias/idempotencia-e-erros.md`](referencias/idempotencia-e-erros.md) |
| Instalar o SDK, nome do pacote por registro, o mesmo envio nas 11 linguagens | [`referencias/linguagens.md`](referencias/linguagens.md) |

## Ao terminar

Entregue ao dono, em português: o que foi feito, o que foi testado (um envio real para o próprio endereço, um repetido com a mesma `Idempotency-Key`, um para destinatário suprimido, um erro de propósito), o que falta (domínio verificado, webhook de entrega, limite do plano) e quem faz cada passo.
