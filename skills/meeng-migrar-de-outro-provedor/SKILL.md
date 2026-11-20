---
name: meeng-migrar-de-outro-provedor
description: Ensina a trocar o envio de e-mail de um sistema que usa outro provedor para o Meeng.dev, a plataforma brasileira de e-mail para desenvolvedores, mantendo o formato do payload de envio, sem derrubar entregas e sem perder reputação. Use quando o pedido for migrar o envio de e-mail para o Meeng.dev, trocar a chamada de envio do provedor atual, mapear campos (destinatário, assunto, corpo, anexos, agendamento, tags), mover domínio e DNS, migrar webhooks e a assinatura, importar contatos e supressões, esquentar o volume aos poucos, ou planejar a virada com plano de volta. Cobre inventário, adaptador de envio, domínio, webhooks, contatos, virada gradual e reversão.
---

# Migrar o envio de e-mail para o Meeng.dev

Trocar de provedor de e-mail dá medo por um bom motivo: se o envio falha, o cliente não recebe o código de acesso, o recibo, o aviso. Esta skill leva a troca em passos pequenos e reversíveis: o sistema continua enviando enquanto o novo provedor é preparado, e a virada é gradual, com volta fácil.

Fale com o dono em português simples. Explique em uma frase termos como "esquentar" (aumentar o volume aos poucos, para os provedores de caixa de entrada confiarem no remetente novo) e "adaptador" (um pedaço de código que traduz a chamada antiga para a nova). Pergunte só o que falta: qual provedor o sistema usa hoje, em que linguagens ele envia, de quais domínios, e quantos e-mails por dia.

## Quando usar

Use quando o sistema já envia e-mail por outro provedor (por API, por SDK ou por SMTP) e o dono quer passar para o Meeng.dev. Para o envio em si, depois da troca, veja `meeng-enviar-email`; domínio, `meeng-dominios`; webhooks, `meeng-webhooks`; SMTP, `meeng-smtp`.

Não use para migrar caixas de entrada (e-mail recebido de pessoas): isto é só o envio feito por sistemas. Nunca cite nem compare marcas ao dono ou nos arquivos do projeto: o assunto é o formato de dados, não o concorrente.

## Regras que valem sempre

1. **Nada de virada de uma vez.** O envio antigo continua funcionando até o fim. A passagem é por **porcentagem** de e-mails, com volta imediata por uma chave de configuração (feature flag), não por um novo deploy.
2. **Mantenha o remetente.** Use o mesmo endereço `From` que os clientes já veem, para que ninguém estranhe. Cadastre o mesmo domínio no Meeng.dev e publique os registros DNS **ao lado** dos antigos: o DKIM do Meeng.dev usa um nome próprio (`meeng._domainkey`), então os dois provedores convivem.
3. **Esquente o volume.** Um domínio ou um servidor que de repente manda tudo por um caminho novo parece suspeito. Comece em 1% a 5% dos e-mails e dobre a cada dia ou dois, olhando as taxas de devolução e de reclamação.
4. **Código e dados em ordem de segurança.** Troque a chave (nova, só para este sistema) e o segredo do webhook em variáveis de ambiente. Nunca grave chave nem segredo no repositório; apague as chaves do provedor antigo só depois de a virada ter estabilizado.
5. **Mantenha a idempotência.** Se o sistema já manda uma chave de idempotência, continue mandando-a para o Meeng.dev (cabeçalho `Idempotency-Key`, de 1 a 256 caracteres, vale 24 horas).
6. **Webhooks mudam de formato e de assinatura.** O corpo, os nomes de evento e a conferência da assinatura são do Meeng.dev; um receptor feito para o outro provedor não confere a assinatura do Meeng.dev. Reescreva o receptor com o SDK (`meeng-webhooks`) e rode os dois em paralelo na virada.
7. **A lista de quem não deve receber vai junto.** Exporte as supressões (bounces duros, reclamações, descadastros) do provedor antigo e carregue-as antes do primeiro envio. Sem isso, o Meeng.dev reenvia para quem já recusou, e a reputação cai logo.
8. **Não importe contatos sem a prova do aceite.** Veja `meeng-contatos-e-campanhas`.
9. O agente não troca o DNS do dono: mostre os registros e peça que ele (ou quem cuida do DNS) os publique. Nenhuma mudança de produção sem a aprovação do dono.

