# Detailed PKI Concepts for Workload Identity

This document provides in-depth technical explanations of PKI concepts, their implementation details, and real-world examples. It serves as a technical companion to the [PKI Guide](./pki_guide.md).

## Table of Contents
1. [Public Key Infrastructure (PKI) Deep Dive](#1-public-key-infrastructure-pki-deep-dive)
2. [Certificate Authority (CA) Operations](#2-certificate-authority-ca-operations)
3. [Certificate Lifecycle Management](#3-certificate-lifecycle-management)
4. [Advanced Key Management](#4-advanced-key-management)
5. [Workload Identity Integration](#5-workload-identity-integration)
6. [Implementation Examples](#6-implementation-examples)

## 1. Public Key Infrastructure (PKI) Deep Dive

### 1.1 Core Principles

#### Asymmetric Cryptography
```python
# Example of asymmetric encryption
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding

# Generate key pair
private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=2048
)
public_key = private_key.public_key()

# Encryption
message = b"Secret message"
ciphertext = public_key.encrypt(
    message,
    padding.OAEP(
        mgf=padding.MGF1(algorithm=hashes.SHA256()),
        algorithm=hashes.SHA256(),
        label=None
    )
)

# Decryption
plaintext = private_key.decrypt(
    ciphertext,
    padding.OAEP(
        mgf=padding.MGF1(algorithm=hashes.SHA256()),
        algorithm=hashes.SHA256(),
        label=None
    )
)
```

#### Digital Signatures
```python
# Example of digital signature
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding

# Sign message
message = b"Message to sign"
signature = private_key.sign(
    message,
    padding.PSS(
        mgf=padding.MGF1(hashes.SHA256()),
        salt_length=padding.PSS.MAX_LENGTH
    ),
    hashes.SHA256()
)

# Verify signature
try:
    public_key.verify(
        signature,
        message,
        padding.PSS(
            mgf=padding.MGF1(hashes.SHA256()),
            salt_length=padding.PSS.MAX_LENGTH
        ),
        hashes.SHA256()
    )
    print("Signature is valid")
except:
    print("Signature is invalid")
```

### 1.2 Certificate Structure

#### X.509 Certificate Fields
```text
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 1234 (0x4d2)
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: C=US, O=Example CA, CN=Example Root CA
        Validity
            Not Before: Jan 1 00:00:00 2024 GMT
            Not After : Jan 1 00:00:00 2025 GMT
        Subject: C=US, O=Example Org, CN=example.com
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
            RSA Public Key: (2048 bit)
        X509v3 extensions:
            X509v3 Basic Constraints: 
                CA:FALSE
            X509v3 Key Usage: 
                Digital Signature, Key Encipherment
            X509v3 Subject Alternative Name: 
                DNS:example.com, DNS:www.example.com
```

## 2. Certificate Authority (CA) Operations

### 2.1 Root CA Setup
```bash
# Generate root CA private key
openssl genrsa -out root-ca.key 4096

# Create root CA certificate
openssl req -x509 -new -nodes -key root-ca.key -sha256 -days 3650 \
    -out root-ca.crt -subj "/C=US/O=Example CA/CN=Example Root CA"

# Create CA configuration
cat > ca.conf << EOF
[ca]
default_ca = CA_default

[CA_default]
dir = ./ca
certs = \$dir/certs
crl_dir = \$dir/crl
database = \$dir/index.txt
new_certs_dir = \$dir/newcerts
certificate = \$dir/ca.crt
serial = \$dir/serial
crl = \$dir/crl.pem
private_key = \$dir/ca.key
RANDFILE = \$dir/private/.rand
x509_extensions = usr_cert
name_opt = ca_default
cert_opt = ca_default
default_days = 365
default_crl_days = 30
default_md = sha256
preserve = no
policy = policy_match

[policy_match]
countryName = match
stateOrProvinceName = match
organizationName = match
organizationalUnitName = optional
commonName = supplied
emailAddress = optional
EOF
```

### 2.2 Intermediate CA Setup
```bash
# Generate intermediate CA private key
openssl genrsa -out intermediate-ca.key 4096

# Create intermediate CA CSR
openssl req -new -key intermediate-ca.key \
    -out intermediate-ca.csr \
    -subj "/C=US/O=Example CA/CN=Example Intermediate CA"

# Sign intermediate CA certificate
openssl ca -config ca.conf -extensions v3_intermediate_ca \
    -days 1825 -notext -md sha256 \
    -in intermediate-ca.csr \
    -out intermediate-ca.crt
```

## 3. Certificate Lifecycle Management

### 3.1 Certificate Generation
```python
# Example of programmatic certificate generation
from cryptography import x509
from cryptography.x509.oid import NameOID
from datetime import datetime, timedelta

# Create certificate
cert = x509.CertificateBuilder().subject_name(
    x509.Name([
        x509.NameAttribute(NameOID.COMMON_NAME, u"example.com"),
    ])
).issuer_name(
    x509.Name([
        x509.NameAttribute(NameOID.COMMON_NAME, u"Example CA"),
    ])
).public_key(
    public_key
).serial_number(
    x509.random_serial_number()
).not_valid_before(
    datetime.utcnow()
).not_valid_after(
    datetime.utcnow() + timedelta(days=365)
).add_extension(
    x509.SubjectAlternativeName([
        x509.DNSName(u"example.com"),
        x509.DNSName(u"www.example.com"),
    ]),
    critical=False,
).sign(private_key, hashes.SHA256())
```

### 3.2 Certificate Validation
```python
# Example of certificate validation
from cryptography.x509 import load_pem_x509_certificate
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding

def validate_certificate(cert_pem, ca_cert_pem):
    # Load certificates
    cert = load_pem_x509_certificate(cert_pem)
    ca_cert = load_pem_x509_certificate(ca_cert_pem)
    
    # Verify signature
    try:
        ca_cert.public_key().verify(
            cert.signature,
            cert.tbs_certificate_bytes,
            padding.PKCS1v15(),
            cert.signature_hash_algorithm
        )
        return True
    except:
        return False
```

## 4. Advanced Key Management

### 4.1 Key Generation and Storage
```python
# Example of secure key generation and storage
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.backends import default_backend
import os

# Generate key with secure parameters
private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=4096,
    backend=default_backend()
)

# Secure storage with encryption
encrypted_pem = private_key.private_bytes(
    encoding=serialization.Encoding.PEM,
    format=serialization.PrivateFormat.PKCS8,
    encryption_algorithm=serialization.BestAvailableEncryption(b'password')
)

# Save to secure storage
with open('private_key.pem', 'wb') as f:
    f.write(encrypted_pem)
```

### 4.2 Key Rotation
```python
# Example of key rotation process
def rotate_keys(current_key, new_key):
    # Generate new key pair
    new_private_key = rsa.generate_private_key(
        public_exponent=65537,
        key_size=4096,
        backend=default_backend()
    )
    
    # Create new certificate
    new_cert = create_certificate(new_private_key)
    
    # Update trust store
    update_trust_store(new_cert)
    
    # Grace period for old key
    schedule_key_removal(current_key, grace_period=7)
```

## 5. Workload Identity Integration

### 5.1 SPIFFE/SPIRE Integration
```yaml
# SPIRE Server Configuration
server:
  bind_address: "0.0.0.0"
  bind_port: 8081
  trust_domain: "example.org"
  data_dir: "/opt/spire/data/server"
  log_level: "INFO"
  ca_key_type: "rsa-2048"
  ca_ttl: "24h"
  jwt_issuer: "spire-server"
  jwt_ttl: "1h"
```

### 5.2 Service Mesh Integration
```yaml
# Istio mTLS Configuration
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: istio-system
spec:
  mtls:
    mode: STRICT
```

## 6. Implementation Examples

### 6.1 Certificate Management
```python
# Example of certificate management system
class CertificateManager:
    def __init__(self, ca_cert, ca_key):
        self.ca_cert = ca_cert
        self.ca_key = ca_key
        
    def issue_certificate(self, subject, public_key):
        cert = create_certificate(
            subject=subject,
            public_key=public_key,
            issuer=self.ca_cert,
            issuer_key=self.ca_key
        )
        return cert
        
    def revoke_certificate(self, cert_serial):
        update_crl(cert_serial)
        notify_revocation(cert_serial)
```

### 6.2 Key Management
```python
# Example of key management system
class KeyManager:
    def __init__(self, storage_backend):
        self.storage = storage_backend
        
    def generate_key(self):
        key = generate_secure_key()
        self.storage.store(key)
        return key
        
    def rotate_key(self, key_id):
        new_key = self.generate_key()
        self.storage.rotate(key_id, new_key)
        return new_key
```

## Best Practices and Recommendations

### 1. Security
- Use hardware security modules (HSMs) for key storage
- Implement key rotation policies
- Monitor certificate expiration
- Regular security audits

### 2. Performance
- Optimize certificate validation
- Implement caching strategies
- Monitor resource usage
- Scale horizontally

### 3. Operations
- Automate certificate management
- Implement monitoring
- Regular backups
- Disaster recovery procedures

## Troubleshooting

### Common Issues
1. Certificate validation failures
2. Key rotation problems
3. Performance issues
4. Integration challenges

### Solutions
1. Check certificate chain
2. Verify key storage
3. Monitor resource usage
4. Review integration logs

## References
- [SPIFFE Documentation](https://spiffe.io/docs/)
- [SPIRE Documentation](https://spiffe.io/docs/latest/spire/)
- [X.509 Certificate Standards](https://tools.ietf.org/html/rfc5280)
- [PKI Best Practices](https://www.nist.gov/publications/pki-best-practices) 