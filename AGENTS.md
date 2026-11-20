# meeng/sdk — repository rules

## Scope and authority

Os SDKs do Meeng.dev, um pacote por linguagem, gerados do contrato da API pelo gerador em Go deste repositório. O contrato é o que o app exporta (`meeng.dev/docs/contracts/api.openapi.json`), a rota é a fonte, e os SDKs seguem o contrato: um método para cada operação pública, com os mesmos nomes em todas as linguagens. O que a pessoa lê, nos README, nos pacotes e no código, é em português do Brasil ou neutro, e **nunca cita outro produto de e-mail**; o gerador recusa um contrato que cite.

Este repositório é filho do vault `meeng-dev/` (a raiz não é Git; este diretório é um repositório próprio). As notas do vault ficam nas pastas numeradas da raiz do vault (`10-adr/`, `60-contracts/`, `70-gaps/`, `90-reference/`); a frente deste trabalho é [[F5-sdk]], o contrato da API é [[CT-public-api]] e a regra de Go é [[ADR-0003-go-only]]. Instruções para agentes ficam só neste `AGENTS.md`; nunca `CLAUDE.md`. Quem edita o vault edita no vault, não aqui.

Antes de mexer, leia: este arquivo, `generator/surface.go`, `generator/webhooks.go` e o `AGENTS.md` do vault (`../AGENTS.md`).

## What is generated and what is written

- `go/`, `typescript/`, `python/`, `php/`, `ruby/`, `java/`, `csharp/`, `elixir/`, `dart/`, `rust/` e `swift/` são **saída**. O gerador esvazia cada pasta e escreve de novo; uma edição ali se perde na próxima rodada. Mude o template.
- `generator/templates/<linguagem>/` é a parte escrita à mão de cada SDK: cliente, erros, webhooks, testes, README. Um arquivo `.tmpl` é renderizado com o IR; qualquer outro é copiado. `_dot_` num nome vira `.`; `__model__`, `__service__` e `__webhook__` escrevem o arquivo uma vez por modelo, por serviço ou por grupo de webhook.
- `generator/` (Go) lê o `openapi.json` num IR neutro (`ir.go`) e o renderiza (`emit.go`, `lang_<linguagem>.go`). O contrato do app é OpenAPI 3.1 com schemas nomeados: o gerador resolve as referências, lê os dois jeitos de escrever um valor anulável (`["string","null"]` e `anyOf` com `null`) e tira o prefixo `/api` dos caminhos (a URL base do cliente é a do app, `https://meeng.dev`).
- `generator/surface.go` e `generator/webhooks.go` existem porque o contrato **não descreve** os nomes que os SDKs dão aos métodos (o `operationId` do app é o do roteador dele) nem os webhooks. O primeiro dá o nome de cada método (`emails.send`, `contacts.bulkCreate`…), os apelidos (`emails.create`, `campaigns.getAll`) e o nome dos modelos de resposta (`Domain`, `Contact`, `ContactBook`, `Campaign`, `CampaignRow`, `Email`); o segundo descreve os 21 eventos de webhook e seus dados (o envelope foi conferido contra `WebhookCalls.go` do app). Uma rota do contrato que o `surface.go` não nomeia, ou uma do `surface.go` que o contrato não tem, **para a geração**: nenhum SDK perde ou renomeia uma rota em silêncio.
- Uma lista que o contrato escreve como schema que se refere a si mesmo (`DomainList` é `{"$ref": ".../DomainList"}`, o que não diz forma nenhuma) é lida como a lista do item que o `surface.go` nomeia em `Response`; qualquer outra referência a si mesma para a geração. Isso é um defeito do contrato do app (a exportação do `arandu-swagger` para `[]T`), não uma decisão: quando o app escrever `{"type": "array", "items": ...}` o gerador lê do mesmo jeito.
- `fakeapi/` e `cmd/fakeapi` são uma **API de mentira feita só do contrato**, sem usar o código do gerador (para não repetirem o mesmo erro): responde cada operação com um corpo que satisfaz o schema, recusa rota, verbo, filtro, cabeçalho ou campo que o contrato não tem, guarda a chave de idempotência (a mesma com outro corpo é 409 `NOT_UNIQUE`), pagina as listas, e um parâmetro de caminho `fake-<status>` ou `flaky-<status>-<n>` escolhe a resposta. `generator/cases.go` e `generator/calls_*.go` descrevem uma vez a chamada de cada método e a escrevem na sintaxe de cada linguagem; é daí que saem o `contract_test` de cada SDK.
- `<linguagem>/examples/` (`Examples/` no Swift) são programas que rodam: enviar, lote, agendar, domínio e webhook. Os modelos estão em `generator/templates/<linguagem>/examples/`; os de Go são `.go.tmpl` para o `go vet ./...` da raiz não os compilar como parte deste módulo.
- `skills/` são as **skills** para assistentes de código (uma pasta por skill, formato do `perata-sh/sdk/skills`), escritas à mão em português. Os exemplos delas são programas inteiros que o gate compila contra estes SDKs; veja "The gates" e `skills/README.md`.
- `Package.swift` e `composer.json` na raiz são escritos à mão (SwiftPM e Packagist leem a raiz do repositório); mantenha-os iguais a `swift/Package.swift` e `php/composer.json`.

