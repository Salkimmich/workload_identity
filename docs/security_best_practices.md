# Security Best Practices

This guide covers security practices for implementing and operating the workload identity system. It is opinionated. Where a decision is close, this guide takes a position and explains it.

## Table of Contents

1. [Core Principles](#core-principles)
2. [Cryptographic Standards](#cryptographic-standards)
3. [Certificate and Key Management](#certificate-and-key-management)
4. [Runtime Security](#runtime-security)
5. [Network Security](#network-security)
6. [Audit and Observability](#audit-and-observability)
7. [Development Security](#development-security)
8. [Security Checklist](#security-checklist)

---

## Core Principles

**Zero trust.** No workload is trusted by default, regardless of network location. Every service interaction requires mutual authentication. Trust is continuously verified, not assumed from prior authentication.

**Least privilege.** Every workload holds the minimum credentials required to do its job. Scope is bounded. Standing access to resources a workload is not currently using is eliminated.

**Short-lived credentials.** The primary control for limiting blast radius is credential lifetime. 15-minute SVIDs are the default. Shorter is better. Never issue credentials that do not expire.

**Crypto agility.** No algorithm choice should be permanently hardcoded. The ability to rotate algorithms — without downtime and without re-engineering — is a security property of the system, not a nice-to-have.

---

## Cryptographic Standards

### Algorithm recommendations

| Use | Current recommendation | Deprecated (do not use for new deployments) |
|---|---|---|
| CA signing key | ECDSA P-384 | RSA-2048, ECDSA P-256 |
| TLS | TLS 1.3 | TLS 1.2 (acceptable legacy), TLS 1.1 and below (prohibited) |
| Symmetric encryption | AES-256-GCM | AES-128, 3DES |
| JWT signing | ES384 (ECDSA P-384) | RS256, HS256 |
| Key exchange | ECDH P-384 | RSA key exchange |

P-256 is not deprecated from a current security perspective — it provides adequate security against classical attacks. The reason to prefer P-384 is **crypto agility**: P-384 gives a larger security margin as the migration to post-quantum algorithms proceeds, and the operational difference is negligible.

### Post-quantum cryptography

NIST finalized its first PQC standards in August 2024: ML-DSA (FIPS 204), ML-KEM (FIPS 203), and SLH-DSA (FIPS 205). HQC was selected in March 2025 as a backup key encapsulation mechanism.

These algorithms are not yet natively supported in SPIRE's built-in CA. When Go's standard library includes ML-DSA support (track the [Go PQC issue](https://github.com/golang/go/issues/64537)), SPIRE will be able to adopt it. In the meantime:

- Use P-384 as the current CA signing algorithm — it provides a stronger classical baseline while PQC support matures
- Confirm your upstream authority (Vault, AWS Private CA, GCP CA Service) has a PQC roadmap
- Implement trust bundle automation now — you will need it for the PQC migration

For full migration guidance, see [Post-Quantum Cryptography Migration Guide](pqc_migration.md).

### TLS configuration

```hcl
# spire-server.conf — TLS hardening
tls_min_version = "TLS1.2"   # TLS 1.3 preferred; 1.2 acceptable for legacy clients

# Cipher suite restriction (TLS 1.3 cipher suites are fixed by the spec)
# For TLS 1.2 fallback, restrict to:
# TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
# TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
```

```yaml
# Network policy — enforce TLS minimum version
network_security:
  tls:
    min_version: "TLS1.3"
    cipher_suites:
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"
```

---

## Certificate and Key Management

### CA signing key

```hcl
# spire-server.conf
ca_key_type = "ec-p384"
ca_ttl = "24h"
default_svid_ttl = "15m"
```

Do not use RSA for the CA signing key in new deployments. ECDSA P-384 is smaller, faster, and provides an equivalent or better security margin.

### SVID TTLs

| Workload type | Recommended TTL | Rationale |
|---|---|---|
| Standard services | 15 minutes | Default. Good balance of security and rotation overhead. |
| AI agents / agentic workloads | 5–15 minutes | Higher risk surface. See [Agentic Identity Guide](agentic_identity.md). |
| Privileged orchestrators | 5 minutes | Smallest feasible TTL for high-value targets. |
| Edge nodes (intermittent connectivity) | Up to 4 hours | Sized to offline window. Accept the trade-off explicitly. |

Never set TTLs longer than 24 hours. The value of short-lived credentials is that compromise windows are bounded.

### Certificate revocation

SPIRE supports certificate revocation through the CRL mechanism. Enable it:

```hcl
# spire-server.conf
revocation_enabled = true
```

Revocation is the emergency stop for a compromised workload. Test it regularly. A revocation mechanism you have never exercised is not a control.

### Key storage

Production deployments should use an HSM or cloud KMS for the CA signing key:

```hcl
# spire-server.conf — HashiCorp Vault upstream authority
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

Never store CA private keys in Kubernetes secrets unencrypted. If you cannot use an HSM or cloud KMS immediately, encrypt the key store with a KMS-backed key and treat rotation as a mandatory near-term action.

---

## Runtime Security

### Kubernetes pod security

All SPIRE components should run with:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

The hardened Helm chart applies these automatically when `recommendations.enabled: true`. See [Deployment Guide](deployment_guide.md).

### Resource limits

Always set resource limits. An identity component without limits can be starved or can cause cluster instability under load:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

### Workload attestation selectors

Use the most specific selectors available for workload registration. More specific selectors reduce the blast radius of a spoofed attestation.

Good:
```bash
-selector k8s:ns:production \
-selector k8s:sa:my-service \
-selector k8s:pod-label:app:my-service \
-selector k8s:pod-label:version:v1-3-2
```

Avoid:
```bash
-selector k8s:ns:production   # Too broad — any pod in production matches
```

---

## Network Security

### Network policies

Restrict SPIRE Server access to only SPIRE Agents and authorized operators:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: spire-server-policy
  namespace: spire-server
spec:
  podSelector:
    matchLabels:
      app: spire-server
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: spire-system
          podSelector:
            matchLabels:
              app: spire-agent
      ports:
        - port: 8081
  egress:
    - {}  # Allow egress for upstream authority connections
```

### mTLS enforcement

All workload-to-workload communication in the mesh must use mTLS with SPIFFE SVIDs. Do not allow plaintext fallback. A service that cannot present a valid SVID should not be able to communicate with production services.

---

## Audit and Observability

### What to log

Every security-relevant event must produce a structured JSON log entry:

```json
{
  "timestamp": "2026-04-15T10:23:01Z",
  "event_type": "svid.issued",
  "spiffe_id": "spiffe://example.org/services/payment-api",
  "node_id": "spiffe://example.org/nodes/k8s-worker-03",
  "ttl_seconds": 900,
  "selector_match": ["k8s:ns:production", "k8s:sa:payment-api"]
}
```

Required event types:
- `svid.issued` — SVID issued to a workload
- `svid.rotated` — SVID renewed before expiry
- `svid.revoked` — SVID explicitly revoked
- `node.attested` — Node successfully attested
- `node.attestation_failed` — Node attestation failure (investigate immediately)
- `policy.denied` — Access denied by policy engine
- `bundle.updated` — Trust bundle rotated or federated bundle refreshed

### Retention

Retain identity audit logs for at minimum 365 days. Many compliance frameworks (NIS2, DORA, HIPAA) require this or longer. Use structured log shipping to a tamper-evident store — do not rely on pod logs alone.

### Alerts

| Condition | Severity | Action |
|---|---|---|
| Node attestation failure rate > 5/min | Critical | Investigate immediately — possible attack |
| SVID rotation failure | Warning | Check agent connectivity to SPIRE Server |
| Trust bundle update failure | Critical | Identity plane degradation |
| CA certificate expiry < 30 days | Warning | Rotate CA |
| Revocation check failure | Warning | CRL infrastructure degradation |

---

## Development Security

### Dependency scanning

Run dependency scanning on every build. The identity infrastructure is a high-value target. A supply chain compromise of a SPIRE dependency is a complete trust plane compromise.

```yaml
# .github/workflows/security.yml
- name: Scan dependencies
  uses: aquasecurity/trivy-action@master
  with:
    scan-type: fs
    severity: CRITICAL,HIGH
    exit-code: 1
```

### Secret scanning

SPIRE configuration files can contain sensitive material (upstream authority credentials, database passwords). Enable secret scanning on the repository:

```bash
# Pre-commit hook
gitleaks protect --staged
```

### Container image scanning

Scan all SPIRE component images before deployment:

```bash
trivy image ghcr.io/spiffe/spire-server:latest
trivy image ghcr.io/spiffe/spire-agent:latest
```

Pin to digest, not tag, in production:

```yaml
image: ghcr.io/spiffe/spire-server@sha256:abc123...
```

---

## Security Checklist

### Before deploying to production

- [ ] CA key stored in HSM or cloud KMS
- [ ] `ca_key_type = "ec-p384"` (not RSA, not P-256)
- [ ] `default_svid_ttl` set to 900 seconds or less
- [ ] Certificate revocation enabled and tested
- [ ] TLS 1.3 enforced on all SPIRE endpoints
- [ ] SPIRE Server running as non-root with read-only root filesystem
- [ ] Network policies restricting SPIRE Server access to agents only
- [ ] Workload selectors as specific as feasible
- [ ] Audit logging enabled and shipping to durable store
- [ ] Alerts configured for attestation failures and bundle update failures
- [ ] Dependency and container image scanning in CI pipeline

### Quarterly review

- [ ] Rotate CA if approaching expiry
- [ ] Review workload registrations — remove stale entries
- [ ] Review SVID TTL settings — can any be shortened?
- [ ] Check SPIRE release notes for security advisories
- [ ] Test revocation end-to-end
- [ ] Verify trust bundle automation is functioning
- [ ] Review audit logs for anomalies

---

## References

- [SPIRE Security Advisories](https://github.com/spiffe/spire/security/advisories)
- [NIST SP 800-207: Zero Trust Architecture](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-207.pdf)
- [Post-Quantum Cryptography Migration Guide](pqc_migration.md)
- [Agentic Identity Guide](agentic_identity.md)
- [SPIFFE/SPIRE Helm Charts Hardened](https://github.com/spiffe/helm-charts-hardened)
