# SDKs por linguagem e a API REST

Os SDKs do Meeng.dev são onze, todos com os mesmos recursos (`emails`, `domains`, `contacts`, `contactBooks`, `campaigns`, `analytics` e `webhooks`), a mesma versão e o mesmo formato de corpo e de erro. Nenhum depende de pacote de terceiros para falar HTTP, JSON ou assinar webhook (Dart e Rust usam o que a comunidade deles usa).

> **Estado dos pacotes.** Os pacotes ainda **não foram publicados** nos registros; os nomes abaixo são os propostos, e só valem depois que a equipe do Meeng.dev publicar. Enquanto isso, use a API REST (abaixo) ou o código do repositório `github.com/meeng-dev/sdk`. Antes de escrever uma instalação no projeto do dono, confira se o pacote existe no registro.

| Linguagem | Pacote proposto | Cliente |
|---|---|---|
| TypeScript e JavaScript (Node 20.3 ou mais novo, Bun, Deno) | `@meeng-dev/sdk` (npm) | `new Meeng()` |
| Go 1.23 ou mais novo | `github.com/meeng-dev/sdk/go` | `meeng.New("")` |
| Python 3.11 ou mais novo | `meeng-sdk` (PyPI) | `Meeng()` |
| PHP 8.1 ou mais novo | `meeng-dev/sdk` (Packagist) | `new Meeng()` |
| Ruby | `meeng-sdk` (RubyGems) | `Meeng::Client.new` |
| Java 17 ou mais novo | `dev.meeng:sdk` (Maven) | `new Meeng()` |
| C# e .NET | `MeengDev.Sdk` (NuGet) | `new MeengClient()` |
| Elixir | `meeng_sdk` (Hex) | `Meeng.new` |
| Dart e Flutter | `meeng_sdk` (pub.dev) | `Meeng()` |
| Rust | `meeng-sdk` (crates.io) | `Client::from_env()` |
| Swift | `https://github.com/meeng-dev/sdk` (SwiftPM, produto `Meeng`) | `MeengClient()` |

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
