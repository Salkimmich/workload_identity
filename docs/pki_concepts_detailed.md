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

### Trust Anchors & PKI Federation

#### Trust Anchors
Trust anchors (root CAs) serve as the foundation of trust in PKI systems. They are the starting point for all certificate validation chains and establish the root of trust for the entire system.

Key aspects of trust anchors:
- Self-signed certificates that serve as the root of trust
- Distributed through trust stores across the system
- Critical for establishing secure communication channels
- Must be protected and managed with highest security

#### PKI Federation
PKI federation enables trust across different domains or organizations by establishing trust relationships between their respective PKI systems.

Federation methods:
1. Cross-certification
   - Direct trust relationships between CAs
   - Bilateral trust establishment
   - Example: Organization A and B exchange root certificates

2. SPIFFE Trust Domain Bundles
   - Federation through trust bundle exchange
   - Supports multiple trust domains
   - Example: Two Kubernetes clusters exchanging trust bundles

Trade-offs:
- Centralized Root
  - Simpler management
  - Single point of control
  - Limited scalability
  - Higher risk of compromise

- Federated Trust
  - More complex management
  - Distributed control
  - Better scalability
  - Reduced blast radius

Security Considerations:
- Clear scoping of trust relationships
- Regular audit of trust anchors
- Monitoring of federation status
- Automated trust bundle updates

### Dynamic Credentials

#### Short-lived Certificates
Modern systems favor ephemeral credentials that are:
- Generated on-demand
- Short validity periods (minutes to hours)
- Automatically rotated
- Revoked immediately after use

Benefits:
- Reduced impact of key compromise
- Improved security posture
- Better compliance with zero-trust principles
- Simplified revocation management

#### Automated Issuance/Revocation
Example workflow for dynamic credential management:

```yaml
# Example: Dynamic Certificate Issuance
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: workload-cert
spec:
  duration: 1h
  renewBefore: 10m
  secretName: workload-tls
  issuerRef:
    name: spire-issuer
    kind: ClusterIssuer
  usages:
    - server auth
    - client auth
```

#### Real-time Revocation
- OCSP stapling for immediate status
- Automated CRL updates
- Trust bundle propagation
- Instant revocation capabilities

### Hardware Security & Attestation

#### Hardware-Backed Key Storage
1. Hardware Security Modules (HSMs)
   - Isolated cryptographic operations
   - Protection against key exfiltration
   - FIPS 140-2 compliance
   - Cloud HSM integration

2. Trusted Platform Modules (TPMs)
   - Hardware root of trust
   - Endorsement Key (EK) for attestation
   - Secure key storage
   - Platform integrity measurement

3. Trusted Execution Environments (TEEs)
   - Intel SGX
   - AMD SEV
   - ARM TrustZone
   - Code integrity verification

#### Attestation Workflow
Example attestation process:

```python
# Example: TPM Attestation
def verify_workload_attestation(attestation_data):
    # Verify TPM quote
    quote = attestation_data['quote']
    if not verify_tpm_quote(quote):
        raise SecurityError("Invalid TPM quote")
    
    # Verify platform state
    pcr_values = attestation_data['pcr_values']
    if not verify_platform_state(pcr_values):
        raise SecurityError("Platform state mismatch")
    
    # Issue certificate if attestation passes
    return issue_workload_certificate(attestation_data['workload_id'])
```

### Workload Identity Integration

#### SPIFFE/SPIRE Integration
Detailed workflow:
1. Node Attestation
   - Platform verification
   - Workload validation
   - Trust domain assignment

2. Identity Issuance
   - SPIFFE ID generation
   - X.509 SVID creation
   - JWT token issuance

3. Trust Domain Federation
   - Bundle exchange
   - Cross-domain trust
   - Policy enforcement

#### Service Mesh Integration
Istio security architecture:
- In-cluster CA (Citadel)
- Automatic certificate distribution
- mTLS enforcement
- External CA integration

Example configuration:

```yaml
# Example: Istio mTLS Configuration
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: STRICT
  selector:
    matchLabels:
      app: my-service
```

#### Cloud IAM Integration
Workload Identity Federation:
- Certificate/OIDC token exchange
- Cloud access token issuance
- Role-based access control
- Audit logging

Example trust configuration:

```yaml
# Example: AWS Workload Identity Federation
apiVersion: iam.cnrm.cloud.google.com/v1beta1
kind: IAMWorkloadIdentityPool
metadata:
  name: my-pool
spec:
  displayName: "My Workload Identity Pool"
  description: "Pool for workload identity federation"
  disabled: false
```

#### CI/CD Pipeline Integration
Secure pipeline practices:
1. OIDC-based authentication
2. Temporary credential issuance
3. Secret management integration
4. Build artifact signing

Example GitHub Actions workflow:

```yaml
# Example: GitHub Actions OIDC
name: Secure Pipeline
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
    steps:
      - uses: actions/checkout@v2
      - name: Authenticate to AWS
        uses: aws-actions/configure-aws-credentials@v1
        with:
          role-to-assume: arn:aws:iam::123456789012:role/github-actions
          aws-region: us-west-2
```

#### Human-to-Workload Trust
Bridging user and service authentication:
1. SSO Integration
   - JWT validation
   - Identity propagation
   - Scope enforcement

2. Service Access
   - mTLS authentication
   - Role-based access
   - Audit logging

3. Delegation Patterns
   - Token exchange
   - Impersonation
   - Permission scoping

## 6. Implementation Examples

### Dynamic Issuance Service
```python
# Example: Dynamic Certificate Issuance
class DynamicIssuanceService:
    def issue_certificate(self, workload_id, attestation):
        # Verify attestation
        if not self.verify_attestation(attestation):
            raise SecurityError("Invalid attestation")
        
        # Generate short-lived certificate
        cert = self.generate_certificate(
            subject=workload_id,
            validity=timedelta(hours=1)
        )
        
        return cert
```

### Federation Trust Bootstrap
```python
# Example: Trust Bundle Exchange
def bootstrap_federation(trust_domain, bundle):
    # Verify bundle signature
    if not verify_bundle_signature(bundle):
        raise SecurityError("Invalid bundle signature")
    
    # Update trust store
    update_trust_store(trust_domain, bundle)
    
    # Configure federation policies
    configure_federation_policies(trust_domain)
```

## Best Practices & Threat Modeling

### Security Best Practices
- Use short-lived credentials
- Maintain minimal trust anchors
- Regular trust anchor audits
- Hardware root of trust
- Automated rotation

### Operational Best Practices
- Automated federation management
- Certificate issuance monitoring
- CA key backup and recovery
- Disaster recovery testing
- Trust bundle updates

### Threat Model
Key threats and mitigations:
1. CA Compromise
   - Mitigation: HSM protection, short-lived certs
   - Detection: Monitoring, audit logs
   - Response: Revocation, key rotation

2. Key Theft
   - Mitigation: Hardware protection, attestation
   - Detection: Usage monitoring
   - Response: Immediate revocation

3. Trust Anchor Abuse
   - Mitigation: Federation scoping
   - Detection: Trust anchor monitoring
   - Response: Trust relationship review

4. Fake Workload Attacks
   - Mitigation: Strong attestation
   - Detection: Anomaly detection
   - Response: Policy enforcement

## References
- [SPIFFE Documentation](https://spiffe.io/docs/)
- [SPIRE Documentation](https://spiffe.io/docs/latest/spire/)
- [X.509 Certificate Standards](https://tools.ietf.org/html/rfc5280)
- [PKI Best Practices](https://www.nist.gov/publications/pki-best-practices) 