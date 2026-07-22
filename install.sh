#!/bin/sh
# Install the bara CLI from GitHub Releases.
#
#   curl -fsSL https://raw.githubusercontent.com/catmargiela/fuckhiggsfield/main/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- --tag v0.1.0 --prefix "$HOME/.local"
set -eu

REPO="catmargiela/fuckhiggsfield"
BIN="bara"
PREFIX="${PREFIX:-/usr/local}"
TAG=""

usage() {
  cat <<EOF
Usage: install.sh [--tag <vX.Y.Z>] [--prefix <dir>]

  --tag     release to install (default: latest)
  --prefix  install root; the binary lands in <prefix>/bin (default: /usr/local)
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --tag) TAG="${2:-}"; shift 2 ;;
    --prefix) PREFIX="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "install.sh: unknown option $1" >&2; usage >&2; exit 1 ;;
  esac
done

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "install.sh: $1 is required" >&2; exit 1; }
}
need uname
need tar

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
else
  echo "install.sh: curl or wget is required" >&2
  exit 1
fi

os=$(uname -s)
case "$os" in
  Darwin) OS=darwin ;;
  Linux) OS=linux ;;
  *) echo "install.sh: unsupported OS $os (use the npm package on Windows)" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "install.sh: unsupported architecture $arch" >&2; exit 1 ;;
esac

if [ -z "$TAG" ]; then
  TAG=$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
  [ -n "$TAG" ] || { echo "install.sh: could not determine the latest release" >&2; exit 1; }
fi

VERSION="${TAG#v}"
ARCHIVE="${BIN}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$TAG/$ARCHIVE"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "Downloading $URL"
fetch_to "$URL" "$TMP/$ARCHIVE" || {
  echo "install.sh: download failed — check that $TAG has a $OS/$ARCH build" >&2
  exit 1
}

# Verify against the release checksums when they are available.
if fetch_to "https://github.com/$REPO/releases/download/$TAG/checksums.txt" "$TMP/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    expected=$(grep " $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')
    actual=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    expected=$(grep " $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')
    actual=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
  else
    expected=""; actual=""
  fi
  if [ -n "$expected" ] && [ "$expected" != "$actual" ]; then
    echo "install.sh: checksum mismatch for $ARCHIVE" >&2
    echo "  expected $expected" >&2
    echo "  actual   $actual" >&2
    exit 1
  fi
  [ -n "$expected" ] && echo "Checksum verified."
fi

tar -xzf "$TMP/$ARCHIVE" -C "$TMP" "$BIN"
chmod +x "$TMP/$BIN"

DEST="$PREFIX/bin"
if mkdir -p "$DEST" 2>/dev/null && [ -w "$DEST" ]; then
  mv "$TMP/$BIN" "$DEST/$BIN"
elif command -v sudo >/dev/null 2>&1; then
  echo "Installing to $DEST (needs sudo)"
  sudo mkdir -p "$DEST"
  sudo mv "$TMP/$BIN" "$DEST/$BIN"
else
  echo "install.sh: cannot write to $DEST — re-run with --prefix \"\$HOME/.local\"" >&2
  exit 1
fi

echo "Installed $BIN $TAG to $DEST/$BIN"
case ":$PATH:" in
  *":$DEST:"*) ;;
  *) echo "Note: $DEST is not on your PATH." ;;
esac
echo "Next: $BIN settings set openai_api_key <key>"
