# API Reference Guide

This document provides detailed information about all APIs available in the workload identity system.

## Table of Contents
1. [Authentication API](#authentication-api)
2. [Identity Management API](#identity-management-api)
3. [Certificate Management API](#certificate-management-api)
4. [Policy Management API](#policy-management-api)
5. [Audit API](#audit-api)
6. [Health API](#health-api)
7. [Error Handling](#error-handling)
8. [Rate Limiting](#rate-limiting)
9. [Versioning](#versioning)

## Authentication API

### 1. Token Issuance
```http
POST /api/v1/auth/token
Content-Type: application/json

{
  "workload_id": "string",
  "service_account": "string",
  "audience": "string",
  "ttl": "string"
}
```

Response:
```json
{
  "token": "string",
  "expires_at": "string",
  "token_type": "string"
}
```

### 2. Token Validation
```http
POST /api/v1/auth/validate
Content-Type: application/json

{
  "token": "string"
}
```

Response:
```json
{
  "valid": boolean,
  "claims": {
    "sub": "string",
    "aud": "string",
    "exp": "integer",
    "iat": "integer"
  }
}
```

## Identity Management API

### 1. Create Identity
```http
POST /api/v1/identities
Content-Type: application/json

{
  "name": "string",
  "type": "string",
  "metadata": {
    "key": "value"
  }
}
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "type": "string",
  "created_at": "string",
  "metadata": {
    "key": "value"
  }
}
```

### 2. List Identities
```http
GET /api/v1/identities
Query Parameters:
  - page: integer
  - limit: integer
  - type: string
```

Response:
```json
{
  "identities": [
    {
      "id": "string",
      "name": "string",
      "type": "string",
      "created_at": "string"
    }
  ],
  "pagination": {
    "total": "integer",
    "page": "integer",
    "limit": "integer"
  }
}
```

## Certificate Management API

### 1. Issue Certificate
```http
POST /api/v1/certificates
Content-Type: application/json

{
  "identity_id": "string",
  "validity_period": "string",
  "key_type": "string",
  "key_size": "integer"
}
```

Response:
```json
{
  "certificate": "string",
  "private_key": "string",
  "valid_from": "string",
  "valid_to": "string",
  "serial_number": "string"
}
```

### 2. Revoke Certificate
```http
POST /api/v1/certificates/{serial_number}/revoke
Content-Type: application/json

{
  "reason": "string"
}
```

Response:
```json
{
  "revoked_at": "string",
  "reason": "string"
}
```

## Policy Management API

### 1. Create Policy
```http
POST /api/v1/policies
Content-Type: application/json

{
  "name": "string",
  "description": "string",
  "rules": [
    {
      "effect": "string",
      "actions": ["string"],
      "resources": ["string"],
      "conditions": {
        "key": "value"
      }
    }
  ]
}
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "created_at": "string",
  "rules": [
    {
      "effect": "string",
      "actions": ["string"],
      "resources": ["string"]
    }
  ]
}
```

### 2. Evaluate Policy
```http
POST /api/v1/policies/evaluate
Content-Type: application/json

{
  "identity_id": "string",
  "action": "string",
  "resource": "string",
  "context": {
    "key": "value"
  }
}
```

Response:
```json
{
  "allowed": boolean,
  "reason": "string",
  "evaluated_policies": ["string"]
}
```

## Audit API

### 1. Query Audit Logs
```http
GET /api/v1/audit/logs
Query Parameters:
  - start_time: string
  - end_time: string
  - identity_id: string
  - action: string
  - resource: string
  - page: integer
  - limit: integer
```

Response:
```json
{
  "logs": [
    {
      "timestamp": "string",
      "identity_id": "string",
      "action": "string",
      "resource": "string",
      "result": "string",
      "metadata": {
        "key": "value"
      }
    }
  ],
  "pagination": {
    "total": "integer",
    "page": "integer",
    "limit": "integer"
  }
}
```

## Health API

### 1. Health Check
```http
GET /api/v1/health
```

Response:
```json
{
  "status": "string",
  "components": {
    "database": "string",
    "certificate_authority": "string",
    "policy_engine": "string"
  },
  "version": "string",
  "uptime": "string"
}
```

### 2. Metrics
```http
GET /api/v1/metrics
```

Response:
```json
{
  "requests_total": "integer",
  "requests_failed": "integer",
  "average_response_time": "float",
  "active_connections": "integer"
}
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": {
      "key": "value"
    }
  }
}
```

### Common Error Codes
- `400`: Bad Request
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Not Found
- `429`: Too Many Requests
- `500`: Internal Server Error

## Rate Limiting

### Rate Limit Headers
```
X-RateLimit-Limit: integer
X-RateLimit-Remaining: integer
X-RateLimit-Reset: string
```

### Default Limits
```yaml
rate_limits:
  authentication:
    requests_per_minute: 60
    burst: 10
  certificate_management:
    requests_per_minute: 30
    burst: 5
  policy_management:
    requests_per_minute: 30
    burst: 5
```

## Versioning

### API Versioning
- All APIs are versioned in the URL path: `/api/v1/`
- Major version changes will increment the version number
- Minor changes will be backward compatible
- Deprecated features will be announced 6 months in advance

### Version Header
```
X-API-Version: string
```

## Conclusion

This guide provides comprehensive API documentation for the workload identity system. Remember to:
- Always include proper authentication
- Handle rate limits appropriately
- Implement proper error handling
- Follow versioning guidelines
- Monitor API usage

For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [Security Best Practices](security_best_practices.md)
- [Deployment Guide](deployment_guide.md) 