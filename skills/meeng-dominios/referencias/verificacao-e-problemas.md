# Verificação e problemas

## Como a verificação funciona

1. `PUT /v1/domains/{id}/verify` marca o domínio como "em verificação" (`isVerifying: true`) e responde `{"message": "Domain verification started"}`. Ele **não** espera o resultado.
2. Enquanto o domínio está em verificação, cada `GET /v1/domains/{id}` **confere o DNS na hora** e grava o resultado. Além disso, uma varredura confere os domínios pendentes a cada 5 minutos, sem você pedir.
3. Quando todos os registros obrigatórios estão certos, o domínio vai a `SUCCESS`, a verificação termina e o webhook `domain.verified` é enviado. Quando algo muda para pior ou melhor sem fechar, sai `domain.updated`.
4. Se o domínio tem mais de **72 horas** e ainda faltam registros, o status vai a `FAILED` e a verificação para. Depois de corrigir o DNS, peça `PUT /v1/domains/{id}/verify` outra vez.

Os campos da resposta que ajudam:

| Campo | O que diz |
|---|---|
| `status` | o estado geral do domínio |
| `dnsRecords[].status` | o estado de cada registro |
| `isVerifying` | `true` enquanto a verificação continua |
| `errorMessage` | por que falhou, quando falhou |
| `lastCheckedTime` | quando o DNS foi conferido pela última vez (só quando a leitura refez a conferência) |
| `dkimStatus`, `spfDetails` | o estado do DKIM e do SPF |
| `dmarcAdded` | `true` quando existe um DMARC |

## Os estados

| Estado | Quer dizer | O que fazer |
|---|---|---|
| `NOT_STARTED` | nada foi conferido ainda, ou o registro é opcional e não existe (DMARC) | peça a verificação |
| `PENDING` | o registro ainda não apareceu no DNS, ou ainda não chegou a quem consultou | espere de 5 minutos a algumas horas; confira com `dig` |
| `SUCCESS` | publicado e certo | nada: o domínio pode enviar |
| `FAILED` | publicado errado, ou ainda ausente depois de 72 horas | corrija o registro apontado e peça a verificação outra vez |
| `TEMPORARY_FAILURE` | o DNS não respondeu na consulta | espere e consulte de novo; não mexa nos registros por isso |

O envio só funciona com o domínio em `SUCCESS`. Em qualquer outro estado, `POST /v1/emails` responde `400` com `Domain: x is not verified`.

## Por que não verifica

Olhe `dnsRecords` e ache o primeiro registro com `status` diferente de `SUCCESS`; o problema quase sempre está nele.

| Sintoma | Causa provável | Correção |
|---|---|---|
| MX continua `PENDING` | nome errado, ou o painel acrescentou o domínio duas vezes | veja o nome com `dig MX mail.seudominio +short`; corrija o campo Nome |
| MX `FAILED` | o MX aponta para outro servidor, ou a prioridade está errada | o valor tem de ser o da API e a prioridade `10`; apague MX que sobraram no mesmo nome |
| SPF `FAILED` | dois registros SPF no mesmo nome, ou falta o `include:` | deixe um só registro `v=spf1`, com o `include:` da API, antes do `~all` |
| SPF `FAILED` e o valor parece certo | aspas coladas dentro do valor, ou espaço no fim | recrie o registro e cole só o texto |
| DKIM `PENDING` ou `FAILED` | a chave foi cortada pelo painel, ou tem quebra de linha | recrie o TXT colando a chave inteira, em uma linha; se o painel pede, divida em partes de até 255 caracteres |
| DKIM em `SUCCESS` e SPF em `PENDING` | cada registro tem o seu tempo para chegar | espere; o DNS não propaga tudo ao mesmo tempo |
| tudo `PENDING` por horas | o domínio usa servidores de DNS de um provedor, mas os registros foram criados em outro | veja onde o domínio **de fato** aponta (`dig NS seudominio`) e publique lá |
| `TEMPORARY_FAILURE` | o servidor de DNS do domínio estava fora do ar | consulte de novo mais tarde |
| `FAILED` com 72 horas ou mais | os registros nunca apareceram | publique e peça `PUT .../verify` |

Se o domínio vai a `SUCCESS` e mesmo assim os e-mails caem no spam, o problema é outro: veja [`dmarc.md`](dmarc.md), a reputação (`analytics.reputationMetrics`, taxas de devolução e de reclamação) e o conteúdo do e-mail.

## Esperar sem consultar a cada segundo

Não dispare um `GET` por segundo: cada um refaz a consulta de DNS, e o DNS não muda tão depressa. Consulte a cada 1 a 5 minutos, ou escute o webhook `domain.verified` (`meeng-webhooks`) e deixe o Meeng.dev avisar.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	ctx := context.Background()

	domains, err := client.Domains.List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, domain := range domains {
		fmt.Printf("%s (id %s): %s\n", domain.Name, strconv.FormatInt(domain.ID, 10), domain.Status)
		for _, record := range domain.DNSRecords {
			if record.Status != "SUCCESS" && record.Name != "_dmarc" { // o DMARC é só recomendado
				fmt.Printf("  falta: %s %s (%s)\n", record.Type, record.Name, record.Status)
			}
		}
		if domain.ErrorMessage != nil {
			fmt.Println("  motivo:", *domain.ErrorMessage)
		}
	}
}
```

## Erros da API de domínios

| Chamada | Status e mensagem | Quer dizer |
|---|---|---|
| `POST /v1/domains` | `500` `something unexpected happened` | o nome não é um domínio válido, a região não existe, ou o nome já está cadastrado (em qualquer conta) |
| `POST /v1/domains` | `403` `DOMAIN` | o plano chegou ao limite de domínios |
| `POST /v1/domains` | `403` `API key is restricted to a single domain and cannot create domains` | a chave é restrita a um domínio |
| `GET /v1/domains/{id}` | erro com `Domain not found` | o id não existe ou é de outra organização |
| `PUT /v1/domains/{id}/verify` | `404` com `{"error": "Domain not found"}` | idem; este erro vem sem `code` |
| `DELETE /v1/domains/{id}` | `403` `API key doesn't have access to this domain` | a chave é restrita a outro domínio |

Um domínio de outra organização é tratado como se não existisse: nunca há `403` revelando que ele existe.
