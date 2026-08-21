#!/usr/bin/env bash
# Installs golangci-lint into ./bin, scoped to this repo — not a system-wide
# install. Re-run to (re)install, or set GOLANGCI_LINT_VERSION to pin a
# different release.
#
# Downloads directly from GitHub releases and verifies the sha256 checksum
# itself (exact filename match) rather than delegating to golangci-lint's
# own install.sh: for this release, that script's checksum lookup matches
# the wrong line in checksums.txt (the archive's .sbom.json entry instead of
# the archive itself, since one filename is a prefix of the other) and fails
# verification of an otherwise-correct download.
set -euo pipefail

VERSION="${GOLANGCI_LINT_VERSION:-v2.12.2}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${REPO_ROOT}/bin"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "${arch}" in
  x86_64) arch="amd64" ;;
  aarch64) arch="arm64" ;;
esac

version_num="${VERSION#v}"
archive="golangci-lint-${version_num}-${os}-${arch}.tar.gz"
base_url="https://github.com/golangci/golangci-lint/releases/download/${VERSION}"

echo "downloading ${archive}..."
curl -sSfL -o "${WORK_DIR}/${archive}" "${base_url}/${archive}"
curl -sSfL -o "${WORK_DIR}/checksums.txt" "${base_url}/golangci-lint-${version_num}-checksums.txt"

expected_line="$(awk -v f="${archive}" '$2 == f' "${WORK_DIR}/checksums.txt")"
if [ -z "${expected_line}" ]; then
  echo "error: no checksum entry for ${archive} in checksums.txt" >&2
  exit 1
fi

echo "verifying checksum..."
(cd "${WORK_DIR}" && shasum -a 256 -c <(echo "${expected_line}"))

mkdir -p "${BIN_DIR}"
tar -xzf "${WORK_DIR}/${archive}" -C "${WORK_DIR}"
cp "${WORK_DIR}/golangci-lint-${version_num}-${os}-${arch}/golangci-lint" "${BIN_DIR}/golangci-lint"
chmod +x "${BIN_DIR}/golangci-lint"

echo "installed: ${BIN_DIR}/golangci-lint"
"${BIN_DIR}/golangci-lint" version
