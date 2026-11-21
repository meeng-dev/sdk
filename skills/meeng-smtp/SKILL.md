---
name: meeng-smtp
description: Ensina a enviar e-mail pelo relay SMTP do Meeng.dev, a plataforma brasileira de e-mail para desenvolvedores, em sistemas que só falam SMTP (WordPress, Laravel, Django, Rails, impressoras, ERPs, scripts, bibliotecas de e-mail). Use quando o pedido for configurar o servidor SMTP de um sistema para o Meeng.dev (host smtp.meeng.dev, portas 587 e 465, usuário meeng e a chave de API como senha), testar a conexão, trocar o SMTP de outro provedor, ou entender por que o login SMTP, o TLS ou o remetente foi recusado. O SMTP é só a reserva: o envio de um sistema é pela API ou por um SDK (meeng-enviar-email), e esta skill vale para o programa que não chama API. Cobre conexão, autenticação, exemplos e erros.
---

# Enviar por SMTP no Meeng.dev

SMTP é o protocolo de e-mail mais antigo e o que quase todo sistema já sabe usar. Quando o sistema do dono não tem como chamar uma API (um WordPress com um plugin de e-mail, um ERP, uma impressora que manda alerta, uma biblioteca de e-mail pronta), ele pode enviar pelo relay SMTP do Meeng.dev: a conta é a mesma, o domínio é o mesmo, e cada e-mail aparece no painel, com status e eventos, como se tivesse sido enviado pela API.

Fale com o dono em português simples. Explique em uma frase o que é "relay SMTP" (um servidor que recebe o e-mail do seu sistema e o entrega ao destinatário) e "porta" (o número da porta de entrada do servidor; cada uma tem um jeito de proteger a conexão). Pergunte só o que falta: qual sistema vai enviar e se o domínio dele já está verificado.

## O SMTP é a reserva, a API é o caminho principal

O relay SMTP está ligado em `smtp.meeng.dev`, nas portas `587` (STARTTLS) e `465` (TLS), e a tela **Configurações › SMTP** do painel (`https://meeng.dev/configuracoes/smtp`) traz o servidor, as portas e o usuário. Mesmo assim, ele existe para o que **não consegue chamar a API**. Para um sistema que dá para alterar, o caminho principal é a API ou um SDK (`meeng-enviar-email`: Go e Node primeiro): devolve o `emailId`, aceita `Idempotency-Key`, agendamento e modelos, e responde com erro estruturado. Um app Arandu nem precisa de SMTP: `MAIL_URL=meeng://<chave>@meeng.dev` envia pela API. Nunca invente host ou porta que a tela não mostra.

## Quando usar

Use quando o sistema só sabe falar SMTP, ou quando trocar a chamada de API por SMTP é mais barato que mexer no código. Quando o sistema é seu e dá para chamar a API, prefira a API: ela devolve o `emailId`, aceita `Idempotency-Key`, agendamento e modelos, e dá erro estruturado. O Meeng.dev não envia anexos por nenhum caminho: uma mensagem SMTP com anexo é recusada com `550 5.7.1`, e o arquivo vai como link no corpo.

Para o domínio (precisa estar verificado), `meeng-dominios`. Para saber a entrega, `meeng-webhooks`.

## Os dados da conexão

| Campo | Valor |
|---|---|
| Servidor (host) | `smtp.meeng.dev` |
| Usuário | `meeng` |
| Senha | **a chave de API** (`mg_...`) |
| Porta com STARTTLS | `587` (a recomendada) |
| Porta com TLS implícito | `465` |
| Autenticação | `PLAIN` ou `LOGIN`, **só depois** de a conexão estar criptografada |

- **STARTTLS** (587): a conexão começa aberta e **sobe** para criptografada antes do login. **TLS implícito** (465): já começa criptografada. As duas são seguras; use a que o sistema do dono oferecer. Nunca use uma porta sem criptografia para enviar a senha: o login é recusado antes do TLS.
- Só a `587` e a `465` respondem para envio. Não use a `25` nem outra porta: a `25` é a de servidor para servidor, e a hospedagem do dono costuma bloqueá-la.
- A **senha é a chave de API**: valem as mesmas regras de chave. Uma chave restrita a um domínio só envia com `from` desse domínio.

## Regras que valem sempre

