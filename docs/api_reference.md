# API Reference Guide

This document provides detailed information about all APIs available in the workload identity system. The API is designed to support modern cloud-native workloads, CI/CD automation, and enhanced security features while maintaining backwards compatibility.

## Overview

The Workload Identity API provides a comprehensive set of endpoints for:
- Authentication and token management
- Identity lifecycle management
- Certificate issuance and revocation
- Policy management and evaluation
- Audit log querying
- System health monitoring

All APIs are versioned under `/api/v1/` and use JSON over HTTPS. Authentication is required for all endpoints, with different levels of access control based on the operation's sensitivity.

### Security Requirements
- All API calls must use HTTPS
- Sensitive operations require mTLS
- Rate limiting is enforced per endpoint
- Audit logging is enabled for all operations

### API Versioning
- Current stable version: v1
- Backwards compatible changes only
- New features added as optional fields
- Breaking changes will be introduced in v2

## Authentication API

### 1. Token Issuance
```http
POST /api/v1/auth/token
Content-Type: application/json
Authorization: Bearer <token>  # or mTLS client certificate

{
  "workload_id": "string",
  "service_account": "string",
  "audience": "string",
  "ttl": "string",
  "claims": {
    "key": "value"
  },
  "oidc_compatible": boolean  # Optional: Request OIDC-compatible token
}
```

Response:
```json
{
  "token": "string",
  "expires_at": "string",
  "token_type": "string",
  "claims": {
    "sub": "string",
    "aud": "string",
    "exp": "integer",
    "iat": "integer",
    "iss": "string",
    "jti": "string"
  }
}
```

#### Security Considerations
- Requires mTLS or valid token for authentication
- TTL is capped by policy (e.g., max 15m for high-sensitivity)
- Tokens are short-lived by default
- Supports OIDC integration for external services

### 2. Token Validation
```http
POST /api/v1/auth/validate
Content-Type: application/json
Authorization: Bearer <token>

{
  "token": "string",
  "verify_signature": boolean,  # Optional: Verify token signature
  "verify_expiry": boolean     # Optional: Verify token expiry
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
    "iat": "integer",
    "iss": "string",
    "jti": "string"
  },
  "reason": "string"  # If invalid, explains why
}
```

#### Security Considerations
- Validates token signature and expiry
- Checks token revocation status
- Verifies token issuer and audience
- Supports custom claim validation

### 3. Token Exchange
```http
POST /api/v1/auth/exchange
Content-Type: application/json
Authorization: Bearer <token>

{
  "token": "string",
  "target_audience": "string",
  "scopes": ["string"]
}
```

Response:
```json
{
  "token": "string",
  "expires_at": "string",
  "token_type": "string",
  "scopes": ["string"]
}
```

#### Security Considerations
- Supports OAuth2 token exchange
- Enforces scope restrictions
- Validates source token
- Maintains audit trail

### 4. Token Revocation
```http
POST /api/v1/auth/revoke
Content-Type: application/json
Authorization: Bearer <token>

{
  "token": "string",
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

#### Security Considerations
- Requires admin privileges
- Propagates to all components
- Updates revocation lists
- Maintains audit trail

## Identity Management API

### 1. Create Identity
```http
POST /api/v1/identities
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "string",
  "type": "string",
  "metadata": {
    "key": "value"
  },
  "ttl": "string",
  "approval_required": boolean,
  "approvers": ["string"]
}
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "type": "string",
  "metadata": {
    "key": "value"
  },
  "created_at": "string",
  "expires_at": "string",
  "status": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Supports approval workflows
- Enforces metadata validation
- Maintains audit trail

### 2. Get Identity
```http
GET /api/v1/identities/{id}
Authorization: Bearer <token>
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "type": "string",
  "metadata": {
    "key": "value"
  },
  "created_at": "string",
  "expires_at": "string",
  "status": "string",
  "last_used": "string"
}
```

#### Security Considerations
- Enforces RBAC
- Filters sensitive metadata
- Tracks access patterns
- Supports audit queries

