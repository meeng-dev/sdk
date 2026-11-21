# Skills do Meeng.dev

Skills para assistentes de código (Claude Code, Codex, Cursor e outros): cada pasta ensina o assistente a fazer uma coisa com o Meeng.dev, com exemplos nos SDKs deste repositório e na API `https://meeng.dev/api/v1`. O formato é o das skills dos outros produtos da HYZIS: uma pasta por skill, um `SKILL.md` com o cabeçalho `name` e `description`, e as referências em `referencias/`.

| Skill | Para quê |
|---|---|
| [`meeng-enviar-email`](meeng-enviar-email/SKILL.md) | enviar e-mail pela API (o caminho principal), por SDK ou pelo `MAIL_URL` de um app Arandu: simples, em lote, agendado e com modelo, com links no lugar de anexos; idempotência, erros e limites; a chave "Só envio" e o plano Grátis |
| [`meeng-dominios`](meeng-dominios/SKILL.md) | cadastrar o domínio de envio, publicar o DNS (MX, SPF, DKIM, DMARC), verificar e resolver o que falta |
| [`meeng-webhooks`](meeng-webhooks/SKILL.md) | receber os avisos (entregue, devolvido, aberto...), conferir a assinatura, tratar repetições |
| [`meeng-contatos-e-campanhas`](meeng-contatos-e-campanhas/SKILL.md) | listas de contatos, dupla confirmação, descadastro, campanhas, LGPD |
| [`meeng-smtp`](meeng-smtp/SKILL.md) | enviar por SMTP, só como reserva, para sistemas que não chamam API (`smtp.meeng.dev`, portas 587 e 465) |
| [`meeng-mcp`](meeng-mcp/SKILL.md) | conectar um assistente de IA ao servidor MCP do Meeng.dev (`https://meeng.dev/mcp`, 44 ferramentas, 2 leituras, 3 roteiros) |
| [`meeng-migrar-de-outro-provedor`](meeng-migrar-de-outro-provedor/SKILL.md) | trocar o envio de outro provedor para o Meeng.dev, aos poucos e com volta |

## Como instalar

Copie a pasta da skill para onde o assistente lê skills: `.claude/skills/` (Claude Code), `.agents/skills/` (Codex e outros) ou o equivalente do cliente. Cada skill é inteira em si, sem depender de outra: copie só as que precisa.

```sh
cp -R skills/meeng-enviar-email ~/.claude/skills/
```

## O que as skills afirmam

- **Estado dos pacotes.** A versão `0.1.0` dos SDKs está publicada em npm (`@meeng-dev/sdk`), Go (`github.com/meeng-dev/sdk/go`), PyPI, RubyGems e crates.io (`meeng-sdk`), Hex e pub.dev (`meeng_sdk`), NuGet (`MeengDev.Sdk`) e Packagist (`meeng-dev/sdk`); o Swift lê a tag do repositório. O Java (Maven Central) ainda não saiu. As skills mandam conferir a versão no registro antes de escrever uma instalação.
- **A API é o caminho principal de envio; o SMTP é a reserva.** `meeng-smtp` só vale para o programa que não chama API, e `meeng-mcp` manda testar o endereço e ler a lista real de ferramentas (`tools/list`) antes de prometer algo.
- **Dois endereços, uma chave.** `https://meeng.dev/api/v1` é a API do Meeng.dev e `https://api.meeng.dev` responde no formato das bibliotecas de outros provedores (basta trocar o endereço base e a chave); um app Arandu usa `MAIL_URL=meeng://<chave>@meeng.dev`. As skills descrevem isso sem citar o nome de nenhum outro produto.
- **Sem anexos.** O Meeng.dev não anexa nem guarda arquivos: as skills ensinam a pôr o link do arquivo no corpo, e nenhum exemplo usa `attachments`.
- **A API é a fonte da verdade.** Em caso de dúvida, o contrato é `https://meeng.dev/api/v1/doc`; as skills mandam o assistente nunca inventar campo, rota ou evento.

## Como mantê-las

- Os exemplos de Go, TypeScript, Python e PHP são **programas inteiros** e **compilam contra os SDKs deste repositório**: `go run ./cmd/skillscheck` (também é o passo `== skills` do `./check.sh` e um job da CI) falha quando um exemplo chama algo que o SDK não tem. Um trecho que não é um programa fica num bloco `<linguagem> nocheck`, e o relatório conta quantos são. Blocos `json` precisam ser JSON válido.
- O cabeçalho de cada `SKILL.md` só tem `name` (igual ao nome da pasta, em minúsculas e hifens) e `description` (uma linha, até 1024 caracteres, dizendo "Use quando ..."). Todo arquivo de `referencias/` é citado no `SKILL.md`, e todo link relativo existe.
- Nenhuma skill cita outro produto de e-mail. A verificação falha se achar o nome de um.
- Prosa em português do Brasil, simples, com a marca escrita "Meeng.dev". Código, identificadores e comentários de código seguem o SDK.
- Mudou o SDK, o contrato ou o comportamento da API? Rode `go run ./cmd/skillscheck` e ajuste a skill no mesmo commit.
