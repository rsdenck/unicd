# unicd -- Unifique Cloud Director CLI

CLI provider para VMware Cloud Director. Gerencia recursos de organizacao (tenant) no vCD: VDCs, redes, VMs, NAT, firewall, catalogos, usuarios e mais.

## Instalacao

```bash
git clone https://github.com/rsdenck/unicd.git
cd unicd
go build -o unicd .
./unicd --help
```

## Autenticacao

Prioridade: **variaveis de ambiente** > **token salvo** > **interativo**.

```bash
# Variaveis de ambiente (recomendado para scripts)
export UNICD_USER="meu.usuario"
export UNICD_PASS="minha-senha"
export UNICD_HOST="vcd.exemplo.com"
export UNICD_ORG="MINHA_ORG"
unicd vm list

# Login interativo
unicd login -u meu.usuario -H vcd.exemplo.com -o MINHA_ORG

# Apenas login (prompt interativo)
unicd login
```

## Comandos

### Sessao e Configuracao
| Comando | Descricao |
|---------|-----------|
| `unicd login [-u user] [-p pass] [-H host] [-o org]` | Autenticar no vCD |
| `unicd logout` | Limpar sessao |
| `unicd whoami` | Mostrar usuario/org atual |
| `unicd version` | Exibir versao da CLI |
| `unicd context list` | Listar contextos salvos |
| `unicd context use <name>` | Ativar contexto |
| `unicd config get` | Exibir configuracao |
| `unicd completion [bash\|zsh\|fish]` | Gerar script de completao |
| `unicd debug info` | Info do ambiente (OS, arch, versao) |
| `unicd debug api <method> <path> [-d data] [-f file]` | Chamada raw a CloudAPI |

### Organizacao (Org)
| Comando | Descricao |
|---------|-----------|
| `unicd org list` | Mostrar organizacao atual |
| `unicd org show` | Detalhes da organizacao |
| `unicd org users list` | Listar usuarios (pendente) |
| `unicd org roles list` | Listar roles (pendente) |
| `unicd org permissions` | Permissoes (pendente) |

### Virtual Datacenter (VDC)
| Comando | Descricao |
|---------|-----------|
| `unicd vdc list` | Listar VDCs |
| `unicd vdc show` | Detalhes (CPU, memoria, alocacao) |
| `unicd vdc storage list` | Listar storage policies |
| `unicd vdc vapp list` | Listar vApps do VDC |
| `unicd vdc vapp delete <name>` | Deletar vApp por nome |

### Redes (Org VDC Networks)
| Comando | Descricao |
|---------|-----------|
| `unicd net list` | Listar redes |
| `unicd net show -n <name>` | Detalhes da rede |
| `unicd net create -n <name> -g <gateway> [-t <type> --dns1 <ip> -p <prefix>]` | Criar rede (isolated/routed) |
| `unicd net delete -n <name>` | Remover rede |

### Edge Gateways
| Comando | Descricao |
|---------|-----------|
| `unicd edge list` | Listar edge gateways |
| `unicd edge show -n <name>` | Detalhes (HA, config) |

### NAT Rules (CloudAPI)
| Comando | Descricao |
|---------|-----------|
| `unicd nat list [-e <edge>]` | Listar regras NAT |
| `unicd nat create dnat -e <edge> -x <ext-ip> -i <int-ip> [--external-port --internal-port -p tcp\|udp]` | Criar DNAT |
| `unicd nat create snat -e <edge> -x <ext-ip> -i <cidr>` | Criar SNAT |
| `unicd nat delete <rule-id> [-e <edge>]` | Remover regra |
| `unicd nat rename <rule-id> <novo-nome> [-e <edge>]` | Renomear regra |

### Firewall Rules (CloudAPI)
| Comando | Descricao |
|---------|-----------|
| `unicd fw list [-e <edge> --json]` | Listar regras de firewall |
| `unicd fw create -e <edge> -n <nome> [-a ACCEPT\|DROP -s <src> -d <dst> -p <proto> --port N]` | Criar regra |
| `unicd fw delete -e <edge> -n <nome>` | Remover regra |

### Virtual Machines
| Comando | Descricao |
|---------|-----------|
| `unicd vm list` | Listar VMs |
| `unicd vm show <name>` | Detalhes e NICs |
| `unicd vm poweron <name>` | Ligar |
| `unicd vm poweroff <name>` | Desligar |
| `unicd vm reboot <name>` | Reiniciar |
| `unicd vm resize <name> --cpu N --memory N` | Alterar CPU/memoria |
| `unicd vm disk-resize <name> -s N [--id N]` | Redimensionar disco |
| `unicd vm delete <name>` | Deletar VM e vApp |
| `unicd vm template [--catalog <name>]` | Listar templates |
| `unicd vm deploy <vmname> <template> [--network --ip --mode --catalog]` | Deploy de VM |
| `unicd vm attach-net <name> --network <net> [--ip --mode --index]` | Anexar NIC |
| `unicd vm sync-nics <source> <target>` | Copiar NICs entre VMs |
| `unicd vm media insert <vm> <catalog> <media>` | Inserir ISO na VM |
| `unicd vm media eject <vm> <catalog> <media>` | Ejetar ISO da VM |

### vApps
| Comando | Descricao |
|---------|-----------|
| `unicd vapp list` | Listar vApps |

### Catalogos
| Comando | Descricao |
|---------|-----------|
| `unicd catalog list` | Listar catalogos |
| `unicd catalog items [-c <catalog>] [<catalog-name>]` | Listar itens (templates/media) |

### Usuarios
| Comando | Descricao |
|---------|-----------|
| `unicd user list` | Listar usuarios |

## Flags Globais
| Flag | Descricao |
|------|-----------|
| `-h, --help` | Ajuda |
| `--no-banner` | Suprimir banner ASCII |

## Variaveis de Ambiente
| Variavel | Descricao |
|----------|-----------|
| `UNICD_USER` | Usuario vCD |
| `UNICD_PASS` | Senha vCD |
| `UNICD_ORG` | Organizacao vCD |
| `UNICD_HOST` | Host vCD |
| `UNICD_API_VERSION` | Versao da API (default: 37.0) |

## Exemplos

```bash
# Listar recursos
unicd vdc list
unicd net list
unicd edge list
unicd vm list

# Criar DNAT
unicd nat create dnat \
  -e RT-EGDE \
  -x 187.85.177.19 \
  -i 10.124.200.10 \
  --external-port 2222 \
  --internal-port 22 \
  --protocol tcp

# Criar regra de firewall
unicd fw create \
  -e RT-EGDE \
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
    vdc.go              # Operacoes de VDC
    vapp.go             # Operacoes de vApp
    vm.go               # Operacoes de VM (power, resize, deploy)
    vm_media.go         # Operacoes de VM media (ISO)
    net.go              # Operacoes de rede
    edge.go             # Operacoes de edge gateway
    nat.go              # Operacoes de NAT (CloudAPI)
    fw.go               # Operacoes de firewall (CloudAPI)
    catalog.go          # Operacoes de catalogo
    user.go             # Operacoes de usuario
    debug.go            # Ferramentas de debug
    query.go            # Consultas (planejado)
    cluster.go          # Clusters (planejado)
    apply.go            # IaC apply (planejado)
  pkg/
    config/config.go    # Gerenciamento de config/sessao
    client/client.go    # Wrapper vCD + RawCloudAPI()
```

## Stack

- **Linguagem:** Go 1.26+
- **CLI Framework:** cobra (spf13/cobra)
- **SDK:** go-vcloud-director v2.26.1 (VMware)
- **API:** vCD CloudAPI (REST) + SDK nativo
