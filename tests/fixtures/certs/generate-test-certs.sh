#!/bin/bash

# Exit on any error
set -e

# Default values
OUTPUT_DIR="./certs"
VALIDITY=365
CA_NAME="Test CA"
ORGANIZATION="Test Org"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --validity)
            VALIDITY="$2"
            shift 2
            ;;
        --ca-name)
            CA_NAME="$2"
            shift 2
            ;;
        --organization)
            ORGANIZATION="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Generate root CA
echo "Generating root CA..."
openssl req -x509 -newkey rsa:4096 -sha256 -days "$VALIDITY" -nodes \
    -keyout "$OUTPUT_DIR/root-ca.key" \
    -out "$OUTPUT_DIR/root-ca.crt" \
    -subj "/CN=$CA_NAME/O=$ORGANIZATION" \
    -addext "basicConstraints=critical,CA:true" \
    -addext "keyUsage=critical,digitalSignature,keyCertSign,cRLSign"

# Generate intermediate CA
echo "Generating intermediate CA..."
openssl req -newkey rsa:2048 -nodes \
    -keyout "$OUTPUT_DIR/intermediate-ca.key" \
    -out "$OUTPUT_DIR/intermediate-ca.csr" \
    -subj "/CN=Intermediate CA/O=$ORGANIZATION"

openssl x509 -req -in "$OUTPUT_DIR/intermediate-ca.csr" \
    -CA "$OUTPUT_DIR/root-ca.crt" \
    -CAkey "$OUTPUT_DIR/root-ca.key" \
    -CAcreateserial \
    -out "$OUTPUT_DIR/intermediate-ca.crt" \
    -days "$VALIDITY" \
    -sha256 \
    -extfile <(echo "basicConstraints=critical,CA:true,pathlen:0")

# Generate workload identity certificate
echo "Generating workload identity certificate..."
openssl req -newkey rsa:2048 -nodes \
    -keyout "$OUTPUT_DIR/workload.key" \
    -out "$OUTPUT_DIR/workload.csr" \
    -subj "/CN=workload-identity/O=$ORGANIZATION" \
    -addext "subjectAltName=DNS:workload-identity,DNS:workload-identity.default.svc.cluster.local" \
    -addext "extendedKeyUsage=serverAuth,clientAuth"

openssl x509 -req -in "$OUTPUT_DIR/workload.csr" \
    -CA "$OUTPUT_DIR/intermediate-ca.crt" \
    -CAkey "$OUTPUT_DIR/intermediate-ca.key" \
    -CAcreateserial \
    -out "$OUTPUT_DIR/workload.crt" \
    -days "$VALIDITY" \
    -sha256 \
    -extfile <(echo "subjectAltName=DNS:workload-identity,DNS:workload-identity.default.svc.cluster.local")

# Generate certificate chain
echo "Generating certificate chain..."
cat "$OUTPUT_DIR/workload.crt" \
    "$OUTPUT_DIR/intermediate-ca.crt" \
    "$OUTPUT_DIR/root-ca.crt" > "$OUTPUT_DIR/workload-chain.crt"

# Set proper permissions
chmod 600 "$OUTPUT_DIR"/*.key
chmod 644 "$OUTPUT_DIR"/*.crt

echo "Certificate generation complete!"
echo "Certificates are stored in: $OUTPUT_DIR" 