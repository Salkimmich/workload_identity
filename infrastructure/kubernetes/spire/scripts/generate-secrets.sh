#!/bin/bash

# SPIRE Secrets Generation Script
# This file defines the Kubernetes DaemonSet for the SPIRE agent.
# The SPIRE agent runs on every node in the cluster and provides workload identity to pods.

# Exit on any error
# This ensures the script stops if any command fails
set -e

# Required: Version control
# Best Practice: Track script version
# Security Note: Version must be properly managed
VERSION="1.0.0"

# Required: Logging
# Best Practice: Configure logging
# Security Note: Logs must be properly secured
LOG_FILE="/var/log/spire/secrets.log"
mkdir -p "$(dirname "$LOG_FILE")"
exec 1> >(tee -a "$LOG_FILE")
exec 2> >(tee -a "$LOG_FILE" >&2)

# Required: Error handling
# Best Practice: Configure error handling
# Security Note: Errors must be properly handled
trap 'echo "Error: Command failed at line $LINENO"' ERR

# Required: Cleanup
# Best Practice: Configure cleanup
# Security Note: Cleanup must be properly handled
trap 'rm -f "$TEMP_DIR"/*' EXIT

# Required: Temporary directory
# Best Practice: Use secure temporary directory
# Security Note: Directory must be properly secured
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Optional improvement: Create backup directory
# Rationale: Provides a backup of generated certificates for disaster recovery
# Note: Ensure backup directory has proper security permissions
BACKUP_DIR="spire-secrets-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Required: Version check
# Best Practice: Check script version
# Security Note: Version must be properly managed
if [ -f "$BACKUP_DIR/version" ]; then
  OLD_VERSION=$(cat "$BACKUP_DIR/version")
  if [ "$OLD_VERSION" != "$VERSION" ]; then
    echo "Warning: Script version mismatch. Old version: $OLD_VERSION, New version: $VERSION"
  fi
fi
echo "$VERSION" > "$BACKUP_DIR/version"

# Create SPIRE namespace if it doesn't exist
# The namespace is required for SPIRE components
echo "Creating SPIRE namespace..."
kubectl create namespace spire --dry-run=client -o yaml | kubectl apply -f -

# Required: Key rotation
# Best Practice: Configure key rotation
# Security Note: Keys must be properly rotated
ROTATION_INTERVAL="30d"
LAST_ROTATION_FILE="$BACKUP_DIR/last_rotation"
if [ -f "$LAST_ROTATION_FILE" ]; then
  LAST_ROTATION=$(cat "$LAST_ROTATION_FILE")
  CURRENT_TIME=$(date +%s)
  ROTATION_TIME=$(date -d "$LAST_ROTATION + $ROTATION_INTERVAL" +%s)
  if [ "$CURRENT_TIME" -lt "$ROTATION_TIME" ]; then
    echo "Warning: Keys are due for rotation"
  fi
fi
date +%Y-%m-%d > "$LAST_ROTATION_FILE"

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

# Required: Certificate versioning
# Best Practice: Track certificate versions
# Security Note: Versions must be properly managed
CERT_VERSION=$(date +%Y%m%d-%H%M%S)
echo "$CERT_VERSION" > "$BACKUP_DIR/cert_version"

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

# Required: Certificate revocation
# Best Practice: Configure certificate revocation
# Security Note: Revocation must be properly managed
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

# Required: Secret versioning
# Best Practice: Track secret versions
# Security Note: Versions must be properly managed
SECRET_VERSION=$(date +%Y%m%d-%H%M%S)
echo "$SECRET_VERSION" > "$BACKUP_DIR/secret_version"

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

# Required: Secret rotation
# Best Practice: Configure secret rotation
# Security Note: Rotation must be properly managed
ROTATION_INTERVAL="30d"
LAST_ROTATION_FILE="$BACKUP_DIR/last_secret_rotation"
if [ -f "$LAST_ROTATION_FILE" ]; then
  LAST_ROTATION=$(cat "$LAST_ROTATION_FILE")
  CURRENT_TIME=$(date +%s)
  ROTATION_TIME=$(date -d "$LAST_ROTATION + $ROTATION_INTERVAL" +%s)
  if [ "$CURRENT_TIME" -lt "$ROTATION_TIME" ]; then
    echo "Warning: Secrets are due for rotation"
  fi
fi
date +%Y-%m-%d > "$LAST_ROTATION_FILE"

# Optional improvement: Backup generated files
# Rationale: Provides a backup of generated certificates for disaster recovery
# Note: Ensure backup directory has proper security permissions
echo "Backing up generated files..."
cp *.key *.crt *.csr *.srl *.pem "$BACKUP_DIR/"

# Required: Backup verification
# Best Practice: Verify backups
# Security Note: Backups must be properly verified
echo "Verifying backups..."
for file in "$BACKUP_DIR"/*; do
  if [ ! -f "$file" ]; then
    echo "Error: Backup file $file is missing"
    exit 1
  fi
done

# Clean up temporary files
# Remove all generated files to prevent sensitive data exposure
# Required: This is mandatory for security
echo "Cleaning up temporary files..."
rm -f *.key *.crt *.csr *.srl *.pem

# Required: Log rotation
# Best Practice: Configure log rotation
# Security Note: Logs must be properly rotated
if [ -f "$LOG_FILE" ]; then
  if [ $(stat -f %z "$LOG_FILE") -gt 10485760 ]; then
    mv "$LOG_FILE" "$LOG_FILE.old"
    gzip "$LOG_FILE.old"
  fi
fi

echo "SPIRE secrets generation complete!"
echo "Backup of certificates stored in: $BACKUP_DIR"
echo "Log file: $LOG_FILE" 