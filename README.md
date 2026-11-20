# SDKs do Meeng.dev

Os SDKs da API do [Meeng.dev](https://meeng.dev), o e-mail para desenvolvedores, num monorepo: um contrato, um gerador em Go e um pacote por linguagem. A API fica em `https://meeng.dev/api`; crie uma chave em [Configurações > Chaves](https://meeng.dev/configuracoes/chaves).

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

- a chave de API e a URL do app (`https://meeng.dev`) lidas de `MEENG_API_KEY` e `MEENG_BASE_URL`, ou passadas ao cliente;
- os recursos `emails` (`send`, `batch`, `get`, `list`, `update`, `cancel`), `domains`, `contacts`, `contactBooks`, `campaigns` e `analytics`;
- `Idempotency-Key` nos envios, só quando o chamador o dá;
- tentativas automáticas e seguras: um 429 sempre se repete; um 5xx ou uma falha de conexão só numa leitura, num `PUT`, num `DELETE` ou numa chamada com chave de idempotência; `maxRetries` e `retryDelay` em todas as linguagens;
- paginação por `page` e `limit`, com `count` e `totalPage` nas respostas;
- erros com `code` e `message`, nos formatos que a API escreve;
- `webhooks(secret)` com `verify` e `constructEvent`, com os cabeçalhos `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`, e o evento tipado pelo `type`;
- exemplos que rodam em `examples/` de cada pasta (enviar, lote, agendar, domínio e webhook) e um teste de contrato contra uma API de mentira feita do contrato.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng(); // a chave vem de MEENG_API_KEY
const { data, error } = await meeng.emails.send({
  from: "Acme <contato@acme.com.br>",
  to: "ana@example.com",
  subject: "Bem-vinda à Acme",
  html: "<p>Olá, Ana! Sua conta está pronta.</p>",
});
```

O README de cada pasta mostra o equivalente na linguagem.

## Skills

A pasta [`skills/`](skills) tem as skills do Meeng.dev para assistentes de código: enviar e-mail, domínios, webhooks, contatos e campanhas, SMTP, MCP e migração de outro provedor. Cada uma traz exemplos nos SDKs deste repositório, e o `./check.sh` compila esses exemplos contra os SDKs, para que uma skill nunca ensine uma chamada que não existe. Veja [`skills/README.md`](skills/README.md).

## Como funciona

`openapi.json` é o contrato que o app exporta (`docs/contracts/api.openapi.json`, mantido igual às rotas pelo `aru openapi:check`). O gerador (`generator/`, em Go) lê o contrato num modelo neutro e escreve cada SDK a partir de `generator/templates/<linguagem>/`. Os nomes dos métodos e os eventos de webhook, que o contrato não traz, estão em `generator/surface.go` e `generator/webhooks.go`.

```sh
./regen.sh               # copia o contrato do app (origin/develop) e escreve os 11 SDKs
go run ./cmd/sdkgen      # escreve os SDKs a partir do openapi.json que está aqui
./check.sh               # os gates de todas as linguagens, o teste de contrato e os exemplos
go run ./cmd/fakeapi     # a API de mentira, feita só do contrato, para rodar os testes de contrato à mão
```

O `./regen.sh` copia o contrato do app, tirando a frase da descrição que cita outro produto (e recusando o contrato que cite um em outro lugar). As regras do repositório, as ferramentas de cada linguagem e as diferenças deliberadas entre o SDK e o contrato estão em [`AGENTS.md`](AGENTS.md).

## Licença

MIT. Veja [`LICENSE.md`](LICENSE.md).
