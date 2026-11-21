# Plano de migração

Seis fases. Cada uma tem um critério de saída: só passe para a seguinte quando ele for verdadeiro. O tempo total típico é de duas a quatro semanas, a maior parte esperando o volume subir com segurança.

## Fase 1: inventário (um dia)

Descubra tudo o que envia e recebe e-mail no sistema, antes de escrever uma linha:

- **Onde se chama o provedor atual.** Procure no código pelo endereço do provedor, pelo nome do pacote do SDK e pela variável de ambiente da chave (`grep -rn` nos três). Inclua filas, tarefas agendadas, scripts de manutenção, e-mails que o próprio framework manda (recuperação de senha, convites).
- **O que cada envio é**: transacional (recibo, código), de lote, com arquivo (vira link), agendado, com modelo, em campanha. Cada um tem um caminho no Meeng.dev ([`mapeamento-de-campos.md`](mapeamento-de-campos.md)).
- **De quais endereços** (`From`) e de quais domínios o sistema envia.
- **Volume**: e-mails por dia e por mês, e o pico por segundo. O limite padrão do Meeng.dev é de 2 requisições por segundo por organização; um pico maior pede lote ou aumento do limite.
- **Webhooks recebidos**: endereços, eventos e o que o sistema faz com cada um.
- **SMTP**: sistemas que enviam por SMTP (plugins, ERPs, impressoras).
- **Listas**: contatos, descadastros e supressões guardados no provedor atual.

**Critério de saída:** uma lista com cada ponto de envio, o domínio, o volume e o dono do código.

## Fase 2: domínio e conta (um a três dias)

1. Crie a conta, a organização e uma **chave de API nova** só para este sistema (`https://meeng.dev/configuracoes/chaves`), de preferência restrita ao domínio de envio.
2. Cadastre o domínio de envio no Meeng.dev (`meeng-dominios`). O DKIM do Meeng.dev usa o nome `meeng._domainkey`, então não colide com o do provedor antigo. Publique os registros do Meeng.dev **sem apagar** os antigos.
3. Aguarde o domínio chegar a `SUCCESS`.
4. Publique um DMARC (`p=none` para começar) se ainda não existe.

**Critério de saída:** `GET /v1/domains/{id}` devolve `status: SUCCESS`; um e-mail de teste do Meeng.dev para o endereço do dono chega à caixa de entrada com `dkim=pass`, `spf=pass` e `dmarc=pass`.

## Fase 3: dados que precisam estar lá antes (um a dois dias)

- **Supressões:** carregue as do provedor antigo (`mapeamento-de-campos.md`, "Supressões").
- **Contatos:** importe só os que têm a prova do aceite, com os que saíram marcados como `subscribed: false` (`meeng-contatos-e-campanhas`).
- **Modelos:** recrie os modelos de e-mail no painel (`https://meeng.dev/painel/modelos`) e anote os `templateId` novos. Compare com o resultado de `render_template`, se houver um assistente conectado por MCP, ou com um envio de teste.

**Critério de saída:** a lista de supressão do Meeng.dev tem os endereços do antigo; os modelos novos geram o mesmo texto.

## Fase 4: adaptador e webhooks (dois a cinco dias)

1. **Escreva o adaptador** de envio (`SKILL.md`, com TypeScript e Go) e ponha a chave de configuração (a porcentagem) em variável de ambiente, começando em **0**.
2. **Troque os receptores de webhook**: escreva o receptor do Meeng.dev com o SDK (`meeng-webhooks`), com idempotência por `(provedor, id)`. Crie o webhook no painel apontando para ele.
3. **Teste em homologação** (ou em produção com 0%): um envio real para o dono, um repetido com a mesma chave (deve voltar o mesmo `emailId`), um com link de arquivo no corpo, um agendado, um para um endereço suprimido, um para um endereço inexistente (deve gerar `email.bounced`).
4. Confira que o webhook novo recebeu `email.delivered`, `email.bounced` e `email.suppressed` nesses testes.

**Critério de saída:** todos os testes acima passam, e o receptor novo grava os eventos.

## Fase 5: virada gradual (uma a duas semanas)

