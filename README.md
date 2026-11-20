# SDKs do meeng

Os SDKs da API do meeng, num monorepo: um contrato, um gerador em Go e um pacote por linguagem. A API do meeng é a do useSend v1, então cada SDK tem o mesmo formato do SDK do useSend: troque o nome do cliente, a chave e a URL, e o resto do código continua igual.

> Nenhum pacote foi publicado. Os nomes abaixo são propostas, aguardando a confirmação do Paulo, e cada pacote está marcado para não sair por engano.

| Linguagem | Pasta | Pacote proposto |
|---|---|---|
| Go | [`go/`](go) | `github.com/meeng-dev/sdk/go` |
| TypeScript / JavaScript | [`typescript/`](typescript) | `@meeng-dev/sdk` (npm) |
| Python | [`python/`](python) | `meeng-sdk` (PyPI) |
| PHP | [`php/`](php) | `meeng-dev/sdk` (Packagist) |
| Ruby | [`ruby/`](ruby) | `meeng-sdk` (RubyGems) |
| Java | [`java/`](java) | `dev.meeng:sdk` (Maven) |
| C# / .NET | [`csharp/`](csharp) | `MeengDev.Sdk` (NuGet) |
| Elixir | [`elixir/`](elixir) | `meeng_sdk` (Hex) |
| Dart / Flutter | [`dart/`](dart) | `meeng_sdk` (pub.dev) |
| Rust | [`rust/`](rust) | `meeng-sdk` (crates.io) |
| Swift | [`swift/`](swift) | `https://github.com/meeng-dev/sdk` (SwiftPM) |

Todos fazem o mesmo, sob a mesma versão (`VERSION`), com o mesmo teste em cada linguagem:

- a chave de API e a URL do app lidas de `MEENG_API_KEY` e `MEENG_BASE_URL`, ou passadas ao cliente;
- os recursos `emails` (`send`, `batch`, `get`, `list`, `update`, `cancel`), `domains`, `contacts`, `contactBooks`, `campaigns` e `analytics`;
- `Idempotency-Key` nos envios, só quando o chamador o dá;
- erros com `code` e `message`, nos formatos que a API escreve;
- `webhooks(secret)` com `verify` e `constructEvent`, no esquema do useSend, com os cabeçalhos `X-Meeng-*` e também os `X-UseSend-*`, e o evento tipado pelo `type`.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng("mg_123");
const { data, error } = await meeng.emails.send({
  from: "hello@acme.com",
  to: "cliente@example.com",
  subject: "Olá",
  html: "<p>Olá!</p>",
});
```

O README de cada pasta mostra o equivalente na linguagem.

## Skills

A pasta [`skills/`](skills) tem as skills do Meeng.dev para assistentes de código: enviar e-mail, domínios, webhooks, contatos e campanhas, SMTP, MCP e migração de outro provedor. Cada uma traz exemplos nos SDKs deste repositório, e o `./check.sh` compila esses exemplos contra os SDKs, para que uma skill nunca ensine uma chamada que não existe. Veja [`skills/README.md`](skills/README.md).

## Como funciona

`openapi.json` é o contrato do useSend v1 com a marca trocada (título e servidor; o resto é idêntico). O gerador (`generator/`, em Go) lê o contrato num modelo neutro e escreve cada SDK a partir de `generator/templates/<linguagem>/`. Os nomes dos métodos e os eventos de webhook, que o contrato não traz, estão em `generator/surface.go` e `generator/webhooks.go`.

```sh
./regen.sh --reference   # refaz o openapi.json do clone do useSend e escreve os 11 SDKs
go run ./cmd/sdkgen      # escreve os SDKs a partir do openapi.json que está aqui
./check.sh               # os gates de todas as linguagens
```

Quando o app meeng.dev exportar o contrato (`docs/contracts/api.openapi.json`), `./regen.sh` passa a copiá-lo de lá. As regras do repositório, as ferramentas de cada linguagem e as diferenças deliberadas entre o SDK e o contrato estão em [`AGENTS.md`](AGENTS.md).

## Licença

MIT. Veja [`LICENSE.md`](LICENSE.md).
