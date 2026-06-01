# unicd — Unifique Cloud Director CLI

CLI provider para VMware Cloud Director, desenvolvido para o tenant **DENCK_ORG** da Unifique.

## Autenticação

```bash
# Login interativo
unicd login -u ranlens.denck -H vcd-tio.unifique.cloud -o DENCK_ORG

# Usar variáveis de ambiente (alternativa)
$env:UNICD_USER = "ranlens.denck"
$env:UNICD_PASS = "sua-senha"
$env:UNICD_ORG  = "DENCK_ORG"
$env:UNICD_HOST = "vcd-tio.unifique.cloud"
unicd login
```

## Comandos

### Autenticação e Configuração
| Comando | Descrição |
|---|---|
| `unicd login [-u user] [-p pass] [-H host] [-o org]` | Autenticar no vCD e salvar sessão |
| `unicd logout` | Limpar sessão local |
| `unicd whoami` | Mostrar usuário/org atual |
| `unicd context list` | Listar contextos salvos |
| `unicd context use <nome>` | Selecionar contexto ativo |
| `unicd config get` | Exibir configuração completa |
| `unicd config set <chave>=<valor>` | Alterar configuração |

### Organização
| Comando | Descrição |
|---|---|
| `unicd org list` | Listar informações da organização |
| `unicd org show` | Detalhes da organização |
| `unicd org users list` | Listar usuários |
| `unicd org roles list` | Listar roles |
| `unicd org permissions` | Exibir permissões |

### Virtual Datacenter (VDC)
| Comando | Descrição |
|---|---|
| `unicd vdc list` | Listar VDCs |
| `unicd vdc show` | Detalhes do VDC (CPU, memória, alocação) |
| `unicd vdc create` | Criar VDC |
| `unicd vdc delete` | Remover VDC |
| `unicd vdc update` | Atualizar VDC |
| `unicd vdc storage list` | Listar storage policies |
| `unicd vdc resource` | Alocação de recursos do VDC |

### Redes (Org VDC Networks)
| Comando | Descrição |
|---|---|
| `unicd net list` | Listar redes (tipo: direct/routed/isolated) |
| `unicd net show --name <nome>` | Detalhes da rede (gateway, netmask, DNS) |
| `unicd net create --name <nome> [--gateway <ip> --type <type> --dns1 <ip> --prefix <n>]` | Criar rede |
| `unicd net delete --name <nome>` | Remover rede |
| `unicd net ipam` | Exibir IPAM |
| `unicd net attach` | Anexar rede a uma VM |

### Edge Gateways
| Comando | Descrição |
|---|---|
| `unicd edge list` | Listar edge gateways |
| `unicd edge show --name <nome>` | Detalhes do edge gateway |

### NAT Rules (CloudAPI)
| Comando | Descrição |
|---|---|
| `unicd nat list` | Listar regras NAT |
| `unicd nat create dnat -e <edge> -x <ext-ip> -i <int-ip> [--external-port <n> --internal-port <n> --protocol tcp\|udp]` | Criar DNAT |
| `unicd nat delete` | Remover regra NAT |
| `unicd nat flush` | Remover todas as regras NAT |
| `unicd nat show` | Detalhes da regra NAT |

### Firewall Rules (CloudAPI)
| Comando | Descrição |
|---|---|
| `unicd fw list` | Listar regras de firewall |
| `unicd fw create -e <edge> -n <nome> [-a ACCEPT\|DROP] [-s <source> -d <dest> -p <proto> --port <n>]` | Criar regra de firewall |
| `unicd fw delete` | Remover regra de firewall |
| `unicd fw enable` | Habilitar firewall |
| `unicd fw disable` | Desabilitar firewall |
| `unicd fw policy` | Policy padrão do firewall |

### Virtual Machines
| Comando | Descrição |
|---|---|
| `unicd vm list` | Listar VMs |
| `unicd vm show` | Detalhes da VM |
| `unicd vm poweron` | Ligar VM |
| `unicd vm poweroff` | Desligar VM |
| `unicd vm reboot` | Reiniciar VM |

### vApps
| Comando | Descrição |
|---|---|
| `unicd vapp list` | Listar vApps |
| `unicd vapp show` | Detalhes do vApp |
| `unicd vapp create` | Criar vApp |
| `unicd vapp delete` | Remover vApp |

### Consultas (Query)
| Comando | Descrição |
|---|---|
| `unicd query run` | Executar query na API do vCD |

### Usuários
| Comando | Descrição |
|---|---|
| `unicd user list` | Listar usuários |
| `unicd user show` | Detalhes do usuário |
| `unicd user create` | Criar usuário |
| `unicd user delete` | Remover usuário |

### IaC (Apply)
| Comando | Descrição |
|---|---|
| `unicd apply file` | Aplicar configuração a partir de arquivo YAML/JSON |

### Debug
| Comando | Descrição |
|---|---|
| `unicd debug info` | Informações do ambiente (OS, arch, versão) |
| `unicd debug api` | Chamada raw à CloudAPI |

### Clusters
| Comando | Descrição |
|---|---|
| `unicd cluster list` | Listar clusters |
| `unicd cluster create` | Criar cluster |

## Flags Globais
| Flag | Descrição |
|---|---|
| `-h, --help` | Ajuda do comando |

## Variáveis de Ambiente
| Variável | Descrição |
|---|---|
| `UNICD_USER` | Usuário vCD |
| `UNICD_PASS` | Senha vCD |
| `UNICD_ORG` | Organização vCD |
| `UNICD_HOST` | Host vCD |
| `UNICD_API_VERSION` | Versão da API (default: 37.0) |

## Exemplos Rápidos

```bash
# Login
unicd login

# Listar recursos
unicd vdc list
unicd net list
unicd edge list

# Criar DNAT
unicd nat create dnat \
  -e TIO-EDGE-DC-SC-01 \
  -x 200.150.100.10 \
  -i 10.0.0.50 \
  --external-port 443 \
  --internal-port 443 \
  --protocol tcp

# Criar regra de firewall
unicd fw create \
  -e TIO-EDGE-DC-SC-01 \
  -n "Liberar HTTPS" \
  -a ACCEPT \
  -s 0.0.0.0/0 \
  -d 10.0.0.50 \
  -p tcp \
  --port 443
```
