# Skills do Meeng.dev

Skills para assistentes de código (Claude Code, Codex, Cursor e outros): cada pasta ensina o assistente a fazer uma coisa com o Meeng.dev, com exemplos nos SDKs deste repositório e na API `https://meeng.dev/api/v1`. O formato é o das skills dos outros produtos da HYZIS: uma pasta por skill, um `SKILL.md` com o cabeçalho `name` e `description`, e as referências em `referencias/`.

| Skill | Para quê |
|---|---|
| [`meeng-enviar-email`](meeng-enviar-email/SKILL.md) | enviar e-mail: simples, em lote, agendado, com anexo e modelo; idempotência, erros e limites |
| [`meeng-dominios`](meeng-dominios/SKILL.md) | cadastrar o domínio de envio, publicar o DNS (MX, SPF, DKIM, DMARC), verificar e resolver o que falta |
| [`meeng-webhooks`](meeng-webhooks/SKILL.md) | receber os avisos (entregue, devolvido, aberto...), conferir a assinatura, tratar repetições |
| [`meeng-contatos-e-campanhas`](meeng-contatos-e-campanhas/SKILL.md) | listas de contatos, dupla confirmação, descadastro, campanhas, LGPD |
| [`meeng-smtp`](meeng-smtp/SKILL.md) | enviar por SMTP, para sistemas que não chamam API (vale quando o relay está ligado na conta) |
| [`meeng-mcp`](meeng-mcp/SKILL.md) | conectar um assistente de IA ao servidor MCP do Meeng.dev |
| [`meeng-migrar-de-outro-provedor`](meeng-migrar-de-outro-provedor/SKILL.md) | trocar o envio de outro provedor para o Meeng.dev, aos poucos e com volta |

## Como instalar

Copie a pasta da skill para onde o assistente lê skills: `.claude/skills/` (Claude Code), `.agents/skills/` (Codex e outros) ou o equivalente do cliente. Cada skill é inteira em si, sem depender de outra: copie só as que precisa.

```sh
cp -R skills/meeng-enviar-email ~/.claude/skills/
```

## O que as skills afirmam

- **Estado dos pacotes.** Os SDKs ainda não foram publicados nos registros; as skills dizem isso e mandam conferir antes de escrever uma instalação.
- **SMTP e MCP só valem ligados.** `meeng-smtp` pede que se confira a tela Configurações › SMTP, e `meeng-mcp` que se teste o endereço e se leia a lista real de ferramentas (`tools/list`) antes de prometer algo.
- **A API é a fonte da verdade.** Em caso de dúvida, o contrato é `https://meeng.dev/api/v1/doc`; as skills mandam o assistente nunca inventar campo, rota ou evento.

## Como mantê-las

- Os exemplos de Go, TypeScript, Python e PHP são **programas inteiros** e **compilam contra os SDKs deste repositório**: `go run ./cmd/skillscheck` (também é o passo `== skills` do `./check.sh` e um job da CI) falha quando um exemplo chama algo que o SDK não tem. Um trecho que não é um programa fica num bloco `<linguagem> nocheck`, e o relatório conta quantos são. Blocos `json` precisam ser JSON válido.
- O cabeçalho de cada `SKILL.md` só tem `name` (igual ao nome da pasta, em minúsculas e hifens) e `description` (uma linha, até 1024 caracteres, dizendo "Use quando ..."). Todo arquivo de `referencias/` é citado no `SKILL.md`, e todo link relativo existe.
- Nenhuma skill cita outro produto de e-mail. A verificação falha se achar o nome de um.
- Prosa em português do Brasil, simples, com a marca escrita "Meeng.dev". Código, identificadores e comentários de código seguem o SDK.
- Mudou o SDK, o contrato ou o comportamento da API? Rode `go run ./cmd/skillscheck` e ajuste a skill no mesmo commit.
