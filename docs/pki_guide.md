# PKI Guide for Workload Identity

This guide covers PKI concepts as they apply to workload identity systems built on SPIFFE/SPIRE. It is a practitioner guide, not a general PKI primer. For post-quantum migration planning, see [pqc_migration.md](pqc_migration.md).

## Table of Contents

1. [Core Concepts](#core-concepts)
2. [Algorithm Recommendations](#algorithm-recommendations)
3. [Certificate Lifecycle Management](#certificate-lifecycle-management)
4. [Key Management](#key-management)
5. [Trust Model in SPIFFE/SPIRE](#trust-model-in-spiffespire)
6. [Integration Patterns](#integration-patterns)
7. [Post-Quantum Readiness](#post-quantum-readiness)
8. [Common Issues](#common-issues)

---

## Core Concepts

### What PKI provides

PKI provides three things for workload identity:

- **Authentication** — proves a workload is who it claims to be
- **Confidentiality** — encrypts communication so only the intended recipient can read it
- **Integrity** — proves a message was not modified in transit

In SPIFFE/SPIRE, the PKI stack underpins all three. The SPIRE Server acts as a CA. SVIDs are X.509 certificates or JWTs. mTLS between workloads uses those certificates for mutual authentication and encrypted channels.

### Components

**Certificate Authority (CA)**
Issues and signs certificates. In SPIRE, the SPIRE Server is the CA — either self-signed or subordinate to an upstream authority (Vault, cloud PKI). The CA's signing key is the root of trust for the entire identity stack. Protect it accordingly.

**SVID (SPIFFE Verifiable Identity Document)**
The certificate issued to a workload. It contains the workload's SPIFFE ID in the Subject Alternative Name (SAN) field. Two formats:
- X.509-SVID: an X.509 certificate with SPIFFE ID in URI SAN
- JWT-SVID: a signed JWT with SPIFFE ID in the `sub` claim and a mandatory `exp` (expiration) claim

**Trust Bundle**
The set of CA certificates that a workload trusts. Workloads use the trust bundle to verify SVIDs presented by other workloads. SPIRE distributes trust bundles automatically via the Workload API.

**Trust Domain**
A logical grouping of workloads that share a CA. Represented as a URI: `spiffe://example.org`. Federation allows workloads in different trust domains to establish mTLS with each other by exchanging trust bundles.

**CRL (Certificate Revocation List)**
A list of revoked certificate serial numbers maintained by the CA. Workloads check the CRL to verify a presented certificate has not been revoked. Short SVID TTLs reduce — but do not eliminate — the need for revocation checking.

---

## Algorithm Recommendations

### Current recommendations

| Use | Algorithm | Key size / curve |
|---|---|---|
| CA signing key | ECDSA | P-384 |
| Leaf SVID signing | ECDSA | P-384 |
| TLS | TLS 1.3 (preferred), TLS 1.2 (acceptable) | — |
| JWT signing | ES384 (ECDSA P-384) | P-384 |
| Symmetric encryption (at rest) | AES-GCM | 256-bit |

### Why P-384, not P-256

P-256 is not broken. It provides adequate security against current classical attacks. P-384 is preferred for new deployments for two reasons:

1. Larger security margin as the cryptographic landscape evolves
2. Better starting position for the transition to post-quantum algorithms — P-384 maintains its security properties longer than P-256 under increasing compute pressure

The performance difference between P-256 and P-384 is negligible for workload identity use cases.

### What to avoid

| Algorithm | Reason |
|---|---|
| RSA (any key size) for new deployments | Deprecated for new use by NIST by 2030. ECDSA is preferred. |
| ECDSA P-256 for CA keys | Acceptable but not preferred. Use P-384. |
| SHA-1 | Broken. Do not use for any signing operation. |
| TLS 1.1 and below | Prohibited. Contains known vulnerabilities. |
| TLS 1.2 for new deployments | Acceptable for legacy compatibility only. Prefer TLS 1.3. |
| HMAC-based JWT signing (HS256, HS384) | Requires shared secret. Incompatible with zero-trust model. Use ES384. |

### SPIRE configuration

```hcl
# spire-server.conf
ca_key_type = "ec-p384"
ca_ttl = "24h"
default_svid_ttl = "15m"
jwt_issuer = "https://spire.example.org"
```

---

## Certificate Lifecycle Management

### Issuance

SPIRE issues SVIDs automatically when a workload attests successfully. No manual CSR process. The flow:

1. Workload contacts SPIRE Agent via Unix domain socket
2. Agent verifies workload identity (via selectors: k8s SA, process attributes, etc.)
3. Agent forwards attestation to SPIRE Server
4. Server issues X.509-SVID and JWT-SVID
5. Agent returns SVIDs to workload
6. Workload uses SVIDs for mTLS and JWT-based authentication

### Rotation

SPIRE rotates SVIDs automatically before they expire. The rotation window is configurable:

```hcl
# spire-server.conf
default_svid_ttl = "15m"
# SPIRE Agent begins rotation at 50% of TTL by default
# A 15-minute TTL means rotation attempt starts at ~7.5 minutes
```

Workloads using the go-spiffe or java-spiffe libraries receive updated SVIDs via the Workload API without restart. Workloads that mount SVIDs as files need to watch for file changes and reload.

### Revocation

SVID revocation is the emergency stop for a compromised workload. Enable CRL support:

```hcl
# spire-server.conf
revocation_enabled = true
```

Revocation propagates to workloads through trust bundle updates. Test your revocation procedure regularly — an untested revocation path is not a control.

For immediate isolation, the fastest path is entry deletion:

```bash
# Remove workload's registration entry — SPIRE stops issuing new SVIDs immediately
spire-server entry delete -entryID <entry-id>
```

Existing SVIDs remain valid until their TTL expires. With 15-minute TTLs, the maximum exposure window after entry deletion is 15 minutes.

### Trust bundle rotation

The CA signing key should be rotated periodically and when a compromise is suspected. SPIRE supports CA rotation with zero downtime:

1. New CA key generated by SPIRE Server
2. New CA certificate added to trust bundle (both old and new are valid)
3. New SVIDs issued with new CA signature
4. Old CA certificate removed from trust bundle after all old SVIDs expire

Trust bundle updates propagate to all workloads automatically via the Workload API.

---

## Key Management

### CA signing key storage

The CA signing key is the most sensitive key in the system. Options in priority order:

**1. Cloud KMS (preferred for cloud deployments)**
```hcl
# spire-server.conf — AWS KMS key manager
KeyManager "aws_kms" {
  plugin_data {
    region = "eu-west-1"
    key_policy_file = "/run/spire/server/kms-key-policy.json"
  }
}
```

**2. HashiCorp Vault (preferred for hybrid/on-premises)**
```hcl
UpstreamAuthority "vault" {
  plugin_data {
    vault_addr = "https://vault.internal:8200"
    pki_mount_point = "pki"
    cert_auth {
      cert_auth_mount_point = "cert"
      client_cert_path = "/run/spire/server/vault-client.crt"
      client_key_path = "/run/spire/server/vault-client.key"
    }
  }
}
```

**3. TPM-backed key manager (for physical infrastructure)**
See [tpm_attestation.md](tpm_attestation.md) for TPM key manager configuration.

**4. Disk (development/testing only)**
Never use disk-backed key storage in production. The key is unprotected.

### Workload key generation

SPIRE generates workload key pairs in-process on the agent. Private keys are never written to disk and never transmitted over the network. This is a core security property of the SPIFFE model — the key generation and CSR happen inside the agent, and only the signed certificate is returned to the workload.

### Key backup

Back up the CA signing key with the same protection as the key itself:
- HSM-backed keys: use the HSM's built-in backup/restore capability
- Cloud KMS: enable multi-region replication where available
- Vault: use Vault's unsealing and key backup procedures

Never store a plaintext backup of a CA signing key. A plaintext backup negates the security of the HSM.

---

## Trust Model in SPIFFE/SPIRE

### Single trust domain

The simplest deployment. All workloads share one CA. SPIRE Server is the CA. Every workload with a valid SVID can establish mTLS with every other workload in the domain.

```
Trust domain: spiffe://example.org
  CA: SPIRE Server
  Workloads: spiffe://example.org/services/*
             spiffe://example.org/agents/*
```

### Federated trust domains

Multiple SPIRE deployments across different clusters, clouds, or organizations. Trust is established by exchanging CA certificates (trust bundles) between deployments.

```
Trust domain A: spiffe://a.example.org
  ←→ federation ←→
Trust domain B: spiffe://b.example.org
```

A workload in domain A with `federatesWith: spiffe://b.example.org` receives domain B's trust bundle and can establish mTLS with workloads in domain B.

Federation configuration in the hardened Helm chart:
```yaml
controllerManager:
  identities:
    clusterSPIFFEIDs:
      default:
        federatesWith:
          - b.example.org
  clusterFederatedTrustDomains:
    b:
      bundleEndpointURL: https://spire-server.b.example.org
      bundleEndpointProfile:
        type: https_spiffe
        endpointSPIFFEID: spiffe://b.example.org/spire/server
      trustDomain: b.example.org
```

### Nested SPIRE

A SPIRE Server can itself be attested by a parent SPIRE Server. This is used for large-scale deployments where a single SPIRE Server cannot serve all workloads, or for delegation across organizational boundaries.

```
Root SPIRE Server (spiffe://root.example.org)
  └── Intermediate SPIRE Server (spiffe://cluster-1.example.org)
        └── Workloads
```

---

## Integration Patterns

### Service mesh

Istio and Linkerd both support SPIRE as an external certificate provider:

```yaml
# Istio: use SPIRE as CA
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    pilot:
      env:
        EXTERNAL_CA: "true"
        USE_TOKEN_FOR_CSR: "true"
```

### Vault authentication

Workloads can authenticate to Vault using their SPIFFE SVID:

```bash
# SPIFFE JWT-SVID authentication to Vault
vault write auth/jwt/login \
  role=my-role \
  jwt="$(cat /run/spire/sockets/agent.sock | spiffe-helper get-jwt)"
```

### Cloud IAM

SPIFFE JWT-SVIDs can be exchanged for cloud provider credentials via OIDC federation. The SPIRE OIDC Discovery Provider exposes the JWKS endpoint that cloud providers need to validate JWT-SVIDs.

See [cicd_oidc_federation.md](cicd_oidc_federation.md) for AWS, GCP, and Azure configuration.

### Application libraries

| Language | Library | Notes |
|---|---|---|
| Go | [github.com/spiffe/go-spiffe](https://github.com/spiffe/go-spiffe) | Official. Handles SVID rotation automatically. |
| Java | [github.com/spiffe/java-spiffe](https://github.com/spiffe/java-spiffe) | Official. |
| Python | Community libraries | No official library. Use the Workload API gRPC interface directly. |
| Rust | Community libraries | No official library. |

---

## Post-Quantum Readiness

NIST finalized its first PQC standards in August 2024. The primary signature algorithm for migration is ML-DSA (FIPS 204), which will replace ECDSA for CA signing when SPIRE and the Go cryptography ecosystem support it.

Current status:
- Go standard library PQC support: in progress (track [golang/go#64537](https://github.com/golang/go/issues/64537))
- SPIRE built-in CA ML-DSA support: not yet available — depends on Go stdlib
- Upstream authorities (Vault, AWS Private CA): check individual vendor roadmaps

What to do now:
1. Use ECDSA P-384 for all new CA deployments — stronger classical baseline
2. Ensure trust bundle automation is working end-to-end — you will need it for the CA rotation when PQC arrives
3. Verify your upstream authority has a PQC roadmap
4. Confirm your TLS stack is crypto-agile — algorithm negotiated from config, not hardcoded

For full migration guidance, see [pqc_migration.md](pqc_migration.md).

---

## Common Issues

### Certificate chain validation failures

**Symptom:** mTLS handshake fails with "certificate signed by unknown authority"

**Causes:**
- Trust bundle not yet propagated to the workload
- Federated trust domain not configured correctly
- SPIRE Agent not running or not connected to SPIRE Server

**Diagnosis:**
```bash
# Check agent connectivity
spire-server agent list

# Check workload's trust bundle
# (via go-spiffe or direct Workload API call)
spiffe-helper get-x509-bundle

# Check federation status
spire-server federation list
```

### SVID rotation failures

**Symptom:** Workload continues serving expired certificates

**Causes:**
- SPIRE Agent not running
- Agent cannot reach SPIRE Server
- Workload not watching for SVID updates (file-based mounting)

**Fix for file-based mounting:** use inotify/fsnotify to watch the SVID file and reload on change. Or migrate to the Workload API via go-spiffe, which handles rotation automatically.

### JWT-SVID validation failures

**Symptom:** JWT-SVID rejected by verifier

**Common causes:**
- Missing `exp` claim — SPIRE now enforces expiration per spec. JWT-SVIDs without `exp` are invalid.
- Audience mismatch — the `aud` claim must match the audience the verifier expects
- Clock skew — ensure NTP is synchronized on all nodes

```bash
# Inspect a JWT-SVID
echo "<jwt>" | cut -d. -f2 | base64 -d | jq .
# Verify exp, aud, sub claims are present and correct
```

### CA key compromise

If the CA signing key is compromised, the entire trust domain is compromised. Response procedure:

1. Immediately notify all workload operators
2. Generate a new CA key pair
3. Issue a new CA certificate
4. Distribute the new trust bundle (SPIRE does this automatically)
5. Revoke all SVIDs issued under the compromised CA (they will expire naturally within TTL)
6. Audit logs to determine what was accessed while the key was compromised

Short SVID TTLs are the primary mitigation — the exposure window is bounded by TTL.

---

## References

- [SPIFFE X509-SVID specification](https://github.com/spiffe/spiffe/blob/main/standards/X509-SVID.md)
- [SPIFFE JWT-SVID specification](https://github.com/spiffe/spiffe/blob/main/standards/JWT-SVID.md)
- [SPIFFE Trust Domain and Bundle](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Trust_Domain_and_Bundle.md)
- [SPIRE configuration reference](https://spiffe.io/docs/latest/deploying/spire_server/)
- [Post-Quantum Cryptography Migration Guide](pqc_migration.md)
- [Security Best Practices](security_best_practices.md)
- [RFC 5280: Internet X.509 PKI Certificate and CRL Profile](https://datatracker.ietf.org/doc/html/rfc5280)
