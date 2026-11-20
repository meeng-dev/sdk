---
name: meeng-mcp
description: Ensina a conectar um assistente de código ou de chat (Claude Code, Claude Desktop, Cursor, VS Code, Codex e qualquer cliente MCP) ao servidor MCP do Meeng.dev em https://meeng.dev/mcp, a plataforma brasileira de e-mail para desenvolvedores, para que o assistente envie e-mail, consulte domínios, registros DNS, modelos, supressões e webhooks da conta, com a chave de API. Use quando o pedido for conectar o Meeng.dev a um assistente de IA, configurar o MCP com uma chave mg_, escolher entre chave FULL e SENDING para o assistente, entender o que o assistente pode e não pode fazer, ou depurar um erro 401, 429 ou de permissão no MCP. Cobre conexão, chave, ferramentas, confirmação, limites e segurança.
---

# Conectar um assistente ao Meeng.dev por MCP

MCP (Model Context Protocol) é o jeito padrão de um assistente de IA usar as ferramentas de um serviço. Conectado ao MCP do Meeng.dev, o assistente do dono pode, por conversa: enviar um e-mail, ver os registros DNS que faltam num domínio, consultar o uso do plano, listar modelos e supressões. Cada pedido passa pelas mesmas regras e permissões da API: o assistente nunca pode mais do que a chave que ele recebeu.

Fale com o dono em português simples. Explique em uma frase o que é MCP (a "tomada" que deixa o assistente usar o Meeng.dev) e o que é a chave (a senha do assistente). Pergunte só o que falta: qual assistente ele usa e se o assistente deve apenas enviar e-mail ou também administrar a conta.

## Estado do servidor: confira antes

O endereço do servidor é `https://meeng.dev/mcp`. Ele só responde depois de publicado na plataforma, e as ferramentas listadas abaixo podem mudar até a publicação. Antes de prometer algo ao dono:

1. Teste o endereço (veja "Testar a conexão"). Se responder `404`, o servidor ainda não está no ar: use a API (`meeng-enviar-email`) e volte aqui depois.
2. Peça ao servidor a **lista real de ferramentas** (`tools/list`, que os clientes mostram ao conectar). A lista do servidor é a fonte da verdade; **nunca invente o nome de uma ferramenta** nem de um argumento.

## Quando usar

Use quando o dono quer que um assistente de IA opere o Meeng.dev, ou quando um desenvolvedor quer perguntar e agir pela conversa ("verifica o que falta no meu domínio", "manda um e-mail de teste para mim"). Quando é o **sistema** do dono que envia e-mail, não um assistente, use a API ou o SDK (`meeng-enviar-email`).

## O que o assistente consegue (e o que não)

O servidor oferece ferramentas e leituras do mesmo conjunto da API, com a chave do dono:

| Área | Para que serve |
|---|---|
| Conta e uso | `show_account` (a organização e a chave em uso) e `get_usage` (o uso do plano) |
| Envio | `send_email` e `send_batch_emails` |
| Domínios | `get_domain_dns_records` (os registros que faltam publicar) e a leitura dos registros DNS dos domínios |
| Modelos | `list_templates`, `get_template`, `render_template` (mostra o resultado com variáveis, sem enviar) |
| Supressões | `list_suppressions`, `add_suppression`, `remove_suppression` |
| Webhooks | `list_webhooks`, `create_webhook` |
| Demais operações da API | domínios, e-mails, contatos e campanhas, conforme o servidor listar |

O MCP não cria chaves de API nem mexe em plano ou cobrança, e o assistente age só dentro da organização da chave: um registro de outra organização responde como se não existisse.

## Regras que valem sempre

1. **A chave é a senha do assistente.** Crie uma chave **só para ele** no painel (`https://meeng.dev/configuracoes/chaves`), com nome que diga isso (`assistente-claude-code`), e apague-a quando não precisar mais. Nunca use a chave de produção do sistema do dono.
2. **Escolha a permissão certa.** Uma chave **SENDING** só envia e-mail: as ferramentas de administração respondem que a chave não pode usá-las. Uma chave **FULL** administra a conta toda (domínios, supressões, webhooks). Para um assistente que só ajuda a testar envio, use SENDING; use FULL só se o dono pediu que o assistente administre.
3. **Restrinja a um domínio** quando o assistente só mexe num (a chave restrita só envia com `from` daquele domínio e só enxerga ele).
4. **Escritas pedem confirmação.** As ferramentas que mudam algo (enviar, adicionar supressão, criar webhook) só executam com `confirm=true`, e o assistente deve mandar isso **só depois de o dono aprovar** naquela conversa. Sem `confirm`, a ferramenta não faz nada e conta o que faria. Isso protege contra o assistente mandar e-mail por engano.
5. **Idempotência:** as escritas aceitam `idempotency_key` (de 1 a 256 caracteres). Repetir a mesma chamada com a mesma chave devolve o primeiro resultado, sem fazer duas vezes. Repita a **mesma** chave ao tentar de novo depois de um erro.
6. **Limites:** cada chave tem um orçamento por minuto (hoje, 120 chamadas, das quais no máximo 30 escritas). Passou: o servidor responde `429` com `Retry-After`. Um assistente que entra em laço não manda e-mail sem fim.
7. **Nada de e-mail em massa pelo assistente.** Para campanhas use `meeng-contatos-e-campanhas`; o assistente serve para tarefas pontuais.
8. **A chave nunca vai na conversa.** Ela fica no arquivo de configuração do cliente (ou numa variável de ambiente), não no texto que o assistente lê. Não cole a chave em chat, issue ou commit.
9. O assistente lê dados de clientes do dono (endereços de e-mail em listas, conteúdo de e-mails). Isso é dado pessoal pela LGPD: conecte só assistentes que o dono pode usar para isso.

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

## Como o assistente deve trabalhar

1. Começar por `show_account` para saber a organização e o tipo de chave, e `get_usage` quando o assunto é limite.
2. Antes de enviar, mostrar ao dono o **rascunho completo** (de, para, assunto, corpo) e esperar a aprovação; só então chamar a ferramenta com `confirm=true` e uma `idempotency_key` descritiva.
3. Para modelos, usar `render_template` para mostrar o resultado antes de qualquer envio.
4. Para domínios, usar a leitura dos registros DNS e pedir que o dono publique o que falta (o assistente não tem acesso ao DNS dele).
5. Dizer ao dono, em português, o que foi feito e o que ficou pendente; ao encontrar um erro, mostrar a mensagem da ferramenta em vez de adivinhar.

## Ao terminar

Entregue ao dono, em português: qual assistente foi conectado, com que chave (nome e permissão, sem o valor), o que o assistente pode fazer, o que foi testado (conexão, lista de ferramentas, uma leitura, um envio de teste confirmado) e o que falta (apagar a chave quando não precisar, restringir ao domínio).