Suba a porcentagem em degraus, esperando um dia inteiro entre um e outro (dois, se o volume é pequeno):

| Degrau | Porcentagem | Olhe antes de subir |
|---|---|---|
| 1 | 1% a 5% | os e-mails chegaram? o webhook gravou `email.delivered`? |
| 2 | 10% | taxa de devolução dura abaixo de 2%; reclamações abaixo de 0,1% |
| 3 | 25% | os mesmos números; nenhum `email.failed` por limite |
| 4 | 50% | os mesmos números; sem erros `429` no registro do sistema |
| 5 | 100% | os mesmos números, por dois dias |

Em cada degrau, olhe:

- `analytics.reputationMetrics` (as taxas de devolução e de reclamação da conta) e `analytics.emailTimeSeries` (os números por dia);
- os eventos `email.bounced`, `email.complained` e `email.failed` no receptor;
- o painel de e-mails (`https://meeng.dev/painel/emails`);
- as reclamações do suporte do dono ("não recebi o e-mail").

**Plano de volta:** se um número passa do limite, ou um tipo de e-mail some, ponha a porcentagem em **0** (uma variável de ambiente, sem deploy) e investigue. O provedor antigo continua pronto até o fim da fase 6. Por isso a chave de configuração decide pelo **id estável do envio**: o mesmo e-mail vai sempre pelo mesmo caminho, e nenhum cliente recebe duplicado na troca.

**Critério de saída:** 100% por dois dias, taxas dentro dos limites, nenhum chamado aberto sobre e-mail que não chegou.

## Fase 6: desligar o antigo (uma a duas semanas depois)

1. Espere pelo menos uma semana de operação estável a 100%.
2. Desligue o receptor de webhook antigo e apague o webhook no provedor antigo.
3. Apague a chave do provedor antigo do ambiente e do cofre de segredos (depois de conferir que nenhum sistema ainda a usa) e revogue-a no painel dele.
4. Remova os registros DNS do provedor antigo (os do DKIM, do SPF e do retorno), **depois** de uma semana sem tráfego por ele. Não apague o DMARC.
5. Remova o código do adaptador e do provedor antigo, e a chave de configuração.
6. Registre a troca (o que mudou e quando) onde a equipe guarda as decisões.

**Critério de saída:** nada do provedor antigo resta no código, no DNS nem nos segredos.

## O que costuma dar errado

| Problema | Causa | Como evitar |
|---|---|---|
| e-mails de recibo caem no spam depois da virada | volume subiu rápido demais num caminho novo | siga os degraus; reduza a porcentagem e espere |
| reclamações e devoluções sobem | a lista de supressão não foi carregada | fase 3, antes da fase 5 |
| e-mails duplicados | duas chamadas, uma para cada provedor, sem decisão estável | a chave de configuração decide por id estável; um e-mail vai por um só caminho |
| o receptor novo rejeita todos os avisos | `SIGNATURE_MISMATCH`: corpo não cru, ou segredo trocado | `meeng-webhooks`, "Erros de assinatura" |
| `403` ao enviar | o domínio do `From` não está verificado, ou a chave é restrita a outro domínio | `meeng-dominios` |
| `429` em horários de pico | passou de 2 requisições por segundo | lote, ou pedir aumento do limite |
| o anexo some | o Meeng.dev não anexa arquivos | hospede o arquivo e ponha o link no corpo (`mapeamento-de-campos.md`) |
| cliente recebe e-mail que já tinha pedido para não receber | contatos que saíram foram importados como inscritos | importe quem saiu com `subscribed: false` |

## Conferência final

- [ ] domínio em `SUCCESS`, com SPF, DKIM e DMARC passando
- [ ] chave de API própria deste sistema, restrita ao domínio, fora do repositório
- [ ] supressões carregadas antes do primeiro envio
- [ ] adaptador testado (simples, lote, link de arquivo, agendado, repetido, suprimido, inexistente)
- [ ] receptor de webhook novo com idempotência e conferência de assinatura
- [ ] virada gradual completa, com taxas dentro dos limites
- [ ] provedor antigo desligado, chaves revogadas, DNS limpo
- [ ] a equipe sabe o plano de volta enquanto o antigo existir