## Roteiro essencial

1. **Inventário**: onde o sistema envia e-mail, com que volume, que formatos (transacional, lote, anexo, agendado, modelos, SMTP) e que webhooks recebe ([`referencias/plano-de-migracao.md`](referencias/plano-de-migracao.md), fase 1).
2. **Domínio e DNS** no Meeng.dev, convivendo com o antigo (`meeng-dominios`).
3. **Adaptador de envio**: uma função que recebe o payload no formato antigo e chama o Meeng.dev ([`referencias/mapeamento-de-campos.md`](referencias/mapeamento-de-campos.md)).
4. **Supressões e contatos** carregados.
5. **Webhooks** novos, em paralelo com os antigos.
6. **Virada gradual** com a chave de configuração: 1%, 5%, 25%, 50%, 100% ([`referencias/plano-de-migracao.md`](referencias/plano-de-migracao.md), fase 5).
7. **Desligar o antigo** só depois de uma a duas semanas estável.

O adaptador resolve a maior parte do trabalho: o resto do sistema continua chamando a mesma função de envio, com o mesmo payload, e só o interior troca de destino. Em TypeScript (o `LegacyPayload` é o formato que o sistema usa hoje, com os campos em `snake_case`):

```ts
import { Meeng } from "@meeng-dev/sdk";

// O formato que o sistema já usa para pedir um envio (ajuste aos campos do seu).
export interface LegacyPayload {
  from: string;
  to: string | string[];
  subject?: string;
  html?: string;
  text?: string;
  cc?: string | string[];
  bcc?: string | string[];
  reply_to?: string | string[];
  scheduled_at?: string;
  headers?: Record<string, string>;
  attachments?: { filename: string; content: string }[]; // conteúdo em base64
  tags?: { name: string; value: string }[];
}

const meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

/** Traduz o payload antigo para o do Meeng.dev, e devolve também o que não tem lugar lá. */
export function paraMeeng(antigo: LegacyPayload): { corpo: Parameters<typeof meeng.emails.send>[0]; perdidos: string[] } {
  const perdidos: string[] = [];
  if (antigo.tags?.length) {
    perdidos.push("tags"); // não há campo de tags: guarde-as no seu banco junto do emailId
  }
  return {
    corpo: {
      from: antigo.from,
      to: antigo.to,
      subject: antigo.subject,
      html: antigo.html,
      text: antigo.text,
      cc: antigo.cc,
      bcc: antigo.bcc,
      replyTo: antigo.reply_to,
      scheduledAt: antigo.scheduled_at,
      headers: antigo.headers,
      attachments: antigo.attachments,
    },
    perdidos,
  };
}

/** A chave de configuração: a fração de e-mails que vai para o Meeng.dev (0 a 100). */
function porcentagemMeeng(): number {
  return Number(process.env.MEENG_PERCENTUAL ?? "0");
}

/** Decide pelo id estável do envio, para que o mesmo e-mail vá sempre para o mesmo provedor. */
function vaiParaMeeng(chave: string): boolean {
  let soma = 0;
  for (const letra of chave) {
    soma = (soma * 31 + letra.charCodeAt(0)) % 100;
  }
  return soma < porcentagemMeeng();
}

export async function enviarEmail(
  antigo: LegacyPayload,
  chave: string, // o id estável do envio (pedido-1042-confirmacao): serve de Idempotency-Key
  enviarPeloProvedorAtual: (payload: LegacyPayload) => Promise<string>,
): Promise<{ provedor: "meeng" | "atual"; id: string }> {
  if (!vaiParaMeeng(chave)) {
    return { provedor: "atual", id: await enviarPeloProvedorAtual(antigo) };
  }
  const { corpo } = paraMeeng(antigo);
  const { data, error } = await meeng.emails.send(corpo, { idempotencyKey: chave });
  if (error) {
    // Plano de volta automático: se o Meeng.dev recusar, o provedor atual envia.
    console.error(`meeng recusou (${error.status} ${error.code}): ${error.message}`);
    return { provedor: "atual", id: await enviarPeloProvedorAtual(antigo) };
  }
  return { provedor: "meeng", id: data.emailId ?? "" };
}
```

