# AGENT.md — Guia para Agentes de IA (unicd)

> Documento de referência **para agentes de IA** que precisam operar a CLI `unicd`
> (Unifique Cloud Director + Object Storage) ou trabalhar no código-fonte dela.
> Leia até o fim antes de executar comandos contra ambientes reais.

---

## 1. O que é a unicd

CLI oficial do Unifique Cloud, escrita em Go (cobra), binário único e estático. Cobre dois domínios:

- **VMware Cloud Director (vCD)** — orgs, VDCs, redes, edge gateways, NAT, firewall, IPSec, VMs, vApps, catálogos, usuários.
- **Object Storage (S3)** — buckets/objetos via endpoint S3-compatível (`s3.unifique.cloud`), implementação nativa (sem dependência externa).

Repositório: `https://github.com/rsdenck/unicd` · Última versão: veja `unicd version` ou as tags do repo.

---

## 2. Regras de ouro para agentes

1. **Sessão primeiro, credencial depois.** Antes de qualquer comando de dados, valide a sessão com `unicd whoami`. Se falhar com `401 Unauthorized` (token expirado), faça login novamente.
2. **Nunca exiba segredos.** Não imprima senhas, tokens (`~/.unicd/session.json`) nem access/secret keys do S3. Passar secret em linha de comando é aceitável em ambiente protegido, mas prefira variáveis de ambiente.
3. **Use `--no-banner` em automação/scripts** — suprime o banner ASCII e facilita o parse da saída.
4. **Não confie em prompts interativos.** Em automação, informe TODAS as flags necessárias (login exige usuário + senha + ORG).
5. **Comandos destrutivos exigem dupla checagem.** `org del`, `vdc delete`, `vm delete`, `vapp delete`, `nat delete`, `fw delete`, `ipsec delete`, `s3 rm`, `s3 rb` removem recursos de verdade. Prefira `--dry-run` quando existir (ex.: `unicd org del --dry-run`, flag `--dry-run` nas transfers S3).
6. **Contexte sempre a org certa.** A CLI opera na org do **contexto ativo** (`unicd context list` / `currentContext`). Troque com `unicd context use <nome>`.
7. **Em caso de erro de API**, use `unicd debug api <method> <path>` para inspecionar a CloudAPI raw e encontrar a causa real.

---

## 3. Instalação e build

```bash
# Binário pronto (releases do GitHub)
# baixar unicd_<versão>_linux_amd64 do release e instalar

# Build a partir do código (Go 1.26+)
git clone https://github.com/rsdenck/unicd.git
cd unicd
go build -o unicd .
sudo install -m 0755 unicd /usr/local/bin/unicd

# Via script de instalação
curl -fsSL https://raw.githubusercontent.com/rsdenck/unicd/main/install.sh | bash
```

### Makefile (útil para manter/versionar o repositório)

| Target | O que faz |
|--------|-----------|
| `make build` | Build local (`./unicd`) |
| `make install` | Build + instala em `/usr/local/bin/unicd` |
| `make fmt` / `make vet` / `make test` | gofmt / vet / testes |
| `make release` | Gera binários multi-plataforma em `dist/` |
| `make clean` | Remove binários e `dist/` |

> A versão embutida vem de `git describe --tags` (via Makefile) ou, nas releases,
> do nome da tag (workflow `.github/workflows/release.yml`). Bucket de dev/sujo aparece
> como `-dirty` / `dev`.

---

## 4. Autenticação

### 4.1 vCD — ordem de resolução: **env vars > token salvo > prompt**

```bash
# Variáveis de ambiente (preferido em CI/scripts)
export UNICD_USER="meu.usuario"
export UNICD_PASS="minha-senha"
export UNICD_HOST="vcd-tio.unifique.cloud"
export UNICD_ORG="MINHA_ORG"
unicd vm list

# Login explícito (usuário, senha e ORG são obrigatórios)
unicd login -H vcd-tio.unifique.cloud -u meu.usuario -p minha-senha -o MINHA_ORG

# Login interativo (só para uso manual)
unicd login
```