1. A chave de API é a senha. Guarde-a como segredo do sistema (variável de ambiente, cofre de segredos do hospedeiro), nunca no código nem no repositório. Se vazou, apague a chave no painel e crie outra.
2. Sempre com criptografia: STARTTLS na 587 ou TLS na 465. Nunca desligue a verificação do certificado para "fazer funcionar".
3. O endereço do remetente (`From`) precisa ser de um **domínio verificado** da conta; senão o e-mail é recusado. É a mesma regra da API.
4. As regras do envio valem iguais às da API: destinatário na lista de supressão não recebe, o limite do plano vale (no plano Grátis, só para endereços da própria organização), e o limite de requisições por segundo da organização também. Em lote, envie com uma pausa curta entre os e-mails, ou use a API de lote.
5. O SMTP não tem `Idempotency-Key`: se o sistema repetir o envio depois de um erro de rede, o e-mail pode sair duas vezes. Para mensagens que não podem duplicar, use a API.
6. Confira o resultado no painel (`https://meeng.dev/painel/emails`): cada e-mail enviado por SMTP aparece com os eventos (`QUEUED`, `SENT`, `DELIVERED`...). Um "ok" do SMTP quer dizer só "aceito na fila".
7. Não envie e-mail de divulgação em massa por SMTP: para isso use as campanhas (`meeng-contatos-e-campanhas`), que trazem descadastro e lotes.

## Roteiro essencial

1. Conferir que a API realmente não serve (se o sistema pode chamar a API, use `meeng-enviar-email`) e que o domínio está verificado.
2. Criar uma chave de API própria para este sistema (de preferência restrita ao domínio dele).
3. Configurar o sistema com os dados da tabela acima ([`referencias/configuracoes-por-sistema.md`](referencias/configuracoes-por-sistema.md)).
4. Testar a conexão e mandar um e-mail de teste para o próprio dono ([`referencias/testar-e-erros.md`](referencias/testar-e-erros.md)).
5. Conferir o e-mail no painel e na caixa de entrada (e no spam), e ver os cabeçalhos de autenticação.

Enviar por SMTP com a biblioteca padrão de Go (STARTTLS na porta 587):

```go
package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

func main() {
	const host, port = "smtp.meeng.dev", "587"
	auth := smtp.PlainAuth("", "meeng", os.Getenv("MEENG_API_KEY"), host) // a senha é a chave de API

	from := "pedidos@envio.exemplo.com.br" // de um domínio verificado
	to := []string{"cliente@example.com"}
	message := strings.Join([]string{
		"From: Loja Exemplo <" + from + ">",
		"To: " + strings.Join(to, ", "),
		"Subject: Pedido 1042 confirmado",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Recebemos o seu pedido 1042.",
	}, "\r\n")

	// SendMail sobe a conexão para TLS (STARTTLS) antes de enviar a senha, e recusa
	// enviá-la por uma conexão sem criptografia.
	if err := smtp.SendMail(host+":"+port, auth, from, to, []byte(message)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("aceito pelo relay; confira o status no painel")
}
```

E com `smtplib`, a biblioteca padrão de Python:

```python
import os
import smtplib
from email.message import EmailMessage

mensagem = EmailMessage()
mensagem["From"] = "Loja Exemplo <pedidos@envio.exemplo.com.br>"  # de um domínio verificado
mensagem["To"] = "cliente@example.com"
mensagem["Subject"] = "Pedido 1042 confirmado"
mensagem.set_content("Recebemos o seu pedido 1042.")

with smtplib.SMTP("smtp.meeng.dev", 587, timeout=30) as servidor:
    servidor.starttls()  # sobe a conexão para TLS antes do login
    servidor.login("meeng", os.environ["MEENG_API_KEY"])  # a senha é a chave de API
    servidor.send_message(mensagem)
```

## Mapa das referências

| Quando você precisa de | Abra |
|---|---|
| Os campos de configuração de WordPress, Laravel, Django, Rails, Node, PHP e outros sistemas | [`referencias/configuracoes-por-sistema.md`](referencias/configuracoes-por-sistema.md) |
| Testar a conexão, ler as respostas do servidor, erros de login, TLS e remetente | [`referencias/testar-e-erros.md`](referencias/testar-e-erros.md) |

## Ao terminar

Entregue ao dono, em português: qual sistema ficou configurado e com que porta, que a chave usada é própria e restrita ao domínio, o resultado do e-mail de teste (status no painel e caixa de entrada), e o que falta (domínio verificado, webhook de entrega, trocar a chave se vazou).
