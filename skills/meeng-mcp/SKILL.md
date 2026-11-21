---
name: meeng-mcp
description: Ensina a conectar um assistente de código ou de chat (Claude Code, Claude Desktop, Cursor, VS Code, Codex e qualquer cliente MCP) ao servidor MCP do Meeng.dev em https://meeng.dev/mcp, a plataforma brasileira de e-mail para desenvolvedores, para que o assistente envie e-mail, consulte domínios, registros DNS, contatos, campanhas, modelos, supressões e webhooks da conta, com a chave de API (44 ferramentas, 2 leituras e 3 roteiros). Use quando o pedido for conectar o Meeng.dev a um assistente de IA, configurar o MCP com uma chave mg_, escolher entre chave FULL e SENDING para o assistente, entender o que o assistente pode e não pode fazer, ou depurar um erro 401, 429 ou de permissão no MCP. Cobre conexão, chave, ferramentas, confirmação, limites e segurança. O servidor serve quem pede e acompanha um envio por conversa; o envio do sistema do dono é pela API ou por um SDK (meeng-enviar-email).
---

# Conectar um assistente ao Meeng.dev por MCP

MCP (Model Context Protocol) é o jeito padrão de um assistente de IA usar as ferramentas de um serviço. Conectado ao MCP do Meeng.dev, o assistente do dono pode, por conversa: enviar um e-mail, ver os registros DNS que faltam num domínio, consultar o uso do plano, listar modelos e supressões. Cada pedido passa pelas mesmas regras e permissões da API: o assistente nunca pode mais do que a chave que ele recebeu.

Fale com o dono em português simples. Explique em uma frase o que é MCP (a "tomada" que deixa o assistente usar o Meeng.dev) e o que é a chave (a senha do assistente). Pergunte só o que falta: qual assistente ele usa e se o assistente deve apenas enviar e-mail ou também administrar a conta.

## Confira o que o servidor oferece

O endereço do servidor é `https://meeng.dev/mcp`. Antes de prometer algo ao dono:

1. Teste o endereço (veja "Testar a conexão"). O servidor está no ar em `https://meeng.dev/mcp` (provado em 21/11/2026 com o cliente MCP oficial: 44 ferramentas, e-mail enviado e `DELIVERED`). Se responder `404`, esta instalação não o tem: use a API (`meeng-enviar-email`) e volte aqui depois.
2. Peça ao servidor a **lista real de ferramentas** (`tools/list`, que os clientes mostram ao conectar). As ferramentas abaixo são as que ele **inclui**, não uma lista fechada: ela cresce com a API. A lista do servidor é a fonte da verdade; **nunca invente o nome de uma ferramenta** nem de um argumento.

## Quando usar

Use quando o dono quer que um assistente de IA opere o Meeng.dev, ou quando um desenvolvedor quer perguntar e agir pela conversa ("verifica o que falta no meu domínio", "manda um e-mail de teste para mim"). Quando é o **sistema** do dono que envia e-mail, não um assistente, use a API ou o SDK (`meeng-enviar-email`): a API é o caminho principal de envio, e o relay SMTP (`meeng-smtp`) é só reserva. O assistente conectado pode, porém, **escrever esse código**: o guia `meeng://docs/api` e o roteiro `send_from_my_system` o levam pelo SDK de Go ou de Node (e dos outros), pelo `MAIL_URL=meeng://` de um app Arandu e pelos dois endereços da API, com a mesma chave.

## O que o assistente consegue (e o que não)

O servidor oferece ferramentas e leituras do mesmo conjunto da API, com a chave do dono:

