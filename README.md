# unicd — Unifique Cloud CLI

<p align="center">
  <strong>CLI oficial para gerenciar recursos no Unifique Cloud</strong><br>
  VMware Cloud Director (vCD) + Object Storage (S3)
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white">
  <img alt="License" src="https://img.shields.io/badge/license-MIT-blue">
  <img alt="Platform" src="https://img.shields.io/badge/platform-linux%20%7C%20windows%20%7C%20macos-lightgrey">
</p>

---

## Sobre

A **unicd** é a CLI do Unifique Cloud para provisionamento e operação de recursos de nuvem. Ela cobre dois grandes domínios:

- **VMware Cloud Director (vCD)** — organizações (tenants), VDCs, redes, edge gateways, regras NAT, firewall, IPSec, VMs, vApps, catálogos e usuários.
- **Object Storage (S3)** — buckets e objetos via endpoint S3 compatível `s3.unifique.cloud`.

Escrita em Go, distribuída como binário único (sem dependências) e pensada tanto para uso interativo quanto para automação/scripts.

## Funcionalidades

| Domínio | Recursos |
|---------|----------|
| **Organização (Org)** | Dados da org, listagem de usuários e roles, permissões por role e remoção completa (tenant offboarding) |
| **VDC** | Listagem/detalhes, criação, atualização, remoção, alocação de recursos, storage policies e vApps |
| **Redes** | Redes org VDC (isolated/routed), detalhes, IPAM, criação e remoção |
| **Edge Gateway** | Listagem e detalhes (HA, configuração) |
| **NAT** | DNAT/SNAT, listagem, criação, remoção e renomeação de regras |
| **Firewall** | Regras (ALLOW/DROP), listagem, criação e remoção |
| **IPSec VPN** | Túneis: listagem, detalhes, criação, remoção e status |
| **Virtual Machines** | Listagem, detalhes, power on/off, reboot, resize de CPU/RAM, resize de disco, deploy a partir de template, attach de rede, sync de NICs, edição e mídia (ISO) |
| **vApps** | Listagem, detalhes, criação e remoção |
| **Catálogos** | Catálogos e itens (templates/mídia) |
| **Usuários** | Listagem, detalhes, criação e remoção |
| **Object Storage (S3)** | Autenticação, configuração, credenciais e execução de operações de objetos (ls, cp, mv, rm, sync, etc.) |

## Instalação

### Build a partir do código

```bash
git clone https://github.com/rsdenck/unicd.git
cd unicd
go build -o unicd .
sudo install -m 0755 unicd /usr/local/bin/unicd
unicd version
```

Requer **Go 1.26+**.

### Script de instalação (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/rsdenck/unicd/main/install.sh | bash
```

## Autenticação

A unicd resolve credenciais na seguinte ordem: **variáveis de ambiente** > **token salvo** > **prompt interativo**.

### VMware Cloud Director

```bash
# Variáveis de ambiente (recomendado para scripts/CI)
export UNICD_USER="meu.usuario"
export UNICD_PASS="minha-senha"
export UNICD_HOST="vcd-tio.unifique.cloud"
export UNICD_ORG="MINHA_ORG"
unicd vm list

# Login interativo
unicd login -u meu.usuario -H vcd-tio.unifique.cloud -o MINHA_ORG

# Apenas login (prompt interativo de usuário e senha)
unicd login
```

### Object Storage (S3)

```bash
# Autenticação interativa (recomendado)
unicd s3 auth

