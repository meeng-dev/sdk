# SDKs por linguagem e a API REST

Os SDKs do Meeng.dev são onze (dez já instaláveis), todos com os mesmos recursos (`emails`, `domains`, `contacts`, `contactBooks`, `campaigns`, `analytics` e `webhooks`), a mesma versão e o mesmo formato de corpo e de erro. Nenhum depende de pacote de terceiros para falar HTTP, JSON ou assinar webhook (Dart e Rust usam o que a comunidade deles usa).

> **Estado dos pacotes (21/11/2026).** A versão `0.1.0` está publicada em: npm, Go (tag `go/v0.1.0`), PyPI, RubyGems, crates.io, Hex, pub.dev, NuGet e Packagist; o Swift lê a tag `v0.1.0` do repositório. **Java (Maven Central) ainda não saiu**: para Java use a API REST (abaixo) ou o código do repositório `github.com/meeng-dev/sdk`. Antes de escrever uma instalação no projeto do dono, confira a versão no registro; para uma linguagem que esta tabela não confirma, não prometa o pacote.

| Linguagem | Pacote | Instalar | Cliente |
|---|---|---|---|
| TypeScript e JavaScript (Node 20.3 ou mais novo, Bun, Deno) | `@meeng-dev/sdk` (npm) | `npm install @meeng-dev/sdk` | `new Meeng()` |
| Go 1.23 ou mais novo | `github.com/meeng-dev/sdk/go` | `go get github.com/meeng-dev/sdk/go` | `meeng.New("")` |
| Python 3.11 ou mais novo | `meeng-sdk` (PyPI) | `pip install meeng-sdk` | `Meeng()` |
| PHP 8.1 ou mais novo | `meeng-dev/sdk` (Packagist) | `composer require meeng-dev/sdk` | `new Meeng()` |
| Ruby | `meeng-sdk` (RubyGems) | `gem install meeng-sdk` | `Meeng::Client.new` |
| C# e .NET | `MeengDev.Sdk` (NuGet) | `dotnet add package MeengDev.Sdk` | `new MeengClient()` |
| Elixir | `meeng_sdk` (Hex) | `{:meeng_sdk, "~> 0.1"}` no `mix.exs` | `Meeng.new` |
| Dart e Flutter | `meeng_sdk` (pub.dev) | `dart pub add meeng_sdk` | `Meeng()` |
| Rust | `meeng-sdk` (crates.io) | `cargo add meeng-sdk` | `Client::from_env()` |
| Swift | `https://github.com/meeng-dev/sdk` (SwiftPM, produto `Meeng`) | pacote SwiftPM pela URL | `MeengClient()` |
| Java 17 ou mais novo | `dev.meeng:sdk` (Maven), **ainda não publicado** | | `new Meeng()` |

Em todos: a chave vem de `MEENG_API_KEY` e o endereço do app de `MEENG_BASE_URL` (opcional: o padrão já é `https://meeng.dev`, sem `/api/v1`), a menos que você os passe ao construir o cliente. Só `https` é aceito; `http` vale para `localhost`, `127.0.0.0/8` e `::1`. Nenhum cliente segue redirecionamento, para a chave nunca ir parar em outro endereço.

O envio, nas linguagens que as outras skills não mostram. Estes trechos vêm dos README dos SDKs; a chave `mg_123` é só um exemplo, use a variável de ambiente.

```ruby
require "meeng"

meeng = Meeng::Client.new # lê MEENG_API_KEY e MEENG_BASE_URL

sent = meeng.emails.send(
  from: "pedidos@envio.exemplo.com.br",
  to: "cliente@example.com",
  subject: "Pedido 1042 confirmado",
  html: "<p>Recebemos o seu pedido 1042.</p>"
)
puts sent[:emailId]
```

```java
import dev.meeng.*;
import java.util.List;

Meeng meeng = new Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL

EmailsSendResponse sent = meeng.emails().send(
    EmailsService.SendParams.builder()
        .from("pedidos@envio.exemplo.com.br")
        .to(List.of("cliente@example.com"))
        .subject("Pedido 1042 confirmado")
        .html("<p>Recebemos o seu pedido 1042.</p>")
        .build());
System.out.println(sent.getEmailId());
```

```csharp
using Meeng;

var meeng = new MeengClient(); // lê MEENG_API_KEY e MEENG_BASE_URL

var sent = await meeng.Emails.SendAsync(new EmailsSendParams
{
    From = "pedidos@envio.exemplo.com.br",
    To = ["cliente@example.com"],
    Subject = "Pedido 1042 confirmado",
    Html = "<p>Recebemos o seu pedido 1042.</p>",
});
Console.WriteLine(sent.EmailId);
```

```elixir
client = Meeng.new() # lê MEENG_API_KEY e MEENG_BASE_URL

{:ok, sent} =
  Meeng.Emails.send(client, %{
    from: "pedidos@envio.exemplo.com.br",
    to: "cliente@example.com",
    subject: "Pedido 1042 confirmado",
    html: "<p>Recebemos o seu pedido 1042.</p>"
  })

sent["emailId"]
```

