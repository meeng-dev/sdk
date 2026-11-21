# Mapeamento de campos

O que muda ao passar a chamada de envio para o Meeng.dev, sem citar marcas: leia a coluna da esquerda como "o que o seu código faz hoje" (a convenção mais comum entre provedores) e a da direita como "o que o Meeng.dev espera". Confira sempre o formato real do seu provedor atual: este quadro é uma lista de conferência, não uma garantia sobre o formato de outro serviço.

## O envio: `POST https://meeng.dev/api/v1/emails`

| Seu código faz hoje | No Meeng.dev | Observação |
|---|---|---|
| `from` | `from` | igual. Um único endereço, com ou sem nome (`Loja <pedidos@envio.exemplo.com.br>`). O domínio precisa estar verificado |
| `to` (texto ou lista) | `to` | igual: aceita um endereço ou uma lista |
| `cc`, `bcc` | `cc`, `bcc` | igual |
| `reply_to` | **`replyTo`** | o Meeng.dev usa `camelCase` |
| `subject` | `subject` | obrigatório, a não ser que use `templateId` |
| `html`, `text` | `html`, `text` | igual. É obrigatório um dos dois |
| `scheduled_at` | **`scheduledAt`** | em ISO 8601 com fuso. Não aceita frases como "daqui a 1 hora" no envio |
| `headers` | `headers` | igual: pares de texto. Alguns cabeçalhos reservados (`from`, `to`, `date`, `message-id`, `content-type`...) são descartados em silêncio |
| `attachments` (em base64 ou por endereço) | **não existe** | o Meeng.dev não anexa nem guarda arquivos: hospede o arquivo e ponha o **link** no corpo (`html` ou `text`). Mandar `attachments` com itens dá `400` (`ZodError`) |
| `tags` | **não existe** | guarde as tags no seu banco, junto do `emailId`, ou mande um cabeçalho `X-` seu para achá-las depois |
| `template` do provedor antigo | `templateId` e `variables` | o template precisa ser recriado no painel do Meeng.dev, e as variáveis são `{{nome}}` |
| a chave de idempotência | cabeçalho `Idempotency-Key` | o mesmo nome de cabeçalho; de 1 a 256 caracteres; vale 24 horas |
| a resposta, com o id do e-mail em `id` | **`emailId`** | `{"emailId": "..."}`. Troque a leitura do id |

A resposta de sucesso é sempre `200`. O SDK TypeScript devolve `{ data, error }`; o de Python devolve `(data, error)` e levanta `MeengHTTPError` por padrão; o de Go devolve `(resposta, erro)`.

### O lote

O lote do Meeng.dev é `POST /api/v1/emails/batch` com uma **lista** de corpos como o acima, de até **100** por chamada. A resposta é `{"data": [{"emailId": "..."}, ...]}`. Se o seu código manda mais de 100, divida em pedaços de 100 e dê a cada pedaço a sua própria `Idempotency-Key`.

### Ler o e-mail depois

`GET /api/v1/emails/{emailId}` devolve o e-mail e os eventos; `GET /api/v1/emails` lista (50 por página, no máximo) com filtro de período e de domínio.

## Erros

