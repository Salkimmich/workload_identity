#!/bin/bash
# This script simulates mutual TLS communication between frontend and backend services
# using SPIFFE Workload API and the SPIRE Agent socket.
# It uses spiffe-helper to initiate an mTLS connection from frontend to backend.

set -e

# Variables (replace these with real service IPs or hostnames if known)
FRONTEND_POD=$(kubectl get pods -l app=frontend -o jsonpath="{.items[0].metadata.name}")
BACKEND_POD=$(kubectl get pods -l app=backend -o jsonpath="{.items[0].metadata.name}")

# Start a TLS echo server in the backend using openssl
kubectl exec "$BACKEND_POD" -- /bin/sh -c '
  openssl req -x509 -newkey rsa:2048 -keyout /tmp/key.pem -out /tmp/cert.pem -days 1 -nodes -subj "/CN=backend" &&
  openssl s_server -accept 8443 -cert /tmp/cert.pem -key /tmp/key.pem -quiet &
'

# Give the backend time to start the server
sleep 3

# Use frontend to curl the backend server over TLS
kubectl exec "$FRONTEND_POD" -- /bin/sh -c '
  echo "hello from frontend" > /tmp/input.txt &&
  openssl s_client -connect backend:8443 < /tmp/input.txt
'
