# Registros DNS, um a um

Tudo o que o dono precisa publicar vem na resposta de `POST /v1/domains` e de `GET /v1/domains/{id}`, no campo `dnsRecords`. Cada item tem `type`, `name`, `value`, `ttl`, `priority` (só no MX), `status` e `recommended`.

Os exemplos abaixo usam o domínio `envio.exemplo.com.br`, um subdomínio de `exemplo.com.br`. Os valores reais (o servidor de entrada, o `include` do SPF e a chave DKIM) mudam: **copie sempre da resposta da API**.

## 1. MX: onde voltam as devoluções

| Campo | Valor |
|---|---|
| Tipo | `MX` |
| Nome | `mail.envio` no exemplo acima (o endereço completo é `mail.envio.exemplo.com.br`), ou só `mail` quando o domínio cadastrado é o principal (`exemplo.com.br`) |
| Valor | o servidor de entrada que a API informa |
| Prioridade | `10` |

Serve para o Meeng.dev receber as mensagens de erro dos servidores de destino (endereço que não existe, caixa cheia) e as reclamações. É esse retorno que alimenta os status `BOUNCED` e `COMPLAINED` e a lista de supressão. Este MX fica num subdomínio (`mail`); **não toca no MX do domínio principal**, onde o e-mail recebido da empresa continua chegando onde chegava.

## 2. SPF: quem pode enviar

| Campo | Valor |
|---|---|
| Tipo | `TXT` |
| Nome | o mesmo do MX (`mail.envio` no exemplo) |
| Valor | `v=spf1 include:... ~all`, como a API informa |

- O SPF diz aos servidores de destino que os servidores do Meeng.dev têm permissão de enviar por esse domínio.
- **Só pode haver um registro SPF por nome.** Se o nome já tem um (`v=spf1 ...`), não crie um segundo: acrescente o `include:` do Meeng.dev ao que existe, antes do `~all`. Dois SPF no mesmo nome fazem o SPF falhar (erro permanente).
- Como o registro fica no subdomínio `mail`, no caso normal não há conflito com o SPF do domínio principal.
- Um SPF pode ter no máximo 10 consultas de DNS (cada `include:` conta). Se o domínio já usa muitos serviços, prefira um subdomínio só para o Meeng.dev.

## 3. DKIM: a assinatura do e-mail

| Campo | Valor |
|---|---|
| Tipo | `TXT` |
| Nome | `meeng._domainkey.envio` no exemplo (só `meeng._domainkey` quando o domínio cadastrado é o principal) |
| Valor | `p=...` com a chave pública, ou `v=DKIM1; k=ed25519; p=...` quando a chave é Ed25519 |

- O Meeng.dev assina cada e-mail com a chave privada, que fica guardada cifrada e nunca sai da plataforma. O destinatário confere a assinatura com esta chave pública.
- A chave RSA é longa (passa de 255 caracteres). O DNS guarda um TXT como pedaços de até 255 caracteres; a maioria dos painéis divide sozinha, outros pedem que você cole o valor inteiro, e alguns exigem dividir à mão em partes entre aspas. Na dúvida, cole o valor inteiro e confira o resultado com a verificação.
- Não troque o valor, não acrescente espaço nem quebra de linha, e não apague a chave enquanto o domínio estiver em uso: sem ela os e-mails perdem a assinatura e o DMARC falha.

## 4. DMARC: a política (recomendado)

| Campo | Valor |
|---|---|
| Tipo | `TXT` |
| Nome | `_dmarc` (no domínio principal, veja [`dmarc.md`](dmarc.md)) |
| Valor | `v=DMARC1; p=none;` |

O domínio funciona sem este registro e a API o marca como `recommended`. Os grandes provedores de caixa de entrada exigem DMARC de quem envia em volume. Comece em `p=none` (só observa) e endureça aos poucos; veja [`dmarc.md`](dmarc.md). Se o domínio já tem um DMARC, **não crie outro**: só pode haver um, e o `dmarcAdded` do domínio vira `true` quando ele existe.

## Como preencher o painel do provedor de DNS

Cada provedor (Registro.br, Cloudflare, Route 53, GoDaddy, Locaweb, HostGator e outros) tem uma tela parecida, mas muda o jeito de escrever o **Nome**:

- Alguns pedem só a parte antes do domínio (`mail`, `meeng._domainkey`, `_dmarc`). É o caso mais comum, e o mesmo formato que a API devolve.
- Outros pedem o nome completo (`envio.exemplo.com.br`). Preencha com o domínio no final.
- Se o painel **acrescenta o domínio sozinho**, não o repita: `envio.exemplo.com.br.exemplo.com.br` é um erro frequente.
- No **Cloudflare**, os registros de e-mail ficam em "somente DNS" (nuvem cinza). MX e TXT não passam pelo proxy, então normalmente já vêm assim.
- **TTL**: a API diz `Auto`. Use o padrão do provedor, ou 3600 segundos. Se estiver corrigindo um erro, 300 segundos faz a correção valer mais depressa.
- **Prioridade** só existe no MX; deixe em branco (ou não preencha) nos TXT.
- **Aspas**: alguns painéis acrescentam aspas ao TXT sozinhos; outros exigem. Se o painel tem um campo de aviso sobre aspas, siga-o. Nunca cole aspas dentro do valor quando o painel já as coloca.

Depois de publicar, confira de fora, sem esperar o Meeng.dev: `dig TXT meeng._domainkey.envio.exemplo.com.br +short` e `dig MX mail.envio.exemplo.com.br +short` (ou `nslookup`). O valor deve voltar igual ao da API. DNS novo pode levar de minutos a algumas horas para valer em todo lugar.

## Restringir uma chave de API a um domínio

Na criação da chave, no painel (`https://meeng.dev/configuracoes/chaves`), é possível limitá-la a **um domínio**. Uma chave restrita:

- só envia com `from` desse domínio (de outro, `403` com `API key does not have access to domain`);
- só enxerga e verifica esse domínio, e não consegue criar outro (`403` com `API key is restricted to a single domain and cannot create domains`);
- lista só os e-mails e as campanhas dele.

É a chave certa para cada serviço do dono: o site de vendas usa a do `vendas.exemplo.com.br`, e o sistema de cobrança usa a do `cobranca.exemplo.com.br`. Se um deles vazar, o estrago fica no domínio dele.

## Remover um domínio

`DELETE /v1/domains/{id}` remove o domínio e **apaga as chaves de API restritas a ele**. Os e-mails já enviados continuam no histórico. Peça a confirmação do dono antes: não há como desfazer. Depois remova os registros DNS que ficaram sobrando, principalmente o MX e o SPF do subdomínio.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

const { data: dominios, error } = await meeng.domains.list();
if (error) {
  throw new Error(error.message);
}
for (const dominio of dominios) {
  const pendentes = dominio.dnsRecords.filter((registro) => registro.status !== "SUCCESS" && !registro.recommended);
  console.log(`${dominio.name}: ${dominio.status}${pendentes.length ? `, faltam ${pendentes.length} registros` : ""}`);
}
```
