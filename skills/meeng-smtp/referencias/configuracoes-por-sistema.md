# Configuração por sistema

Em todos os casos os dados são os mesmos: servidor `smtp.meeng.dev`, usuário `meeng`, senha igual à **chave de API** (`mg_...`), porta `587` com STARTTLS (ou `465` com TLS implícito), autenticação ligada. O endereço do remetente (`From`) é de um domínio **verificado** da conta.

A senha vem sempre de variável de ambiente ou do cofre de segredos do sistema; os exemplos usam um marcador (`SUA_CHAVE_AQUI`) que você troca pelo mecanismo de segredo do projeto, nunca pela chave colada no arquivo versionado.

## WordPress

Instale um plugin de SMTP (WP Mail SMTP, FluentSMTP ou parecido) e escolha a opção **outro SMTP** (SMTP genérico):

| Campo do plugin | Valor |
|---|---|
| Servidor SMTP | `smtp.meeng.dev` |
| Criptografia | TLS (STARTTLS) com a porta `587`, ou SSL com a porta `465` |
| Porta | `587` |
| Autenticação | ligada |
| Usuário | `meeng` |
| Senha | a chave de API |
| E-mail do remetente | um endereço do domínio verificado, e marque a opção de forçar o remetente |

Se o plugin guarda a senha no banco, ele tem uma opção de defini-la em `wp-config.php` (constante); prefira essa.

## Laravel

```ini
MAIL_MAILER=smtp
MAIL_HOST=smtp.meeng.dev
MAIL_PORT=587
MAIL_USERNAME=meeng
MAIL_PASSWORD=SUA_CHAVE_AQUI
MAIL_FROM_ADDRESS=pedidos@envio.exemplo.com.br
MAIL_FROM_NAME="Loja Exemplo"
```

- Laravel 10 e anteriores: acrescente `MAIL_ENCRYPTION=tls` (STARTTLS na 587). Para a porta 465, `MAIL_ENCRYPTION=ssl`.
- Laravel 11 e posteriores: com a porta 587 nada mais é preciso (o STARTTLS é usado quando o servidor oferece); para a porta 465, `MAIL_SCHEME=smtps`.

## Django

```python nocheck
EMAIL_BACKEND = "django.core.mail.backends.smtp.EmailBackend"
EMAIL_HOST = "smtp.meeng.dev"
EMAIL_PORT = 587
EMAIL_USE_TLS = True  # STARTTLS; na porta 465 use EMAIL_USE_SSL = True e tire o EMAIL_USE_TLS
EMAIL_HOST_USER = "meeng"
EMAIL_HOST_PASSWORD = os.environ["MEENG_API_KEY"]
DEFAULT_FROM_EMAIL = "Loja Exemplo <pedidos@envio.exemplo.com.br>"
SERVER_EMAIL = DEFAULT_FROM_EMAIL
```

## Ruby on Rails

```ruby
# config/environments/production.rb
config.action_mailer.delivery_method = :smtp
config.action_mailer.smtp_settings = {
  address: "smtp.meeng.dev",
  port: 587,
  user_name: "meeng",
  password: ENV.fetch("MEENG_API_KEY"),
  authentication: :plain,
  enable_starttls_auto: true
}
config.action_mailer.default_options = { from: "Loja Exemplo <pedidos@envio.exemplo.com.br>" }
```

Para a porta 465, troque `enable_starttls_auto` por `tls: true`.

## Node.js (Nodemailer)

```ts nocheck
import nodemailer from "nodemailer";

const transporte = nodemailer.createTransport({
  host: "smtp.meeng.dev",
  port: 587,
  secure: false, // STARTTLS: a conexão sobe para TLS; na porta 465, secure: true
  requireTLS: true, // nunca envie a senha sem criptografia
  auth: { user: "meeng", pass: process.env.MEENG_API_KEY },
});

await transporte.sendMail({
  from: "Loja Exemplo <pedidos@envio.exemplo.com.br>",
  to: "cliente@example.com",
  subject: "Pedido 1042 confirmado",
  text: "Recebemos o seu pedido 1042.",
});
```

