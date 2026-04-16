# Changelog

All notable changes to the Workload Identity project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- None

### Changed
- None

### Deprecated
- None

### Removed
- None

### Fixed
- None

### Security
- None

## [1.0.0] - 2026-04-16

### Added
- New documentation: agentic identity, PQC migration, CI/CD OIDC federation, TPM attestation, and resources.
- New `examples/ci-cd/` workflows for secretless GitHub Actions deployments to AWS, GCP, and Azure.
- New `examples/kubernetes/` projected token examples aligned with Kubernetes 1.24+ behavior.

### Changed
- Replaced `docs/security_best_practices.md`, `docs/compliance_guide.md`, and `docs/pki_guide.md` with updated versions.
- Updated SPIRE deployment image pins to `1.13.3` in Kubernetes manifests.
- Updated `docs/developer_guide.md`, `docs/deployment_guide.md`, and `README.md` for current prerequisites and hardened guidance.
- Updated `VERSION` from `0.1.0` to `1.0.0`.

### Security
- Updated Kubernetes workload examples to disable legacy service account token automount and use projected service account tokens.
- Updated documentation to reinforce JWT-SVID expiration (`exp`) claim requirements.

## [0.1.0] - YYYY-MM-DD
- Initial release