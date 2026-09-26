#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 0 ]]; then
	printf 'Usage: %s\n' "$0" >&2
	exit 2
fi

command -v kubectl >/dev/null || { echo 'kubectl is required' >&2; exit 1; }

namespace=${KUBE_NAMESPACE:-anon3anon}
output="anon3anon-$(date -u +%Y%m%dT%H%M%SZ).db"
if [[ -e $output ]]; then
	printf 'Backup already exists: %s\n' "$output" >&2
	exit 1
fi

pods=$(kubectl -n "$namespace" get pods -l app=anon3anon --field-selector=status.phase=Running -o name)
if [[ $(printf '%s\n' "$pods" | sed '/^$/d' | wc -l) -ne 1 ]]; then
	echo 'Expected exactly one running anon3anon pod' >&2
	exit 1
fi

umask 077
tmp=$(mktemp "${output}.tmp.XXXXXXXX")
trap 'rm -f "$tmp"' EXIT

# The app uses WAL mode, so a plain copy of the main database file can omit writes.
kubectl -n "$namespace" exec "$pods" -c anon3anon -- sh -ec '
	tmp=$(mktemp /data/anon3anon-backup.XXXXXXXX)
	trap '\''rm -f "$tmp"'\'' EXIT
	sqlite3 /data/anon3anon.db ".backup $tmp"
	if [ "$(sqlite3 "$tmp" "PRAGMA integrity_check;")" != ok ]; then
		echo "Backup failed SQLite integrity check" >&2
		exit 1
	fi
	cat "$tmp"
' > "$tmp"

if [[ ! -s $tmp ]]; then
	echo 'Backup is empty' >&2
	exit 1
fi
ln "$tmp" "$output"
printf 'Saved backup to %s\n' "$output"
