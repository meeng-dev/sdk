---
name: meeng-contatos-e-campanhas
description: Ensina a montar listas de contatos e mandar campanhas de e-mail em massa pelo Meeng.dev, a plataforma brasileira de e-mail para desenvolvedores, com os SDKs oficiais (TypeScript, Go, Python, PHP e as demais linguagens) ou a API REST em https://meeng.dev/api/v1. Use quando o pedido for criar uma lista de contatos, importar contatos (um a um ou em massa), confirmar a inscrição (dupla confirmação), descadastrar, criar uma campanha ou newsletter, personalizar com variáveis, agendar, pausar ou retomar uma campanha, ler os números de entrega, abertura e clique, ou cumprir a LGPD no envio em massa. Cobre contatos, listas, campanhas, descadastro e métricas.
---

# Contatos e campanhas no Meeng.dev

Um e-mail transacional (um pedido, um código de acesso) vai de uma pessoa para a outra e se envia com `meeng-enviar-email`. Uma **campanha** é o mesmo e-mail para **uma lista inteira** de contatos: um boletim, um aviso de novidade, uma promoção. Esta skill cobre as listas, os contatos e as campanhas.

Fale com o dono em português simples. Quando um termo técnico for inevitável (dupla confirmação, descadastro, lote), explique em uma frase. Pergunte só o que falta: de onde vêm os contatos, se eles pediram para receber, qual é o conteúdo da campanha e quando enviar.

## Quando usar

Use quando o trabalho é guardar contatos no Meeng.dev e mandar e-mail para eles em massa. Para mensagem avulsa, `meeng-enviar-email`; para o remetente (domínio verificado), `meeng-dominios`; para saber o que aconteceu com cada e-mail (entregue, devolvido, clicado), `meeng-webhooks`.

Não use para comprar ou raspar listas de contatos: o Meeng.dev é só para quem recebeu o consentimento da pessoa. Para algo que não está aqui, a fonte da verdade é o contrato `https://meeng.dev/api/v1/doc`. Nunca invente campo ou rota.

## Os três objetos

- **Lista de contatos** (`contactBooks`): um grupo de pessoas com o mesmo assunto (clientes, boletim, lista de espera). Tem nome, variáveis de personalização e as regras da confirmação de inscrição.
- **Contato** (`contacts`): uma pessoa na lista. Tem e-mail, nome, sobrenome, propriedades livres e o estado de inscrição.
- **Campanha** (`campaigns`): um e-mail enviado a todos os contatos inscritos de uma lista, em lotes, com contadores de entrega, abertura e clique.

## Regras que valem sempre

1. **Só envie a quem pediu.** A LGPD exige uma base legal para cada envio, e para e-mail de divulgação a base é o consentimento da pessoa. Guarde de onde e quando cada contato aceitou (`properties`). Não importe lista comprada, raspada ou de outro negócio.
2. **A dupla confirmação vem ligada.** Uma lista nova confirma a inscrição: o contato novo fica **pendente** (`subscribed: false`, sem motivo de saída) e recebe um e-mail com o link de confirmação, que vale 7 dias. Só quem confirma entra nas campanhas. O e-mail sai de um domínio verificado da conta (ou do endereço `doubleOptInFrom` da lista, que também precisa ser de um domínio verificado). Só desligue (`doubleOptInEnabled: false`) para contatos que já confirmaram em outro lugar e que você consegue provar.
3. **Toda campanha leva o link de descadastro.** O texto `{{meeng_unsubscribe_url}}` no HTML vira, para cada contato, o link próprio dele. Sem ele, criar a campanha dá `400` (`Campaign must include an unsubscribe link before sending`). Ponha o link num lugar visível (um `<a>` no rodapé).
4. **Quem saiu não volta sozinho.** Um contato que se descadastrou, deu bounce duro ou reclamou **não é reinscrito** ao ser adicionado de novo por `POST` ou `PUT`: continua fora. Só um `PATCH` com `subscribed: true` o reinscreve, e isso só pode acontecer quando a própria pessoa pediu.
5. **Importação em massa é assíncrona.** `contacts.bulkCreate` aceita até 1000 contatos e responde na hora com a contagem de enfileirados; o processamento vem depois e **os erros por contato não voltam na resposta**. Confira o resultado listando a lista depois.
6. **Teste antes de mandar para todos.** Crie a campanha como rascunho (sem `sendNow` nem `scheduledAt`), confira no painel (`https://meeng.dev/painel/campanhas`), e envie primeiro para uma lista pequena só com o time do dono.
7. **Campanha sai em lotes** (500 contatos por lote por padrão; de 1 a 100000 em `batchSize`). Se algo der errado, **pause** (`campaigns.pause`) e investigue; retome depois.
8. O remetente (`from`) de uma campanha é validado como no envio simples: domínio da conta e verificado.
9. Não registre a lista de contatos em log aberto: são dados pessoais.

## Roteiro essencial

