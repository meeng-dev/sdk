# Listas de contatos e contatos

## A lista de contatos (`contactBooks`)

| Campo | Tipo | O que é |
|---|---|---|
| `id` | texto | o id da lista |
| `name` | texto | o nome (obrigatório ao criar) |
| `emoji` | texto | o ícone mostrado no painel |
| `variables` | lista de textos | os nomes de variável que o conteúdo pode usar (`{{empresa}}`) |
| `properties` | objeto texto para texto | propriedades livres da lista |
| `doubleOptInEnabled` | booleano | **nasce `true`**: contato novo precisa confirmar a inscrição |
| `doubleOptInFrom` | texto ou `null` | endereço que envia o e-mail de confirmação; precisa ser de um domínio verificado |
| `doubleOptInSubject` | texto | o assunto da confirmação (padrão `Confirme a sua inscrição`) |
| `doubleOptInContent` | texto | o conteúdo da confirmação; precisa conter o marcador `{{doubleOptInUrl}}` |
| `_count` | objeto | `{ contacts: N }`, só ao ler uma lista ou listar |

Rotas: `GET /v1/contactBooks`, `POST /v1/contactBooks`, `GET`, `PATCH` e `DELETE /v1/contactBooks/{id}`. Apagar uma lista apaga os contatos dela; as campanhas que a usavam ficam sem lista.

**Variáveis**: letras, números e sublinhado (`empresa`, `plano_atual`), sem repetir (sem diferenciar maiúsculas). Os nomes `email`, `firstName` e `lastName` já existem e são reservados. Um nome inválido dá `400` (`Variable "x" contains invalid characters...` ou `is reserved`).

## O contato (`contacts`)

| Campo | O que é |
|---|---|
| `id` | o id do contato |
| `email` | o endereço; a API não confere o formato. Único por lista |
| `firstName`, `lastName` | nome e sobrenome, ou `null` |
| `subscribed` | `true` quando está inscrito e recebe campanhas |
| `unsubscribeReason` | por que saiu: `UNSUBSCRIBED`, `BOUNCED` ou `COMPLAINED`; `null` se não saiu |
| `properties` | propriedades livres, texto para texto |
| `contactBookId`, `createdAt`, `updatedAt` | a lista e as datas |

**Pendente de confirmação** é o contato com `subscribed: false` e `unsubscribeReason: null`: ainda não clicou no link da confirmação. Ele não recebe campanha.

### As rotas

| Rota | O que faz |
|---|---|
| `POST /v1/contactBooks/{lista}/contacts` | cria o contato, ou atualiza o que tem o mesmo e-mail. Resposta: `{ "contactId": "..." }` |
| `PUT /v1/contactBooks/{lista}/contacts/{id}` | igual ao `POST`: o contato é achado pelo **e-mail**, e o `{id}` do caminho é ignorado |
| `PATCH /v1/contactBooks/{lista}/contacts/{id}` | muda só os campos enviados; **é o único que reinscreve** (`subscribed: true`) |
| `GET /v1/contactBooks/{lista}/contacts` | lista; filtros `emails` e `ids` (separados por vírgula), `page`, `limit` (5000 por padrão). Devolve uma lista simples, sem total |
| `GET` e `DELETE /v1/contactBooks/{lista}/contacts/{id}` | lê e remove um contato |
| `POST /v1/contactBooks/{lista}/contacts/bulk` | adiciona até **1000** contatos de uma vez, de forma assíncrona |
| `DELETE /v1/contactBooks/{lista}/contacts/bulk` | remove de 1 a 1000 contatos por `contactIds` |

### Propriedades e variáveis

`properties` guarda qualquer par texto para texto. As chaves que coincidem com uma variável da lista (sem diferenciar maiúsculas) são guardadas com a grafia da variável e **entram nas campanhas** (`{{empresa}}`). Chaves fora das variáveis da lista são guardadas, mas só servem de dado seu: não aparecem no conteúdo. Ao atualizar, as propriedades novas são **mescladas** por cima das existentes; omitir uma chave não a apaga.

Guarde em `properties` a prova do consentimento: de onde o contato veio e quando aceitou (`origem: formulario-site`, `aceite_em: 2026-12-01T10:30:00-03:00`). Em caso de dúvida do titular, é isso que você mostra.

## Dupla confirmação (padrão)

1. Você adiciona o contato (`POST` ou `bulk`).
2. Ele nasce **pendente** e o Meeng.dev manda o e-mail de confirmação, de um domínio verificado da conta, com o link que vale **7 dias**. O contato pendente que é adicionado de novo recebe a confirmação outra vez.
3. Quando clica no link, o contato vira `subscribed: true` e passa a receber as campanhas. O evento `contact.updated` é enviado.
4. Se a conta **não tem domínio verificado**, o contato é criado mas o e-mail de confirmação não sai (a falha fica no registro do sistema): verifique um domínio primeiro (`meeng-dominios`).

Desligar a confirmação (`doubleOptInEnabled: false` na lista) faz o contato nascer inscrito. Só faça isso se o consentimento já foi confirmado e você consegue provar (por exemplo, o clique de um link de confirmação no seu próprio sistema).

`subscribed: false` mandado de propósito no `POST` grava o contato como descadastrado (`UNSUBSCRIBED`) e não envia confirmação.

## Descadastro

