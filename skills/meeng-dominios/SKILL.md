---
name: meeng-dominios
description: Ensina a cadastrar, publicar o DNS, verificar e manter o domínio de envio no Meeng.dev, a plataforma brasileira de e-mail para desenvolvedores, pelo painel, pelos SDKs oficiais ou pela API REST em https://meeng.dev/api/v1. Use quando o pedido for adicionar um domínio para enviar e-mail, descobrir quais registros DNS (MX, SPF, DKIM e DMARC) publicar, conferir por que o domínio não verifica (PENDING, FAILED, TEMPORARY_FAILURE), configurar DMARC, escolher entre domínio e subdomínio de envio, restringir uma chave de API a um domínio, remover um domínio, ou resolver o erro "Domain is not verified" no envio. Cobre criação, registros DNS, verificação, DMARC e remoção.
---

# Domínios de envio no Meeng.dev

Um e-mail só sai pelo Meeng.dev se o endereço de quem envia (`from`) for de um domínio que a conta cadastrou e **verificou**. Verificar é provar, pelo DNS, que o domínio é seu, e publicar as assinaturas (SPF e DKIM) que fazem o e-mail chegar à caixa de entrada em vez do spam. Esta skill leva o dono do zero até um domínio com status `SUCCESS`.

Fale com o dono em português simples. Explique em uma frase o que é DNS (a lista de endereços e regras de um domínio, que fica no lugar onde o domínio foi comprado ou no serviço que cuida dele). Pergunte só o que falta: qual domínio ou subdomínio vai enviar e quem acessa o DNS dele.

## Quando usar

Use quando o trabalho é deixar um domínio pronto para enviar, ou descobrir por que o envio responde `Domain: x is not verified`. Para o envio em si use `meeng-enviar-email`; para saber quando o domínio ficou pronto sem consultar, `meeng-webhooks` (eventos `domain.verified` e `domain.updated`).

Não use para o e-mail de **recebimento** do domínio (caixa de entrada da empresa): os registros do Meeng.dev ficam num subdomínio de retorno (`mail`) e não mexem no MX do domínio principal. Para algo que não está aqui, a fonte da verdade é o contrato `https://meeng.dev/api/v1/doc`. Nunca invente campo ou status.

## Os quatro registros

Ao criar um domínio, a API devolve `dnsRecords`: o que publicar no DNS, na ordem do painel.

| Para quê | Tipo | Nome (relativo ao domínio) | Valor |
|---|---|---|---|
| Retorno de devoluções (bounces) | `MX`, prioridade `10` | `mail` | o servidor de entrada do Meeng.dev |
| SPF: quem pode enviar | `TXT` | `mail` | `v=spf1 include:... ~all` |
| DKIM: assinatura do e-mail | `TXT` | `meeng._domainkey` | `p=...` (a chave pública) |
| DMARC: política (recomendado) | `TXT` | `_dmarc` | `v=DMARC1; p=none;` |

- Em domínios que usam o **retorno por CNAME**, os dois primeiros registros (MX e SPF) vêm como **um só `CNAME`** no nome `mail`, apontando para o domínio de retorno do Meeng.dev ([`referencias/registros-dns.md`](referencias/registros-dns.md)). Publique o que a API devolver, pelo `type` de cada item.
- O **nome** é relativo ao domínio cadastrado: para `exemplo.com.br` o MX é `envio.exemplo.com.br`. Se o cadastro foi de um subdomínio (`envio.exemplo.com.br`), os nomes ganham o sufixo (`mail.envio`, `meeng._domainkey.envio`); copie o que a API devolveu, nunca monte à mão.
- Copie o **valor** exatamente como a API devolveu. A chave DKIM é longa; alguns painéis de DNS pedem para dividi-la em pedaços de até 255 caracteres.
- `ttl` vem como `Auto`: use o padrão do seu provedor de DNS (3600 segundos serve).
- O DMARC é marcado `recommended`: o domínio funciona sem ele, mas os provedores de caixa de entrada cada vez mais o exigem. Veja [`referencias/dmarc.md`](referencias/dmarc.md).

Detalhes por tipo de registro, e como preencher o painel dos provedores de DNS mais comuns, em [`referencias/registros-dns.md`](referencias/registros-dns.md).

## Regras que valem sempre

1. Cadastre um **subdomínio de envio** (`envio.exemplo.com.br`) e não o domínio principal. A reputação de quem envia em massa fica separada da do e-mail da empresa, e os registros não colidem com os que o domínio já tem.
2. O nome do domínio é **único na plataforma inteira**: dois clientes não cadastram o mesmo. Se der erro ao criar um domínio que parece certo, veja se ele não está em outra conta do dono.
3. Publique os registros **exatamente** como a API devolveu. Um espaço a mais, uma aspa colada ou um SPF repetido faz a verificação falhar.
4. Um domínio só pode ter **um** registro SPF por nome. Se já existe um SPF no mesmo nome, junte os `include:` num só registro; dois registros SPF invalidam os dois.
5. Verifique, espere e confira o status pelo `GET`; não assuma que publicou certo. Só `status: SUCCESS` libera o envio.
6. Depois de **72 horas** sem os registros certos, o domínio passa a `FAILED` e para de ser conferido sozinho. Publique e peça a verificação de novo (`PUT /v1/domains/{id}/verify`).
7. Apagar um domínio apaga também as **chaves de API restritas a ele**. Avise o dono antes.
8. O agente nunca tem acesso ao DNS do dono: mostre os registros e peça que ele (ou quem cuida do DNS) os publique.

## Roteiro essencial

