# API Reference

This document provides detailed information about the workload identity system's API endpoints, request/response formats, and authentication requirements.

## Table of Contents
1. [Authentication](#authentication)
2. [Identity Management](#identity-management)
3. [Certificate Management](#certificate-management)
4. [Policy Management](#policy-management)
5. [Error Handling](#error-handling)

## Authentication

### 1. Authentication Methods
```yaml
# Supported Authentication Methods
authentication:
  methods:
    - name: "mTLS"
      description: "Mutual TLS authentication"
      required: true
    - name: "JWT"
      description: "JSON Web Token authentication"
      required: false
```

### 2. Authentication Headers
```http
# Example Authentication Headers
Authorization: Bearer <jwt_token>
X-SPIFFE-ID: spiffe://example.org/workload/123
```

## Identity Management

### 1. Issue Identity
```http
POST /api/v1/identities
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "workload_id": "workload-123",
    "type": "kubernetes",
    "metadata": {
        "namespace": "default",
        "service_account": "default"
    }
}
```

Response:
```json
{
    "identity_id": "id-123",
    "spiffe_id": "spiffe://example.org/workload/123",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2024-01-02T00:00:00Z"
}
```

### 2. Validate Identity
```http
GET /api/v1/identities/{identity_id}
Authorization: Bearer <jwt_token>
```

Response:
```json
{
    "identity_id": "id-123",
    "spiffe_id": "spiffe://example.org/workload/123",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2024-01-02T00:00:00Z",
    "metadata": {
        "namespace": "default",
        "service_account": "default"
    }
}
```

### 3. Revoke Identity
```http
DELETE /api/v1/identities/{identity_id}
Authorization: Bearer <jwt_token>
```

Response:
```json
{
    "identity_id": "id-123",
    "status": "revoked",
    "revoked_at": "2024-01-01T12:00:00Z",
    "reason": "manual_revocation"
}
```

## Certificate Management

### 1. Issue Certificate
```http
POST /api/v1/certificates
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "identity_id": "id-123",
    "key_type": "RSA",
    "key_size": 2048,
    "validity_period": "24h"
}
```

Response:
```json
{
    "certificate_id": "cert-123",
    "identity_id": "id-123",
    "certificate": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
    "valid_from": "2024-01-01T00:00:00Z",
    "valid_to": "2024-01-02T00:00:00Z"
}
```

### 2. Validate Certificate
```http
GET /api/v1/certificates/{certificate_id}
Authorization: Bearer <jwt_token>
```

Response:
```json
{
    "certificate_id": "cert-123",
    "identity_id": "id-123",
    "status": "valid",
    "valid_from": "2024-01-01T00:00:00Z",
    "valid_to": "2024-01-02T00:00:00Z",
    "subject": "CN=workload-123",
    "issuer": "CN=SPIRE CA"
}
```

### 3. Revoke Certificate
```http
DELETE /api/v1/certificates/{certificate_id}
Authorization: Bearer <jwt_token>

{
    "reason": "compromise"
}
```

Response:
```json
{
    "certificate_id": "cert-123",
    "status": "revoked",
    "revoked_at": "2024-01-01T12:00:00Z",
    "reason": "compromise"
}
```

## Policy Management

### 1. Create Policy
```http
POST /api/v1/policies
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "name": "workload-policy",
    "description": "Policy for workload access",
    "rules": [
        {
            "effect": "allow",
            "resources": ["api:read"],
            "conditions": {
                "namespace": "default",
                "service_account": "default"
            }
        }
    ]
}
```

Response:
```json
{
    "policy_id": "policy-123",
    "name": "workload-policy",
    "version": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "status": "active"
}
```

### 2. Evaluate Policy
```http
POST /api/v1/policies/evaluate
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "identity_id": "id-123",
    "resource": "api:read",
    "action": "get"
}
```

Response:
```json
{
    "allowed": true,
    "policy_id": "policy-123",
    "reason": "matched_rule",
    "evaluated_at": "2024-01-01T00:00:00Z"
}
```

### 3. Update Policy
```http
PUT /api/v1/policies/{policy_id}
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "name": "workload-policy",
    "description": "Updated policy for workload access",
    "rules": [
        {
            "effect": "allow",
            "resources": ["api:read", "api:write"],
            "conditions": {
                "namespace": "default",
                "service_account": "default"
            }
        }
    ]
}
```

Response:
```json
{
    "policy_id": "policy-123",
    "name": "workload-policy",
    "version": 2,
    "updated_at": "2024-01-01T12:00:00Z",
    "status": "active"
}
```

## Error Handling

### 1. Error Response Format
```json
{
    "error": {
        "code": "invalid_request",
        "message": "Invalid request parameters",
        "details": {
            "field": "workload_id",
            "reason": "required"
        }
    }
}
```

### 2. Common Error Codes
```yaml
error_codes:
  - code: "invalid_request"
    status: 400
    description: "Invalid request parameters"
  - code: "unauthorized"
    status: 401
    description: "Authentication required"
  - code: "forbidden"
    status: 403
    description: "Permission denied"
  - code: "not_found"
    status: 404
    description: "Resource not found"
  - code: "conflict"
    status: 409
    description: "Resource conflict"
  - code: "internal_error"
    status: 500
    description: "Internal server error"
```

### 3. Rate Limiting
```yaml
rate_limits:
  default:
    requests_per_second: 100
    burst: 200
  authentication:
    requests_per_second: 10
    burst: 20
```

## Conclusion

This API reference provides comprehensive documentation for the workload identity system. For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [Security Best Practices](security_best_practices.md)
- [Developer Guide](developer_guide.md)
- [Deployment Guide](deployment_guide.md)