### 3. Update Identity
```http
PATCH /api/v1/identities/{id}
Content-Type: application/json
Authorization: Bearer <token>

{
  "metadata": {
    "key": "value"
  },
  "ttl": "string",
  "status": "string"
}
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "type": "string",
  "metadata": {
    "key": "value"
  },
  "updated_at": "string",
  "expires_at": "string",
  "status": "string"
}
```

#### Security Considerations
- Requires appropriate permissions
- Validates metadata changes
- Enforces TTL limits
- Maintains change history

### 4. Delete Identity
```http
DELETE /api/v1/identities/{id}
Authorization: Bearer <token>
```

Response:
```json
{
  "deleted_at": "string",
  "reason": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Revokes all associated tokens
- Maintains audit trail
- Supports recovery window

## Certificate Management API

### 1. Issue Certificate
```http
POST /api/v1/certificates
Content-Type: application/json
Authorization: Bearer <token>

{
  "identity_id": "string",
  "csr": "string",
  "validity_period": "string",
  "key_type": "string",
  "key_size": "integer",
  "san": ["string"],
  "auto_renew": boolean
}
```

Response:
```json
{
  "certificate": "string",
  "private_key": "string",
  "ca_chain": ["string"],
  "valid_from": "string",
  "valid_to": "string",
  "serial_number": "string"
}
```

#### Security Considerations
- Requires mTLS
- Validates CSR
- Enforces key strength
- Supports auto-renewal

### 2. Revoke Certificate
```http
POST /api/v1/certificates/{serial}/revoke
Content-Type: application/json
Authorization: Bearer <token>

{
  "reason": "string",
  "revocation_time": "string"
}
```

Response:
```json
{
  "revoked_at": "string",
  "reason": "string",
  "crl_url": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Updates CRL/OCSP
- Propagates revocation
- Maintains audit trail

### 3. Get Certificate
```http
GET /api/v1/certificates/{serial}
Authorization: Bearer <token>
```

Response:
```json
{
  "certificate": "string",
  "ca_chain": ["string"],
  "valid_from": "string",
  "valid_to": "string",
  "serial_number": "string",
  "status": "string",
  "revocation_info": {
    "revoked_at": "string",
    "reason": "string"
  }
}
```

#### Security Considerations
- Enforces RBAC
- Validates certificate
- Checks revocation status
- Supports audit queries

### 4. List Certificates
```http
GET /api/v1/certificates
Authorization: Bearer <token>

Query Parameters:
- identity_id: string
- status: string
- valid_from: string
- valid_to: string
- limit: integer
- offset: integer
```

Response:
```json
{
  "certificates": [
    {
      "serial_number": "string",
      "identity_id": "string",
      "valid_from": "string",
      "valid_to": "string",
      "status": "string"
    }
  ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

#### Security Considerations
- Enforces RBAC
- Supports pagination
- Filters sensitive data
- Rate limited

## Policy Management API

### 1. Create Policy
```http
POST /api/v1/policies
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "string",
  "description": "string",
  "rules": [
    {
      "effect": "string",
      "action": "string",
      "resource": "string",
      "conditions": {
        "key": "value"
      }
    }
  ],
  "metadata": {
    "key": "value"
  },
  "enforcement_mode": "string"
}
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "rules": [
    {
      "effect": "string",
      "action": "string",
      "resource": "string",
      "conditions": {
        "key": "value"
      }
    }
  ],
  "metadata": {
    "key": "value"
  },
  "enforcement_mode": "string",
  "created_at": "string",
  "updated_at": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Validates policy syntax
- Enforces policy limits
- Maintains audit trail

### 2. Evaluate Policy
```http
POST /api/v1/policies/evaluate
Content-Type: application/json
Authorization: Bearer <token>

{
  "identity": {
    "id": "string",
    "type": "string",
    "attributes": {
      "key": "value"
    }
  },
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
  "matched_policies": ["string"],
  "evaluation_time": "string"
}
```

#### Security Considerations
- Requires authentication
- Caches results
- Rate limited
- Logs decisions

