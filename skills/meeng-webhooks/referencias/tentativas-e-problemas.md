# Tentativas, repetições e problemas

## Como a entrega funciona

- O Meeng.dev faz um `POST` em JSON no endereço do webhook, com os cabeçalhos `X-Meeng-Signature`, `X-Meeng-Timestamp`, `X-Meeng-Event` e `X-Meeng-Call`, e o `User-Agent` `Meeng-Webhook/1.0`.
- **Sucesso é uma resposta 2xx.** Qualquer outra resposta, ou uma demora de mais de **10 segundos**, é uma falha dessa tentativa. **Redirecionamentos não são seguidos**: um `301` ou `302` conta como falha. Cadastre o endereço final, com `https` e a barra do jeito que o servidor espera.
- Cada aviso tem **até 6 tentativas**, com espera crescente entre elas: 5, 10, 20, 40 e 80 segundos depois da falha anterior. A tentativa vai no campo `attempt` do aviso e no cabeçalho `X-Meeng-Retry` (`true` a partir da segunda).
- O `id` do aviso (e o cabeçalho `X-Meeng-Call`) é o **mesmo em todas as tentativas**: é a chave para ignorar repetidos.
- Se um endpoint falha **30 avisos seguidos** (cada um já sem tentativas sobrando), ele é **desligado sozinho** (status desligado automaticamente). Corrija o servidor e ligue-o de novo no painel.
- Os avisos já concluídos ficam guardados por 30 dias na tela do webhook, com a resposta do seu servidor (os primeiros 4 KB) e o último erro.
- A segurança da entrega: o Meeng.dev **só chama endereços públicos**. Um endereço que resolve para rede interna, `localhost` ou IP privado é recusado.

Para testar localmente, exponha o servidor da máquina por um túnel HTTPS público (o endereço do túnel vai no webhook) ou use um webhook de teste só durante o desenvolvimento; apague-o depois.

## Um aviso não chegou

Siga a ordem:

1. **O endpoint está ligado?** No painel (Webhooks), veja o status. Desligado automaticamente depois de 30 falhas seguidas: corrija e religue.
2. **O evento está assinado?** Um endpoint só recebe os eventos que escolheu (vazio recebe todos) e, se marcou domínios, só os desses domínios.
3. **O e-mail é do mesmo domínio?** O filtro por domínio ignora os eventos dos outros.
4. **Olhe a tela do webhook.** Cada aviso mostra as tentativas, o status que o seu servidor respondeu e o último erro. Tentar de novo está a um botão de distância.
5. **O evento existe?** `email.opened` e `email.clicked` só acontecem com o rastreio ligado no domínio, e só quando o destinatário abre ou clica.
6. **Responda 2xx, sem redirecionar.** Um receptor que responde `301` (de `http` para `https`, ou com uma barra a mais) falha em todas as tentativas. Um receptor que devolve 200 só no fim de um trabalho longo estoura os 10 segundos. Responda primeiro e processe depois.

## Erros de assinatura

O SDK recusa com um código. O que cada um quer dizer:

| Código | Causa | O que fazer |
|---|---|---|
| `MISSING_SIGNATURE` | falta o cabeçalho de assinatura | a chamada não veio do Meeng.dev, ou um proxy removeu o cabeçalho; confira os cabeçalhos que chegam ao servidor |
| `MISSING_TIMESTAMP` | falta o cabeçalho de hora | idem |
| `INVALID_SIGNATURE_FORMAT` | o cabeçalho não começa com `v1=` | idem; confira se um proxy o alterou |
| `INVALID_TIMESTAMP` | a hora não é um número | idem |
| `TIMESTAMP_OUT_OF_RANGE` | a hora difere mais de 5 minutos da do seu servidor | acerte o relógio do servidor (NTP) |
| `SIGNATURE_MISMATCH` | a assinatura não bate | o corpo foi alterado antes da conferência (JSON reescrito), ou o segredo está errado (de outro webhook, de outro ambiente, antigo depois de "gerar novo segredo") |
| `INVALID_BODY` | o corpo não é texto nem bytes válidos | passe o corpo cru |
| `INVALID_JSON` | a assinatura está certa, mas o corpo não é JSON | raro; guarde o corpo e veja a tela do aviso |

`SIGNATURE_MISMATCH` é o erro mais comum, e quase sempre por um destes motivos:

1. **O corpo não é o cru.** Um middleware (`express.json()`, `BodyParser`, `request.json()`) leu e reescreveu o JSON. Leia o corpo em texto ou bytes e confira antes de interpretar.
2. **Segredo de outro webhook.** Cada webhook tem o seu. Confira se a variável de ambiente é a do webhook que está chamando.
3. **Segredo trocado.** Depois de "gerar novo segredo" o anterior deixa de assinar na hora.
4. **Espaço ou quebra de linha no segredo** copiado para a variável de ambiente.
5. **Codificação.** O corpo tem de ser conferido como os mesmos bytes, em UTF-8.

## Idempotência: o mesmo aviso duas vezes

Mesmo com tudo certo, o mesmo aviso pode chegar mais de uma vez (uma resposta que se perdeu, um servidor que reiniciou entre gravar e responder). Trate assim:

1. Grave o `id` do aviso numa tabela com **chave única** antes de agir.
2. Se a gravação falhar por chave repetida, o aviso já foi tratado: responda 200 e pare.
3. Faça o efeito (marcar entregue, descadastrar) e a gravação do `id` na **mesma transação**, para que uma falha no meio não deixe o efeito feito sem o registro, ou o registro sem o efeito.

```sql
CREATE TABLE meeng_avisos (
  id         text PRIMARY KEY,         -- o id do aviso (X-Meeng-Call)
  tipo       text NOT NULL,
  criado_em  timestamptz NOT NULL DEFAULT now()
);
-- INSERT ... ON CONFLICT (id) DO NOTHING: zero linhas afetadas quer dizer repetido.
```

## Ordem dos avisos

Os avisos de um mesmo e-mail podem chegar fora de ordem: o `email.delivered` antes do `email.sent`, o `email.opened` antes do `email.delivered`. Guarde o maior status já visto e só avance. A ordem de precedência dos status é: `SCHEDULED`, `QUEUED`, `SENT`, `DELIVERY_DELAYED`, `BOUNCED`, `REJECTED`, `RENDERING_FAILURE`, `DELIVERED`, `OPENED`, `CLICKED`, `COMPLAINED`, `FAILED`, `CANCELLED`, `SUPPRESSED`. Um `BOUNCED` que chega depois de `DELIVERED` é registrado como fato, mas não derruba o status que o Meeng.dev mostra.

## Segurança do receptor

- Só `POST`. Responda 404 ou 405 a qualquer outro método.
- Limite o tamanho do corpo (1 MB sobra) antes de ler.
- Responda 401 a assinatura inválida, **sem** explicar demais no corpo.
- Não exponha o endereço do receptor em lugar público; a assinatura é a garantia, não o sigilo do endereço.
- Mantenha o segredo fora do repositório. Para trocar sem perder avisos: crie um segundo webhook com outro segredo, troque o receptor, e só então apague o antigo.
