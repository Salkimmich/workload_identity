# TPM 2.0 Node Attestation

This guide describes TPM-backed node attestation for SPIRE in physical, edge, and constrained environments.

## Why TPM Attestation

TPM-backed identity anchors trust in hardware measurements and reduces reliance on metadata-only node trust.

## Use Cases

- Bare metal clusters.
- Edge nodes with intermittent connectivity.
- Air-gapped or sovereign infrastructure.

## Design Principles

- Bind node identity to TPM-attested evidence.
- Keep attestation policy explicit and versioned.
- Isolate bootstrap and rotation channels.

## Deployment Notes

- Ensure TPM availability and secure boot posture on all nodes.
- Standardize PCR policy across node classes.
- Validate failure behavior for PCR drift and firmware updates.

## Operational Practices

- Track attestation failures as security events.
- Rotate attestation credentials with node lifecycle actions.
- Require remediation playbooks for drift, tampering, and RMA scenarios.
