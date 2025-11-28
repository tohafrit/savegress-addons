# HealthSync

HIPAA-compliant healthcare data synchronization and compliance engine for Savegress CDC platform.

## Features

- **FHIR R4 Support**
  - Patient, Practitioner, Organization resources
  - Clinical resources (Observation, Condition, Procedure)
  - Medication management
  - Diagnostic reports
  - Immunization records

- **HIPAA Compliance**
  - PHI identification and classification
  - Minimum necessary principle enforcement
  - 18 Safe Harbor identifiers tracking
  - Automated compliance validation
  - Violation detection and remediation

- **Audit Logging**
  - HIPAA-compliant audit trail
  - All PHI access logged
  - Security event tracking
  - 6-year retention
  - Real-time alerting

- **Patient Consent Management**
  - Consent directive storage
  - Purpose-based access control
  - Consent verification
  - Access request workflow
  - Expiration management

- **Data Anonymization**
  - Safe Harbor de-identification
  - Limited Dataset support
  - Date shifting
  - K-anonymity verification
  - Text redaction

## Quick Start

### Using Docker Compose

```bash
# Start all services
make docker-run

# View logs
make docker-logs

# Stop services
make docker-stop
```

### Local Development

```bash
# Install dependencies
make deps
make tidy

# Run in development mode
make dev

# Run tests
make test
```

## Configuration

HealthSync uses YAML configuration with environment variable substitution:

```yaml
server:
  port: 3005
  environment: ${ENVIRONMENT:-production}

compliance:
  hipaa_enabled: true
  minimum_necessary: true
  breach_notification: true
  retention_period: 52560h # 6 years

audit:
  enabled: true
  retention_days: 2190 # 6 years
  detail_level: full
  realtime_alerts: true

consent:
  required: true
  default_policy: deny
  allowed_purposes:
    - treatment
    - payment
    - operations
    - research
  expiration_days: 365

encryption:
  at_rest_enabled: true
  in_transit_enabled: true
  algorithm: AES-256-GCM
```

## API Endpoints

### FHIR Resources

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthsync/fhir/Patient` | Search patients |
| POST | `/api/v1/healthsync/fhir/Patient` | Create patient |
| GET | `/api/v1/healthsync/fhir/Patient/{id}` | Get patient |
| PUT | `/api/v1/healthsync/fhir/Patient/{id}` | Update patient |
| DELETE | `/api/v1/healthsync/fhir/Patient/{id}` | Delete patient |
| GET | `/api/v1/healthsync/fhir/Observation` | Search observations |
| POST | `/api/v1/healthsync/fhir/Observation` | Create observation |
| GET | `/api/v1/healthsync/fhir/Encounter` | Search encounters |

### Compliance

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthsync/compliance/violations` | List violations |
| GET | `/api/v1/healthsync/compliance/violations/{id}` | Get violation |
| POST | `/api/v1/healthsync/compliance/violations/{id}/resolve` | Resolve violation |
| POST | `/api/v1/healthsync/compliance/validate` | Validate resource |
| POST | `/api/v1/healthsync/compliance/phi-scan` | Scan for PHI |
| POST | `/api/v1/healthsync/compliance/minimum-necessary` | Check minimum necessary |
| GET | `/api/v1/healthsync/compliance/stats` | Get compliance stats |

### Audit

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthsync/audit/events` | List audit events |
| GET | `/api/v1/healthsync/audit/events/{id}` | Get audit event |
| GET | `/api/v1/healthsync/audit/stats` | Get audit stats |

### Consent

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthsync/consent` | List consents |
| POST | `/api/v1/healthsync/consent` | Create consent |
| GET | `/api/v1/healthsync/consent/{id}` | Get consent |
| POST | `/api/v1/healthsync/consent/{id}/revoke` | Revoke consent |
| POST | `/api/v1/healthsync/consent/check` | Check access |

