# SPIFFE Workload Identity mTLS Demo

This repository provides a complete example of setting up secure mutual TLS (mTLS) authentication between two Kubernetes workloads using [SPIFFE](https://spiffe.io/) and [SPIRE](https://spiffe.io/spire/). It includes configuration for identity issuance, deployment manifests, and verification utilities.

---

## 📂 File Structure

```bash
.
├── core/
│   ├── apply-spiffe-config.sh         # Applies all SPIFFE-related configurations
│   ├── cleanup-spiffe-config.sh       # Deletes SPIFFE configs for manual cleanup
│   ├── install-all.sh                 # Complete setup installer for workloads + SPIFFE
│   ├── uninstall-all.sh               # Deletes all resources and cleans the environment
│   ├── verify-all.sh                  # Verifies all components are correctly installed
├── workloads/
│   ├── frontend-deployment.yaml       # Kubernetes Deployment for the frontend
│   ├── backend-deployment.yaml        # Kubernetes Deployment for the backend
│   ├── frontend-service.yaml          # Service exposing the frontend
│   ├── backend-service.yaml           # Service exposing the backend
├── spiffe/
│   ├── sidecar-config.json            # Envoy config for SPIFFE-enabled mTLS
│   ├── spiffe-sidecar-configmap.yaml  # Mounts the sidecar config into workloads
│   ├── frontend-registration-entry.yaml # SPIRE registration for frontend workload
│   ├── backend-clusterspiffeid.yaml     # SPIRE ClusterSPIFFEID for backend workload
├── test/
│   ├── test-mtls.sh                   # Script to send requests from frontend to backend using mTLS
```

---

## 🚀 Getting Started

### Step 1: Install Everything

Run the full setup:

```bash
cd core
./install-all.sh
```

This will:
- Deploy frontend and backend workloads
- Apply SPIFFE registration and identity bindings
- Mount SPIRE sidecar config and config maps
- Expose frontend/backend services

---

### Step 2: Verify the Installation

Use this script to verify that everything is up and running correctly:

```bash
./verify-all.sh
```

It will:
- Check if all SPIRE and workload pods are running
- Confirm sockets and sidecar configuration
- Print active SPIFFE registration entries
- Try querying the SPIRE Agent socket

---

### Step 3: Test Mutual TLS Communication

To simulate and test frontend-backend secure communication using mTLS:

```bash
cd test
./test-mtls.sh
```

This sends an HTTP request from frontend to backend and checks the response.

---

### Step 4: Clean Up

If you want to **remove all installed resources** and reset your environment:

```bash
./uninstall-all.sh
```

This is useful when:
- Starting over with a clean cluster
- Re-deploying a fresh configuration
- Finishing your demo or test session

---

## 🛠 Script Breakdown

| Script                  | Purpose                                                     |
|-------------------------|-------------------------------------------------------------|
| `install-all.sh`        | Installs workloads, SPIFFE entries, and config maps         |
| `verify-all.sh`         | Ensures all services and SPIRE components are functioning   |
| `test-mtls.sh`          | Simulates workload communication over mTLS via SPIFFE       |
| `uninstall-all.sh`      | Deletes all deployed resources and resets the environment   |
| `apply-spiffe-config.sh`| Applies only SPIFFE identity entries                        |
| `cleanup-spiffe-config.sh` | Removes only SPIFFE identity entries                  |

---

## 📘 Additional Notes

- All SPIRE entries assume a trust domain of `spiffe://example.org`
- The SPIRE Agent must be running and accessible via `/run/spire/sockets/agent.sock`
- You can extend this demo to include AWS federation or OIDC discovery if desired.

---

For more about SPIFFE, SPIRE, and Workload Identity:
- [SPIFFE Documentation](https://spiffe.io/docs/latest/)
- [SPIRE Docs](https://spiffe.io/spire/)
- [Confidential Computing Consortium](https://confidentialcomputing.io/)
