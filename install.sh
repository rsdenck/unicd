#!/usr/bin/env bash
#
# Instalador da unicd — Unifique Cloud CLI
#
# Uso:
#   curl -fsSL https://raw.githubusercontent.com/rsdenck/unicd/main/install.sh | bash
#   ./install.sh [--version v1.2.0] [--dir /usr/local/bin]
#
set -euo pipefail

REPO="rsdenck/unicd"
BIN="unicd"
INSTALL_DIR="/usr/local/bin"
VERSION=""

info()  { printf '\033[0;32m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[0;33m!!\033[0m %s\n' "$*" >&2; }
fail()  { printf '\033[0;31mxx\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Instalador da ${BIN} — Unifique Cloud CLI

Opcoes:
  --version <vX.Y.Z>   Versao especifica (default: ultima release)
  --dir <caminho>      Diretorio de instalacao (default: ${INSTALL_DIR})
  -h, --help           Exibir esta ajuda
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift 2 ;;
    --dir)     INSTALL_DIR="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) fail "Opcao desconhecida: $1" ;;
  esac
done

need() { command -v "$1" >/dev/null 2>&1 || fail "Dependencia ausente: $1"; }
need curl
need tar

# Detecta sistema operacional e arquitetura
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux)  os="linux" ;;
  darwin) os="darwin" ;;
  *) fail "Sistema operacional nao suportado: $os" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) fail "Arquitetura nao suportada: $arch" ;;
esac

# Resolve a versao
if [ -z "$VERSION" ]; then
  info "Buscando a ultima versao..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -oE '"tag_name"[[:space:]]*:[[:space:]]*"v[^"]+"' \
    | head -n1 | sed -E 's/.*"(v[^"]+)"/\1/')"
  [ -n "$VERSION" ] || fail "Nao foi possivel determinar a ultima versao. Use --version."
fi
VERSION="${VERSION#v}"

ext=""
[ "$os" = "windows" ] && ext=".exe"
asset="${BIN}_${VERSION}_${os}_${arch}${ext}"
url="https://github.com/${REPO}/releases/download/v${VERSION}/${asset}"

info "Baixando ${BIN} ${VERSION} (${os}/${arch})..."
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
curl -fL "$url" -o "${tmp}/${BIN}"

install_cmd="install"
if command -v install >/dev/null 2>&1; then
  if [ -w "$INSTALL_DIR" ]; then
    install -m 0755 "${tmp}/${BIN}" "${INSTALL_DIR}/${BIN}"
  elif command -v sudo >/dev/null 2>&1; then
    sudo install -m 0755 "${tmp}/${BIN}" "${INSTALL_DIR}/${BIN}"
  else
    fail "Sem permissao para escrever em ${INSTALL_DIR}. Use --dir ou execute com sudo."
  fi
else
  mkdir -p "$INSTALL_DIR"
  cp "${tmp}/${BIN}" "${INSTALL_DIR}/${BIN}"
  chmod 0755 "${INSTALL_DIR}/${BIN}"
fi

info "Instalado em ${INSTALL_DIR}/${BIN}"
"${INSTALL_DIR}/${BIN}" --no-banner version || warn "Falha ao executar a versao instalada."
