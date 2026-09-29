# Melhorias pendentes — unicd CLI

## 1. `unicd --version` como flag global ✅ (implementado)

**Problema:** `unicd --version` não funcionava, apenas `unicd version` (subcomando).

**Solução:** Adicionado campo `Version` no `rootCmd` do cobra, permitindo `unicd --version`.

---

## 2. Teste de conectividade de porta ✅ (implementado)

**Problema:** Não havia como testar se uma porta estava aberta/accessível diretamente pela CLI.

**Solução:** Adicionado comando `unicd debug port-check <host> <port>` para verificar conectividade TCP.

---

## 3. Detalhes do edge gateway (IP público) ✅ (implementado)

**Problema:** `unicd edge show` não exibia o IP público nem as interfaces de rede do edge gateway.

**Solução:** Adicionado comando `unicd edge interfaces` para listar interfaces, IPs e subnets do edge.

---

## 4. Comando `unicd fw show` ✅ (implementado)

**Problema:** Não havia comando para ver detalhes de uma regra de firewall específica.

**Solução:** Adicionado `unicd fw show -n <nome>` para exibir detalhes completos de uma regra.

---

## 5. Comando `unicd nat show` ✅ (implementado)

**Problema:** Não havia comando para ver detalhes de uma regra NAT específica.

**Solução:** Adicionado `unicd nat show <rule-id>` para exibir detalhes completos de uma regra.

---

## 6. Regras de firewall — especificar porta/protocolo

**Problema:** A regra `RDP_ALLOW_DENCK` foi criada sem especificar porta/protocolo (`applicationPortProfiles: null`), o que pode causar comportamento inesperado.

**Sugestão:** Garantir que `fw create` sempre exija `--port` e `--protocol` para regras de entrada, ou alertar o usuário quando não forem especificados.

---

## 7. Validação de IP público em regras NAT

**Problema:** Não há validação se o IP público informado em regras NAT/SNAT pertence realmente ao edge gateway.

**Sugestão:** Validar o IP público contra as interfaces do edge antes de criar regras.

---

## 8. Teste de regra de firewall/NAT

**Problema:** Não há como validar se uma regra de firewall ou NAT está funcionando corretamente após criação.

**Sugestão:** Adicionar `unicd debug test-rule <tipo> <nome/ID>` para testar conectividade através da regra.

---

## 9. Output formatado para scripts (JSON/YAML)

**Problema:** Alguns comandos não suportam output em JSON/YAML para automação.

**Sugestão:** Adicionar flags `--json` e `--yaml` a todos os comandos de listagem e show.

---

## 10. Verificação de capacidades da CLI

**Status:** Verificado que a CLI já suporta:
- ✅ Criar NAT (DNAT/SNAT)
- ✅ Criar redes
- ✅ Deletar redes
- ✅ Editar redes
- ✅ Criar vApp
- ✅ Deletar vApp
- ✅ Aumentar/diminuir recursos da VM (memória e CPU)
- ✅ Deletar interface da VM
- ✅ Adicionar novas interfaces na VM

**Pendente:**
- Criar/Deletar/Editar Edge (não implementado)
- Ler tasks (não implementado)
- Editar catálogo (não implementado)
- Ver IPs públicos livres e usados (não implementado)
- Setar interface na VM ou vApp (parcialmente implementado)
--------------------------------------------------------------------------------------------
# EXECUTE TODAS AS MELHORIAS ACIMA! A CLI "unicd" DEVE SER COMPLETA E CAPAZ DE TUDO!
# CONFIRMAR TAMBME SE A CLI PODE:
- CRIAR NAT
- CRIAR SNAT
- CRIAR REDES
- DELETAR REDES
- EDITAR REDES
- CRIAR EDGE
- DELETAR EDGE
- EDITAR EDGE
- LER AS TASKS
- EDITAR CATALOGO
- VER OS IPS PUBLICOS LIVRES E USADOS!
- CRIAR VAPP
- DELETAR VAPP
- SETAR UMA INTERFACE NA VM OU NO VAPP
- AUMENTAR OU DOMINUI RECURSOS DA VM (MEMORIA E CPU)
- DELETAR A INTERFACE DA VM, ADD NOVAS INTERFACES NA VM!
