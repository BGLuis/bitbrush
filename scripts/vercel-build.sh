#!/usr/bin/env bash
set -e

echo "=========================================="
echo "  BitBrush - Build de Produção para Vercel"
echo "=========================================="

ARCH="$(uname -m)"
if [ "$ARCH" = "x86_64" ]; then
  GO_ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
  GO_ARCH="arm64"
else
  GO_ARCH="amd64"
fi

# 1. Garantir Go no PATH (necessário como base tanto para Go quanto para TinyGo)
if ! command -v go &> /dev/null; then
  echo "==> Go não detectado no PATH. Baixando Go portátil..."
  GO_VERSION="1.27.1"
  GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
  mkdir -p /tmp/gobin
  curl -fsSL "https://go.dev/dl/${GO_TAR}" -o "/tmp/${GO_TAR}"
  tar -C /tmp/gobin --strip-components=1 -xzf "/tmp/${GO_TAR}"
  export PATH="/tmp/gobin/bin:$PATH"
fi
echo "==> Go ativo: $(go version)"

# 2. Tentativa de compilação com TinyGo 0.42.0 (otimização de 44% no tamanho: 3.1MB vs 5.5MB)
BUILD_SUCCESS=0
WASM_OUT="web/public/main.wasm"
WASM_EXEC="web/public/wasm_exec.js"
mkdir -p web/public

if ! command -v tinygo &> /dev/null; then
  echo "==> Baixando TinyGo 0.42.0..."
  TINYGO_TAR="tinygo0.42.0.linux-${GO_ARCH}.tar.gz"
  if curl -fsSL "https://github.com/tinygo-org/tinygo/releases/download/v0.42.0/${TINYGO_TAR}" -o "/tmp/${TINYGO_TAR}"; then
    rm -rf /tmp/tinygo
    mkdir -p /tmp/tinygo
    tar -C /tmp/tinygo --strip-components=1 -xzf "/tmp/${TINYGO_TAR}"
    export PATH="/tmp/tinygo/bin:$PATH"
  fi
fi

if command -v tinygo &> /dev/null; then
  echo "==> Compilando WASM com TinyGo ($(tinygo version))..."
  if tinygo build -o "$WASM_OUT" -target=wasm ./cmd/wasm; then
    TINY_ROOT="$(tinygo env TINYGOROOT)"
    cp "${TINY_ROOT}/targets/wasm_exec.js" "$WASM_EXEC"
    echo "==> [SUCESSO] WASM gerado com TinyGo! Tamanho: $(du -h "$WASM_OUT" | cut -f1)"
    BUILD_SUCCESS=1
  else
    echo "==> [AVISO] Compilação TinyGo falhou. Acionando fallback para Go padrão..."
  fi
fi

# 3. Fallback para Go padrão se o TinyGo não completou
if [ "$BUILD_SUCCESS" -ne 1 ]; then
  echo "==> Compilando WASM com Go padrão..."
  GOOS=js GOARCH=wasm go build -o "$WASM_OUT" ./cmd/wasm
  GOROOT_DIR="$(go env GOROOT)"
  if [ -f "${GOROOT_DIR}/lib/wasm/wasm_exec.js" ]; then
    cp "${GOROOT_DIR}/lib/wasm/wasm_exec.js" "$WASM_EXEC"
  else
    cp "${GOROOT_DIR}/misc/wasm/wasm_exec.js" "$WASM_EXEC"
  fi
  echo "==> [SUCESSO] WASM gerado com Go padrão! Tamanho: $(du -h "$WASM_OUT" | cut -f1)"
fi

# 4. Compilar Frontend Svelte 5 / Vite 8
echo "==> Instalando dependências e compilando frontend estático..."
cd web
npm install
npm run build

echo "=========================================="
echo "  Build finalizado com sucesso em web/dist!"
echo "=========================================="
