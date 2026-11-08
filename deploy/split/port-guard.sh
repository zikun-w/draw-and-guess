#!/usr/bin/env bash
set -euo pipefail
# Only replace this application's table; submit the change atomically.
{
    if nft list table inet drawguess_guard >/dev/null 2>&1; then
        printf 'delete table inet drawguess_guard\n'
    fi
    cat /etc/drawguess/port-guard.nft
} | nft -f -