1. **Criar a lista** com `contactBooks.create` e as `variables` que o conteúdo vai usar (`empresa`, `plano`).
2. **Adicionar os contatos**: um a um (`contacts.create`) ou em lotes de até 1000 (`contacts.bulkCreate`). Detalhes em [`referencias/contatos.md`](referencias/contatos.md).
3. **Esperar a confirmação** dos contatos novos (dupla confirmação) e conferir quantos estão inscritos.
4. **Criar a campanha** como rascunho, com o link de descadastro ([`referencias/campanhas.md`](referencias/campanhas.md)).
5. **Enviar** agora (`sendNow`) ou **agendar** (`schedule` com a data em ISO 8601).
6. **Acompanhar** pelos contadores (`campaigns.get`), por webhook e pelas métricas de reputação (`analytics.reputationMetrics`).
7. **Respeitar os pedidos de saída**: o descadastro pelo link é automático; espelhe-o no seu sistema pelo evento `contact.updated` (`meeng-webhooks`).

Criar uma lista e importar contatos, em TypeScript:

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

const lista = await meeng.contactBooks.create({
  name: "Boletim de novidades",
  emoji: "📬",
  variables: ["empresa"], // vira {{empresa}} no conteúdo da campanha
});
if (lista.error) {
  throw new Error(`${lista.error.code}: ${lista.error.message}`);
}

const inscritos = [
  { email: "ana@example.com", firstName: "Ana", properties: { empresa: "Padaria Central", origem: "formulario-site-2026-12-01" } },
  { email: "bruno@example.com", firstName: "Bruno", properties: { empresa: "Oficina do Bruno", origem: "formulario-site-2026-12-01" } },
];

// Até 1000 por chamada. A resposta só conta quantos entraram na fila de processamento.
const importacao = await meeng.contacts.bulkCreate(lista.data.id, inscritos);
if (importacao.error) {
  throw new Error(`${importacao.error.code}: ${importacao.error.message}`);
}
console.log(importacao.data.message); // "Queued 2 contacts for processing"
```

Criar a campanha, conferir e enviar, em Go:

```go
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("") // lê MEENG_API_KEY e MEENG_BASE_URL
	ctx := context.Background()

	html := `<p>Olá, {{firstName,fallback=amigo}}! Chegou a novidade de dezembro.</p>
<p><a href="{{meeng_unsubscribe_url}}">Cancelar a inscrição</a></p>`
	previewText := "A novidade de dezembro chegou"

	// Sem sendNow nem scheduledAt: fica como rascunho, para conferir no painel.
	campaign, err := client.Campaigns.Create(ctx, meeng.CampaignsCreateParams{
		Name:          "Boletim de dezembro",
		From:          "Loja Exemplo <novidades@envio.exemplo.com.br>",
		Subject:       "A novidade de dezembro chegou",
		PreviewText:   &previewText,
		ContactBookID: "ID_DA_LISTA",
		HTML:          &html,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("rascunho criado:", campaign.ID, campaign.Status)

	// Depois de conferir no painel: agendar para segunda-feira às 9h (horário de Brasília).
	at := "2026-12-07T09:00:00-03:00"
	if _, err := client.Campaigns.Schedule(ctx, campaign.ID, meeng.CampaignsScheduleParams{ScheduledAt: &at}); err != nil {
		log.Fatal(err)
	}
}
```

Em Python e PHP:

```python
from meeng import Meeng

meeng = Meeng()  # lê MEENG_API_KEY e MEENG_BASE_URL

html = (
    "<p>Olá, {{firstName,fallback=amigo}}! Chegou a novidade de dezembro.</p>"
    '<p><a href="{{meeng_unsubscribe_url}}">Cancelar a inscrição</a></p>'
)

campanha, _ = meeng.campaigns.create(
    {
        "name": "Boletim de dezembro",
        "from": "novidades@envio.exemplo.com.br",
        "subject": "A novidade de dezembro chegou",
        "contactBookId": "ID_DA_LISTA",
        "html": html,
    }
)
if campanha is None:  # só acontece com raise_on_error=False
    raise RuntimeError("o Meeng.dev não respondeu")
print(campanha["id"], campanha["status"])  # DRAFT: confira no painel antes de agendar
```

```php
<?php

use Meeng\Meeng;

$meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

$campanha = $meeng->campaigns->create([
    'name' => 'Boletim de dezembro',
    'from' => 'novidades@envio.exemplo.com.br',
    'subject' => 'A novidade de dezembro chegou',
    'contactBookId' => 'ID_DA_LISTA',
    'html' => '<p>Olá, {{firstName,fallback=amigo}}!</p><p><a href="{{meeng_unsubscribe_url}}">Cancelar a inscrição</a></p>',
]);
echo $campanha['id'], ' ', $campanha['status'], PHP_EOL; // DRAFT
```

## Mapa das referências

| Quando você precisa de | Abra |
|---|---|
| Campos da lista e do contato, dupla confirmação, propriedades e variáveis, importação em lotes, descadastro, reinscrição | [`referencias/contatos.md`](referencias/contatos.md) |
| Campos da campanha, estados, personalização, agendar, pausar, retomar, contadores, erros | [`referencias/campanhas.md`](referencias/campanhas.md) |
| LGPD no envio em massa: consentimento, descadastro, retenção, o que registrar | [`referencias/lgpd-e-boas-praticas.md`](referencias/lgpd-e-boas-praticas.md) |

## Ao terminar

Entregue ao dono, em português: qual lista e quantos contatos foram importados (e quantos estão confirmados), o estado da campanha (rascunho, agendada ou enviada), o que foi testado (rascunho conferido, envio para o time, link de descadastro funcionando), os números de entrega e o que falta (conferir a taxa de devolução depois do primeiro envio).
