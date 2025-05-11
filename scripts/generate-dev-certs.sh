#!/bin/bash

# Development Certificate Generation Script
# This script generates development certificates for the workload identity system

set -e

# Create certificates directory
CERT_DIR="certs/dev"
mkdir -p "$CERT_DIR"

# Function to generate a certificate
generate_cert() {
    local name=$1
    local subject=$2
    local san=$3
    local days=$4
    
    echo "Generating $name certificate..."
    
    # Generate private key
    openssl genrsa -out "$CERT_DIR/$name.key" 2048
    
    # Generate CSR
    openssl req -new \
        -key "$CERT_DIR/$name.key" \
        -out "$CERT_DIR/$name.csr" \
        -subj "$subject" \
        -addext "subjectAltName=$san" \
        -addext "keyUsage=critical,digitalSignature,keyEncipherment" \
        -addext "extendedKeyUsage=serverAuth,clientAuth"
    
    # Sign certificate
    openssl x509 -req \
        -in "$CERT_DIR/$name.csr" \
        -CA "$CERT_DIR/root-ca.crt" \
        -CAkey "$CERT_DIR/root-ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/$name.crt" \
        -days "$days" \
        -sha256 \
        -extfile <(echo "subjectAltName=$san")
}

echo "Generating development certificates..."

# Generate root CA
echo "Generating root CA..."
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
    -keyout "$CERT_DIR/root-ca.key" \
    -out "$CERT_DIR/root-ca.crt" \
    -subj "/CN=Development Root CA" \
    -addext "basicConstraints=critical,CA:true" \
    -addext "keyUsage=critical,digitalSignature,keyCertSign,cRLSign" \
    -addext "extendedKeyUsage=serverAuth,clientAuth"

# Generate server certificate
generate_cert "server" \
    "/CN=spire-server" \
    "DNS:spire-server,DNS:spire-server.spire,DNS:localhost" \
    365

# Generate agent certificate
generate_cert "agent" \
    "/CN=spire-agent" \
    "DNS:spire-agent,DNS:localhost" \
    365

# Generate workload certificate
generate_cert "workload" \
    "/CN=workload" \
    "DNS:workload,DNS:localhost" \
    365

# Set proper permissions
chmod 600 "$CERT_DIR"/*.key
chmod 644 "$CERT_DIR"/*.crt
chmod 644 "$CERT_DIR"/*.csr
chmod 644 "$CERT_DIR"/*.srl

# Create Kubernetes secrets
echo "Creating Kubernetes secrets..."

# Create namespace if it doesn't exist
kubectl create namespace spire --dry-run=client -o yaml | kubectl apply -f -

# Create server secrets
kubectl create secret generic spire-server-certs \
    --namespace spire \
    --from-file=server.key="$CERT_DIR/server.key" \
    --from-file=server.crt="$CERT_DIR/server.crt" \
    --from-file=root-ca.crt="$CERT_DIR/root-ca.crt" \
    --dry-run=client -o yaml | kubectl apply -f -

# Create agent secrets
kubectl create secret generic spire-agent-certs \
    --namespace spire \
    --from-file=agent.key="$CERT_DIR/agent.key" \
    --from-file=agent.crt="$CERT_DIR/agent.crt" \
    --from-file=root-ca.crt="$CERT_DIR/root-ca.crt" \
    --dry-run=client -o yaml | kubectl apply -f -

echo "Development certificates generated successfully!"
echo "Certificates are stored in: $CERT_DIR"
echo "Kubernetes secrets have been created in the spire namespace." 