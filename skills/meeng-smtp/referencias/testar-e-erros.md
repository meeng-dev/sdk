# Testar a conexão e entender os erros

## Antes de testar

1. A tela **Configurações › SMTP** do painel traz o servidor, as portas e o usuário (o relay está ligado).
2. Existe um domínio **verificado** na conta (`meeng-dominios`) e o `From` do teste é dele.
3. A chave de API existe e está completa (`mg_...`, sem espaço no começo ou no fim).

## Testar do terminal

Ferramenta `swaks` (um cliente de teste de SMTP, disponível nos gerenciadores de pacote):

```sh
swaks --server smtp.meeng.dev --port 587 --tls \
  --auth PLAIN --auth-user meeng --auth-password "$MEENG_API_KEY" \
  --from pedidos@envio.exemplo.com.br --to voce@example.com \
  --header "Subject: Teste do Meeng.dev" --body "Se você leu isto, o SMTP funciona."
```

A saída mostra a conversa linha a linha. O que se espera, nesta ordem: `220` (o servidor atende), `250` com a lista de recursos, `220 ... Ready to start TLS`, `235` (login aceito), `250` no remetente e no destinatário, e `250` no fim dos dados (mensagem aceita).

Para testar só a porta e o TLS, sem login:

```sh
openssl s_client -starttls smtp -connect smtp.meeng.dev:587 -crlf -quiet
```

Depois do certificado, digite `EHLO teste` e veja a lista de recursos (entre eles `AUTH`, que só aparece depois do TLS). `QUIT` sai.

Para a porta com TLS implícito: `openssl s_client -connect smtp.meeng.dev:465 -crlf -quiet`.

## Conferir o resultado

1. Abra **E-mails** no painel (`https://meeng.dev/painel/emails`): o e-mail do teste aparece com os eventos. `QUEUED` e `SENT` mostram que saiu; `DELIVERED` mostra que o servidor do destinatário aceitou.
2. Veja a caixa de entrada **e o spam**. Abra "mostrar original" e procure `dkim=pass`, `spf=pass` e `dmarc=pass` no cabeçalho `Authentication-Results`.

## O que o servidor responde

O SMTP responde com um código de três dígitos. A regra geral: começa com **2** deu certo; com **4** é uma falha temporária (tente de novo depois); com **5** é permanente (corrija antes de tentar).

| Código | Quer dizer | Causa provável aqui |
|---|---|---|
| `220` | o servidor atende | normal |
| `235` | login aceito | normal |
| `250` | comando aceito, ou mensagem aceita no fim dos dados | normal |
| `421` | serviço indisponível agora, ou conexão encerrada | instante de sobrecarga ou limite de conexões; espere e tente de novo |
| `451`, `452` | falha temporária | tente de novo depois; se repetir por horas, verifique o limite do plano |
| `530` | falta autenticação, ou é preciso subir o TLS primeiro | o sistema tentou o login sem STARTTLS; ligue a criptografia (porta 587 com STARTTLS ou 465 com TLS) |
| `535` | login recusado | usuário errado (deve ser `meeng`), chave errada, incompleta, apagada ou com espaço sobrando |
| `550`, `553`, `554` | recusa permanente da mensagem ou do endereço | `From` de domínio que não é da conta ou não verificado; destinatário na lista de supressão; chave restrita a outro domínio |

## Os problemas mais comuns

| Sintoma | O que fazer |
|---|---|
| a conexão não abre (timeout) | a hospedagem bloqueia a porta de saída. A `25` costuma ser bloqueada; use a `587` ou a `465`. Teste com `openssl s_client` de dentro do mesmo servidor |
| `530` ou erro de "starttls" | ligue a criptografia no sistema: STARTTLS na 587 ou SSL/TLS na 465. Não dá para logar sem ela |
| `535` na primeira tentativa | confira o usuário (`meeng`) e a chave inteira. Cole a chave de novo, sem espaço. Se a chave foi apagada no painel, crie outra |
| "certificate verify failed" | o sistema não tem a lista de autoridades de certificado, ou o relógio dele está errado. Instale `ca-certificates` e acerte a hora. Nunca desligue a verificação do certificado |
| o SMTP aceita, mas o e-mail não chega | veja o status no painel: `SUPPRESSED` (endereço bloqueado), `BOUNCED` (o servidor do destinatário recusou) ou `FAILED` (limite do plano). `SENT` sem `DELIVERED` por muito tempo é um adiamento do servidor de destino |
| o e-mail cai no spam | domínio sem DMARC, conteúdo só de imagem, assunto em maiúsculas, `From` que não bate com o domínio verificado. Veja `meeng-dominios` |
| o remetente aparece diferente do que o sistema configurou | alguns sistemas trocam o `From` pelo do usuário de login; configure o remetente no sistema e force o uso |
| cada e-mail do lote demora | o relay respeita o limite de requisições por segundo da organização; envie com pausas, ou use a API de lote |

## Segurança

- A senha do SMTP é a chave de API. Crie uma chave **só para este sistema**, restrita ao domínio dele, e apague-a quando o sistema for desligado.
- Não deixe a chave no código, no repositório nem na tela de configuração de um plugin que mostra a senha em texto.
- Se o sistema é de terceiros (um ERP de um fornecedor), dê a ele uma chave própria: se ele for comprometido, você apaga só aquela.