## PHP (PHPMailer)

```php nocheck
<?php

use PHPMailer\PHPMailer\PHPMailer;

$mail = new PHPMailer(true);
$mail->isSMTP();
$mail->Host = 'smtp.meeng.dev';
$mail->SMTPAuth = true;
$mail->Username = 'meeng';
$mail->Password = (string) getenv('MEENG_API_KEY');
$mail->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS; // na porta 465: ENCRYPTION_SMTPS
$mail->Port = 587;
$mail->CharSet = 'UTF-8';

$mail->setFrom('pedidos@envio.exemplo.com.br', 'Loja Exemplo');
$mail->addAddress('cliente@example.com');
$mail->Subject = 'Pedido 1042 confirmado';
$mail->Body = 'Recebemos o seu pedido 1042.';
$mail->send();
```

## Java (Spring Boot)

```ini
spring.mail.host=smtp.meeng.dev
spring.mail.port=587
spring.mail.username=meeng
spring.mail.password=${MEENG_API_KEY}
spring.mail.properties.mail.smtp.auth=true
spring.mail.properties.mail.smtp.starttls.enable=true
spring.mail.properties.mail.smtp.starttls.required=true
```

Para a porta 465, use `spring.mail.port=465` e `spring.mail.properties.mail.smtp.ssl.enable=true` no lugar das linhas de `starttls`.

## C# e .NET (MailKit)

```csharp
using MailKit.Net.Smtp;
using MailKit.Security;
using MimeKit;

var message = new MimeMessage();
message.From.Add(new MailboxAddress("Loja Exemplo", "pedidos@envio.exemplo.com.br"));
message.To.Add(new MailboxAddress("", "cliente@example.com"));
message.Subject = "Pedido 1042 confirmado";
message.Body = new TextPart("plain") { Text = "Recebemos o seu pedido 1042." };

using var client = new SmtpClient();
await client.ConnectAsync("smtp.meeng.dev", 587, SecureSocketOptions.StartTls); // 465: SslOnConnect
await client.AuthenticateAsync("meeng", Environment.GetEnvironmentVariable("MEENG_API_KEY"));
await client.SendAsync(message);
await client.DisconnectAsync(true);
```

## Postfix, Exim e outros servidores de e-mail do próprio servidor

Para um servidor que envia e-mail do sistema (alertas do `cron`, relatórios) por meio do Meeng.dev, configure o Postfix como cliente de relay:

```ini
# /etc/postfix/main.cf
relayhost = [smtp.meeng.dev]:587
smtp_sasl_auth_enable = yes
smtp_sasl_password_maps = hash:/etc/postfix/sasl_passwd
smtp_sasl_security_options = noanonymous
smtp_tls_security_level = encrypt
smtp_tls_CAfile = /etc/ssl/certs/ca-certificates.crt
```

```ini
# /etc/postfix/sasl_passwd (permissão 600; depois, postmap /etc/postfix/sasl_passwd)
[smtp.meeng.dev]:587 meeng:SUA_CHAVE_AQUI
```

O remetente dos e-mails do servidor precisa ser de um domínio verificado: ajuste `myorigin` e use `smtp_generic_maps` para reescrever `root@servidor` para um endereço do domínio verificado.

## Impressoras, scanners, câmeras e sistemas fechados

Esses aparelhos têm uma tela "servidor de e-mail (SMTP)" com os mesmos campos: servidor, porta, usuário, senha, tipo de segurança (escolha **STARTTLS** com a 587, ou **SSL/TLS** com a 465) e o endereço do remetente. Se o aparelho só sabe a porta 25 ou não tem criptografia, ele não consegue logar: use um servidor intermediário (o Postfix acima) ou troque a forma de aviso.
