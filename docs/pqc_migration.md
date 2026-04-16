# Post-Quantum Cryptography Migration Guide

This guide outlines a staged migration path toward post-quantum readiness for workload identity systems.

## Scope

Focus areas:

- Certificate hierarchy and key algorithms.
- Trust bundle distribution.
- mTLS interoperability across mixed environments.
- Compliance posture for regulated sectors.

## Migration Phases

### Phase 1: Inventory and Baseline

- Enumerate all PKI dependencies and supported algorithms.
- Record TLS termination points and certificate lifetimes.
- Identify systems that cannot support crypto agility.

### Phase 2: Hybrid Readiness

- Introduce crypto-agile configuration paths.
- Test hybrid certificate and signature strategies in non-production.
- Validate trust bundle size and propagation behavior.

### Phase 3: Controlled Rollout

- Roll out hybrid profiles by environment.
- Monitor handshake failures and latency regressions.
- Keep rollback plans per domain and namespace.

### Phase 4: Policy Enforcement

- Enforce approved algorithm policy in CI/CD.
- Block deprecated algorithm issuance.
- Align audit evidence with control frameworks.

## Compliance Notes

PQC planning affects requirements under CRA, EU AI Act, NIS2, and DORA where resilience and secure lifecycle controls are in scope.

## Practical Recommendations

- Prefer short-lived credentials to limit algorithm-transition blast radius.
- Keep bundle publishers and caches tuned for larger artifacts.
- Test federation partners for hybrid compatibility before production enforcement.