```dart
import 'package:meeng_sdk/meeng_sdk.dart';

Future<void> main() async {
  final meeng = Meeng(); // lê MEENG_API_KEY e MEENG_BASE_URL
  try {
    final sent = await meeng.emails.send(
      const EmailsSendParams(
        from: 'pedidos@envio.exemplo.com.br',
        to: ['cliente@example.com'],
        subject: 'Pedido 1042 confirmado',
        html: '<p>Recebemos o seu pedido 1042.</p>',
      ),
    );
    print(sent.emailId);
  } on MeengException catch (error) {
    print(error);
  } finally {
    meeng.close();
  }
}
```

```rust
use meeng::{Client, EmailsSendParams};

fn main() -> Result<(), meeng::Error> {
    let client = Client::from_env()?; // MEENG_API_KEY e MEENG_BASE_URL

    let sent = client.emails().send(
        &EmailsSendParams::new(vec!["cliente@example.com".to_owned()], "pedidos@envio.exemplo.com.br")
            .subject("Pedido 1042 confirmado")
            .html("<p>Recebemos o seu pedido 1042.</p>"),
    )?;
    println!("{:?}", sent.email_id);
    Ok(())
}
```

```swift
import Meeng

let meeng = MeengClient() // lê MEENG_API_KEY e MEENG_BASE_URL

let sent = try await meeng.emails.send(
    EmailsSendParams(
        to: ["cliente@example.com"],
        from: "pedidos@envio.exemplo.com.br",
        subject: "Pedido 1042 confirmado",
        html: "<p>Recebemos o seu pedido 1042.</p>"
    )
)
print(sent.emailID ?? "")
```

## Bibliotecas de outros provedores: só o endereço base e a chave

O mesmo serviço também responde em **`https://api.meeng.dev`**, sem o prefixo `/api/v1`, no formato das bibliotecas oficiais de envio de e-mail de outros provedores (Node, Python, Go, PHP e outras): caminhos `/emails`, `/emails/batch`, `/domains`, `/audiences`, `/contacts` e `/broadcasts`, corpo em `snake_case` (`reply_to`, `scheduled_at`) e erros no formato `{"statusCode", "name", "message"}`. A **mesma chave `mg_...`** abre os dois endereços, e as regras são as mesmas (domínio verificado, supressão, plano, `Idempotency-Key`).

Para um sistema que já usa uma dessas bibliotecas, a troca é só de configuração, sem adaptador: a chave nova na variável de ambiente que ele já lê, e o endereço base no lugar do que a biblioteca usa (a maioria lê uma variável de ambiente que termina em `_BASE_URL`, ou aceita uma opção `baseUrl` no cliente). Confira o nome certo na documentação da biblioteca do dono; a skill `meeng-migrar-de-outro-provedor` leva o resto da troca. Para código **novo**, use o SDK do Meeng.dev, que cobre toda a API e o `Idempotency-Key`.

```sh
curl -sS https://api.meeng.dev/emails \
  -H "Authorization: Bearer $MEENG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"from":"pedidos@envio.exemplo.com.br","to":["cliente@example.com"],"subject":"Pedido 1042 confirmado","html":"<p>Recebemos o seu pedido 1042.</p>"}'
```

O que a tradução **não** guarda: `tags` e `topic_id` são aceitos e ignorados, e anexos não existem (link no corpo, como em todo o Meeng.dev).

## App Arandu: `MAIL_URL`

Um app Arandu (framework `v0.72.0` ou mais novo) liga o envio sem código: no `.env`, `MAIL_URL=meeng://<chave>@meeng.dev`, com a chave `mg_...` no lugar de `<chave>` e o host `meeng.dev` (o framework chama `POST /api/v1/emails`). A chave vai só no `.env` ou no segredo de produção, nunca no repositório. Em produção o framework recusa subir com `MAIL_URL=log://` ou sem `MAIL_URL`, de propósito. Um host diferente (um ambiente de teste) é `meeng://<chave>@host`, e uma porta vale (`@127.0.0.1:8443`).

## Sem SDK: a API REST

Qualquer linguagem fala com a API por HTTPS e JSON. O endereço é `https://meeng.dev/api/v1`; o cabeçalho é `Authorization: Bearer $MEENG_API_KEY`; e o corpo é o mesmo que os SDKs montam.

```sh
curl -sS https://meeng.dev/api/v1/emails \
  -H "Authorization: Bearer $MEENG_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: pedido-1042-confirmacao" \
  -d '{
    "from": "pedidos@envio.exemplo.com.br",
    "to": "cliente@example.com",
    "subject": "Pedido 1042 confirmado",
    "html": "<p>Recebemos o seu pedido 1042.</p>",
    "text": "Recebemos o seu pedido 1042."
  }'
```

A resposta é `{"emailId": "..."}`. O contrato completo, em OpenAPI, fica em `https://meeng.dev/api/v1/doc` e a página interativa em `https://meeng.dev/api/v1/ui`. Não exponha a chave: a chamada acima roda no servidor ou na máquina do desenvolvedor, nunca no navegador do cliente final.