| Área | Ferramentas que o servidor inclui |
|---|---|
| Conta e uso | `show_account` (a organização e a chave em uso), `get_usage` (o uso do plano e os limites) |
| Envio | `send_email` e `send_batch_emails` (até 100 e-mails por chamada, com agendamento) |
| E-mails | `list_emails`, `get_email`, `update_email` (reagendar), `cancel_email` |
| Domínios | `list_domains`, `create_domain`, `get_domain`, `get_domain_dns_records` (os registros que faltam publicar, em texto para seguir), `verify_domain`, `delete_domain` |
| Contatos | `list_contact_books`, `get_contact_book`, `create_contact_book` (nasce com a confirmação em duas etapas ligada), `update_contact_book`, `delete_contact_book`, `list_contacts`, `get_contact`, `create_contact`, `upsert_contact` (acha o contato pelo e-mail, sem duplicar), `update_contact`, `delete_contact`, `bulk_create_contacts` (até 1000, em segundo plano), `bulk_delete_contacts` |
| Campanhas | `list_campaigns`, `get_campaign`, `create_campaign` (rascunho, com o link de descadastro no corpo), `schedule_campaign`, `pause_campaign`, `resume_campaign`, `delete_campaign` |
| Modelos | `list_templates`, `get_template`, `render_template` (mostra o resultado com variáveis, sem enviar) |
| Supressões | `list_suppressions`, `add_suppression`, `remove_suppression` |
| Webhooks | `list_webhooks`, `create_webhook` (o segredo de assinatura não volta: o dono o lê no painel) |
| Números | `get_email_metrics` (envios por dia), `get_reputation_metrics` (devolução e reclamação) |

Também há leituras (resources): o guia da API (`meeng://docs/api`: chave, envio com exemplos em Go e Node, SDKs publicados, os dois endereços da API, `MAIL_URL`, "Só envio", erros) e os registros DNS dos domínios da conta (`meeng://dominios/dns`); e três roteiros prontos (prompts): `send_first_email`, `set_up_domain` e `send_from_my_system`. Confira tudo com `tools/list`, `resources/list` e `prompts/list`.

**O que fica no painel, fora do MCP:** criar e apagar chaves (inclusive a chave "Só envio"), a equipe e a assinatura, **exportar os dados da organização** (Configurações, só o dono) e o botão **"Configurar automaticamente"** da página do domínio, que publica os registros pela Cloudflare ou por um provedor compatível. O assistente mostra os registros com `get_domain_dns_records` e diz que o botão existe; quem o usa é a pessoa. Edição de modelos e a gestão de webhooks além de listar e cadastrar também são do painel.

**Sem anexos:** o Meeng.dev não anexa arquivos, e `send_email` não tem o campo. Para mandar um documento, o assistente põe o **link** do arquivo no corpo (`html` ou `text`); um link que expira, se o documento é sensível.

O MCP não cria chaves de API nem mexe em plano ou cobrança, e o assistente age só dentro da organização da chave: um registro de outra organização responde como se não existisse.

## Regras que valem sempre

1. **A chave é a senha do assistente.** Crie uma chave **só para ele** no painel (`https://meeng.dev/configuracoes/chaves`), com nome que diga isso (`assistente-claude-code`), e apague-a quando não precisar mais. Nunca use a chave de produção do sistema do dono.
2. **Escolha a permissão certa.** Uma chave **SENDING** (a opção "Só envio" do painel) só envia e-mail (`send_email` e `send_batch_emails`): as demais ferramentas respondem que a chave não pode usá-las. Uma chave **FULL** administra a conta toda (domínios, contatos, campanhas, supressões, webhooks). Para um assistente que só ajuda a testar envio, use SENDING; use FULL só se o dono pediu que o assistente administre.
3. **Restrinja a um domínio** quando o assistente só mexe num (a chave restrita só envia com `from` daquele domínio e só enxerga ele).
4. **O que envia ou apaga pede confirmação.** `send_email`, `send_batch_emails`, `schedule_campaign`, `update_email`, `cancel_email`, `delete_domain`, `delete_contact`, `bulk_delete_contacts`, `delete_contact_book`, `delete_campaign` e `remove_suppression` só executam com `confirm=true`, e o assistente deve mandar isso **só depois de o dono aprovar** naquela conversa. Sem `confirm`, a ferramenta não faz nada e conta o que faria. Isso protege contra o assistente mandar e-mail por engano. Criar um domínio, uma lista, um contato (um a um ou em lote), um rascunho de campanha, uma supressão ou um webhook não envia nada e não pede `confirm`, mas conta como escrita no limite.
5. **Idempotência:** `send_email` e `send_batch_emails` aceitam `idempotency_key` (de 1 a 256 caracteres). Repetir o mesmo envio com a mesma chave devolve o primeiro resultado, sem enviar duas vezes. Repita a **mesma** chave ao tentar de novo depois de um erro de rede. As outras ferramentas que criam não têm essa proteção: depois de um erro, o assistente lista antes de criar de novo, para não duplicar.
6. **Limites:** cada chave tem um orçamento por minuto (hoje, 120 chamadas, das quais no máximo 30 escritas). Passou: o servidor responde `429` com `Retry-After`. Um assistente que entra em laço não manda e-mail sem fim.
7. **Nada de e-mail em massa pelo assistente.** Para campanhas use `meeng-contatos-e-campanhas`; o assistente serve para tarefas pontuais.
8. **No plano Grátis, o envio vale só para endereços da própria organização.** Um destinatário de fora volta como recusa com o motivo em português; o assistente conta isso à pessoa em vez de tentar de novo.
9. **A chave nunca vai na conversa.** Ela fica no arquivo de configuração do cliente (ou numa variável de ambiente), não no texto que o assistente lê. Não cole a chave em chat, issue ou commit.
10. O assistente lê dados de clientes do dono (endereços de e-mail em listas, conteúdo de e-mails). Isso é dado pessoal pela LGPD: conecte só assistentes que o dono pode usar para isso.

