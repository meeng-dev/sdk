# DMARC

O DMARC é a regra que o dono do domínio publica dizendo ao servidor de destino **o que fazer** com um e-mail que diz ser do domínio mas não passa na conferência de SPF e DKIM: aceitar assim mesmo, mandar para o spam ou recusar. Sem ele, qualquer um pode se passar pelo domínio da empresa (golpe de falsa cobrança, por exemplo); com ele, o domínio fica protegido e a reputação de envio melhora.

## O que o Meeng.dev publica por padrão

O registro recomendado é:

```
_dmarc    TXT    v=DMARC1; p=none;
```

`p=none` quer dizer "só observe": nada muda para quem recebe, mas os provedores passam a mandar relatórios. É o ponto de partida seguro.

## Onde o registro fica

O DMARC do **domínio principal** vale para todos os subdomínios, a menos que um subdomínio tenha o próprio. Por isso:

- O nome é `_dmarc.<domínio principal>` (`_dmarc.exemplo.com.br`), mesmo quando o envio sai de um subdomínio (`envio.exemplo.com.br`). O nome que a API devolve já é o certo para o domínio cadastrado; se cadastrou um subdomínio e o domínio principal já tem DMARC, o do principal já cobre o envio.
- **Só pode existir um registro DMARC por nome.** Se o domínio já tem um, não publique outro: ajuste o que existe.

## O caminho de `none` a `reject`

Faça em etapas, olhando os relatórios entre uma e outra.

1. **`p=none` com relatórios** (semanas 1 e 2). Acrescente um endereço para receber os relatórios agregados: `v=DMARC1; p=none; rua=mailto:dmarc@exemplo.com.br;`. Eles chegam em XML, uma vez por dia, de cada provedor que recebeu e-mail do domínio. Use um endereço que alguém leia, ou um serviço que leia os relatórios para você.
2. **Corrigir quem não passa.** O relatório lista cada origem de e-mail do domínio (o Meeng.dev, o sistema de cobrança, a ferramenta de marketing, o servidor antigo) e se ela passa em SPF e DKIM *alinhados* com o domínio do `from`. Quem aparece falhando e é legítimo precisa ser configurado (SPF ou DKIM do serviço dele); quem é desconhecido é o que o DMARC vai barrar.
3. **`p=quarantine; pct=25`** (semanas 3 e 4). Manda para o spam 25% dos e-mails que falham. Suba o `pct` a 50, 100 enquanto nada legítimo aparecer falhando.
4. **`p=reject`**. Recusa o que falha. É o estado final de um domínio saudável, e o que mais protege a marca.

Em cada etapa, espere pelo menos uma semana de relatórios. Nunca pule de `none` direto para `reject` num domínio que envia por vários sistemas: e-mail legítimo some.

## Alinhamento: por que o Meeng.dev passa

O DMARC passa quando o SPF **ou** o DKIM passa e o domínio que ele conferiu é o mesmo (ou subdomínio) do domínio do `from`. No Meeng.dev:

- o **DKIM** é assinado com o domínio do `from` (a chave que você publicou em `meeng._domainkey`), então fica alinhado;
- o **SPF** é conferido no domínio de retorno (`mail`), que é subdomínio do domínio do `from`, então também fica alinhado no modo padrão (`aspf` relaxado).

Por isso, com SPF e DKIM em `SUCCESS` e `from` do domínio cadastrado, o DMARC passa. Se um e-mail seu falha no DMARC, o `from` provavelmente não é do domínio cadastrado, ou o DKIM foi apagado.

## Ler o status no Meeng.dev

`dmarcAdded` é `true` quando o registro existe, e o item `_dmarc` de `dnsRecords` mostra o status. O DMARC é opcional: com ele ausente o domínio ainda chega a `SUCCESS` e envia. Aparecem `PENDING` ou `NOT_STARTED` no item enquanto ele não foi publicado, e isso **não** impede o envio.

```ts
import { Meeng } from "@meeng-dev/sdk";

const meeng = new Meeng();

const { data, error } = await meeng.domains.list();
if (error) {
  throw new Error(error.message);
}
for (const dominio of data) {
  if (dominio.status === "SUCCESS" && !dominio.dmarcAdded) {
    console.log(`${dominio.name}: verificado, mas sem DMARC. Publique v=DMARC1; p=none; em _dmarc.`);
  }
}
```

## Dicas

- Use um subdomínio só para o Meeng.dev: o relatório de DMARC fica limpo, e endurecer a política do principal não depende deste envio.
- Se um serviço de terceiros envia e-mail "em nome" do domínio sem estar configurado, o DMARC o barra: é o efeito esperado quando a política é `quarantine` ou `reject`.
- Depois de endurecer, acompanhe `analytics.reputationMetrics` (taxas de devolução e de reclamação) por alguns dias.
