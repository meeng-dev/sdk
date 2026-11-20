# Os eventos

São 20 eventos de negócio (13 de e-mail, 3 de contato, 4 de domínio) mais o `webhook.test`, que o botão "Testar" do painel manda. Quem assina um endpoint escolhe quais receber; **sem escolher nenhum, recebe todos**.

## O formato comum

Todo aviso é `{ id, type, version, createdAt, teamId, attempt, data }`:

| Campo | O que é |
|---|---|
| `id` | o id da entrega; o mesmo em todas as tentativas. Use para ignorar repetidos |
| `type` | o evento, como `email.delivered` |
| `version` | a versão do formato do `data` (`2026-01-18`) |
| `createdAt` | quando o aviso foi criado (UTC, com milissegundos) |
| `teamId` | o número da organização dona do evento |
| `attempt` | qual tentativa é esta, de 1 a 6 |
| `data` | o objeto do evento; varia por família, abaixo |

## E-mail

`email.queued`, `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.bounced`, `email.rejected`, `email.rendering_failure`, `email.complained`, `email.failed`, `email.cancelled`, `email.suppressed`, `email.opened` e `email.clicked`.

O `data` de todos tem a base:

| Campo | O que é |
|---|---|
| `id` | o `emailId` devolvido no envio |
| `status` | `QUEUED`, `SENT`, `DELIVERY_DELAYED`, `DELIVERED`, `BOUNCED`, `REJECTED`, `RENDERING_FAILURE`, `COMPLAINED`, `FAILED`, `CANCELLED`, `SUPPRESSED`, `OPENED`, `CLICKED` ou `SCHEDULED` |
| `from`, `to` | o remetente e a lista de destinatários |
| `occurredAt` | quando o fato aconteceu (UTC) |
| `subject` | o assunto |
| `domainId` | o id do domínio do `from`, ou `null` |
| `campaignId`, `contactId` | só quando o e-mail veio de uma campanha |
| `templateId`, `metadata` | opcionais; `metadata` traz detalhes do evento |

E o que cada evento acrescenta:

| Evento | O que aconteceu | Campo extra no `data` |
|---|---|---|
| `email.queued` | aceito e na fila de envio | nenhum |
| `email.sent` | saiu do Meeng.dev rumo ao servidor do destinatário | nenhum |
| `email.delivery_delayed` | o servidor do destinatário adiou; o Meeng.dev tenta de novo | `metadata.delayedRecipients` |
| `email.delivered` | o servidor do destinatário **aceitou** a mensagem (resposta 250) | nenhum |
| `email.bounced` | o e-mail foi devolvido | `bounce`: `type` (`Transient`, `Permanent`, `Undetermined`), `subType` (`General`, `NoEmail`, `Suppressed`, `OnAccountSuppressionList`, `MailboxFull`, `MessageTooLarge`, `ContentRejected`, `AttachmentRejected`) e `message` (a resposta do servidor) |
| `email.rejected` | recusado antes de sair | `metadata.reason` |
| `email.rendering_failure` | o template falhou ao montar o corpo | nenhum |
| `email.complained` | o destinatário marcou como spam | `metadata.feedbackType` |
| `email.failed` | falha do lado do envio (limite do plano, por exemplo) | `failed.reason` |
| `email.cancelled` | cancelado antes de sair | nenhum |
| `email.suppressed` | não enviado: o destinatário está na lista de supressão | `suppression`: `type` (`Bounce`, `Complaint`, `Manual`), `reason` e `source` |
| `email.opened` | a imagem de rastreio foi aberta (só com o rastreio de abertura ligado no domínio) | `open`: `timestamp`, `userAgent`, `ip` |
| `email.clicked` | o destinatário clicou num link (só com o rastreio de cliques ligado) | `click`: `timestamp`, `url`, `userAgent`, `ip` |

Abertura e clique não são prova de leitura: programas de segurança e a proteção de privacidade de alguns leitores abrem e clicam em tudo. Não use `email.opened` para decisão que importa.

## Contato

`contact.created`, `contact.updated` e `contact.deleted`. O `data`:

| Campo | O que é |
|---|---|
| `id`, `email` | o contato |
| `contactBookId` | a lista de contatos |
| `subscribed` | se está inscrito |
| `firstName`, `lastName` | nome, ou `null` |
| `properties` | as propriedades livres |
| `createdAt`, `updatedAt` | datas |

## Domínio

`domain.created`, `domain.verified`, `domain.updated` e `domain.deleted`. O `data`: `id` (número), `name`, `status`, `region`, `clickTracking`, `openTracking`, `subdomain`, `dkimStatus`, `spfDetails`, `dmarcAdded`, `createdAt`, `updatedAt`. `domain.verified` é o aviso de que o domínio chegou a `SUCCESS` e já pode enviar.

## Teste

`webhook.test`, com o `data` `{ test: true, webhookId, sentAt }`. É o que o botão "Testar" manda; use para provar que o endereço, a assinatura e o segredo estão certos antes de assinar eventos de verdade.

## O que fazer com cada evento

| Evento | Ação típica |
|---|---|
| `email.delivered` | marcar o e-mail como entregue no seu sistema |
| `email.bounced` com `bounce.type` `Permanent` | endereço inválido: pare de enviar para ele e avise o responsável pelo cadastro. O Meeng.dev já o põe na lista de supressão |
| `email.bounced` com `Transient` (caixa cheia, servidor fora do ar) | não é definitivo; não apague o cadastro por isso |
| `email.complained` | o destinatário não quer mais: descadastre-o de tudo que é marketing, agora |
| `email.failed` | alertar quem cuida do sistema; se for limite do plano, ver `meeng-enviar-email` |
| `email.suppressed` | o endereço está bloqueado; não insista, avise o cadastro |
| `email.delivery_delayed` | só registrar; chega um `delivered` ou `bounced` depois |
| `domain.verified` | liberar o envio daquele domínio no seu sistema |
| `contact.updated` com `subscribed: false` | espelhar o descadastro no seu sistema |

Para o `to` de um evento, lembre que é uma lista; um e-mail com vários destinatários gera um aviso com todos eles (e `bounce` e `complaint` valem para o destinatário que de fato falhou).

## Em TypeScript: o tipo do evento muda com o `type`

O SDK tipa `evento.data` conforme o `type`. Dentro de cada `case`, os campos extras existem:

```ts
import type { WebhookEvent } from "@meeng-dev/sdk";

export function descrever(evento: WebhookEvent): string {
  switch (evento.type) {
    case "email.bounced":
      return `devolvido (${evento.data.bounce.type}): ${evento.data.to.join(", ")}`;
    case "email.failed":
      return `falhou: ${evento.data.failed.reason}`;
    case "email.suppressed":
      return `bloqueado (${evento.data.suppression.type}): ${evento.data.suppression.reason}`;
    case "email.clicked":
      return `clique em ${evento.data.click.url}`;
    case "domain.verified":
      return `domínio ${evento.data.name} verificado`;
    case "webhook.test":
      return "teste recebido";
    default:
      return evento.type;
  }
}
```
