#!/bin/bash
# Script de configuracao VYOS para RT-EDGE-01 e RT-EDGE-02
# Modo de uso:
#   scp vyos-configure.sh vyos@RT-EDGE-01:~/
#   ssh vyos@RT-EDGE-01
#   chmod +x vyos-configure.sh
#   ./vyos-configure.sh 1   # 1 = Primary (RT-EDGE-01), 2 = Secondary (RT-EDGE-02)
#
# Topologia:
#   eth0: NET_IS     10.124.0.0/24   (gestao)
#   eth1: NET_RT     10.124.10.0/30  (transit para vCD Edge)
#   eth2: NET_VLAN100 10.124.100.0/24
#   eth3: NET_VLAN101 10.124.101.0/24
#   eth4: NET_VLAN102 10.124.102.0/24
#   eth5: NET_VLAN103 10.124.103.0/24
#   eth6: NET_VLAN104 10.124.104.0/24
#
# Esquema de IPs:
#   vCD Edge Gateway: 10.124.10.1
#   RT-EDGE-01:       .2 em cada rede (prioridade 100, MASTER)
#   RT-EDGE-02:       .3 em cada rede (prioridade 90, BACKUP)
#   VRRP VIP:         .1 em cada rede (gateway padrao das VMs)

set -euo pipefail

NODE_ID="${1:-}"
if [[ "$NODE_ID" != "1" && "$NODE_ID" != "2" ]]; then
    echo "Uso: $0 <1|2>"
    echo "  1 = RT-EDGE-01 (Primary, MASTER)"
    echo "  2 = RT-EDGE-02 (Secondary, BACKUP)"
    exit 1
fi

if [[ "$NODE_ID" == "1" ]]; then
    HOSTNAME="RT-EDGE-01"
    PRIORITY=100
    IP_SUFFIX=2
    VRRP_PREEMPT=true
else
    HOSTNAME="RT-EDGE-02"
    PRIORITY=90
    IP_SUFFIX=3
    VRRP_PREEMPT=false
fi

# Gateway do vCD Edge na NET_RT
VCD_EDGE_GW="10.124.10.1"

echo "=== Configurando $HOSTNAME (node $NODE_ID, prioridade $PRIORITY) ==="

/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper begin

# Hostname
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set system host-name "$HOSTNAME"

# DNS
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set system name-server 1.1.1.1
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set system name-server 8.8.8.8

# SSH
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set service ssh port 22

# --- Interfaces ---

# eth0: NET_IS
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth0 address "10.124.0.${IP_SUFFIX}/24"
if [[ "$NODE_ID" == "1" ]]; then
    /opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth0 vrrp vrrp-group 1 virtual-address 10.124.0.1/24
    /opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth0 vrrp vrrp-group 1 priority "$PRIORITY"
    /opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth0 vrrp vrrp-group 1 preempt true
fi

# eth1: NET_RT (transit - sem VRRP)
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth1 address "10.124.10.${IP_SUFFIX}/30"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth1 description "TRANSIT-vCD-Edge"

# eth2: NET_VLAN100
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth2 address "10.124.100.${IP_SUFFIX}/24"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth2 vrrp vrrp-group 100 virtual-address 10.124.100.1/24
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth2 vrrp vrrp-group 100 priority "$PRIORITY"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth2 vrrp vrrp-group 100 preempt "$VRRP_PREEMPT"

# eth3: NET_VLAN101
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth3 address "10.124.101.${IP_SUFFIX}/24"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth3 vrrp vrrp-group 101 virtual-address 10.124.101.1/24
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth3 vrrp vrrp-group 101 priority "$PRIORITY"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth3 vrrp vrrp-group 101 preempt "$VRRP_PREEMPT"

# eth4: NET_VLAN102
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth4 address "10.124.102.${IP_SUFFIX}/24"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth4 vrrp vrrp-group 102 virtual-address 10.124.102.1/24
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth4 vrrp vrrp-group 102 priority "$PRIORITY"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth4 vrrp vrrp-group 102 preempt "$VRRP_PREEMPT"

# eth5: NET_VLAN103
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth5 address "10.124.103.${IP_SUFFIX}/24"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth5 vrrp vrrp-group 103 virtual-address 10.124.103.1/24
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth5 vrrp vrrp-group 103 priority "$PRIORITY"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth5 vrrp vrrp-group 103 preempt "$VRRP_PREEMPT"

# eth6: NET_VLAN104
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth6 address "10.124.104.${IP_SUFFIX}/24"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth6 vrrp vrrp-group 104 virtual-address 10.124.104.1/24
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth6 vrrp vrrp-group 104 priority "$PRIORITY"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth6 vrrp vrrp-group 104 preempt "$VRRP_PREEMPT"

# --- Roteamento ---
# Default route via vCD Edge
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set protocols static route 0.0.0.0/0 next-hop "$VCD_EDGE_GW"

# --- NAT (Masquerade) para saida internet ---
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set nat source rule 100 description "NAT-SAIDA-Internet"
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set nat source rule 100 outbound-interface eth1
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set nat source rule 100 source address 10.124.0.0/16
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set nat source rule 100 translation address masquerade

# --- Firewall ---
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-IN default-action drop
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-IN rule 10 action accept
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-IN rule 10 state established enable
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-IN rule 10 state related enable

/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL default-action drop
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 10 action accept
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 10 state established enable
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 10 state related enable
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 20 action accept
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 20 protocol icmp
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 30 action accept
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 30 destination port 22
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set firewall name WAN-LOCAL rule 30 protocol tcp
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth1 firewall in name WAN-IN
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper set interfaces ethernet eth1 firewall local name WAN-LOCAL

echo ""
echo "=== Aplicando configuracao ==="
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper commit
/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper save

echo ""
echo "=== $HOSTNAME configurado com sucesso! ==="
echo "IPs configurados:"
echo "  eth0 (NET_IS):     10.124.0.${IP_SUFFIX}/24   (VRRP VIP: 10.124.0.1)"
echo "  eth1 (NET_RT):     10.124.10.${IP_SUFFIX}/30"
echo "  eth2 (VLAN100):    10.124.100.${IP_SUFFIX}/24 (VRRP VIP: 10.124.100.1)"
echo "  eth3 (VLAN101):    10.124.101.${IP_SUFFIX}/24 (VRRP VIP: 10.124.101.1)"
echo "  eth4 (VLAN102):    10.124.102.${IP_SUFFIX}/24 (VRRP VIP: 10.124.102.1)"
echo "  eth5 (VLAN103):    10.124.103.${IP_SUFFIX}/24 (VRRP VIP: 10.124.103.1)"
echo "  eth6 (VLAN104):    10.124.104.${IP_SUFFIX}/24 (VRRP VIP: 10.124.104.1)"
echo "  Default route:     0.0.0.0/0 via $VCD_EDGE_GW"
echo "  Priority VRRP:     $PRIORITY"