## The contract

`openapi.json` é o **contrato do app meeng.dev**, `docs/contracts/api.openapi.json`, que o `aru openapi:check` do app mantém igual às rotas reais (a rota é a fonte; [[GAP-openapi-source]]), sem mais nada: servidor `https://meeng.dev`, caminhos `/api/v1/...`. O `regen.sh` o copia (`./regen.sh` do `origin/develop` do app, depois de buscá-lo; `./regen.sh <ref>`; `./regen.sh --worktree`) e escreve os 11 SDKs; depois, `./check.sh`. **Nunca edite o `openapi.json` à mão.**

`generator.SanitizeContract` roda na cópia: tira da `info.description` a frase que nomeia outro produto e **recusa** o contrato se algum outro lugar o nomeia (isso se corrige no app, não aqui). O padrão dos nomes é montado em pedaços no código, para que nenhum arquivo deste repositório os escreva. Se o contrato trouxer rota, campo ou tipo novo, o gerador para até alguém nomeá-lo em `surface.go`.

Diferenças deliberadas entre os SDKs e o contrato escrito, todas vindas do código do app:
- o erro é `{error: {code, message}}`; também se lê `{error: "texto"}` (o `openapi.json` declara assim os 403 e 404, e o `PUT /domains/{id}/verify` responde assim) e o erro de validação do zod, `{success: false, error: {issues, name}}`, que vira o código `ZodError` e a primeira falha como mensagem (`from: Required`);
- `to`, `cc`, `bcc` e `replyTo` aceitam texto ou lista de textos na API: os SDKs enviam a lista e leem um texto solto como lista de um;
- as rotas `POST`/`PUT` sem corpo mandam `{}`;
- o `Idempotency-Key` só vale em `emails.send` e `emails.batch`, só é enviado quando o chamador o dá (nenhum SDK inventa uma chave) e a API o guarda por 24 horas;
- `page` e `limit` são inteiros nos SDKs mesmo onde o contrato os escreve como texto; `domainId` de `emails.list` é uma lista, enviada como um `domainId` por valor.

## Names the SDKs share

