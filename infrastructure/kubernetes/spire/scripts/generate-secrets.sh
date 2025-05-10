#!/bin/bash

# SPIRE Secrets Generation Script
# This script generates the necessary secrets for SPIRE deployment, including:
# - Root CA certificate and key
# - Server certificate and key
# - Agent certificate and key
# It also creates the SPIRE namespace and cleans up temporary files.

# Exit on any error
set -e

# Create SPIRE namespace if it doesn't exist
echo "Creating SPIRE namespace..."
kubectl create namespace spire --dry-run=client -o yaml | kubectl apply -f -

# Generate root CA
echo "Generating root CA..."
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout root-ca.key -out root-ca.crt \
  -subj "/CN=SPIRE Root CA" \
  -addext "basicConstraints=critical,CA:true"

# Generate server key and CSR
echo "Generating server key and CSR..."
openssl req -newkey rsa:2048 -nodes -keyout server.key \
  -out server.csr \
  -subj "/CN=spire-server" \
  -addext "subjectAltName=DNS:spire-server,DNS:spire-server.spire"

# Sign server certificate
echo "Signing server certificate..."
openssl x509 -req -in server.csr \
  -CA root-ca.crt -CAkey root-ca.key \
  -CAcreateserial -out server.crt \
  -days 365 -sha256 \
  -extfile <(echo "subjectAltName=DNS:spire-server,DNS:spire-server.spire")

# Generate agent key and CSR
echo "Generating agent key and CSR..."
openssl req -newkey rsa:2048 -nodes -keyout agent.key \
  -out agent.csr \
  -subj "/CN=spire-agent" \
  -addext "subjectAltName=DNS:spire-agent"

# Sign agent certificate
echo "Signing agent certificate..."
openssl x509 -req -in agent.csr \
  -CA root-ca.crt -CAkey root-ca.key \
  -CAcreateserial -out agent.crt \
  -days 365 -sha256 \
  -extfile <(echo "subjectAltName=DNS:spire-agent")

# Create server secrets
echo "Creating server secrets..."
kubectl create secret generic spire-server-certs \
  --namespace spire \
  --from-file=server.key=server.key \
  --from-file=server.crt=server.crt \
  --from-file=root-ca.crt=root-ca.crt \
  --dry-run=client -o yaml | kubectl apply -f -

# Create agent secrets
echo "Creating agent secrets..."
kubectl create secret generic spire-agent-certs \
  --namespace spire \
  --from-file=agent.key=agent.key \
  --from-file=agent.crt=agent.crt \
  --from-file=root-ca.crt=root-ca.crt \
  --dry-run=client -o yaml | kubectl apply -f -

# Clean up temporary files
echo "Cleaning up temporary files..."
rm -f *.key *.crt *.csr *.srl

echo "SPIRE secrets generation complete!" 