- **Pelo link do e-mail.** O marcador `{{meeng_unsubscribe_url}}` de cada campanha vira um link próprio de cada contato, que leva a uma página pública do Meeng.dev (`/cancelar-inscricao`), sem login, e descadastra o contato. O contato vira `subscribed: false` com motivo `UNSUBSCRIBED`, o webhook `contact.updated` sai, e ele não recebe mais campanhas daquela lista. 
- **Pelo botão do provedor (um clique).** Todo e-mail de campanha sai com `List-Unsubscribe: <https://meeng.dev/api/unsubscribe-oneclick?id=...&hash=...>` e `List-Unsubscribe-Post: List-Unsubscribe=One-Click` (RFC 8058), que o Gmail e o Yahoo mostram como "Cancelar inscrição". O provedor faz um `POST` nesse endereço, sem login nem confirmação, e o contato daquela lista sai na hora (`subscribed: false`, `UNSUBSCRIBED`, o webhook `contact.updated` e o registro de atividades). O link é assinado: ele só descadastra o contato de quem recebeu o e-mail, e repetir o clique não muda nada.
- **Por devolução e reclamação.** Um bounce duro marca o contato como `BOUNCED`; uma reclamação de spam, `COMPLAINED`. Em ambos o contato sai da lista de envio e o endereço entra na lista de supressão da conta.
- **Pela API.** `PATCH` com `subscribed: false` (motivo `UNSUBSCRIBED`), por exemplo quando a pessoa pede a saída por outro canal.

**Reinscrever** (`PATCH` com `subscribed: true`) só quando a própria pessoa pediu para voltar. Não use para "limpar" a lista de quem saiu: reinscrever quem se descadastrou é a forma mais rápida de receber reclamações e perder a reputação de envio.

## Importar uma lista grande

O `bulk` aceita 1000 por chamada e é assíncrono: os contatos entram numa fila, são processados um a um pelas mesmas regras do `POST`, e **erros por contato não voltam na resposta** (a resposta é só `{ "message": "Queued N contacts for processing", "count": N }`). Para um CSV grande:

1. Valide antes: endereço com `@` e domínio, sem duplicados, sem linhas em branco. A API não valida o formato.
2. Divida em pedaços de 1000 e envie em sequência, respeitando o limite de requisições (2 por segundo por padrão).
3. Confira o total no fim (`contactBooks.get`, `_count.contacts`) contra o que você enviou.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

type Contato = { email: string; firstName?: string; properties?: Record<string, string> };

async function importar(listaId: string, contatos: Contato[]): Promise<number> {
  let enfileirados = 0;
  for (let i = 0; i < contatos.length; i += 1000) {
    const pedaco = contatos.slice(i, i + 1000);
    const { data, error } = await meeng.contacts.bulkCreate(listaId, pedaco);
    if (error) {
      throw new Error(`pedaço ${i / 1000 + 1}: ${error.code} ${error.message}`);
    }
    enfileirados += data.count;
    await new Promise((resolve) => setTimeout(resolve, 600)); // menos de 2 requisições por segundo
  }
  return enfileirados;
}

const total = await importar("ID_DA_LISTA", [
  { email: "ana@example.com", firstName: "Ana", properties: { origem: "formulario-site" } },
  { email: "bruno@example.com", firstName: "Bruno", properties: { origem: "formulario-site" } },
]);
console.log(`${total} contatos na fila de processamento`);
```

Ler os contatos de uma lista, em Go (`List` devolve uma fatia simples e o filtro por e-mail aceita vários separados por vírgula):

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

	contacts, err := client.Contacts.List(ctx, "ID_DA_LISTA", &meeng.ContactsListQuery{Emails: "ana@example.com,bruno@example.com"})
	if err != nil {
		log.Fatal(err)
	}
	for _, contact := range contacts {
		state := "pendente de confirmação"
		switch {
		case contact.Subscribed:
			state = "inscrito"
		case contact.UnsubscribeReason != nil:
			state = "fora (" + *contact.UnsubscribeReason + ")"
		}
		fmt.Println(contact.Email, state)
	}
}
```

Reinscrever, só a pedido da pessoa, em TypeScript:

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

// A própria pessoa pediu para voltar, por escrito: guarde esse registro no seu sistema.
const { error } = await meeng.contacts.update("ID_DA_LISTA", "ID_DO_CONTATO", {
  subscribed: true,
  properties: { reinscrito_em: "2026-12-01", reinscrito_por: "pedido-por-e-mail" },
});
if (error) {
  throw new Error(`${error.code}: ${error.message}`);
}
```

## Erros que você vai ver

| Status e mensagem | Quer dizer |
|---|---|
| `404` `Contact book not found for this team` | a lista não existe, ou é de outra organização |
| `404` `Contact not found` | o contato não está nessa lista |
| `400` `Cannot add more than 1000 contacts in a single bulk request` | o lote passou de 1000 |
| `400` `Cannot delete more than 1000 contacts in a single request` | idem para a remoção |
| `400` `Double opt-in email content must include the {{doubleOptInUrl}} placeholder` | o conteúdo da confirmação perdeu o marcador do link |
| `400` `Variable "x" is reserved` | `email`, `firstName` e `lastName` já existem |
| `403` `CONTACT_BOOK` | o plano chegou ao limite de listas |
