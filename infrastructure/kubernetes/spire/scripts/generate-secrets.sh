#!/bin/bash

# SPIRE Secrets Generation Script
# This file defines the Kubernetes DaemonSet for the SPIRE agent.
# The SPIRE agent runs on every node in the cluster and provides workload identity to pods.

# Exit on any error
# This ensures the script stops if any command fails
set -e

# Optional improvement: Create backup directory
# Rationale: Provides a backup of generated certificates for disaster recovery
# Note: Ensure backup directory has proper security permissions
BACKUP_DIR="spire-secrets-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Create SPIRE namespace if it doesn't exist
# The namespace is required for SPIRE components
echo "Creating SPIRE namespace..."
kubectl create namespace spire --dry-run=client -o yaml | kubectl apply -f -

# Generate root CA
# The root CA is the trust anchor for the entire SPIRE deployment
# It signs all other certificates in the system
# Required: These settings are mandatory for security
echo "Generating root CA..."
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout root-ca.key -out root-ca.crt \
  -subj "/CN=SPIRE Root CA" \
  -addext "basicConstraints=critical,CA:true" \
  -addext "keyUsage=critical,digitalSignature,keyCertSign,cRLSign" \
  -addext "extendedKeyUsage=serverAuth,clientAuth"

# Generate server key and CSR
# The server certificate is used for agent-server communication
# It must be valid for the server's DNS names
# Required: These settings are mandatory for security
echo "Generating server key and CSR..."
openssl req -newkey rsa:2048 -nodes -keyout server.key \
  -out server.csr \
  -subj "/CN=spire-server" \
  -addext "subjectAltName=DNS:spire-server,DNS:spire-server.spire" \
  -addext "keyUsage=critical,digitalSignature,keyEncipherment" \
  -addext "extendedKeyUsage=serverAuth,clientAuth"

# Sign server certificate
# The server certificate is signed by the root CA
# It's valid for 365 days
# Required: These settings are mandatory for security
echo "Signing server certificate..."
openssl x509 -req -in server.csr \
  -CA root-ca.crt -CAkey root-ca.key \
  -CAcreateserial -out server.crt \
  -days 365 -sha256 \
  -extfile <(echo "subjectAltName=DNS:spire-server,DNS:spire-server.spire")

# Generate agent key and CSR
# The agent certificate is used for agent-server communication
# It must be valid for the agent's DNS name
# Required: These settings are mandatory for security
echo "Generating agent key and CSR..."
openssl req -newkey rsa:2048 -nodes -keyout agent.key \
  -out agent.csr \
  -subj "/CN=spire-agent" \
  -addext "subjectAltName=DNS:spire-agent" \
  -addext "keyUsage=critical,digitalSignature,keyEncipherment" \
  -addext "extendedKeyUsage=serverAuth,clientAuth"

# Sign agent certificate
# The agent certificate is signed by the root CA
# It's valid for 365 days
# Required: These settings are mandatory for security
echo "Signing agent certificate..."
openssl x509 -req -in agent.csr \
  -CA root-ca.crt -CAkey root-ca.key \
  -CAcreateserial -out agent.crt \
  -days 365 -sha256 \
  -extfile <(echo "subjectAltName=DNS:spire-agent")

# Optional improvement: Generate CRL
# Rationale: Enables certificate revocation for compromised certificates
# Note: Requires periodic CRL updates and distribution
echo "Generating CRL..."
openssl ca -config <(echo "[ca]
default_ca = CA_default
[CA_default]
dir = .
certs = .
crl_dir = .
database = index.txt
new_certs_dir = .
certificate = root-ca.crt
serial = serial
crl = crl.pem
private_key = root-ca.key
RANDFILE = .rand
x509_extensions = usr_cert
name_opt = ca_default
cert_opt = ca_default
default_days = 365
default_crl_days = 30
default_md = sha256
preserve = no
policy = policy_anything
[policy_anything]
countryName = optional
stateOrProvinceName = optional
localityName = optional
organizationName = optional
organizationalUnitName = optional
commonName = supplied
emailAddress = optional
[usr_cert]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment") \
  -gencrl -out crl.pem

# Create server secrets
# The server secrets are stored in a Kubernetes secret
# They include the server's key, certificate, and the root CA
# Required: These settings are mandatory for security
echo "Creating server secrets..."
kubectl create secret generic spire-server-certs \
  --namespace spire \
  --from-file=server.key=server.key \
  --from-file=server.crt=server.crt \
  --from-file=root-ca.crt=root-ca.crt \
  --from-file=crl.pem=crl.pem \
  --dry-run=client -o yaml | kubectl apply -f -

# Create agent secrets
# The agent secrets are stored in a Kubernetes secret
# They include the agent's key, certificate, and the root CA
# Required: These settings are mandatory for security
echo "Creating agent secrets..."
kubectl create secret generic spire-agent-certs \
  --namespace spire \
  --from-file=agent.key=agent.key \
  --from-file=agent.crt=agent.crt \
  --from-file=root-ca.crt=root-ca.crt \
  --from-file=crl.pem=crl.pem \
  --dry-run=client -o yaml | kubectl apply -f -

# Optional improvement: Backup generated files
# Rationale: Provides a backup of generated certificates for disaster recovery
# Note: Ensure backup directory has proper security permissions
echo "Backing up generated files..."
cp *.key *.crt *.csr *.srl *.pem "$BACKUP_DIR/"

# Clean up temporary files
# Remove all generated files to prevent sensitive data exposure
# Required: This is mandatory for security
echo "Cleaning up temporary files..."
rm -f *.key *.crt *.csr *.srl *.pem

echo "SPIRE secrets generation complete!"
echo "Backup of certificates stored in: $BACKUP_DIR" 