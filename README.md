# Chirpy - REST API

A RESTful API for a Twitter-like microblogging service built with Go. Users can create chirps (short messages), manage accounts, and enjoy premium features.

## Features

- **User Management**: Create accounts, authenticate, and manage user profiles
- **Chirps**: Create, retrieve, and delete short messages (max 140 characters)
- **Authentication**: JWT-based authentication with refresh token support
- **Filtering**: Query chirps by author or sort by creation date
- **Premium Features**: Chirpy Red subscription support via webhook integration
- **Content Moderation**: Automatic profanity filtering for prohibited words
- **Metrics**: Admin dashboard with server usage metrics

## API Endpoints

### Health & Metrics
- `GET /api/healthz` - Health check
- `GET /admin/metrics` - View server metrics (admin only)
- `POST /admin/reset` - Reset metrics (admin only)

### Users
- `POST /api/users` - Create a new user
- `PUT /api/users` - Update user email and password (requires JWT)
- `POST /api/login` - Authenticate user and get tokens

### Authentication
- `POST /api/refresh` - Get a new JWT token using refresh token
- `POST /api/revoke` - Revoke refresh token (logout)

### Chirps
- `GET /api/chirps` - Get all chirps (supports `sort` query: `asc`/`desc`, default `asc`)
- `GET /api/chirps?author_id={id}` - Get chirps by specific author
- `GET /api/chirps?sort=desc` - Get all chirps sorted descending by creation date
- `GET /api/chirps/{chirpID}` - Get a specific chirp
- `POST /api/chirps` - Create a new chirp (requires JWT)
- `DELETE /api/chirps/{chirpID}` - Delete a chirp (requires JWT, author only)

### Webhooks
- `POST /api/polka/webhooks` - Handle Polka payment webhooks for Chirpy Red upgrades

## Setup

### Prerequisites
- Go 1.26.2+
- PostgreSQL database
- `.env` file with configuration

### Environment Variables
```
DB_URL=postgres://user:password@localhost:5432/chirpy?sslmode=disable
PLATFORM=dev
SECRET=your-jwt-secret-key
POLKA_KEY=your-polka-api-key
```

### Building
```bash
go build -o httpserver.exe
```

### Running
```bash
./httpserver.exe
```

Server runs on `http://localhost:8080`

## Database

The API uses PostgreSQL with sqlc for type-safe SQL queries. Database schema includes:
- Users table with hashed passwords
- Chirps table with timestamps and user relationships
- Refresh tokens table for session management
- Chirpy Red subscription status

## Authentication

Protected endpoints require a JWT token in the `Authorization` header:
```
Authorization: Bearer {token}
```

## Content Moderation

The following words are automatically censored:
- kerfuffle
- sharbert
- fornax

Censored words are replaced with `****` in chirp responses.