- A sessão (token) fica em `~/.unicd/session.json`; contextos em `~/.unicd/config.json`.
- **Tokens expiram** → erro `401 Unauthorized`. Solução: rodar `unicd login` novamente com as credenciais.
- Multi-org: `unicd login` cria/reutiliza contextos. Ative com `unicd context use <org>`.

### 4.2 Object Storage (S3)

```bash
unicd s3 auth                                      # interativo (recomendado)
unicd s3 auth -u <access-key> -p <secret-key> --endpoint s3.unifique.cloud
unicd s3 config                                    # conferir configuração salva
unicd s3 logout                                    # remover credenciais
```

Credenciais salvas em `~/.unicd/s3.credentials` / `s3.json` (permissão restrita). Também podem vir de env: `UNICD_S3_ACCESS_KEY`, `UNICD_S3_SECRET`, `UNICD_S3_ENDPOINT`, `UNICD_S3_REGION`.

---

## 5. Grupos de comandos (referência rápida)

### Sessão / configuração
| Comando | Uso |
|---------|-----|
| `unicd whoami` | Mostra usuário/org/sessão atuais |
| `unicd login [-H host] -u user [-p pass] -o org` | Autentica no vCD |
| `unicd logout` | Limpa sessão local |
| `unicd context list` / `context use <name>` | Gerencia contextos (orgs) |
| `unicd config get` | Configuração atual |
| `unicd version` | Versão, commit e data de build |
| `unicd debug info` / `debug api <method> <path>` | Diagnóstico / chamada CloudAPI raw |
| `unicd completion bash\|zsh\|fish` | Shell completion |

### vCD — recursos
| Grupo | Comandos-chave |
|-------|----------------|
| **org** | `org list`, `org show`, `org users list`, `org roles list`, `org permissions`, `org del [--dry-run] [--yes]` |
| **vdc** | `vdc list`, `vdc show`, `vdc create --name ...`, `vdc update`, `vdc delete [--force] [--recursive]`, `vdc resource`, `vdc storage list`, `vdc vapp list/delete` |
| **net** | `net list`, `net show -n <name>`, `net create -n <name> -g <gw> [-t isolated\|routed] [-p <prefix>]`, `net delete -n <name>`, `net ipam -n <name>`, `net attach -n <name> --vm <vm>` |
| **edge** | `edge list`, `edge show -n <name>` |
| **nat** (CloudAPI) | `nat list [-e <edge>]`, `nat create dnat\|snat ...`, `nat delete <id>`, `nat rename <id> <new>` |
| **fw** (CloudAPI) | `fw list [-e <edge>] [--json]`, `fw create -e <edge> -n <nome> [-a ALLOW\|DROP] [-s src] [-d dst] [-p proto] [--port N]`, `fw delete -e <edge> -n <nome>` |
| **ipsec** | `ipsec list/show/create/delete/status` (sempre com `-e <edge>`) |
| **vm** | `vm list`, `vm show <name>`, `vm poweron/off/reboot <name>`, `vm resize <name> --cpu N --memory N`, `vm disk-resize <name> -s N`, `vm edit <name> [...flags]`, `vm delete <name>`, `vm template`, `vm deploy <vm> <template>`, `vm attach-net`, `vm sync-nics <src> <dst>`, `vm media insert/eject` |
| **vapp** | `vapp list/show/create/delete` |
| **catalog** | `catalog list`, `catalog items [-c <catalog>]` |
| **user** | `user list`, `user show <name>`, `user create --name ...`, `user delete <name>` |

> Observação: `unicd user list` lista via AdminOrg (role, enabled, provider type, email) — não via antiga query XML.

### S3 — operações de objeto
`unicd s3 ls|cp|mv|rm|mb|rb|cat|du|head|presign|sync|pipe`