1. **Cadastrar o domínio**: `POST /v1/domains` com `name` e `region` (use `us-east-1`, a região padrão que o painel mostra).
2. **Mostrar os registros** (`dnsRecords`) ao dono, um por linha: tipo, nome, valor, prioridade.
3. **Esperar a publicação**: o DNS leva de minutos a algumas horas para valer.
4. **Pedir a verificação**: `PUT /v1/domains/{id}/verify`.
5. **Conferir**: `GET /v1/domains/{id}` até o `status` virar `SUCCESS`. Enquanto o domínio está sendo verificado, cada `GET` refaz a checagem do DNS; além disso, uma varredura confere os domínios pendentes a cada 5 minutos.
6. **Mandar um e-mail de teste** (`meeng-enviar-email`).
7. **Subir o DMARC** de `p=none` para uma política mais firme, depois de uns dias olhando os relatórios ([`referencias/dmarc.md`](referencias/dmarc.md)).

Exemplo completo em TypeScript: cadastra, mostra o que publicar, pede a verificação e espera.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

const criado = await meeng.domains.create({ name: "envio.exemplo.com.br", region: "us-east-1" });
if (criado.error) {
  // 500 "something unexpected happened": nome que não é domínio, região desconhecida ou nome já cadastrado.
  throw new Error(`${criado.error.code}: ${criado.error.message}`);
}
const dominio = criado.data;

console.log(`Publique estes ${dominio.dnsRecords.length} registros no DNS de ${dominio.name}:`);
for (const registro of dominio.dnsRecords) {
  const prioridade = registro.priority ? ` prioridade ${registro.priority}` : "";
  const sugestao = registro.recommended ? " (recomendado)" : "";
  console.log(`${registro.type}  ${registro.name}${prioridade}  ${registro.value}${sugestao}`);
}

// Depois que o dono publicar:
const pedido = await meeng.domains.verify(dominio.id);
if (pedido.error) {
  throw new Error(pedido.error.message);
}

for (let tentativa = 1; tentativa <= 12; tentativa++) {
  await new Promise((resolve) => setTimeout(resolve, 5 * 60 * 1000)); // a varredura roda a cada 5 minutos
  const atual = await meeng.domains.get(dominio.id);
  if (atual.error) {
    throw new Error(atual.error.message);
  }
  console.log(`status do domínio: ${atual.data.status}`);
  if (atual.data.status === "SUCCESS") {
    break;
  }
  if (atual.data.status === "FAILED") {
    const pendentes = atual.data.dnsRecords.filter((r) => r.status !== "SUCCESS");
    console.log("ainda faltam:", pendentes.map((r) => `${r.type} ${r.name}`).join(", "));
    break;
  }
}
```

O mesmo em Go (o `id` do domínio é número, e os métodos pedem o id como texto):

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("") // lê MEENG_API_KEY e MEENG_BASE_URL
	ctx := context.Background()

	domain, err := client.Domains.Create(ctx, meeng.DomainsCreateParams{Name: "envio.exemplo.com.br", Region: "us-east-1"})
	if err != nil {
		log.Fatal(err)
	}
	for _, record := range domain.DNSRecords {
		priority := ""
		if record.Priority != nil {
			priority = " prioridade " + *record.Priority
		}
		fmt.Printf("%s  %s%s  %s\n", record.Type, record.Name, priority, record.Value)
	}

	id := strconv.FormatInt(domain.ID, 10)
	// Depois que o dono publicar os registros:
	if _, err := client.Domains.Verify(ctx, id); err != nil {
		log.Fatal(err)
	}
	for attempt := 0; attempt < 12; attempt++ {
		time.Sleep(5 * time.Minute)
		current, err := client.Domains.Get(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("status do domínio:", current.Status)
		if current.Status == "SUCCESS" || current.Status == "FAILED" {
			return
		}
	}
}
```

Em Python:

```python
import time

from meeng import Meeng

meeng = Meeng()  # lê MEENG_API_KEY e MEENG_BASE_URL

domain, _ = meeng.domains.create({"name": "envio.exemplo.com.br", "region": "us-east-1"})
if domain is None:  # só acontece com raise_on_error=False
    raise RuntimeError("o Meeng.dev não respondeu")
for record in domain["dnsRecords"]:
    print(record["type"], record["name"], record["value"])

# Depois que o dono publicar os registros:
meeng.domains.verify(domain["id"])
for tentativa in range(12):
    time.sleep(5 * 60)
    current, _ = meeng.domains.get(domain["id"])
    if current is not None and current["status"] in ("SUCCESS", "FAILED"):
        print("status do domínio:", current["status"])
        break
```

Em PHP:

```php
<?php

use Meeng\Meeng;

$meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

$domain = $meeng->domains->create(['name' => 'envio.exemplo.com.br', 'region' => 'us-east-1']);
foreach ($domain['dnsRecords'] as $record) {
    echo $record['type'], ' ', $record['name'], ' ', $record['value'], PHP_EOL;
}

// Depois que o dono publicar os registros:
$meeng->domains->verify($domain['id']);
```

## Mapa das referências

| Quando você precisa de | Abra |
|---|---|
| Cada registro em detalhe, dividir a chave DKIM, preencher o painel do provedor de DNS, restringir uma chave a um domínio, remover um domínio | [`referencias/registros-dns.md`](referencias/registros-dns.md) |
| Status do domínio e de cada registro, por que não verifica, o que fazer em cada caso | [`referencias/verificacao-e-problemas.md`](referencias/verificacao-e-problemas.md) |
| Política DMARC, relatórios, o caminho de `none` a `reject` | [`referencias/dmarc.md`](referencias/dmarc.md) |

## Ao terminar

Entregue ao dono, em português: qual domínio ficou pronto (e com que status), quais registros ele publicou, o resultado do e-mail de teste, o que falta (DMARC mais firme, webhook `domain.verified`) e quem faz cada passo.
