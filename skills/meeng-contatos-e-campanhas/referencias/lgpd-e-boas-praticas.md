# LGPD e boas práticas no envio em massa

Esta página resume o que o dono precisa fazer para mandar e-mail em massa dentro da LGPD (Lei nº 13.709/2018) e sem perder a reputação de envio. Não é parecer jurídico: para uma dúvida do negócio, o dono deve falar com quem cuida da conformidade dele (o encarregado ou o jurídico).

## Quem é quem

O **dono do sistema** que usa o Meeng.dev decide para quem enviar e com que objetivo: ele é o **controlador** dos dados dos contatos. O Meeng.dev trata esses dados para enviar os e-mails a mando dele: é o **operador**. Quem responde ao titular que reclama do e-mail é, em primeiro lugar, o dono do sistema.

## Consentimento e base legal

- E-mail de **divulgação** (boletim, promoção, novidade) pede uma base legal. A mais comum é o **consentimento**: a pessoa aceitou receber, de forma livre e informada, para uma finalidade que ela conhece.
- E-mail **transacional** (recibo, código de acesso, aviso de entrega do pedido que ela fez) tem outra base, a execução do contrato, e não depende de aceite para divulgação. **Não misture**: não aproveite um e-mail de recibo para empurrar promoção.
- A **dupla confirmação** (que o Meeng.dev traz ligada) é a prova mais forte do consentimento: a pessoa digitou o endereço e depois clicou no link enviado a ele. Mantenha-a ligada em listas de divulgação.
- Guarde **de onde** e **quando** cada contato aceitou (`properties`: `origem`, `aceite_em`, o texto do formulário que ele viu). É o que se mostra se alguém perguntar "como vocês conseguiram meu e-mail?".
- **Não importe** lista comprada, raspada de site, ou de outro negócio sem o aceite da pessoa para este envio. Além de irregular, estraga a reputação do domínio na primeira campanha.

## Transparência e descadastro

- Diga no formulário de captura **o que** a pessoa vai receber e **com que frequência**.
- Todo e-mail de divulgação tem o **link de descadastro** visível (o Meeng.dev exige o marcador em toda campanha) e **funciona sem login**.
- Respeite o pedido **na hora**: o link do Meeng.dev já tira o contato da lista. Se a pessoa pedir a saída por outro canal (resposta ao e-mail, telefone), marque `subscribed: false` por `contacts.update` no mesmo dia.
- Se o dono tem **outros sistemas** com a mesma pessoa, espelhe o descadastro (escute `contact.updated` em `meeng-webhooks`).

## Direitos do titular

A pessoa pode pedir confirmação de que você a trata, acesso, correção, eliminação e outros direitos (arts. 17 a 22). Para os contatos que estão no Meeng.dev:

- **Acesso e correção:** `contacts.list` com `emails=...` e `contacts.update`.
- **Eliminação:** `contacts.delete` apaga o contato da lista. Para o histórico de e-mails já enviados, peça a eliminação à equipe do Meeng.dev (`ajuda@meeng.dev`).
- **Lista de supressão:** quem pediu para nunca mais receber (ou deu bounce duro) fica numa lista de supressão da conta: é o que garante que o pedido seja respeitado mesmo se o contato for importado de novo. A supressão da conta é separada da lista de contatos: apagar o contato não a limpa, e isso protege quem pediu para não receber.

Responda ao titular no prazo que a lei e a ANPD definem para cada direito, e guarde o registro do pedido e da resposta.

## Segurança dos dados

- Os contatos são dados pessoais. A chave de API que lê e escreve contatos (`mg_...`) fica só no servidor.
- Prefira uma chave por serviço, restrita ao domínio de envio dele (`meeng-dominios`).
- Não copie a lista de contatos para planilhas soltas, nem registre e-mails em log aberto.
- Importe só os campos de que o envio precisa: e-mail, nome e as variáveis do conteúdo. Não leve CPF, data de nascimento ou outros dados sensíveis para uma lista de e-mail.
- Em caso de incidente com dados (vazamento da chave, lista exposta), troque a chave no painel na hora e avise o responsável pela conformidade do dono: pode haver dever de comunicar à ANPD e aos titulares.

## Reputação de envio

Quem manda e-mail em massa vive da reputação do domínio. O que a protege:

1. Só mande a quem pediu (acima).
2. Comece com listas pequenas e **aumente aos poucos**: um domínio novo que manda 50 mil e-mails no primeiro dia é tratado como suspeito. Duplique o volume a cada envio, observando os números.
3. **Limpe a lista**: contatos que não abrem nada há 6 a 12 meses são candidatos a sair, ou a uma campanha de "ainda quer receber?".
4. Fique **abaixo de 2%** de devolução dura e **abaixo de 0,1%** de reclamação por campanha. Passou: pare e investigue.
5. Use um **subdomínio de envio** só para divulgação, separado do que envia recibos e códigos de acesso (`meeng-dominios`). Se a divulgação tiver problema, o transacional continua entregando.
6. Mande no horário em que o público lê, com frequência constante. Picos grandes e raros parecem spam.
7. DKIM, SPF e DMARC em ordem (`meeng-dominios`).
