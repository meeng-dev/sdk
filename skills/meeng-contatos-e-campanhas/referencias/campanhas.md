# Campanhas

Uma campanha manda o mesmo conteúdo, com os dados de cada pessoa, a todos os contatos **inscritos** de uma lista, em lotes. É criada por `POST /v1/campaigns` e pode ser enviada na hora, agendada, pausada e retomada.

## Campos para criar

| Campo | Tipo | Regra |
|---|---|---|
| `name` | texto | **obrigatório**: o nome interno, só você vê |
| `from` | texto | **obrigatório**: o remetente, de um domínio verificado da conta |
| `subject` | texto | **obrigatório**: o assunto |
| `contactBookId` | texto | **obrigatório**: a lista que recebe |
| `html` | texto | o corpo em HTML. `html` **ou** `content` é obrigatório |
| `content` | texto | o corpo no formato do editor do painel (JSON). Pela API, prefira `html` |
| `previewText` | texto | a prévia que aparece ao lado do assunto na caixa de entrada |
| `replyTo`, `cc`, `bcc` | texto ou lista | endereços extras |
| `sendNow` | booleano | `true` envia assim que criar |
| `scheduledAt` | texto | agenda; use ISO 8601 com fuso (`2026-12-07T09:00:00-03:00`) |
| `batchSize` | inteiro | contatos por lote, de 1 a 100000; padrão 500 |

Sem `sendNow` nem `scheduledAt`, a campanha fica como **rascunho** (`DRAFT`). A resposta é a campanha criada.

### Regras de validação

- `html` e `content` ausentes: `400` com `Either content or html must be provided.`
- Lista inexistente: **`400`** com `Contact book not found` (não é 404, neste caso).
- Remetente: as mesmas regras do envio simples (`From email is invalid`, domínio errado ou não verificado, chave restrita a outro domínio).
- **Sem o link de descadastro**: `400` com `Campaign must include an unsubscribe link before sending`. O marcador é `{{meeng_unsubscribe_url}}`, no `html` ou no `content`.
- Se `sendNow` ou `scheduledAt` falhar (por exemplo, nenhum contato inscrito), a campanha **já foi criada** e fica como rascunho, e a resposta é o erro. Releia a lista de campanhas antes de tentar de novo, para não criar duas.

## Personalização

O conteúdo aceita marcadores entre chaves duplas, sem diferenciar maiúsculas e minúsculas, com texto padrão opcional:

| Marcador | Vira |
|---|---|
| `{{firstName}}`, `{{lastName}}`, `{{email}}` | o nome, o sobrenome e o e-mail do contato |
| `{{empresa}}` e as demais variáveis da lista | o valor de `properties` do contato |
| `{{firstName,fallback=amigo}}` | o nome, ou `amigo` quando o contato não tem nome |
| `{{meeng_unsubscribe_url}}` | o link de descadastro próprio de cada contato |

Um marcador sem valor e sem texto padrão fica como está no e-mail, e isso é feio: use sempre `fallback`. Em campanha, as variáveis vêm do **contato** e da lista, não de um campo `variables` no corpo.

## Estados

| Estado | Quer dizer |
|---|---|
| `DRAFT` | rascunho; nada foi enviado |
| `SCHEDULED` | agendada para o horário de `scheduledAt` |
| `RUNNING` | enviando, lote por lote |
| `PAUSED` | pausada; retome com `resume` |
| `SENT` | todos os contatos foram processados |

## Enviar, agendar, pausar, retomar

| Rota | O que faz |
|---|---|
| `POST /v1/campaigns/{id}/schedule` | agenda para `scheduledAt`, ou envia agora quando o corpo vem sem horário. O corpo precisa ser um objeto, mesmo vazio (`{}`). Aceita `batchSize` |
| `POST /v1/campaigns/{id}/pause` | pausa uma campanha agendada ou em andamento |
| `POST /v1/campaigns/{id}/resume` | retoma uma pausada |
| `DELETE /v1/campaigns/{id}` | apaga a campanha |
| `GET /v1/campaigns/{id}` | lê a campanha com os contadores |
| `GET /v1/campaigns` | lista, 30 por página, da mais nova à mais velha; filtros `status`, `search` e `page` |