### Access Requests

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthsync/access-requests` | List pending requests |
| POST | `/api/v1/healthsync/access-requests` | Create request |
| POST | `/api/v1/healthsync/access-requests/{id}/approve` | Approve request |
| POST | `/api/v1/healthsync/access-requests/{id}/deny` | Deny request |

### Anonymization

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/healthsync/anonymize/patient` | Anonymize patient |
| POST | `/api/v1/healthsync/anonymize/resource` | Anonymize resource |
| POST | `/api/v1/healthsync/anonymize/text` | Redact PHI from text |
| POST | `/api/v1/healthsync/anonymize/k-anonymity` | Check k-anonymity |

## HIPAA Safe Harbor Identifiers

HealthSync tracks and protects all 18 HIPAA Safe Harbor identifiers:

1. Names
2. Geographic data smaller than state
3. Dates (except year) related to individual
4. Phone numbers
5. Fax numbers
6. Email addresses
7. Social Security numbers
8. Medical record numbers
9. Health plan beneficiary numbers
10. Account numbers
11. Certificate/license numbers
12. Vehicle identifiers and serial numbers
13. Device identifiers and serial numbers
14. Web URLs
15. IP addresses
16. Biometric identifiers
17. Full-face photographs
18. Any other unique identifying number

## Anonymization Methods

### Safe Harbor Method
Removes or generalizes all 18 identifier types:
- Names removed
- Dates generalized to year only
- ZIP codes truncated to 3 digits
- All direct identifiers removed

### Limited Dataset Method
Allows certain data elements:
- Dates (admission, discharge, birth, death)
- City, state, ZIP code
- Age in years

Direct identifiers still removed.

### Date Shifting
Consistent date shifting per patient:
- Preserves intervals between dates
- Random shift within configurable range
- Same shift applied to all dates for one patient

## Compliance Validators

### Minimum Necessary
Enforces access to only data needed for intended purpose.

### PHI Exposure
Detects unprotected PHI in resources.

### Data Integrity
Validates required fields and audit trail metadata.

### Access Control
Checks for security labels and access restrictions.

### Encryption
Validates encryption of sensitive fields.

## Integration with Savegress CDC

HealthSync consumes CDC events from the Savegress platform:

```go
// CDC event handler with HIPAA compliance
func (e *Engine) HandleCDCEvent(event CDCEvent) {
    // Validate compliance
    result := e.compliance.ValidateResource(event.Data, event.ResourceType)
    if !result.Valid {
        e.recordViolation(result)
        return
    }

    // Log access
    e.audit.LogAccess(event)

    // Process event
    e.processResource(event)
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HealthSync                              │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Compliance │  │   Audit    │  │  Consent   │            │
│  │   Engine   │  │   Logger   │  │  Manager   │            │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘            │
│        │               │               │                    │
│  ┌─────┴───────────────┴───────────────┴─────┐             │
│  │            Anonymization Engine            │             │
│  │  ┌──────────┐  ┌───────────┐  ┌─────────┐ │             │
│  │  │Safe Harbor│  │Date Shift │  │K-Anon   │ │             │
│  │  └──────────┘  └───────────┘  └─────────┘ │             │
│  └───────────────────────────────────────────┘             │
├─────────────────────────────────────────────────────────────┤
│                    FHIR R4 Data Layer                        │
├─────────────────────────────────────────────────────────────┤
│                   Savegress CDC Platform                     │
└─────────────────────────────────────────────────────────────┘
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HEALTHSYNC_CONFIG` | Path to config file | - |
| `ENVIRONMENT` | Environment name | production |
| `JWT_SECRET` | JWT signing secret | - |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `REDIS_URL` | Redis connection URL | - |
| `HIPAA_ENABLED` | Enable HIPAA compliance | true |
| `AUDIT_ENABLED` | Enable audit logging | true |
| `CONSENT_REQUIRED` | Require consent for access | true |

## Regulatory Compliance

HealthSync is designed to help organizations comply with:

- **HIPAA** - Health Insurance Portability and Accountability Act
- **HITECH** - Health Information Technology for Economic and Clinical Health Act
- **42 CFR Part 2** - Confidentiality of Substance Use Disorder Patient Records
- **State Privacy Laws** - Various state-specific requirements

## License

Proprietary - Savegress Platform