### 3. Update Policy
```http
PATCH /api/v1/policies/{id}
Content-Type: application/json
Authorization: Bearer <token>

{
  "description": "string",
  "rules": [
    {
      "effect": "string",
      "action": "string",
      "resource": "string",
      "conditions": {
        "key": "value"
      }
    }
  ],
  "metadata": {
    "key": "value"
  },
  "enforcement_mode": "string"
}
```

Response:
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "rules": [
    {
      "effect": "string",
      "action": "string",
      "resource": "string",
      "conditions": {
        "key": "value"
      }
    }
  ],
  "metadata": {
    "key": "value"
  },
  "enforcement_mode": "string",
  "updated_at": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Validates changes
- Maintains version history
- Notifies affected services

### 4. Delete Policy
```http
DELETE /api/v1/policies/{id}
Authorization: Bearer <token>
```

Response:
```json
{
  "deleted_at": "string",
  "reason": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Checks dependencies
- Maintains audit trail
- Supports recovery window

## Audit API

### 1. Query Audit Logs
```http
GET /api/v1/audit/logs
Authorization: Bearer <token>

Query Parameters:
- start_time: string
- end_time: string
- identity_id: string
- action: string
- resource: string
- status: string
- limit: integer
- offset: integer
```

Response:
```json
{
  "logs": [
    {
      "id": "string",
      "timestamp": "string",
      "identity_id": "string",
      "action": "string",
      "resource": "string",
      "status": "string",
      "details": {
        "key": "value"
      }
    }
  ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

#### Security Considerations
- Requires admin privileges
- Supports pagination
- Rate limited
- Retention policy enforced

### 2. Export Audit Logs
```http
POST /api/v1/audit/export
Content-Type: application/json
Authorization: Bearer <token>

{
  "start_time": "string",
  "end_time": "string",
  "format": "string",
  "filters": {
    "key": "value"
  }
}
```

Response:
```json
{
  "export_id": "string",
  "status": "string",
  "download_url": "string",
  "expires_at": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Asynchronous processing
- Secure download URLs
- Limited retention

### 3. Get Audit Statistics
```http
GET /api/v1/audit/stats
Authorization: Bearer <token>

Query Parameters:
- start_time: string
- end_time: string
- group_by: string
```

Response:
```json
{
  "total_events": "integer",
  "by_action": {
    "key": "integer"
  },
  "by_status": {
    "key": "integer"
  },
  "by_identity": {
    "key": "integer"
  },
  "time_series": [
    {
      "timestamp": "string",
      "count": "integer"
    }
  ]
}
```

#### Security Considerations
- Requires admin privileges
- Cached results
- Rate limited
- Data retention policy

### 4. Configure Audit Settings
```http
PUT /api/v1/audit/settings
Content-Type: application/json
Authorization: Bearer <token>

{
  "retention_period": "string",
  "storage_location": "string",
  "encryption": {
    "enabled": boolean,
    "key_id": "string"
  },
  "notifications": {
    "enabled": boolean,
    "channels": ["string"]
  }
}
```

Response:
```json
{
  "retention_period": "string",
  "storage_location": "string",
  "encryption": {
    "enabled": boolean,
    "key_id": "string"
  },
  "notifications": {
    "enabled": boolean,
    "channels": ["string"]
  },
  "updated_at": "string"
}
```

#### Security Considerations
- Requires admin privileges
- Validates settings
- Maintains audit trail
- Supports encryption

## Health API

### 1. Health Check
```http
GET /api/v1/health
Authorization: Bearer <token>
```

Response:
```json
{
  "status": "string",
  "components": {
    "database": {
      "status": "string",
      "latency": "string",
      "last_sync": "string"
    },
    "certificate_authority": {
      "status": "string",
      "issuer_status": "string",
      "crl_status": "string"
    },
    "policy_engine": {
      "status": "string",
      "cache_status": "string",
      "last_update": "string"
    },
    "key_management": {
      "status": "string",
      "hsm_status": "string",
      "key_rotation": "string"
    }
  },
  "version": "string",
  "uptime": "string",
  "last_maintenance": "string"
}
```

#### Security Considerations
- Requires authentication
- Rate limited
- Cached results
- Minimal sensitive data

### 2. Metrics
```http
GET /api/v1/metrics
Authorization: Bearer <token>

Query Parameters:
- start_time: string
- end_time: string
- interval: string
```

Response:
```json
{
  "requests": {
    "total": "integer",
    "failed": "integer",
    "by_endpoint": {
      "key": "integer"
    },
    "by_status": {
      "key": "integer"
    }
  },
  "performance": {
    "average_response_time": "float",
    "p95_response_time": "float",
    "p99_response_time": "float"
  },
  "resources": {
    "cpu_usage": "float",
    "memory_usage": "float",
    "disk_usage": "float"
  },
  "security": {
    "failed_auth_attempts": "integer",
    "rate_limit_hits": "integer",
    "policy_violations": "integer"
  }
}
```

#### Security Considerations
- Requires admin privileges
- Rate limited
- Aggregated data only
- Retention policy enforced

## Metrics API

### Get System Metrics
```http
GET /api/v1/metrics/system
```

Retrieves system-wide metrics for monitoring and compliance.

#### Query Parameters
- `timeframe` (string, required): Time range for metrics (e.g., "1h", "24h", "7d")
- `metrics` (string[], optional): Specific metrics to retrieve
- `aggregation` (string, optional): Aggregation method (e.g., "avg", "max", "min")

#### Response
```json
{
  "metrics": {
    "availability": {
      "value": 99.99,
      "threshold": 99.9,
      "status": "healthy"
    },
    "response_time": {
      "value": 150,
      "threshold": 200,
      "unit": "ms",
      "status": "healthy"
    },
    "error_rate": {
      "value": 0.05,
      "threshold": 0.1,
      "unit": "percent",
      "status": "healthy"
    }
  },
  "timestamp": "2024-03-20T10:00:00Z"
}
```

#### Security Considerations
- Requires metrics.read permission
- Rate limited to prevent abuse
- Metrics are cached for performance
- Sensitive metrics require elevated permissions

### Get Compliance Metrics
```http
GET /api/v1/metrics/compliance
```

Retrieves compliance-specific metrics and status.

#### Query Parameters
- `framework` (string, required): Compliance framework (e.g., "GDPR", "HIPAA", "PCI-DSS")
- `timeframe` (string, required): Time range for metrics
- `controls` (string[], optional): Specific controls to check

#### Response
```json
{
  "framework": "GDPR",
  "status": "compliant",
  "metrics": {
    "data_protection": {
      "encryption_coverage": 100,
      "access_controls": 98,
      "audit_coverage": 100
    },
    "privacy": {
      "data_minimization": 95,
      "purpose_limitation": 100,
      "storage_limitation": 100
    }
  },
  "last_updated": "2024-03-20T10:00:00Z"
}
```

#### Security Considerations
- Requires compliance.read permission
- Metrics are signed for integrity
- Historical data is retained for audit
- Access is logged for compliance

## Automation API

### Run Compliance Check
```http
POST /api/v1/automation/compliance/check
```

Initiates an automated compliance check.

#### Request Body
```json
{
  "framework": "GDPR",
  "scope": {
    "systems": ["identity", "certificate", "policy"],
    "controls": ["access", "encryption", "audit"]
  },
  "options": {
    "generate_report": true,
    "notify_on_failure": true,
    "remediation": "automatic"
  }
}
```

#### Response
```json
{
  "check_id": "comp-check-123",
  "status": "running",
  "estimated_completion": "2024-03-20T10:05:00Z",
  "report_url": "/api/v1/reports/comp-check-123"
}
```

#### Security Considerations
- Requires automation.execute permission
- Checks are rate limited
- Results are signed for integrity
- Access is logged for audit

### Get Automation Status
```http
GET /api/v1/automation/status/{check_id}
```

Retrieves the status of an automated compliance check.

#### Response
```json
{
  "check_id": "comp-check-123",
  "status": "completed",
  "results": {
    "total_controls": 50,
    "passed": 48,
    "failed": 2,
    "remediated": 1
  },
  "report_url": "/api/v1/reports/comp-check-123",
  "completed_at": "2024-03-20T10:04:30Z"
}
```

#### Security Considerations
- Requires automation.read permission
- Status updates are real-time
- Results are cached for performance
- Access is logged for audit

## Risk Management API

### Assess Risk
```http
POST /api/v1/risk/assess
```

Performs a risk assessment for the system.

#### Request Body
```json
{
  "scope": {
    "systems": ["identity", "certificate", "policy"],
    "threats": ["unauthorized_access", "data_breach", "service_disruption"]
  },
  "options": {
    "include_controls": true,
    "include_remediation": true,
    "risk_threshold": "high"
  }
}
```

#### Response
```json
{
  "assessment_id": "risk-assess-123",
  "status": "completed",
  "results": {
    "overall_risk": "medium",
    "threats": [
      {
        "name": "unauthorized_access",
        "risk_level": "low",
        "controls": ["mfa", "audit_logging"],
        "effectiveness": 95
      }
    ],
    "recommendations": [
      {
        "threat": "data_breach",
        "action": "enhance_encryption",
        "priority": "high"
      }
    ]
  },
  "completed_at": "2024-03-20T10:00:00Z"
}
```

#### Security Considerations
- Requires risk.assess permission
- Assessments are rate limited
- Results are signed for integrity
- Access is logged for audit

### Get Risk Metrics
```http
GET /api/v1/risk/metrics
```

Retrieves risk-related metrics and trends.

#### Query Parameters
- `timeframe` (string, required): Time range for metrics
- `metrics` (string[], optional): Specific metrics to retrieve

#### Response
```json
{
  "metrics": {
    "risk_levels": {
      "critical": 0,
      "high": 2,
      "medium": 5,
      "low": 15
    },
    "control_effectiveness": {
      "access_control": 98,
      "encryption": 100,
      "monitoring": 95
    },
    "incident_frequency": {
      "value": 0.5,
      "unit": "per_month",
      "trend": "decreasing"
    }
  },
  "last_updated": "2024-03-20T10:00:00Z"
}
```

#### Security Considerations
- Requires risk.read permission
- Metrics are cached for performance
- Historical data is retained for audit
- Access is logged for compliance

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": {
      "key": "value"
    },
    "request_id": "string",
    "timestamp": "string"
  }
}
```

### Common Error Codes
- `400`: Bad Request - Invalid parameters or request format
- `401`: Unauthorized - Missing or invalid authentication
- `403`: Forbidden - Insufficient permissions
- `404`: Not Found - Resource doesn't exist
- `409`: Conflict - Resource state conflict
- `422`: Unprocessable Entity - Validation failed
- `429`: Too Many Requests - Rate limit exceeded
- `500`: Internal Server Error - Unexpected server error
- `503`: Service Unavailable - Service maintenance or overload

### Error Handling Best Practices
1. Always include request_id for tracking
2. Provide clear, actionable error messages
3. Include relevant details for debugging
4. Follow consistent error format
5. Log all errors with appropriate severity

## Rate Limiting

### Rate Limit Headers
```
X-RateLimit-Limit: integer
X-RateLimit-Remaining: integer
X-RateLimit-Reset: string
X-RateLimit-Burst: integer
```

### Default Limits
```yaml
rate_limits:
  authentication:
    requests_per_minute: 60
    burst: 10
    per_ip: true
  certificate_management:
    requests_per_minute: 30
    burst: 5
    per_identity: true
  policy_management:
    requests_per_minute: 30
    burst: 5
    per_tenant: true
  audit_logs:
    requests_per_minute: 20
    burst: 3
    per_user: true
```

### Rate Limit Best Practices
1. Use appropriate limits per endpoint
2. Implement burst handling
3. Consider different limit types (IP, user, tenant)
4. Provide clear rate limit headers
5. Log rate limit violations

## Versioning

### API Versioning
- All APIs are versioned in the URL path: `/api/v1/`
- Major version changes will increment the version number
- Minor changes will be backward compatible
- Deprecated features will be announced 6 months in advance
- Breaking changes require new major version

### Version Header
```
X-API-Version: string
```

### Versioning Best Practices
1. Maintain backward compatibility
2. Document all changes
3. Provide migration guides
4. Support multiple versions
5. Monitor version usage

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
- [Developer Guide](developer_guide.md)