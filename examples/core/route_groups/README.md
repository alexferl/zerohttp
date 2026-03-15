# Route Groups Example

Demonstrates organizing routes with groups and nested middleware.

## Running

```bash
go run .
```

## Endpoints

| Endpoint                     | Description      | Auth       |
|------------------------------|------------------|------------|
| `GET /`                      | Public welcome   | No         |
| `GET /users`                 | List all users   | No         |
| `POST /users`                | Create user      | No         |
| `GET /users/{id}`            | Get user by ID   | No         |
| `PUT /users/{id}`            | Update user      | No         |
| `DELETE /users/{id}`         | Delete user      | No         |
| `GET /admin/dashboard`       | Admin dashboard  | Basic Auth |
| `GET /admin/settings`        | Admin settings   | Basic Auth |
| `POST /admin/users/{id}/ban` | Ban user         | Basic Auth |
| `GET /v2/public/status`      | Public v2 status | No         |
| `GET /v2/profile`            | User profile     | Basic Auth |
| `PUT /v2/profile`            | Update profile   | Basic Auth |

## Features

- **Group middleware** - Apply middleware to a set of routes
- **Nested groups** - Groups within groups for hierarchical middleware
- **Isolation** - Each group has its own router instance

## Basic Auth

Default credentials for protected routes:
- Username: `admin`
- Password: `admin`
