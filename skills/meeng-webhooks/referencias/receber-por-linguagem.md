# Receber o webhook em cada linguagem

A regra é a mesma em toda linguagem: **ler o corpo cru**, conferir a assinatura com o SDK, gravar o `id`, responder 2xx. O que muda é como o framework entrega o corpo cru e os cabeçalhos. O verificador de cada SDK recebe o corpo (texto ou bytes) e os cabeçalhos do jeito que a linguagem os tem.

O segredo vem de uma variável de ambiente (`MEENG_WEBHOOK_SECRET`), nunca do código.

## TypeScript e JavaScript

O verificador aceita os `Headers` do `fetch` e o objeto de cabeçalhos do Node, e o corpo como texto ou bytes. Em **Express**, o corpo cru exige `express.raw`, na rota do webhook, antes de qualquer `express.json()`:

```ts nocheck
import express from "express";
import { Meeng } from "@meeng-dev/sdk";

const app = express();
const webhooks = new Meeng().webhooks(process.env.MEENG_WEBHOOK_SECRET ?? "");

app.post("/webhooks/meeng", express.raw({ type: "application/json" }), (req, res) => {
  try {
    const evento = webhooks.constructEvent(req.body, { headers: req.headers }); // req.body é um Buffer
    console.log(evento.type, evento.id);
    res.sendStatus(200);
  } catch {
    res.sendStatus(401);
  }
});
```

Em Next.js (App Router), Hono, Bun, Deno e Workers, use `await request.text()` como no exemplo da skill principal. No Next.js com Pages Router é preciso desligar o leitor de corpo da rota (`export const config = { api: { bodyParser: false } }`) e ler o fluxo cru.

O mesmo receptor com o servidor padrão do Node, sem framework, conferido pela verificação das skills:

```ts
import { createServer } from "node:http";

import { Meeng, WebhookVerificationError } from "@meeng-dev/sdk";

const webhooks = new Meeng().webhooks(process.env.MEENG_WEBHOOK_SECRET ?? "");

createServer(async (req, res) => {
  if (req.method !== "POST" || req.url !== "/webhooks/meeng") {
    res.writeHead(404).end();
    return;
  }
  const pedacos: Buffer[] = [];
  for await (const pedaco of req) {
    pedacos.push(pedaco as Buffer);
  }
  const corpo = Buffer.concat(pedacos); // o corpo cru

  try {
    const evento = webhooks.constructEvent(corpo, { headers: req.headers });
    console.log(evento.type, evento.id);
    res.writeHead(200).end("ok");
  } catch (erro) {
    if (erro instanceof WebhookVerificationError) {
      res.writeHead(401).end(erro.code);
      return;
    }
    res.writeHead(500).end();
  }
}).listen(8080);
```

## Go

O exemplo da skill principal usa `net/http`. Com **Gin** ou **Echo**, leia o corpo com `io.ReadAll(c.Request.Body)` (não use `c.BindJSON`, que consome o corpo) e passe `c.Request.Header`. Com **Chi** e **Fiber** vale a mesma regra: o corpo em bytes, antes de decodificar.

## Python

A parte que decide é uma função que não depende de framework: recebe o corpo cru e os cabeçalhos e devolve o status HTTP.

```python
import os

from meeng import Meeng, WebhookVerificationError

meeng = Meeng()  # lê MEENG_API_KEY; aqui só serve para criar o verificador
webhooks = meeng.webhooks(os.environ.get("MEENG_WEBHOOK_SECRET", ""))

processados: set[str] = set()  # em produção, uma tabela com o id como chave única


def receber(corpo: bytes, cabecalhos: dict[str, str]) -> int:
    """Devolve o status HTTP da resposta ao Meeng.dev."""
    try:
        evento = webhooks.construct_event(corpo, headers=cabecalhos)
    except WebhookVerificationError as erro:
        print("assinatura recusada:", erro.code)
        return 401

    if evento["id"] in processados:
        return 200  # repetido: já foi tratado
    processados.add(evento["id"])

    if evento["type"] == "email.delivered":
        print("entregue", evento["data"]["id"])
    elif evento["type"] == "email.bounced":
        print("devolvido", evento["data"]["bounce"]["type"])
    elif evento["type"] == "email.complained":
        print("marcado como spam", evento["data"]["to"])
    return 200
```