`scheduledAt` aceita uma data ISO 8601 (com ou sem fuso; sem fuso, a hora é lida no fuso do servidor) e também frases curtas em inglês como `tomorrow 9am` ou `in 2 hours`. **Prefira sempre ISO com fuso**: é a única forma sem ambiguidade de horário.

A resposta de `GET /v1/campaigns` é `{ "campaigns": [...], "totalPage": N }`; o `totalPage` é o número de páginas. O SDK entrega a lista das campanhas da página pedida e o total de páginas.

### Contadores

`GET /v1/campaigns/{id}` traz, entre outros: `total` (contatos da campanha), `sent`, `delivered`, `opened`, `clicked`, `unsubscribed`, `bounced`, `hardBounced` e `complained`. Abertura e clique só existem com o rastreio ligado no domínio.

O que olhar depois do primeiro envio (regras práticas do mercado, não limites da plataforma):

- `hardBounced / total` acima de **2%** é sinal de lista suja: pause, limpe a lista (os endereços inválidos já estão na supressão) e investigue de onde vieram.
- `complained / delivered` acima de **0,1%** é sinal de público que não pediu o e-mail: pause e revise a origem dos contatos e o conteúdo.
- `analytics.reputationMetrics` mostra as taxas de devolução e de reclamação de toda a conta.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

const { data: campanha, error } = await meeng.campaigns.get("ID_DA_CAMPANHA");
if (error) {
  throw new Error(`${error.code}: ${error.message}`);
}

const taxaDeBounce = campanha.total > 0 ? campanha.hardBounced / campanha.total : 0;
console.log(`${campanha.status}: ${campanha.sent} de ${campanha.total} enviados, ${campanha.delivered} entregues`);

if (campanha.status === "RUNNING" && taxaDeBounce > 0.02) {
  await meeng.campaigns.pause(campanha.id);
  console.log("pausada: mais de 2% de devoluções duras; revise a lista antes de retomar");
}
```

Listar as campanhas e agendar, em Go:

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

	page, err := client.Campaigns.List(ctx, &meeng.CampaignsListQuery{Status: "DRAFT"})
	if err != nil {
		log.Fatal(err)
	}
	for _, campaign := range page.Campaigns {
		fmt.Println(campaign.ID, campaign.Name, campaign.Status)
	}

	// Sem horário: envia agora. Com horário (ISO 8601 com fuso): agenda.
	if _, err := client.Campaigns.Schedule(ctx, "ID_DA_CAMPANHA", meeng.CampaignsScheduleParams{}); err != nil {
		log.Fatal(err)
	}
}
```

## HTML de e-mail que chega

- Largura de até 600 pixels, uma coluna, tabelas para o layout: leitores de e-mail ignoram boa parte do CSS moderno.
- Todo CSS em linha (`style="..."`); sem JavaScript, sem formulário, sem vídeo.
- Imagens com `alt`, hospedadas em `https` por você. Não dependa de imagem para entender a mensagem.
- Um **botão** de ação claro e um link de descadastro **visível** no rodapé.
- Texto equilibrado: e-mail só de imagem, com muitas maiúsculas ou pontos de exclamação, cai no spam.
- Identifique quem envia no rodapé (nome da empresa e CNPJ, ou nome e endereço), como a boa prática de transparência pede.

## Erros que você vai ver

| Status e mensagem | Quer dizer |
|---|---|
| `400` `Campaign must include an unsubscribe link before sending` | falta `{{meeng_unsubscribe_url}}` |
| `400` `Contact book not found` | `contactBookId` errado, ou de outra organização |
| `400` `Invalid content JSON` | `content` não é um JSON válido; use `html` |
| `400` `Invalid date format: ...` | `scheduledAt` ilegível; use ISO 8601 |
| `404` `Campaign not found` | a campanha não existe, ou é de outra organização |
| `403` `API key does not have access to domain: x` | a chave é restrita a outro domínio |
