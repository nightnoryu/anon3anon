#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 || $1 != --yes ]]; then
	printf 'Usage: %s --yes BACKUP_FILE\n' "$0" >&2
	exit 2
fi

command -v kubectl >/dev/null || { echo 'kubectl is required' >&2; exit 1; }

backup=$2
if [[ ! -f $backup || ! -r $backup || ! -s $backup ]]; then
	printf 'Cannot read backup: %s\n' "$backup" >&2
	exit 1
fi
namespace=${KUBE_NAMESPACE:-anon3anon}
deployment=deployment/anon3anon
replicas=$(kubectl -n "$namespace" get "$deployment" -o jsonpath='{.spec.replicas}')
if [[ ! $replicas =~ ^[0-9]+$ || $replicas -ne 1 ]]; then
	echo 'Expected an anon3anon deployment with one replica' >&2
	exit 1
fi
image=$(kubectl -n "$namespace" get "$deployment" -o jsonpath='{.spec.template.spec.containers[?(@.name=="anon3anon")].image}')
if [[ -z $image ]]; then
	echo 'Could not find the anon3anon container image' >&2
	exit 1
fi

pod="anon3anon-restore-$(date +%s)-${RANDOM}"
pod_created=false
scaled_down=false
cleanup() {
	if [[ $pod_created == true ]]; then
		kubectl -n "$namespace" delete pod "$pod" --ignore-not-found --wait=true >&2 || true
	fi
	if [[ $scaled_down == true ]]; then
		echo 'Restore did not finish; deployment is left scaled to zero for inspection' >&2
	fi
}

kubectl -n "$namespace" scale "$deployment" --replicas=0
scaled_down=true
trap cleanup EXIT

for ((attempt = 0; attempt < 60; attempt++)); do
	if [[ -z $(kubectl -n "$namespace" get pods -l app=anon3anon -o name) ]]; then
		break
	fi
	sleep 2
done
if [[ -n $(kubectl -n "$namespace" get pods -l app=anon3anon -o name) ]]; then
	echo 'Timed out waiting for the app pod to stop' >&2
	exit 1
fi

kubectl -n "$namespace" create -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $pod
spec:
  restartPolicy: Never
  securityContext:
    runAsUser: 10001
    runAsGroup: 10001
    fsGroup: 10001
  containers:
    - name: restore
      image: $image
      command: ["sleep", "3600"]
      volumeMounts:
        - name: data
          mountPath: /data
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: anon3anon-data
EOF
pod_created=true
kubectl -n "$namespace" wait --for=condition=Ready "pod/$pod" --timeout=120s
kubectl -n "$namespace" cp "$backup" "$pod:/data/anon3anon.db.restore" -c restore

# The old connection is closed. Drop its WAL files before installing the backup.
kubectl -n "$namespace" exec "$pod" -c restore -- sh -ec '
	test -s /data/anon3anon.db.restore
	if [ "$(sqlite3 /data/anon3anon.db.restore "PRAGMA integrity_check;")" != ok ]; then
		echo "Transferred backup failed SQLite integrity check" >&2
		exit 1
	fi
	rm -f /data/anon3anon.db-wal /data/anon3anon.db-shm
	mv /data/anon3anon.db.restore /data/anon3anon.db
'

kubectl -n "$namespace" delete pod "$pod" --wait=true
pod_created=false
kubectl -n "$namespace" scale "$deployment" --replicas="$replicas"
scaled_down=false
kubectl -n "$namespace" rollout status "$deployment" --timeout=300s
echo 'Database restored and deployment is ready'
