# Compliance Guide

This guide covers compliance requirements and controls for the workload identity system. It maps specific controls to regulatory frameworks and provides implementation guidance for each.

**Framework coverage:** ISO 27001, SOC 2 Type II, GDPR, HIPAA, EU Cyber Resilience Act (CRA), EU AI Act, DORA, NIS2.

## Table of Contents

1. [Security Controls](#security-controls)
2. [Audit Requirements](#audit-requirements)
3. [Data Protection](#data-protection)
4. [Access Control](#access-control)
5. [Monitoring and Reporting](#monitoring-and-reporting)
6. [ISO 27001](#iso-27001)
7. [SOC 2 Type II](#soc-2-type-ii)
8. [GDPR](#gdpr)
9. [HIPAA](#hipaa)
10. [EU Cyber Resilience Act (CRA)](#eu-cyber-resilience-act-cra)
11. [EU AI Act](#eu-ai-act)
12. [DORA](#dora-digital-operational-resilience-act)
13. [NIS2](#nis2-directive)
14. [Control Mapping Summary](#control-mapping-summary)

---

## Security Controls

### Certificate Management

```yaml
certificate_security:
  key_protection:
    algorithm: "EC-P384"        # Minimum. See security_best_practices.md.
    storage: "HSM"              # Or cloud KMS. Never plain Kubernetes secret.
    rotation: "90d"
  certificate_lifecycle:
    validity: "24h"             # SVIDs. Not the CA cert.
    svid_ttl: "15m"            # Default SVID TTL.
    renewal_threshold: "1h"
    revocation_check: "CRL"
  access_control:
    roles:
      - name: "cert-admin"
        permissions: ["issue", "revoke"]
      - name: "cert-user"
        permissions: ["request", "renew"]
```

### Identity Verification

```yaml
identity_verification:
  attestation:
    methods:
      - "k8s-psat"            # Kubernetes Projected Service Account Token
      - "aws-iid"             # AWS Instance Identity Document
      - "azure-msi"           # Azure Managed Service Identity
      - "tpm-devid"           # TPM 2.0 Device Identity (edge/physical)
    validation:
      timeout: "5s"
      retries: 3
  lifecycle:
    issuance:
      max_validity: "15m"
      renewal_threshold: "5m"
    revocation:
      automatic: true
      triggers:
        - "compromise"
        - "expiration"
        - "workload_deregistration"
```

---

## Audit Requirements

Every compliance framework covered in this guide requires audit logging. The requirements overlap substantially. Implement once to satisfy all.

### Minimum required audit events

```yaml
audit_logging:
  events:
    - "svid.issued"
    - "svid.revoked"
    - "node.attested"
    - "node.attestation_failed"
    - "policy.evaluated"
    - "policy.denied"
    - "bundle.updated"
    - "federation.established"
    - "admin.configuration_changed"
  format:
    type: "json"
    fields:
      - "timestamp"
      - "event_type"
      - "spiffe_id"
      - "node_id"
      - "action"
      - "result"
      - "client_ip"
      - "policy_version"
  storage:
    retention: "365d"           # Minimum. DORA requires 5 years for some events.
    encryption: true
    tamper_evident: true        # Immutable log store. Not pod logs.
```

### Audit trail integrity

Audit logs must be tamper-evident. Ship logs to an immutable store (S3 with Object Lock, GCS with retention policy, a dedicated SIEM) within seconds of generation. Do not rely on Kubernetes pod logs — they are deleted when pods restart.

---

## Data Protection

### Encryption at rest

```yaml
encryption_at_rest:
  algorithm: "AES-256-GCM"
  key_rotation: "90d"
  storage: "HSM"              # Or cloud KMS
  scope:
    - "CA private keys"
    - "SPIRE Server datastore"
    - "Audit log archives"
```

### Encryption in transit

```yaml
encryption_in_transit:
  protocol: "TLS1.3"          # Minimum TLS 1.2 for legacy clients
  cipher_suites:
    - "TLS_AES_256_GCM_SHA384"
    - "TLS_CHACHA20_POLY1305_SHA256"
  certificate_validation: "strict"
  mtls:
    required: true            # All service-to-service communication
    svid_based: true          # SPIFFE SVID, not static certs
```

---

## Access Control

### RBAC configuration

```yaml
rbac:
  roles:
    - name: "spire-admin"
      permissions:
        - "entries:create"
        - "entries:delete"
        - "agents:attest"
        - "bundles:refresh"
    - name: "spire-operator"
      permissions:
        - "entries:list"
        - "agents:list"
        - "bundles:list"
  enforcement:
    mode: "strict"
    audit: true
    mfa_required: true        # For admin role
```

### Policy enforcement

```yaml
policy_enforcement:
  engine: "OPA"               # Or Cedar, Kyverno
  rules:
    - name: "identity_validation"
      conditions:
        - "attestation_valid"
        - "svid_not_expired"
        - "selector_match"
      actions:
        - "allow"
        - "audit"
    - name: "access_control"
      conditions:
        - "spiffe_id_authorized"
        - "within_time_window"
      actions:
        - "allow"
        - "audit"
```

---

## Monitoring and Reporting

### Compliance metrics

```yaml
compliance_monitoring:
  metrics:
    - name: "svid_issuance_total"
      type: "counter"
      labels: ["spiffe_id", "result"]
    - name: "svid_ttl_remaining"
      type: "gauge"
      labels: ["spiffe_id"]
    - name: "node_attestation_failures_total"
      type: "counter"
      labels: ["method", "reason"]
    - name: "policy_decisions_total"
      type: "counter"
      labels: ["policy", "result"]
  alerts:
    - name: "certificate_expiring_soon"
      threshold: "30d"
      severity: "warning"
    - name: "attestation_failure_spike"
      threshold: 5
      window: "5m"
      severity: "critical"
    - name: "policy_denial_spike"
      threshold: 10
      window: "1m"
      severity: "warning"
```

---

## ISO 27001

ISO 27001:2022 Annex A controls addressed by this workload identity system:

**A.5.15 — Access control**
SPIFFE-based identity enforces access control at the network layer. No workload can connect to another service without a valid SVID. Policy engines (OPA) enforce which SPIFFE IDs may access which resources.

**A.5.17 — Authentication information**
SVIDs replace static passwords and API keys for service authentication. Short TTLs (15 minutes default) ensure credentials cannot accumulate. Automatic rotation eliminates manual credential management.

**A.8.20 — Networks security**
mTLS with SVID-based authentication enforces mutual authentication on all service-to-service communication. Network policies restrict SPIRE Server access to authorized agents only.

**A.8.24 — Use of cryptography**
All cryptographic operations use ECDSA P-384 (CA signing) and AES-256-GCM (symmetric). TLS 1.3 is enforced. Algorithm selection is documented and reviewed as part of the cryptographic controls policy.

**A.8.25 — Secure development lifecycle**
Dependency scanning, container image scanning, and secret scanning are required in the CI pipeline. See [Development Security](security_best_practices.md#development-security).

---

## SOC 2 Type II

Trust Services Criteria addressed:

**CC6.1 — Logical and physical access controls**
SPIFFE IDs provide cryptographically verifiable identities for every workload. Access control policies are enforced at the identity layer. There are no static credentials to mismanage.

**CC6.3 — Role-based access**
OPA policies map SPIFFE IDs to permitted actions. Every access decision is audited. Role changes are tracked in the SPIRE entry registry.

**CC6.7 — Transmission and disposal of data**
All service communication uses mTLS. SVIDs expire automatically. There is no manual disposal process for credentials — they simply stop working.

**CC7.1 — Vulnerability management**
SPIRE versions are tracked. Security advisories are monitored. Container images are scanned before deployment. See the monitoring configuration above for alert thresholds.

**CC7.2 — Anomaly and threat detection**
Attestation failure spikes, policy denial spikes, and unusual SVID issuance patterns generate alerts. All events are logged to a tamper-evident store.

**CC9.2 — Vendor risk management**
Third-party services that participate in the SPIRE trust domain are evaluated as vendors. Trust bundle federation with external domains is documented and reviewed.

---

## GDPR

GDPR (Regulation 2016/679) does not directly regulate infrastructure, but workload identity infrastructure handles authentication flows that touch personal data systems.

**Article 25 — Data protection by design and by default**
Zero-trust architecture implemented through SPIFFE/SPIRE is a technical measure for data protection by design. Every workload must authenticate before accessing data systems. Least-privilege access policies limit the scope of any single workload's data access.

**Article 32 — Security of processing**
mTLS with SVID-based authentication implements the Article 32 requirement for "appropriate technical measures to ensure a level of security appropriate to the risk." Specifically:
- Encryption in transit: mTLS with TLS 1.3
- Confidentiality assurance: cryptographic authentication prevents unauthorized access
- Ability to ensure ongoing integrity: trust bundle rotation, certificate revocation

**Article 33 — Notification of a personal data breach**
Workload identity audit logs provide the forensic trail required to determine the scope and timeline of a breach. Logs must be retained for at least the GDPR statutory period. Enable structured logging with tamper-evident storage.

**Processor agreements (Article 28)**
If you use a cloud provider as your upstream certificate authority (AWS Private CA, GCP CA Service, Azure Key Vault), that provider is a data processor. Ensure a data processing agreement is in place.

---

## HIPAA

HIPAA Security Rule (45 CFR Part 164) applies to covered entities and business associates handling electronic protected health information (ePHI).

**§164.312(a) — Access control**
Implementation specification: unique user identification, automatic logoff, encryption and decryption.

SPIFFE SVIDs provide unique cryptographic identities for every workload. SVIDs expire automatically (automatic logoff equivalent for service identities). All ePHI in transit is encrypted via mTLS.

**§164.312(b) — Audit controls**
Hardware, software, and procedural mechanisms to record and examine activity in information systems containing ePHI.

The audit logging configuration above satisfies this requirement. Ensure logs include the SPIFFE ID of every workload that accessed ePHI systems. Retain logs for 6 years (HIPAA minimum).

**§164.312(d) — Person or entity authentication**
Verify that a person or entity seeking access to ePHI is who they claim to be.

SPIFFE SVIDs cryptographically verify workload identity. mTLS ensures mutual authentication. A workload cannot impersonate another workload without its SVID private key, which is generated in-process and never transmitted.

**§164.312(e) — Transmission security**
Guard against unauthorized access to ePHI in transit.

TLS 1.3 with mTLS enforced on all service communication. No plaintext fallback.

**§164.308(a)(1) — Risk analysis**
Workload identity infrastructure must be included in the HIPAA risk analysis. Key risks: CA compromise, SPIRE Server availability, SVID rotation failures. Mitigations and residual risks should be documented.

---

## EU Cyber Resilience Act (CRA)

The Cyber Resilience Act (Regulation 2024/2847) entered into force December 2024. Most compliance obligations apply from December 2027. Vulnerability disclosure requirements apply from September 2026.

### Article 13: Essential cybersecurity requirements

**13.1(b) — No known exploitable vulnerabilities at time of market placement**
Track [SPIRE security advisories](https://github.com/spiffe/spire/security/advisories). Maintain an SBOM for your workload identity deployment. Patch within the timeframe required by severity.

**13.1(d) — Confidentiality protection**
mTLS with SPIFFE SVIDs encrypts all service communication. AES-256-GCM for data at rest. This satisfies the "state-of-the-art" encryption requirement.

**13.1(e) — Integrity protection**
SVID-based authentication ensures commands and configurations are executed only by attested workloads. Policy engines prevent unauthorized manipulation.

**13.1(f) — Minimize attack surface**
Zero-trust architecture eliminates implicit network trust. Short-lived credentials eliminate standing access. Network policies restrict component-to-component communication.

### Article 14: Vulnerability handling

CRA Article 14 requires manufacturers to:
- Handle vulnerabilities throughout the support lifecycle
- Provide security updates
- Report actively exploited vulnerabilities to ENISA within 24 hours

For workload identity infrastructure operators:
- Subscribe to SPIRE security advisories
- Maintain a defined patch SLA by severity (e.g., Critical: 24h, High: 7 days, Medium: 30 days)
- Automated SVID rotation means a patched SPIRE version propagates new trust material without manual intervention

### SBOM and workload identity

CRA implies SBOM requirements for identifying components with known vulnerabilities. SPIFFE IDs can anchor SBOM attribution: each deployed workload's SPIFFE ID links to its component inventory, creating a traceable chain from runtime identity to software provenance.

---

## EU AI Act

The EU AI Act (Regulation 2024/1689) applies to AI systems in the EU. High-risk AI systems (Annex III) have the most demanding requirements.

Workload identity is a supporting control for AI Act compliance, not a direct regulatory target. The most relevant application is in securing agentic AI infrastructure. See [Agentic Identity Guide](agentic_identity.md) for implementation patterns.

### Article 9: Risk management system

Providers of high-risk AI systems must implement a continuous risk management system. Workload identity contributes to Article 9 compliance:

- **Access control for AI model serving**: SPIFFE-based identity ensures only authorized workloads can query model endpoints. This limits unauthorized access to the AI system.
- **Audit trail for automated decisions**: mTLS with SVID-based authentication creates an identity record for every service interaction. For AI systems making consequential decisions, this audit trail supports transparency obligations.
- **Agentic AI systems**: AI agents operating autonomously require identity controls per [Agentic Identity Guide](agentic_identity.md). For high-risk AI systems with agentic components, these controls are Article 9 requirements.

### Article 13: Transparency

For AI systems making consequential decisions, logs must record:
- Which AI workload (by SPIFFE ID) processed a specific request
- The policy version governing the AI workload's access at the time of decision
- The timestamp and outcome

Configure the audit logging above to capture these fields for all AI workload interactions.

### Article 17: Quality management system

Article 17 requires a quality management system covering the full AI lifecycle. Security controls are part of this system. Key workload identity contributions:
- Certificate lifecycle management demonstrates ongoing security control
- Automated SVID rotation demonstrates controls do not degrade over time
- SPIRE monitoring metrics feed into quality system KPIs

---

## DORA: Digital Operational Resilience Act

DORA (Regulation 2022/2554) applies to financial entities in the EU and their critical ICT third-party providers. It entered into force January 2025.

### Article 6: ICT Risk Management Framework

**Article 8 — Identification**
SPIFFE IDs provide a runtime inventory of all active workloads. Every workload with a valid SVID is an identified, authenticated asset. Workloads without SVIDs are anomalies — detectable in monitoring.

**Article 9 — Protection and prevention**
mTLS with SPIFFE-based identity enforces zero-trust communication. No lateral movement is possible without a valid SVID. Short TTLs limit the window of any credential compromise.

**Article 10 — Detection**
SVID issuance anomalies (unusual rate, unexpected nodes, unexpected IP ranges) are detection signals for ICT incidents. Configure alerts per the monitoring section above.

**Article 11 — Response and recovery**
SVID revocation provides immediate isolation capability. A compromised workload can be cut off from all services within seconds by revoking its SVID. Document and test this procedure.

### Article 28: ICT Third-Party Risk Management

Third-party services participating in your SPIRE trust domain are ICT third parties under DORA. This includes:
- Upstream certificate authorities (Vault, cloud PKI services)
- Federated trust domain partners

Each requires a documented ICT third-party risk assessment. Trust bundle federation with external domains requires contractual security obligations aligned with DORA Article 28.

### Operational Resilience Testing

DORA Article 26 requires digital operational resilience testing. Include workload identity infrastructure in scope:
- Test SPIRE Server HA failover
- Test SVID issuance under load
- Test revocation propagation latency
- Test CA compromise and rotation scenarios
- Document results and remediation

---

## NIS2 Directive

NIS2 (Directive 2022/2555) replaced NIS1 and expanded its scope significantly. Member states transposed it by October 2024.

### Article 21: Cybersecurity Risk Management Measures

**21.2(e) — Security in network and information systems acquisition, development, and maintenance**
Short-lived SVIDs and automated rotation address vulnerability accumulation. Automated trust bundle distribution means security updates propagate without manual intervention.

**21.2(h) — Human resources security, security of supply chains, and access control policies**
SPIFFE-based access control implements identity-driven policy enforcement. Every access decision is logged and auditable.

**21.2(i) — Use of cryptographic solutions**
mTLS with SVID-based identity implements cryptographic authentication for all service communication. Algorithm choices are documented in [Security Best Practices](security_best_practices.md).

### Incident Reporting

NIS2 requires early warning within 24 hours and initial notification within 72 hours of a significant incident. Workload identity infrastructure supports rapid incident response:
- SVID issuance logs provide a timestamped record of what was active and what it accessed
- mTLS handshake logs provide a complete service communication record at incident time
- Revocation capability allows rapid isolation of compromised workloads

---

## Control Mapping Summary

| Control Area | ISO 27001 | SOC 2 | GDPR | HIPAA | CRA | EU AI Act | DORA | NIS2 |
|---|---|---|---|---|---|---|---|---|
| Workload authentication | A.5.15 | CC6.1 | Art. 32 | §164.312(d) | Art. 13 | Art. 9 | Art. 9 | Art. 21 |
| Encryption in transit | A.8.24 | CC6.7 | Art. 32 | §164.312(e) | Art. 13 | Art. 9 | Art. 9 | Art. 21 |
| Access control | A.5.15 | CC6.3 | Art. 25 | §164.312(a) | Art. 13 | Art. 9 | Art. 9 | Art. 21 |
| Audit logging | A.8.15 | CC7.2 | Art. 30 | §164.312(b) | Art. 14 | Art. 13 | Art. 10 | Art. 23 |
| Vulnerability management | A.8.8 | CC7.1 | Art. 32 | §164.308(a)(1) | Art. 14 | Art. 17 | Art. 6 | Art. 21 |
| Incident response | A.5.26 | CC7.3 | Art. 33 | §164.308(a)(6) | Art. 14 | — | Art. 11 | Art. 23 |
| Third-party risk | A.5.19 | CC9.2 | Art. 28 | §164.308(b) | — | — | Art. 28 | Art. 21 |
| Cryptographic controls | A.8.24 | CC6.7 | Art. 32 | §164.312(e) | Art. 13 | Art. 9 | Art. 9 | Art. 21 |

---

## References

- [Security Best Practices](security_best_practices.md)
- [Architecture Guide](architecture_guide.md)
- [Post-Quantum Cryptography Migration Guide](pqc_migration.md)
- [Agentic Identity Guide](agentic_identity.md)
- [EU CRA full text](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32024R2847)
- [EU AI Act full text](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32024R1689)
- [DORA full text](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32022R2554)
- [NIS2 full text](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32022L2555)
- [NIST SP 800-207: Zero Trust Architecture](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-207.pdf)
