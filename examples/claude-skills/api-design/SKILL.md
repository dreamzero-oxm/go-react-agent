---
name: api-design
version: 1.0
description: |
  Provides REST API design guidance including endpoint structure,
  HTTP methods, status codes, and documentation best practices.
tags:
  - api
  - rest
  - design
  - http
---

# API Design Skill

This skill provides guidance for designing RESTful APIs.

## REST Principles

### Resource Identification
- Use nouns, not verbs: `/users` not `/getUsers`
- Use plural nouns for collections: `/users`, `/posts`
- Hierarchical structure: `/users/{id}/posts/{post_id}`

### HTTP Methods

| Method | Purpose | Safe? | Idempotent? |
|--------|---------|-------|-------------|
| GET | Retrieve resource | Yes | Yes |
| POST | Create resource | No | No |
| PUT | Update/Replace resource | No | Yes |
| PATCH | Partial update | No | No |
| DELETE | Remove resource | No | Yes |

### Status Code Guide

#### Success Codes
- `200 OK` - Successful GET, PUT, PATCH
- `201 Created` - Successful POST
- `204 No Content` - Successful DELETE, PUT with no response body

#### Client Error Codes
- `400 Bad Request` - Malformed request
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Valid auth but insufficient permissions
- `404 Not Found` - Resource doesn't exist
- `409 Conflict` - Conflict with current state
- `422 Unprocessable Entity` - Semantic errors

#### Server Error Codes
- `500 Internal Server Error` - Unexpected server error
- `503 Service Unavailable` - Service temporarily down

## Request/Response Design

### Request Format
```json
{
  "data": {
    "type": "user",
    "attributes": {
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

### Response Format
```json
{
  "data": {
    "type": "user",
    "id": "123",
    "attributes": {
      "name": "John Doe",
      "email": "john@example.com"
    }
  },
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

### Error Response Format
```json
{
  "errors": [
    {
      "code": "VALIDATION_ERROR",
      "title": "Invalid input",
      "detail": "Email is required",
      "source": { "pointer": "/data/attributes/email" }
    }
  ]
}
```

## Best Practices

### Versioning
- Use URL versioning: `/api/v1/users`
- Or header versioning: `Accept: application/vnd.api+json; version=1`

### Pagination
```
GET /users?page=1&per_page=50
```

Response should include:
```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 50,
    "total": 150,
    "total_pages": 3
  }
}
```

### Filtering and Sorting
```
GET /users?status=active&sort=created_at:desc
```

### Rate Limiting
Return headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1609459200
```

## Security Considerations

1. **Always use HTTPS** in production
2. **Validate all input** - never trust client data
3. **Use authentication** - API keys, OAuth2, JWT
4. **Implement rate limiting** - prevent abuse
5. **Sanitize error messages** - don't leak sensitive info
6. **Use CORS carefully** - limit allowed origins

## Documentation Best Practices

1. **Provide examples** for every endpoint
2. **Document all error codes**
3. **Keep docs in sync with code**
4. **Include request/response schemas**
5. **Document authentication requirements**
