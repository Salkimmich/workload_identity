#!/bin/bash
# verify-all.sh
# This script verifies the status of SPIFFE-related components and workloads

set -e

echo "=== Verifying SPIFFE mTLS Environment ==="

echo "--- Checking SPIRE Server and Agent pods ---"
kubectl get pods -l app=spire-server
kubectl get pods -l app=spire-agent

echo "--- Checking Workload Pods ---"
kubectl get pods -l app=frontend
kubectl get pods -l app=backend

echo "--- Verifying SPIRE Agent Socket Mount ---"
kubectl exec $(kubectl get pod -l app=frontend -o jsonpath='{.items[0].metadata.name}') -- ls /run/spire/sockets

echo "--- Listing SPIRE Registration Entries ---"
kubectl exec -it $(kubectl get pod -l app=spire-server -o jsonpath='{.items[0].metadata.name}') -- /opt/spire/bin/spire-server entry show

echo "--- Checking SPIFFE IDs via SPIRE Workload API ---"
kubectl exec $(kubectl get pod -l app=frontend -o jsonpath='{.items[0].metadata.name}') -- curl -s --unix-socket /run/spire/sockets/agent.sock http://localhost/ | jq .

echo "=== Verification Complete ==="
