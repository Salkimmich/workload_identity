# Agentic Identity Guide

This guide defines workload identity patterns for AI agents and autonomous systems using SPIFFE and SPIRE.

## Why Agentic Identity

Agentic systems chain tools, call external APIs, and operate across trust domains. Static API keys do not provide enough runtime assurance. Agents need short-lived, auditable, and policy-scoped identities.

## Identity Model

- Assign one SPIFFE ID per agent role, not per application binary.
- Use labels/selectors for role binding, environment, and risk tier.
- Keep SVID TTLs short for high-autonomy agents.

Example SPIFFE IDs:

- `spiffe://example.org/agents/data-pipeline/reader`
- `spiffe://example.org/agents/incident-response/operator`

## Runtime Controls

- Enforce mTLS for all agent-to-service and agent-to-agent traffic.
- Require policy checks before tool invocation.
- Gate high-risk actions behind just-in-time authorization.
- Emit identity-linked audit events for every privileged action.

## Kubernetes Pattern

- Set `automountServiceAccountToken: false`.
- Use projected service account tokens with explicit audience.
- Use `expirationSeconds: 300` for high-risk agent workloads.
- Add stable pod labels for SPIRE selectors.

## Migration from API Keys

1. Inventory agent endpoints currently using static keys.
2. Map each key scope to a SPIFFE ID and policy set.
3. Add SPIRE attestation and SVID retrieval path.
4. Run dual-stack auth during cutover.
5. Remove static keys and enforce identity-only access.

## Operational Checklist

- Rotation interval defined and tested.
- Access policy tied to SPIFFE IDs and role labels.
- End-to-end tracing includes identity subject.
- Revocation runbook validated in staging.