Flags comuns de transferência (`cp/mv/sync/pipe`): `-r`, `-c <n>` (concorrência), `--part-size`, `--storage-class`, `--metadata`, `--no-clobber`, `--if-size-differ`, `--include/--exclude`, `--dry-run`.

---

## 6. Fluxos de trabalho típicos (receitas para agentes)

### 6.1 Listar VMs de uma org (passo a passo)
```bash
# 1. Garantir sessão ativa na org certa
unicd whoami
unicd context list

# 2. Se 401/login ausente:
unicd login -H vcd-tio.unifique.cloud -u <user> -p <pass> -o <org>

# 3. Listar
unicd --no-banner vm list
unicd --no-banner vm show <nome-da-vm>   # detalhes + NICs
```

### 6.2 Entender rede/FW/NAT de um tenant
```bash
unicd vdc list
unicd edge list                          # pega o nome do edge (ex.: RT-EDGE)
unicd net list                           # redes org VDC
unicd fw list -e <edge> --json
unicd nat list -e <edge>
```

### 6.3 Deploy de VM a partir de template
```bash
unicd vm template --catalog "Linux Templates"
unicd vm deploy app01 ubuntu-22.04 --network NET_VLAN100 --ip 10.124.100.50 --mode MANUAL
```

### 6.4 Object storage
```bash
unicd s3 auth
unicd s3 ls s3://meu-bucket
unicd s3 cp ./backup.sql s3://meu-bucket/backup.sql
unicd s3 presign s3://meu-bucket/arq.zip -e 1h -X GET
```

### 6.5 Diagnóstico de erro
```bash
unicd debug info
unicd debug api GET /api/admin/orgs         # exemplo de chamada raw
```
Sempre capture `unicd version` (commit+built) para reportar problemas.

---

## 7. Variáveis de ambiente

| Variável | Descrição | Default |
|----------|-----------|---------|
| `UNICD_USER` / `UNICD_PASS` | Usuário/senha vCD | — |
| `UNICD_ORG` | Organização vCD | — |
| `UNICD_HOST` | Host vCD | — |
| `UNICD_API_VERSION` | Versão da API | `37.0` |
| `UNICD_S3_ENDPOINT` | Endpoint S3 | `s3.unifique.cloud` |
| `UNICD_S3_ACCESS_KEY` / `UNICD_S3_SECRET` | Credenciais S3 | — |
| `UNICD_S3_REGION` | Região S3 | `us-east-1` |

---

## 8. Estrutura do repositório (para agentes que mantêm o código)

```
unicd/
  main.go               # entry point + banner
  cmd/                  # comandos cobra (um arquivo por domínio)
  pkg/config/           # config/sessão (~/.unicd)
  pkg/client/           # wrapper vCD + RawCloudAPI()
  install.sh            # instalador
  Makefile              # build/install/release
  .github/workflows/    # ci.yml + release.yml
```

**Ao alterar código, o CI exige:** `gofmt -l .` limpo, `go vet ./...` sem erros, build ok e testes ok. Não commite se o CI for quebrar.

---

## 9. Versionamento (como publicar mudanças)

1. Faça as alterações e valide: `gofmt -l .`, `go vet ./...`, `go build -o unicd .`, `go test ./...`.
2. Atualize `cmd/version.go` (Version) para a nova versão se for release.
3. Commit com mensagem descritiva (convenção: `feat(...)`, `fix(...)`, etc.) e push para `main`.
4. **Release:** crie a tag `vX.Y.Z` e envie (`git tag v1.4.4 && git push origin v1.4.4`).
   O workflow `release.yml` compila os binários multi-plataforma, gera checksums e publica a release no GitHub (tags `v*`).
5. Confirme o status da release em Actions → Release, e a última tag com `git ls-remote --tags origin`.

> Uma mudança só está **versionada** quando: commitada **e** pushada na `main` (e, se aplicável, tag/release).
> Sempre confira o estado local com `git status` (mudanças não commitadas e não pushadas).