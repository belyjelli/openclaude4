#!/usr/bin/env bash
# Interactive picker: choose a .gz in the same directory as this script, decompress,
# install into /usr/local/bin with mode 0755.
# Only lists archives for this host OS/arch (e.g. *_darwin_arm64.gz on Apple silicon).
# On macOS, removes com.apple.quarantine from the installed file when present.
# Usage: bash install-from-gz.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

shopt -s nullglob
gz_candidates=("$SCRIPT_DIR"/*.gz)
shopt -u nullglob

if [[ ${#gz_candidates[@]} -eq 0 ]]; then
  printf 'No .gz files found in %s\n' "$SCRIPT_DIR" >&2
  exit 1
fi

detect_os() {
  case "$(uname -s)" in
    Darwin) printf '%s\n' darwin ;;
    Linux) printf '%s\n' linux ;;
    MINGW* | MSYS* | CYGWIN*) printf '%s\n' windows ;;
    *) printf '%s\n' unknown ;;
  esac
}

detect_arch() {
  local a
  a="$(uname -m)"
  case "$a" in
    arm64 | aarch64) printf '%s\n' arm64 ;;
    x86_64 | amd64) printf '%s\n' amd64 ;;
    *) printf '%s\n' "$a" ;;
  esac
}

CURRENT_OS="$(detect_os)"
CURRENT_ARCH="$(detect_arch)"

matches_this_machine() {
  local base="$1"
  case "${CURRENT_OS}/${CURRENT_ARCH}" in
    darwin/arm64)
      [[ "$base" == *"_darwin_arm64"* ]]
      ;;
    darwin/amd64)
      [[ "$base" == *"_darwin_amd64"* ]]
      ;;
    linux/arm64)
      [[ "$base" == *"_linux_arm64"* ]]
      ;;
    linux/amd64)
      [[ "$base" == *"_linux_amd64"* ]]
      ;;
    windows/amd64)
      [[ "$base" == *"_windows_amd64"* ]] || [[ "$base" == *"_windows_amd64.exe"* ]]
      ;;
    *)
      return 1
      ;;
  esac
}

gz_files=()
for c in "${gz_candidates[@]}"; do
  base="$(basename "$c" .gz)"
  if matches_this_machine "$base"; then
    gz_files+=("$c")
  fi
done

if [[ ${#gz_files[@]} -eq 0 ]]; then
  printf 'No .gz for this host (%s/%s) in %s\n' "$CURRENT_OS" "$CURRENT_ARCH" "$SCRIPT_DIR" >&2
  exit 1
fi

sorted_gz=()
while IFS= read -r line; do
  [[ -n "$line" ]] && sorted_gz+=("$line")
done < <(printf '%s\n' "${gz_files[@]}" | sort)
gz_files=("${sorted_gz[@]}")

file_desc() {
  if command -v file >/dev/null 2>&1; then
    file -b "$1" 2>/dev/null || true
  fi
}

install_dest_basename() {
  local gz_path="$1" tmpbin="$2" stem desc
  stem="$(basename "$gz_path" .gz)"
  stem="${stem%%_*}"
  [[ -z "$stem" ]] && stem="occli"

  desc="$(file_desc "$tmpbin")"
  case "$desc" in
    *PE32* | *PE\ executable*)
      if [[ "$CURRENT_OS" == windows ]]; then
        printf '%s\n' "${stem}.exe"
      else
        printf '%s\n' ""
      fi
      ;;
    *)
      printf '%s\n' "$stem"
      ;;
  esac
}

need_sudo() {
  local dir dest
  dest="$1"
  dir="$(dirname "$dest")"
  [[ -w "$dir" ]] || return 0
  [[ -e "$dest" && ! -w "$dest" ]] && return 0
  return 1
}

strip_quarantine_macos() {
  local dest="$1"
  [[ "$(uname -s)" != Darwin ]] && return 0
  if need_sudo "$dest"; then
    sudo xattr -d com.apple.quarantine "$dest" 2>/dev/null || true
  else
    xattr -d com.apple.quarantine "$dest" 2>/dev/null || true
  fi
}

printf 'Showing .gz for this host (%s/%s) in %s\n\n' "$CURRENT_OS" "$CURRENT_ARCH" "$SCRIPT_DIR"

n=${#gz_files[@]}
for i in "${!gz_files[@]}"; do
  idx=$((i + 1))
  printf '%2d) %s\n' "$idx" "$(basename "${gz_files[i]}")"
done
printf '%2d) Quit\n' "$((n + 1))"

gz_path=""
while true; do
  read -r -p "Enter number: " choice || exit 1
  if [[ "$choice" == "$((n + 1))" ]] || [[ "$choice" == [Qq]* ]]; then
    printf 'Cancelled.\n'
    exit 0
  fi
  if [[ "$choice" =~ ^[0-9]+$ ]] && ((choice >= 1 && choice <= n)); then
    gz_path="${gz_files[$((choice - 1))]}"
    break
  fi
  printf 'Invalid choice; enter 1–%d (or Q to quit).\n' "$((n + 1))"
done

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/occli-gz.XXXXXX")"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

tmpbin="$tmpdir/extracted"
gzip -dc -- "$gz_path" >"$tmpbin"
chmod 755 "$tmpbin"

dest_name="$(install_dest_basename "$gz_path" "$tmpbin")"
if [[ -z "$dest_name" ]]; then
  printf 'This archive is a Windows PE binary; pick a %s/%s build on this host.\n' "$CURRENT_OS" "$CURRENT_ARCH" >&2
  exit 1
fi

dest="/usr/local/bin/$dest_name"
printf '\nInstalling to %s\n' "$dest"

if need_sudo "$dest"; then
  sudo cp -f "$tmpbin" "$dest"
  sudo chmod 755 "$dest"
else
  cp -f "$tmpbin" "$dest"
  chmod 755 "$dest"
fi

strip_quarantine_macos "$dest"

printf 'Done: %s (%s)\n' "$dest" "$(file_desc "$dest" | tr '\n' ' ')"