## Conectar

O servidor fala o protocolo MCP (revisão `2024-11-05`) por HTTP: o cliente manda os pedidos por `POST https://meeng.dev/mcp`, com a chave no cabeçalho `Authorization: Bearer mg_...`.

### Claude Code

```sh
claude mcp add --transport http meeng https://meeng.dev/mcp \
  --header "Authorization: Bearer $MEENG_API_KEY"
```

Confira com `claude mcp list` e, dentro do Claude Code, `/mcp`. Por padrão o servidor vale só para você, neste projeto, e o shell troca `$MEENG_API_KEY` pela chave antes de gravar a configuração. Para compartilhar com o time (`--scope project`), o comando grava num arquivo do projeto que **vai para o repositório**: aí **não grave a chave**. Passe o texto literal, entre aspas simples, para que o Claude Code leia a variável de ambiente de cada pessoa na hora de conectar:

```sh
claude mcp add --transport http --scope project meeng https://meeng.dev/mcp \
  --header 'Authorization: Bearer ${MEENG_API_KEY}'
```

### Cursor, VS Code e outros clientes que leem um arquivo JSON

Cursor lê `.cursor/mcp.json` (ou `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "meeng": {
      "url": "https://meeng.dev/mcp",
      "headers": { "Authorization": "Bearer ${env:MEENG_API_KEY}" }
    }
  }
}
```

O VS Code lê `.vscode/mcp.json`, e pede a chave numa janela, sem gravá-la no arquivo:

```json
{
  "inputs": [
    { "type": "promptString", "id": "meeng-key", "description": "Chave de API do Meeng.dev (mg_...)", "password": true }
  ],
  "servers": {
    "meeng": {
      "type": "http",
      "url": "https://meeng.dev/mcp",
      "headers": { "Authorization": "Bearer ${input:meeng-key}" }
    }
  }
}
```

### Codex

No arquivo `~/.codex/config.toml`, com a chave numa variável de ambiente (confira a sintaxe na documentação da versão instalada):

```toml
[mcp_servers.meeng]
url = "https://meeng.dev/mcp"
bearer_token_env_var = "MEENG_API_KEY"
```

### Claude Desktop e clientes que só falam por programa local (stdio)

Alguns clientes só abrem um programa local. Use uma ponte como o `mcp-remote`, que fala com o servidor remoto e conversa com o cliente por stdio:

```json
{
  "mcpServers": {
    "meeng": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://meeng.dev/mcp", "--header", "Authorization:${MEENG_AUTH}"],
      "env": { "MEENG_AUTH": "Bearer COLE_A_CHAVE_AQUI" }
    }
  }
}
```

Esse arquivo guarda a chave em texto no computador do dono: mantenha-o fora de repositórios e com permissão só do usuário. Confira na documentação do cliente se ele já aceita servidores remotos por URL (nesse caso, a forma sem ponte é melhor).

### Outros clientes

Qualquer cliente MCP que aceite um servidor remoto por URL e cabeçalhos serve: endereço `https://meeng.dev/mcp`, cabeçalho `Authorization: Bearer mg_...`. Se o cliente só aceita OAuth, não conecte por ele: o Meeng.dev autentica pela chave de API.

## Testar a conexão

Sem cliente nenhum, com `curl`, faça o aperto de mão e liste as ferramentas:

```sh
curl -sS https://meeng.dev/mcp \
  -H "Authorization: Bearer $MEENG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"teste","version":"1"}}}'

curl -sS https://meeng.dev/mcp \
  -H "Authorization: Bearer $MEENG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
```

A primeira resposta traz o nome e a versão do servidor; a segunda, a lista de ferramentas com a descrição e os argumentos de cada uma. Use essa lista como a verdade.

| Resposta | O que fazer |
|---|---|
| `401` | a chave não foi enviada, está incompleta, foi apagada ou não é do Meeng.dev; confira `Authorization: Bearer mg_...` e se a chave ainda existe em Configurações › Chaves |
| `404` | o servidor MCP ainda não está publicado neste endereço |
| `429` com `Retry-After` | o orçamento por minuto da chave acabou; espere |
| a ferramenta responde que a chave "não pode usar" | a chave é SENDING; crie uma FULL se o dono quiser que o assistente administre |

### Testar com um cliente MCP de verdade

Para provar a conexão como um cliente faz (aperto de mão, lista, uma leitura, um envio e o acompanhamento até `DELIVERED`), use o SDK oficial de MCP em Node (`npm install @modelcontextprotocol/sdk`). O envio vai para um endereço do próprio dono, primeiro como prévia e depois com `confirm`:

```ts nocheck
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

const client = new Client({ name: "teste", version: "1" });
await client.connect(
  new StreamableHTTPClientTransport(new URL("https://meeng.dev/mcp"), {
    requestInit: { headers: { Authorization: `Bearer ${process.env.MEENG_API_KEY}` } },
  }),
);

console.log((await client.listTools()).tools.length, "ferramentas");
const chamar = async (name: string, args: Record<string, unknown>) =>
  (await client.callTool({ name, arguments: args })) as { isError?: boolean; content: { text: string }[] };

const mensagem = { from: "teste@envio.exemplo.com.br", to: "dono@exemplo.com.br", subject: "Teste do MCP", text: "Funcionou." };
console.log((await chamar("send_email", mensagem)).content[0].text); // prévia: nada foi enviado
const enviado = await chamar("send_email", { ...mensagem, confirm: true, idempotency_key: "teste-mcp-001" });
const { emailId } = JSON.parse(enviado.content[0].text);
const email = await chamar("get_email", { email_id: emailId });
console.log(JSON.parse(email.content[0].text).latestStatus); // repita até DELIVERED
await client.close();
```

## Como o assistente deve trabalhar

1. Começar por `show_account` para saber a organização e o tipo de chave, e `get_usage` quando o assunto é limite.
2. Antes de enviar, mostrar ao dono o **rascunho completo** (de, para, assunto, corpo) e esperar a aprovação; só então chamar a ferramenta com `confirm=true` e uma `idempotency_key` descritiva.
3. Para modelos, usar `render_template` para mostrar o resultado antes de qualquer envio.
4. Para domínios, usar a leitura dos registros DNS e pedir que o dono publique o que falta (o assistente não tem acesso ao DNS dele).
5. Dizer ao dono, em português, o que foi feito e o que ficou pendente; ao encontrar um erro, mostrar a mensagem da ferramenta em vez de adivinhar.

## Ao terminar

Entregue ao dono, em português: qual assistente foi conectado, com que chave (nome e permissão, sem o valor), o que o assistente pode fazer, o que foi testado (conexão, lista de ferramentas, uma leitura, um envio de teste confirmado) e o que falta (apagar a chave quando não precisar, restringir ao domínio).