- **Cliente:** `Meeng` (Go `meeng.New`, Java `new Meeng`, C# `MeengClient`, Ruby `Meeng::Client`, Rust `meeng::Client`, Swift `MeengClient`, Elixir `Meeng.new`); TypeScript `new Meeng(apiKey?, baseUrl?)`.
- **Recursos:** `emails` (`send`, `batch`, `get`, `list`, `update`, `cancel`), `domains`, `contacts`, `contactBooks`, `campaigns`, `analytics` e `webhooks`. O TypeScript devolve `{data, error}`; o Python devolve `(data, error)` e levanta `MeengHTTPError` por padrão (`raise_on_error`); as demais linguagens usam o jeito delas (exceção, `Result`, erro de retorno).
- **Ambiente:** `MEENG_API_KEY` e `MEENG_BASE_URL`. A URL base é a do **app** (`https://meeng.dev`; a API fica em `https://meeng.dev/api`), sem `/api/v1`: o cliente acrescenta o prefixo `/api` do contrato e os caminhos `/v1/...`. Só `https` é aceito; `http` vale para `localhost`, `127.0.0.0/8` e `::1`, para a chave nunca viajar em texto puro. Nenhum cliente segue redirecionamento.
- **Webhooks:** `meeng.webhooks(secret)` com `verify` e `constructEvent` (a grafia de cada linguagem). O esquema é: `v1=` mais o HMAC-SHA256 em hexadecimal de `"{timestamp}.{corpo}"` com o segredo `whsec_…` como chave literal; o timestamp em milissegundos; tolerância de 5 minutos (negativa desliga); códigos de erro `MISSING_SIGNATURE`, `MISSING_TIMESTAMP`, `INVALID_SIGNATURE_FORMAT`, `INVALID_TIMESTAMP`, `TIMESTAMP_OUT_OF_RANGE`, `SIGNATURE_MISMATCH`, `INVALID_BODY` e `INVALID_JSON`. Os cabeçalhos são `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`, lidos em qualquer caixa e **só esses**. O envelope é `{id, type, version, createdAt, teamId, data, attempt}`; o `id` é o da chamada e vale para deduplicar. Mude o verificador e o app (`WebhookSignature.go`) juntos, ou nenhum.
- **Tentativas:** todo cliente repete a chamada que pode ser repetida, igual nas onze linguagens: `maxRetries` (2; 0 envia uma vez só) e `retryDelay` (500 ms, dobrando, no máximo 30 s, trocado pelo `Retry-After` da API). Um 429 repete para qualquer chamada, porque a chamada não foi processada; 500, 502, 503, 504 ou falha de conexão repetem só leitura (`GET`), `PUT`, `DELETE` ou chamada com `Idempotency-Key`; 409 e o resto nunca. Quatro testes por linguagem dizem isso, e o teste de contrato o prova por soquete de verdade (`flaky-429-2`).

## The gates

`./check.sh` roda todos; nada está pronto até ele sair com zero. Cada passo imprime `== <linguagem>`.
- gerador: `gofmt`, `go vet`, `go test -race` (o contrato lido, a superfície, o texto hostil, o determinismo, a API de mentira, as chamadas dos testes de contrato) e `go run ./cmd/sdkgen`;
- depois do gerador o `check.sh` sobe a **API de mentira** (`cmd/fakeapi`, só do contrato) e exporta `MEENG_FAKE_URL`. Em cada linguagem: os testes, o **teste de contrato** (chama os 33 métodos, mais erro tipado, 429 repetido até passar, 503 repetido numa leitura, chave de idempotência e campo obrigatório), e então `fakeapi -coverage`, que **falha se alguma operação do contrato ficou sem chamada**; um método que falta ou um teste pulado quebra o gate. Depois os **exemplos** rodam contra a mesma API, e ela é zerada;
- go: `go vet` e `go test -race`;
- typescript: `tsc --noEmit` do código, dos testes e dos exemplos, e `node --test`;
- python: `unittest` com `-W error` e `mypy --strict` em 3.11 (código e exemplos);
- php: `tests/run.php` (inclui o contrato) e PHPStan nível 8 (código e exemplos);
- ruby: minitest sob `ruby -w` (todos os `test/*_test.rb`);
- java: `mvn verify`;
- csharp: `dotnet test` (net8.0 e net10.0 compilam, com aviso como erro) e o projeto `examples/`;
- elixir: `mix compile` e `mix test`, com `--warnings-as-errors`;
- dart: `dart format --language-version=3.8 --set-exit-if-changed`, `dart analyze --fatal-infos` e `dart test`;
- rust: `cargo fmt --check`, `cargo clippy -D warnings` e `cargo test` (os exemplos de `examples/` entram no `--all-targets`);
- swift: `swift test` em `swift/`, `swift build` na raiz e `swift build --product MeengExamples` em `swift/` (só macOS: CryptoKit e Network). O alvo `MeengExamples` só existe em `swift/Package.swift`; o `Package.swift` da raiz não o tem, para quem depende do pacote não receber um executável.
- skills: `go run ./cmd/skillscheck` (o passo `== skills`, o último, e um job da CI): o cabeçalho de cada `SKILL.md` só com `name` e `description`, os links relativos, todo arquivo de `referencias/` citado, nenhum nome de outro produto de e-mail, os blocos `json` válidos, e cada exemplo em Go e em TypeScript compilado contra `go/` e `typescript/src` (Python por `compile` e `mypy --strict`, PHP por `php -l`). Um exemplo é um programa inteiro; um trecho vai num bloco `<linguagem> nocheck`, que o relatório conta. Mudou o contrato ou um SDK: rode o passo e corrija a skill no mesmo commit.

Uma ferramenta que falta **para o gate** (sai com erro), a não ser que se passe `SKIP_MISSING=1`, e então ela é nomeada e pulada: uma linguagem que ninguém verificou não está verificada. `MYPY` e `PHPSTAN` apontam essas ferramentas quando não estão no `PATH` (o `PHPSTAN` pode ser um `.phar`; ele roda com `--memory-limit=1G`). `CHECK_DRIFT=1` (na CI) também falha quando um SDK commitado difere do que o gerador escreve. Use `GOWORK=off`: o `go.work` do vault não vale aqui. Use `$(go env GOROOT)/bin/gofmt`.

O `cmd/sdkgen` roda `dart format` sobre o Dart gerado, com versão de linguagem fixa; sem o Dart no `PATH` ele avisa e deixa o Dart como está.

## Rules

- Todo SDK faz a mesma coisa, sob a mesma versão (`VERSION`). Um recurso entra em todas as linguagens, com o mesmo teste em cada uma, ou em nenhuma. Os testes de cada linguagem cobrem os mesmos casos (envio com chave de idempotência, corpo `{}`, lote, filtros com lista, escape de caminho, lista sem objeto em volta, `DELETE` com corpo, `PUT` com id, os formatos de erro, destinatário texto ou lista, ambiente, `https`, redirecionamento, as quatro situações de repetição e as seis situações do webhook), mais o teste de contrato contra a API de mentira.
- Nenhuma dependência de tempo de execução onde a biblioteca padrão faz HTTP, JSON e HMAC. As exceções deliberadas são as do Peráta: Dart usa `http` e `crypto` do dart.dev, Rust usa `serde`, `serde_json`, `ureq`, `hmac` e `sha2`. Não acrescente outras.
- Código, comentários, erros e commits em inglês; os README em português do Brasil. Um commit diz o que mudou e por quê.
- Identidade: `Paulo R. Lima <paulorlima@hyz.is>` (o meeng é produto), no `git config` local, nunca em `--author`; `core.hooksPath` aponta para `/Users/paulorlima9/Developer/arandu-io/.githooks`. Nunca atribuição a IA: sem `Co-Authored-By`, sem rodapé "Generated with", sem emoji de robô.
- `declared` não é `implemented`, e `implemented` não é `tested`: diga o que rodou e o que foi pulado.
- Push, criação de repositório ou organização no GitHub, publicação em registro de pacotes, cadastro em serviço de terceiros e deploy exigem autorização do Paulo. Fluxo develop → main, sem force-push.

## Names

Os nomes públicos são **propostas** até o Paulo confirmar ([[GAP-public-names]]; regra global `public-names-ask-first`). O padrão é o do GitHub, `<organização>/<repositório>`: organização `meeng-dev`, pacote `sdk`.

| Registro | Nome proposto |
|---|---|
| Go | `github.com/meeng-dev/sdk/go` |
| npm | `@meeng-dev/sdk` |
| PyPI, RubyGems, crates.io | `meeng-sdk` |
| Packagist | `meeng-dev/sdk` |
| Maven | `dev.meeng:sdk` |
| NuGet | `MeengDev.Sdk` |
| Hex, pub.dev | `meeng_sdk` |
| Swift (SwiftPM) | `https://github.com/meeng-dev/sdk`, produto `Meeng` |

Nenhum pacote foi publicado, e cada um está marcado para não sair por engano: `"private": true` no npm, o classificador `Private :: Do Not Upload` no PyPI, `publish = false` no Cargo, `publish_to: none` no pub.dev, `allowed_push_host` inválido no RubyGems e `IsPackable` falso no NuGet. Quem publicar tira a marca junto com a confirmação do nome. Nunca renomeie nem acrescente um pacote sem a confirmação do Paulo.

## Release

Uma versão para todas as linguagens. Ainda não há release: a `VERSION` é `0.1.0`, o primeiro conjunto. Quando houver autorização: suba o `VERSION`, rode `./regen.sh` e `./check.sh`, faça o commit no `develop` e avance o `main` depois da CI; tag `vX.Y.Z` e `go/vX.Y.Z` no mesmo commit. O fluxo de publicação por registro (confiança por OIDC onde houver) será escrito quando os nomes forem confirmados.