Ligue ao framework passando o corpo cru. Estes trechos são a cola de cada framework (não são conferidos pela verificação automática das skills):

```python nocheck
# Flask
from flask import Flask, request

app = Flask(__name__)

@app.post("/webhooks/meeng")
def meeng_webhook():
    status = receber(request.get_data(), dict(request.headers))  # get_data() é o corpo cru
    return "", status
```

```python nocheck
# FastAPI
from fastapi import FastAPI, Request, Response

app = FastAPI()

@app.post("/webhooks/meeng")
async def meeng_webhook(request: Request) -> Response:
    status = receber(await request.body(), dict(request.headers))  # body() é o corpo cru
    return Response(status_code=status)
```

Em Django use `request.body` (e `@csrf_exempt` na view, porque o aviso vem de fora e leva a própria assinatura).

## PHP

```php
<?php

use Meeng\Meeng;
use Meeng\WebhookVerificationException;

$meeng = new Meeng(); // lê MEENG_API_KEY; aqui só serve para criar o verificador
$webhooks = $meeng->webhooks((string) getenv('MEENG_WEBHOOK_SECRET'));

$corpo = (string) file_get_contents('php://input'); // o corpo cru
$cabecalhos = function_exists('getallheaders') ? getallheaders() : [];

try {
    $evento = $webhooks->constructEvent($corpo, $cabecalhos);
} catch (WebhookVerificationException $erro) {
    http_response_code(401);
    echo $erro->errorCode;
    exit;
}

// Ignore os avisos repetidos: grave $evento['id'] numa tabela com chave única.
switch ($evento['type']) {
    case 'email.delivered':
        error_log('entregue ' . $evento['data']['id']);
        break;
    case 'email.bounced':
        error_log('devolvido ' . implode(',', $evento['data']['to']));
        break;
}
http_response_code(200);
```

No Laravel, leia `$request->getContent()` (o corpo cru) e `$request->headers->all()`, e tire a rota do `VerifyCsrfToken`. No Symfony, `$request->getContent()`.

## Outras linguagens

Ruby, Java, C#, Elixir, Dart, Rust e Swift têm `webhooks(secret)` com `verify` e `constructEvent` (com a grafia da linguagem), com os mesmos códigos de erro. Os nomes exatos estão no README de cada SDK (`github.com/meeng-dev/sdk`, pasta da linguagem). Em Rails, leia `request.raw_post`; em Phoenix, guarde o corpo cru num `Plug` antes do `Plug.Parsers`; em ASP.NET, `Request.EnableBuffering()` e leia o `Body` como texto.

## Conferir a assinatura à mão (só se não houver SDK)

O esquema, para quem não pode usar o SDK:

1. Leia `X-Meeng-Signature` (`v1=<hex>`) e `X-Meeng-Timestamp` (milissegundos desde 1970).
2. Recuse se a diferença entre o timestamp e a hora do seu servidor passa de 5 minutos.
3. Calcule `HMAC-SHA256` de `"<timestamp>.<corpo cru>"`, com o segredo `whsec_...` **inteiro como chave literal** (sem decodificar), e escreva em hexadecimal.
4. Compare `"v1=" + hex` com o cabeçalho **em tempo constante** (`hmac.compare_digest`, `crypto.timingSafeEqual`, `hash_equals`, `subtle.ConstantTimeCompare`).

```go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

// verify diz se o aviso é do Meeng.dev: signature é o cabeçalho X-Meeng-Signature,
// timestampMs o X-Meeng-Timestamp, e body o corpo cru.
func verify(secret, signature, timestampMs string, body []byte, now time.Time) bool {
	ts, err := strconv.ParseInt(timestampMs, 10, 64)
	if err != nil {
		return false
	}
	if math.Abs(float64(now.UnixMilli()-ts)) > float64((5 * time.Minute).Milliseconds()) {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestampMs + "."))
	mac.Write(body)
	expected := "v1=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func main() {
	secret := "whsec_exemplo"
	body := []byte(`{"id":"1","type":"webhook.test"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	signature := "v1=" + hex.EncodeToString(mac.Sum(nil))

	fmt.Println(verify(secret, signature, ts, body, time.Now()))       // true
	fmt.Println(verify(secret, signature, ts, []byte("{}"), time.Now())) // false
}
```
