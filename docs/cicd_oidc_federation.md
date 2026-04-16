# CI/CD OIDC Federation Guide

This guide covers secretless CI/CD authentication using OIDC federation with cloud IAM.

## Goals

- Remove long-lived cloud credentials from pipelines.
- Bind cloud access to workload identity claims.
- Restrict deployment trust to approved environments.

## Required Claims

Enforce claim conditions on:

- repository
- branch or environment
- audience
- subject

For production paths, prefer environment-gated subjects over branch-only matching.

## Provider Patterns

- **AWS:** GitHub OIDC provider + `AssumeRoleWithWebIdentity`.
- **GCP:** Workload Identity Pool + provider attribute mapping.
- **Azure:** Federated credentials bound to managed identity.

## Hardening

- Grant least privilege roles per pipeline purpose.
- Use short session durations and explicit audiences.
- Keep deploy and build identities separate.
- Log federated principal and workflow metadata.

## Examples

Reference workflows in:

- `examples/ci-cd/github-actions-aws.yaml`
- `examples/ci-cd/github-actions-gcp.yaml`
- `examples/ci-cd/github-actions-azure.yaml`
