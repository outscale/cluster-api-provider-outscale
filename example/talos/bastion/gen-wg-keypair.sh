#!/bin/bash
set -eu

privkey="$(wg genkey)"
pubkey="$(wg pubkey <<<"$privkey")"

cat <<EOF
{
	"privkey": "$privkey",
	"pubkey": "$pubkey"
}
EOF