Atenção ao plano de volta automático: ele só vale para **recusa** da API (um `4xx` ou `5xx`). Se a chamada chegou ao Meeng.dev mas a resposta se perdeu, o e-mail pode ter sido aceito; por isso a mesma `chave` serve de `Idempotency-Key` no Meeng.dev, e o provedor antigo também deve receber a sua chave de idempotência se ele a oferece.

O mesmo adaptador em Go:

```go
package main

import (
	"context"
	"fmt"
	"log"

	meeng "github.com/meeng-dev/sdk/go"
)

// LegacyPayload é o formato que o sistema já usa para pedir um envio.
type LegacyPayload struct {
	From        string
	To          []string
	Subject     string
	HTML        string
	Text        string
	ReplyTo     []string
	ScheduledAt string
	Headers     map[string]string
	Attachments []LegacyAttachment
}

// LegacyAttachment é um anexo, com o conteúdo em base64.
type LegacyAttachment struct{ Filename, Content string }

// ToMeeng traduz o payload antigo para o do Meeng.dev.
func ToMeeng(old LegacyPayload) meeng.EmailsSendParams {
	params := meeng.EmailsSendParams{
		From:    old.From,
		To:      meeng.StringList(old.To),
		ReplyTo: meeng.StringList(old.ReplyTo),
	}
	if len(old.Headers) > 0 {
		params.Headers = make(map[string]any, len(old.Headers))
		for name, value := range old.Headers {
			params.Headers[name] = value
		}
	}
	if old.Subject != "" {
		params.Subject = &old.Subject
	}
	if old.HTML != "" {
		params.HTML = &old.HTML
	}
	if old.Text != "" {
		params.Text = &old.Text
	}
	if old.ScheduledAt != "" {
		params.ScheduledAt = &old.ScheduledAt
	}
	for _, a := range old.Attachments {
		params.Attachments = append(params.Attachments, meeng.EmailsSendAttachment{Filename: a.Filename, Content: a.Content})
	}
	return params
}

func main() {
	client := meeng.New("") // lê MEENG_API_KEY e MEENG_BASE_URL

	sent, err := client.Emails.Send(context.Background(), ToMeeng(LegacyPayload{
		From:    "Loja Exemplo <pedidos@envio.exemplo.com.br>",
		To:      []string{"cliente@example.com"},
		Subject: "Pedido 1042 confirmado",
		Text:    "Recebemos o seu pedido 1042.",
	}), meeng.WithIdempotencyKey("pedido-1042-confirmacao"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.EmailID)
}
```

## Mapa das referências

| Quando você precisa de | Abra |
|---|---|
| Campo a campo do envio, a resposta, os erros, os webhooks e a assinatura: o que muda e o que continua igual | [`referencias/mapeamento-de-campos.md`](referencias/mapeamento-de-campos.md) |
| As fases da migração com a lista de conferência, o esquenta, o plano de volta e o que apagar no fim | [`referencias/plano-de-migracao.md`](referencias/plano-de-migracao.md) |

## Ao terminar

Entregue ao dono, em português: onde o envio passou a ir pelo Meeng.dev e em que porcentagem, o que ficou do provedor antigo e até quando, as taxas de devolução e de reclamação observadas, o que foi testado (envio real, anexo, agendado, repetido com a mesma chave, webhook novo) e o que falta (chegar a 100%, apagar as chaves antigas, descadastrar o webhook antigo).
