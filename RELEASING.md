# Releasing

Uma versão para todos os SDKs (`VERSION`). Empurrar a tag `vX.Y.Z` roda
`.github/workflows/release.yml`, que publica essa versão em cada registro abaixo.
Cada job pergunta antes ao registro se a versão já existe e não faz nada se
existir; assim uma versão publicada à mão antes, ou uma segunda rodada da tag,
nunca falha por duplicidade. Os jobs são independentes: um registro que falha
não para os outros, e um job que falhou pode ser rodado sozinho de novo.

## What gates a release

- O primeiro job, `guard`, falha se o commit da tag não for ancestral de
  `origin/main`, e todo job de publicação depende dele. Uma tag num commit que não
  está no `main` não publica nada. Avance o `main` primeiro, depois crie a tag.
- Todo job de publicação roda num ambiente do GitHub, para que revisores
  obrigatórios possam segurar a publicação: `release` para npm, RubyGems,
  crates.io, pub.dev, NuGet e Maven Central, `pypi` para o PyPI e `hex` para o
  Hex (esses dois são os ambientes que os publicadores confiáveis deles citam, e
  um job só tem um). Os publicadores dos demais deixam o ambiente em branco, o que
  aceita qualquer um; então citar `release` não muda o que eles aceitam.

## Each release

1. Suba o `VERSION`, rode `./regen.sh` e `./check.sh`, faça o commit no
   `develop` e, depois da CI, avance o `main`.
2. Nesse commit, crie e empurre as duas tags:

   ```sh
   git tag vX.Y.Z && git tag go/vX.Y.Z
   git push origin vX.Y.Z go/vX.Y.Z
   ```

   - `vX.Y.Z` roda o fluxo (npm, PyPI, RubyGems, crates.io, Hex, pub.dev, NuGet,
     Maven Central) e é o que o Packagist e o Swift leem.
   - `go/vX.Y.Z` é a versão do módulo Go.

## What the release installs

Os jobs que seguram um token OIDC instalam só ferramentas fixadas, para que uma
publicação comprometida de uma ferramenta não alcance a credencial de
publicação:

- npm: `typescript/package-lock.json` está no repositório e é instalado com
  `npm ci --ignore-scripts`; o próprio npm é instalado numa versão exata (12.2.1;
  a publicação confiável pede o npm 11.5.1 ou mais novo, e o npm 12 pede o Node
  22.22.2, 24.15 ou mais novo, que o `node-version: '24'` dá). Suba-a de propósito.
  Depois de mudar `typescript/package.json`, rode
  `npm install --package-lock-only --ignore-scripts` lá e faça o commit do lock.
- PyPI: as ferramentas de construção estão em `.github/release/python-build.txt`,
  com hash travado (`pip-compile --generate-hashes --allow-unsafe --strip-extras
  python-build.in`, a partir de `.github/release/`).
- Hex: `mix local.hex 2.6.0`.
- Toda action está fixada num SHA de commit, mantido em dia pelo Dependabot.

## One-time setup, per registry

O pacote pertence à organização `meeng-dev` em todo registro que tem
organização. Em todo formulário de publicador confiável os valores são os
mesmos: dono `meeng-dev`, repositório `sdk`, fluxo `release.yml`; o ambiente fica
em branco, a menos que este arquivo diga outra coisa (em branco aceita qualquer
ambiente, então os jobs que rodam em `release` casam). Ligar um publicador ao
`release` é um passo seguinte, quando os revisores do ambiente estiverem
definidos; PyPI e Hex ficam em `pypi` e `hex` até seus publicadores serem
movidos. Os nomes estão fixados em `AGENTS.md`.

| Registry | Package | Auth in the workflow |
| --- | --- | --- |
| npm | `@meeng-dev/sdk` | OIDC |
| PyPI | `meeng-sdk` | OIDC |
| RubyGems | `meeng-sdk` | OIDC |
| crates.io | `meeng-sdk` | OIDC |
| Hex | `meeng_sdk` | OIDC (secret as fallback) |
| pub.dev | `meeng_sdk` | OIDC |
| NuGet | `MeengDev.Sdk` | OIDC + one secret |
| Maven Central | `dev.meeng:sdk` | secrets |
| Packagist, Go, Swift | | the tag itself, no workflow |

Um publicador confiável só pode ser acrescentado a um pacote que já existe no
npm, no crates.io, no Hex e no pub.dev; por isso a primeira versão de um pacote
nesses registros sai à mão, e todas as seguintes saem pela tag. O RubyGems e o
PyPI aceitam um publicador para um pacote que ainda não existe.

### GitHub (repository meeng-dev/sdk)