O Meeng.dev responde `{"error": {"code": "...", "message": "..."}}`, com o `code` entre `BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `METHOD_NOT_ALLOWED`, `NOT_UNIQUE`, `RATE_LIMITED` e `INTERNAL_SERVER_ERROR`; e `{"success": false, "error": {"issues": [...], "name": "ZodError"}}` quando um campo está faltando ou com o tipo errado. Troque as comparações por código do erro antigo (`if erro.name == ...`) pelo par status e `code`. A tabela completa está em `meeng-enviar-email`.

- **Limite de requisições:** 2 por segundo por organização, por padrão (e `429` com `Retry-After` quando passa). Se o sistema manda rajadas maiores, use o lote ou peça à equipe do Meeng.dev que aumente o limite da organização.
- **Idempotência repetida com corpo diferente:** `409` `NOT_UNIQUE`. Se o seu código já faz isso, confira que a chave é derivada do envio.
- **Destinatário bloqueado:** `200` com e-mail `SUPPRESSED` (não é erro de API). Veja o evento `email.suppressed`.

## Webhooks

| O receptor antigo faz | No Meeng.dev |
|---|---|
| lê o tipo do evento num campo do corpo | `type` (`email.delivered`, `email.bounced`...) e o cabeçalho `X-Meeng-Event` |
| deduplica pelo id do aviso | `id` do aviso (igual ao `X-Meeng-Call`), o mesmo em todas as tentativas |
| acha o e-mail pelo id | `data.id` é o `emailId` devolvido no envio |
| confere uma assinatura de outro formato | `X-Meeng-Signature`: `v1=` mais o HMAC-SHA256 em hexadecimal de `{X-Meeng-Timestamp}.{corpo}` com o segredo `whsec_...` como chave (sem decodificar). Use o SDK (`constructEvent`) |
| espera os campos do corpo do provedor antigo | o corpo do Meeng.dev: `data.to` é lista, `data.occurredAt` é a hora do fato, `data.bounce`, `data.failed`, `data.suppression`, `data.open` e `data.click` trazem os detalhes. Veja `meeng-webhooks` |

Os eventos de e-mail correspondentes: enviado `email.sent`, entregue `email.delivered`, adiado `email.delivery_delayed`, devolvido `email.bounced`, reclamação de spam `email.complained`, aberto `email.opened`, clicado `email.clicked`, falhou `email.failed`, recusado antes de sair `email.rejected`, bloqueado por supressão `email.suppressed`. Há também os de contato (`contact.created`, `contact.updated`, `contact.deleted`) e de domínio (`domain.created`, `domain.verified`, `domain.updated`, `domain.deleted`).

**Na virada, rode os dois receptores.** Cada provedor chama o seu endereço (`/webhooks/atual` e `/webhooks/meeng`), e os dois gravam o resultado na mesma tabela, com a chave única `(provedor, id do aviso)`. Quando o antigo parar de enviar, desligue o primeiro.

## Contatos e listas

- A lista de contatos do Meeng.dev tem **dupla confirmação ligada** por padrão. Para contatos que já confirmaram no provedor antigo e que você consegue provar, crie a lista com `doubleOptInEnabled: false` e importe-os com `subscribed: true`; para os demais, deixe a confirmação ligada.
- Importe em lotes de 1000 (`contacts.bulkCreate`), com a prova do aceite em `properties`. O processamento é assíncrono e erros por contato não voltam na resposta: confira o total no fim.
- Os contatos que **saíram** (descadastrados, devolvidos, reclamação) devem ser importados com `subscribed: false`, para nunca receberem.
- Variáveis de personalização: no Meeng.dev são `{{firstName}}`, `{{lastName}}`, `{{email}}` e as variáveis da lista (`{{empresa}}`), com `{{nome,fallback=amigo}}` para o texto padrão; o link de descadastro é `{{meeng_unsubscribe_url}}`. Troque os marcadores do conteúdo antigo.

## Supressões

A lista de endereços que nunca devem receber (bounces duros, reclamações, bloqueios) do provedor antigo precisa ir para o Meeng.dev **antes** do primeiro envio. Use a tela de supressões do painel (`https://meeng.dev/painel/supressoes`), que aceita adicionar endereços em quantidade, ou, com um assistente conectado por MCP, a ferramenta `add_suppression` (`meeng-mcp`).

## SMTP

Se o sistema envia por SMTP, troque o servidor, o usuário e a senha: veja `meeng-smtp`. O usuário é `meeng` e a senha é a chave de API. O endereço do remetente continua o mesmo, de um domínio verificado.

## Exemplo: o mesmo envio, antes e depois, em Python

```python
from typing import Any

from meeng import Meeng
from meeng.models import EmailsSendParams

meeng = Meeng()  # lê MEENG_API_KEY e MEENG_BASE_URL


def para_meeng(antigo: dict[str, Any]) -> EmailsSendParams:
    """Traduz um payload de envio no formato antigo (snake_case) para o do Meeng.dev."""
    corpo: EmailsSendParams = {"from": antigo["from"], "to": antigo["to"]}
    if antigo.get("subject") is not None:
        corpo["subject"] = antigo["subject"]
    if antigo.get("html") is not None:
        corpo["html"] = antigo["html"]
    if antigo.get("text") is not None:
        corpo["text"] = antigo["text"]
    if antigo.get("cc") is not None:
        corpo["cc"] = antigo["cc"]
    if antigo.get("bcc") is not None:
        corpo["bcc"] = antigo["bcc"]
    if antigo.get("reply_to") is not None:
        corpo["replyTo"] = antigo["reply_to"]
    if antigo.get("scheduled_at") is not None:
        corpo["scheduledAt"] = antigo["scheduled_at"]
    if antigo.get("headers") is not None:
        corpo["headers"] = antigo["headers"]
    if antigo.get("attachments"):
        # O Meeng.dev não recebe anexos: troque o arquivo por um link no corpo.
        raise ValueError("e-mail com anexo: ponha o link do arquivo no corpo")
    return corpo


payload_antigo = {
    "from": "Loja Exemplo <pedidos@envio.exemplo.com.br>",
    "to": ["cliente@example.com"],
    "subject": "Pedido 1042 confirmado",
    "text": "Recebemos o seu pedido 1042.",
    "reply_to": "atendimento@exemplo.com.br",
}

data, _ = meeng.emails.send(para_meeng(payload_antigo), options={"idempotency_key": "pedido-1042-confirmacao"})
if data is None:  # só acontece com raise_on_error=False
    raise RuntimeError("o Meeng.dev não respondeu")
print(data["emailId"])
```
