---
title: Add Health Check Endpoint
type: feature
priority: normal
labels:
  - api
  - infrastructure
---

# Add Health Check Endpoint

Add a `/health` endpoint that reports the application's current status. This is needed for load balancer health checks and monitoring.

## Requirements

- Return HTTP 200 with a JSON response when the service is healthy
- Include basic metadata: service name, version, and uptime
- Response time should be under 50ms (no expensive checks)
- No authentication required on this endpoint

## Expected Response

```json
{
  "status": "ok",
  "service": "my-app",
  "version": "1.0.0",
  "uptime": "2h15m"
}
```

## Constraints

- Use the existing HTTP framework/router already in the project
- Follow the project's existing patterns for route registration
- Add a corresponding test