- Ambientes `release`, `pypi` e `hex` (criados pela primeira rodada que os
  cita). Acrescente revisores obrigatórios aos três e restrinja a implantação às
  tags `v*` (regras de proteção de implantação, tags selecionadas); até lá nada
  segura uma publicação para aprovação. `pypi` e `hex` precisam manter os nomes
  que os publicadores confiáveis abaixo usam.
- Proteção de branch no `main` e um conjunto de regras de tag para `v*` e `go/v*`
  (só mantenedores criam) reforçam o job `guard`.
- Segredos do repositório: só os listados em NuGet, Maven Central e, se usado, Hex.

### npm

1. A organização `meeng-dev` é dona de `@meeng-dev/sdk`.
2. Configurações do pacote, Trusted Publisher, GitHub Actions: organização
   `meeng-dev`, repositório `sdk`, fluxo `release.yml`, ambiente em branco.

### PyPI

A organização `meeng-dev` é dona de `meeng-sdk`. Publicador confiável (GitHub):
dono `meeng-dev`, repositório `sdk`, fluxo `release.yml`, ambiente `pypi`.

### RubyGems

Gem `meeng-sdk` (da organização `meeng-dev`). Página da gem, Trusted publishers,
GitHub Actions: dono `meeng-dev`, repositório `sdk`, fluxo `release.yml`,
ambiente em branco.

### crates.io

Crate `meeng-sdk` (dono: um time do GitHub de `meeng-dev`). Configurações da
crate, Trusted Publishing, acrescentar GitHub: dono `meeng-dev`, repositório
`sdk`, fluxo `release.yml`, ambiente em branco.

### Hex

Pacote `meeng_sdk`, da organização `meeng-dev` do Hex. Como dono completo, com
segundo fator ligado, abra a aba "Workload identities" do pacote e acrescente:
dono `meeng-dev`, repositório `sdk`, fluxo `release.yml` (maiúsculas e minúsculas
contam), ambiente `hex` (exigido enquanto o repositório for público).

Se isso ainda não estiver feito, acrescente o segredo `HEX_API_KEY` do
repositório (uma chave com permissão de publicar o pacote); o job o usa quando
existe.

### pub.dev

1. Primeira versão à mão, a partir de `dart/`: `dart pub publish`, sob o
   publicador verificado escolhido para `meeng-dev` (confirme o nome do
   publicador antes de criá-lo, veja `AGENTS.md`).
2. Package Admin, Automated publishing, publicação pelo GitHub Actions:
   repositório `meeng-dev/sdk`, padrão de tag `v{{version}}`. Não exija um
   ambiente: o job roda em `release`.

### NuGet

1. A conta ou organização `meeng-dev` no nuget.org é dona de `MeengDev.Sdk`
   (reserve o prefixo `MeengDev.`).
2. Trusted Publishing, nova política: dona da política a conta ou organização
   `meeng-dev`, dono do repositório `meeng-dev`, repositório `sdk`, arquivo do
   fluxo `release.yml` (só o nome do arquivo), ambiente em branco, escopo para
   permitir pacotes e versões novas (glob `MeengDev.*`).
3. Segredo do repositório `NUGET_USER`: o nome de perfil no nuget.org (não o
   e-mail).

### Maven Central

1. Conta no Sonatype Central Portal; registre o namespace `dev.meeng` e
   verifique-o com o registro DNS TXT em `meeng.dev` (sem tocar no SPF, MX ou
   DMARC do domínio).
2. Portal, Account, Generate User Token. Segredos do repositório
   `MAVEN_CENTRAL_USERNAME` e `MAVEN_CENTRAL_TOKEN` (as duas metades do token).
3. Crie uma chave GPG e publique a metade pública em `keyserver.ubuntu.com` ou
   `keys.openpgp.org`. Segredos do repositório `MAVEN_GPG_PRIVATE_KEY`
   (`gpg --armor --export-secret-keys <id>`) e `MAVEN_GPG_PASSPHRASE`.

O job roda `mvn -Prelease deploy` em `java/` (fontes, Javadoc e assinaturas vêm do
perfil `release` do template do gerador) e espera até o portal publicar a
versão, o que pode levar vários minutos.

### Packagist, Go, Swift

Nada no fluxo. O Packagist lê a tag `v` pelo `composer.json` da raiz, o Swift pelo
`Package.swift` da raiz, e o proxy do Go lê a tag `go/vX.Y.Z`.

## Checking a release without publishing

Localmente, na pasta correspondente: `gem build meeng-sdk.gemspec`,
`cargo package --allow-dirty`, `mix hex.build`, `dart pub publish --dry-run`,
`dotnet pack src/Meeng/Meeng.csproj -c Release` e
`mvn -Prelease -Dgpg.skip=true package`.