# Autenticação via flags
unicd s3 auth -u <access-key> -p <secret-key> --endpoint s3.unifique.cloud
```

As credenciais ficam salvas localmente com permissão restrita (`~/.unicd/`). Use `unicd s3 config` para conferir e `unicd s3 logout` para remover.

As operações de objetos são implementadas nativamente pela própria CLI, sem dependências externas.

## Comandos

### Sessão e Configuração

| Comando | Descrição |
|---------|-----------|
| `unicd login [-u user] [-p pass] [-H host] [-o org]` | Autenticar no vCD |
| `unicd logout` | Limpar sessão local |
| `unicd whoami` | Mostrar usuário/org/sessão atual |
| `unicd version` | Exibir versão da CLI (commit e data de build) |
| `unicd context list` | Listar contextos salvos |
| `unicd context use <name>` | Ativar contexto |
| `unicd config get` | Exibir configuração atual |
| `unicd completion [bash\|zsh\|fish]` | Gerar script de completion |
| `unicd debug info` | Informações do ambiente (OS, arch, versão) |
| `unicd debug api <method> <path> [-d data] [-f file]` | Chamada raw à CloudAPI |

### Organização (Org)

| Comando | Descrição |
|---------|-----------|
| `unicd org list` | Dados da organização atual |
| `unicd org show` | Detalhes da organização |
| `unicd org users list` | Listar usuários |
| `unicd org roles list` | Listar roles |
| `unicd org permissions` | Permissões por role |
| `unicd org del [--dry-run] [--yes]` | Remover organização completamente |

### Virtual Datacenter (VDC)

| Comando | Descrição |
|---------|-----------|
| `unicd vdc list` | Listar VDCs |
| `unicd vdc show` | Detalhes (CPU, memória, alocação) |
| `unicd vdc create --name ...` | Criar VDC (admin) |
| `unicd vdc update --name ... [--new-name ...]` | Atualizar VDC (admin) |
| `unicd vdc delete [--force] [--recursive]` | Remover VDC (admin) |
| `unicd vdc resource` | Alocação de recursos do VDC |
| `unicd vdc storage list` | Listar storage policies |
| `unicd vdc vapp list` | Listar vApps do VDC |
| `unicd vdc vapp delete <name>` | Remover vApp por nome |

### Redes

| Comando | Descrição |
|---------|-----------|
| `unicd net list` | Listar redes |
| `unicd net show -n <name>` | Detalhes da rede |
| `unicd net create -n <name> -g <gateway> [-t isolated\|routed] [-p <prefix>] [--dns1 <ip>]` | Criar rede |
| `unicd net delete -n <name>` | Remover rede |
| `unicd net ipam -n <name>` | Detalhes de IPAM da rede |
| `unicd net attach -n <name> --vm <vm> [--ip <ip>] [--mode MANUAL\|DHCP\|POOL]` | Anexar rede a uma VM |

### Edge Gateways

| Comando | Descrição |
|---------|-----------|
| `unicd edge list` | Listar edge gateways |
| `unicd edge show -n <name>` | Detalhes (HA, configuração) |

### NAT (CloudAPI)

| Comando | Descrição |
|---------|-----------|
| `unicd nat list [-e <edge>]` | Listar regras NAT |
| `unicd nat create dnat -e <edge> -x <ext-ip> -i <int-ip> [--external-port --internal-port -p tcp\|udp]` | Criar DNAT |
| `unicd nat create snat -e <edge> -x <ext-ip> -i <cidr>` | Criar SNAT |
| `unicd nat delete <rule-id> [-e <edge>]` | Remover regra |
| `unicd nat rename <rule-id> <novo-nome> [-e <edge>]` | Renomear regra |

### Firewall (CloudAPI)

| Comando | Descrição |
|---------|-----------|
| `unicd fw list [-e <edge>] [--json]` | Listar regras de firewall |
| `unicd fw create -e <edge> -n <nome> [-a ALLOW\|DROP] [-s <src>] [-d <dst>] [-p <proto>] [--port N]` | Criar regra |
| `unicd fw delete -e <edge> -n <nome>` | Remover regra |

### IPSec VPN

| Comando | Descrição |
|---------|-----------|
| `unicd ipsec list [-e <edge>]` | Listar túneis IPSec |
| `unicd ipsec show -n <name> [-e <edge>]` | Detalhes do túnel |
| `unicd ipsec create --peer-ip <ip> --psk <key> ... [-e <edge>]` | Criar túnel IPSec |
| `unicd ipsec delete -n <name> [-e <edge>]` | Remover túnel |
| `unicd ipsec status -n <name> [-e <edge>]` | Status do túnel |

### Virtual Machines

| Comando | Descrição |
|---------|-----------|
| `unicd vm list` | Listar VMs |
| `unicd vm show <name>` | Detalhes e NICs |
| `unicd vm poweron <name>` | Ligar |
| `unicd vm poweroff <name>` | Desligar |
| `unicd vm reboot <name>` | Reiniciar |
| `unicd vm resize <name> --cpu N --memory N` | Alterar CPU/memória |
| `unicd vm disk-resize <name> -s N [--id N]` | Redimensionar disco |
| `unicd vm edit <name> [--name --description --hostname --script <file>]` | Editar nome, descrição, hostname ou script de customização |
| `unicd vm delete <name>` | Deletar VM e vApp |
| `unicd vm template [--catalog <name>]` | Listar templates disponíveis |
| `unicd vm deploy <vmname> <template> [--network --ip --mode --catalog]` | Deploy de VM a partir de template |
| `unicd vm attach-net <name> --network <net> [--ip --mode --index]` | Anexar NIC |
| `unicd vm sync-nics <source> <target>` | Copiar layout de NICs entre VMs |
| `unicd vm media insert <vm> <catalog> <media>` | Inserir ISO na VM |
| `unicd vm media eject <vm> <catalog> <media>` | Ejetar ISO da VM |

### vApps

| Comando | Descrição |
|---------|-----------|
| `unicd vapp list` | Listar vApps |
| `unicd vapp show <name>` | Detalhes do vApp |
| `unicd vapp create <name>` | Criar vApp |
| `unicd vapp delete <name>` | Remover vApp |

### Catálogos

| Comando | Descrição |
|---------|-----------|
| `unicd catalog list` | Listar catálogos |
| `unicd catalog items [-c <catalog>] [<catalog-name>]` | Listar itens (templates/mídia) |

### Usuários

| Comando | Descrição |
|---------|-----------|
| `unicd user list` | Listar usuários via CloudAPI |
| `unicd user show <name>` | Detalhes do usuário |
| `unicd user create --name ... [--role ...] [--password ...]` | Criar usuário |
| `unicd user delete <name>` | Remover usuário |

### Object Storage (S3)

| Comando | Descrição |
|---------|-----------|
| `unicd s3 auth [-u access-key] [-p secret] [--endpoint] [--region] [--insecure] [--check=false]` | Autenticar no object storage |
| `unicd s3 config` | Mostrar configuração S3 salva |
| `unicd s3 logout` | Remover credenciais S3 |
| `unicd s3 ls [s3://bucket[/prefix]] [-H] [--sum]` | Listar buckets ou objetos |
| `unicd s3 cp <origem> <destino> [-r]` | Copiar arquivos/objetos |
| `unicd s3 mv <origem> <destino> [-r]` | Mover arquivos/objetos |
| `unicd s3 rm s3://bucket/chave [...] [-r]` | Remover objetos |
| `unicd s3 mb s3://bucket` | Criar bucket |
| `unicd s3 rb s3://bucket` | Remover bucket vazio |
| `unicd s3 cat s3://bucket/chave` | Imprimir objeto em stdout |
| `unicd s3 du s3://bucket[/prefix] [-H]` | Tamanho total de um prefixo |
| `unicd s3 head s3://bucket/chave` | Metadados do objeto |
| `unicd s3 presign s3://bucket/chave [-e duração] [-X GET\|PUT]` | Gerar URL pré-assinada |
| `unicd s3 sync <origem> <destino> [--delete] [--exact-timestamps]` | Sincronizar diretório <-> bucket (via simples) |
| `unicd s3 pipe s3://bucket/chave` | Enviar stdin para um objeto |

Flags comuns de transferência (`cp`, `mv`, `sync`, `pipe`):

| Flag | Descrição |
|------|-----------|
| `-r, --recursive` | Recursivo em diretórios/prefixos |
| `-c, --concurrency <n>` | Transferências em paralelo (default 5) |
| `--part-size <tam>` | Tamanho da parte no multipart (ex.: `8MB`) |
| `--storage-class <classe>` | Classe de armazenamento do objeto |
| `--content-type <mime>` | Content-Type |
| `--content-encoding <enc>` | Content-Encoding |
| `--cache-control <valor>` | Cache-Control |
| `--metadata chave=valor` | Metadado do usuário (repetível) |
| `--expires <RFC3339>` | Data de expiração |
| `--no-clobber` | Não sobrescrever destino existente |
| `--if-size-differ` | Transferir apenas quando o tamanho diferir |
| `--include <glob>` / `--exclude <glob>` | Filtrar chaves (repetível) |
| `--dry-run` | Simular sem transferir |
| `--use-list-objects-v1` | Listar com V1 (default; exigido por alguns serviços) |

## Flags Globais

| Flag | Descrição |
|------|-----------|
| `-h, --help` | Ajuda |
| `--no-banner` | Suprimir banner ASCII |

## Variáveis de Ambiente

| Variável | Descrição |
|----------|-----------|
| `UNICD_USER` | Usuário vCD |
| `UNICD_PASS` | Senha vCD |
| `UNICD_ORG` | Organização vCD |
| `UNICD_HOST` | Host vCD |
| `UNICD_API_VERSION` | Versão da API (default: `37.0`) |
| `UNICD_S3_ENDPOINT` | Endpoint do object storage (default: `s3.unifique.cloud`) |
| `UNICD_S3_ACCESS_KEY` | Access Key do object storage |
| `UNICD_S3_SECRET` | Secret Key do object storage |
| `UNICD_S3_REGION` | Região do object storage (default: `us-east-1`) |

## Exemplos

```bash
# Listar recursos
unicd vdc list
unicd net list
unicd edge list
unicd vm list

# Criar DNAT
unicd nat create dnat \
  -e RT-EDGE \
  -x 187.85.177.19 \
  -i 10.124.200.10 \
  --external-port 2222 \
  --internal-port 22 \
  --protocol tcp

# Criar regra de firewall
unicd fw create \
  -e RT-EDGE \
  -n "ALLOW_SSH" \
  -a ALLOW \
  -s 0.0.0.0/0 \
  -d 10.124.200.10 \
  -p tcp \
  --port 22

# Gerenciar VM
unicd vm show RT-EDGE-01
unicd vm resize RT-EDGE-01 --cpu 4 --memory 8192
unicd vm attach-net RT-EDGE-01 --network NET_VLAN100 --ip 10.124.100.2 --mode MANUAL

# Deploy de VM
unicd vm template --catalog "Linux Templates"
unicd vm deploy app01 "ubuntu-22.04" --network NET_VLAN100 --ip 10.124.100.50 --mode MANUAL

# Inserir ISO
unicd vm media insert minha-vm "ISO Catalog" "ubuntu-22.04.iso"

# Object Storage (S3)
unicd s3 auth

unicd s3 ls s3://meu-bucket
unicd s3 cp s3://meu-bucket/arquivo.zip ./
unicd s3 cp ./backup.sql s3://meu-bucket/backup.sql
unicd s3 sync ./data s3://meu-bucket/data

# Object Storage via variáveis de ambiente
export UNICD_S3_ACCESS_KEY="sua-access-key"
export UNICD_S3_SECRET="sua-secret"
export UNICD_S3_ENDPOINT="s3.unifique.cloud"
unicd s3 ls s3://meu-bucket
```

## Estrutura do Projeto

```
unicd/
  main.go               # Entry point + banner
  cmd/
    root.go             # Comando raiz (cobra)
    common.go           # getClientFromContext() + stubCmd()
    login.go            # Autenticacao
    logout.go           # Limpar sessao
    whoami.go           # Info do usuario
    version.go          # Versao da CLI
    config.go           # Gerenciar configuracao
    context.go          # Gerenciar contextos (multi-org)
    completion.go       # Shell completion (bash/zsh/fish)
    org.go              # Operacoes de organizacao
    org_del.go          # Remocao completa de org (tenant offboarding)
    vdc.go              # Operacoes de VDC
    vapp.go             # Operacoes de vApp
    vm.go               # Operacoes de VM (power, resize, deploy)
    vm_media.go         # Operacoes de VM media (ISO)
    net.go              # Operacoes de rede
    edge.go             # Operacoes de edge gateway
    nat.go              # Operacoes de NAT (CloudAPI)
    fw.go               # Operacoes de firewall (CloudAPI)
    ipsec.go            # Operacoes de IPSec VPN
    catalog.go          # Operacoes de catalogo
    user.go             # Operacoes de usuario
    s3.go               # Operacoes de object storage (S3)
    debug.go            # Ferramentas de debug
    query.go            # Consultas (planejado)
    cluster.go          # Clusters (planejado)
    apply.go            # IaC apply (planejado)
  pkg/
    config/config.go    # Gerenciamento de config/sessao
    client/client.go    # Wrapper vCD + RawCloudAPI()
  install.sh            # Instalador para Linux/macOS
```

## Stack

- **Linguagem:** Go 1.26+
- **CLI Framework:** [cobra](https://github.com/spf13/cobra)
- **SDK vCD:** [go-vcloud-director](https://github.com/vmware/go-vcloud-director) v2.26.1
- **API:** vCD CloudAPI (REST) + SDK nativo
- **Object Storage:** serviço S3-compatível `s3.unifique.cloud`

## Desenvolvimento

```bash
# Formatação e análise estática
gofmt -l .
go vet ./...

# Build
go build -o unicd .

# Testes
go test ./...
```

O projeto usa **GitHub Actions** para integração contínua (build + vet) e release automatizada de binários ao criar uma tag `v*`.

## Licença

Distribuído sob a licença MIT. Veja [LICENSE](LICENSE) para mais detalhes.
