# APIary - Provider-Agnostic Compute Control Plane

APIary is GlueOps' unified control plane for provisioning and managing compute infrastructure across multiple providers (AWS, Hetzner, Proxmox, and others). It provides a common REST API with built-in multi-tenancy, fine-grained access controls, and enterprise-grade security features.

## Features

- **Multi-Provider Support**: Unified API for AWS, Hetzner, Proxmox, and extensible to additional providers
- **Authentication & Authorization**: Multiple auth methods (password, OAuth2, WebAuthn/passkeys, TOTP)
- **Organization & Team Management**: Role-based access control (owner, admin, member) with invitation workflows
- **Tenant Isolation**: Encrypted per-tenant databases with automatic provisioning and lifecycle management
- **Credential Management**: Support for both static secrets and federated role assumption (OIDC)
- **Security Policies**: Configurable platform and organization-level policies (MFA, passkey requirements, password complexity, session lifetime)
- **Audit Logging**: Comprehensive authentication and action audit trails
- **Email & Notifications**: Queued email delivery with encryption for sensitive content
- **Session Management**: Secure token-based sessions with refresh token rotation
- **Domain Cohort Analysis**: Abuse prevention through domain reputation analysis
- **API-First**: Full OpenAPI specification with auto-generated SDKs (TypeScript, Go)

## Tech Stack

**Backend**
- Go 1.26 with Cobra CLI framework
- PostgreSQL with GORM ORM
- Huma/v2 REST API framework (OpenAPI 3.0)
- River job queue for async processing
- Atlas + Goose for database migrations
- WebAuthn for passkey authentication
- JWT with RS256 signing

**Frontend**
- React 19 with TypeScript
- Vite build tool
- TailwindCSS for styling
- React Router for SPA navigation
- React Query for server state
- Generated TypeScript SDK (Hey API)

**Infrastructure**
- Docker & Docker Compose for local development
- PostgreSQL 18
- pgBouncer for connection pooling
- Mailpit for local email testing
- GoReleaser for automated releases

## Prerequisites

- **Go 1.26+** (for backend development)
- **Node.js 18+** & **Yarn 3+** (for frontend)
- **PostgreSQL 18+** (or Docker for local dev)
- **Docker & Docker Compose** (recommended for local setup)
- **psql** CLI tool (optional, for direct database access)

Optional for code generation:
- `atlas` CLI for schema migrations
- `openapi-generator-cli` for SDK generation
- `tfplugingen-openapi` & `tfplugingen-framework` for Terraform provider generation

## Setup & Installation

### Local Development with Docker

1. **Clone and prepare environment**:
   ```bash
   git clone https://github.com/glueops/apiary.git
   cd apiary
   cp .env.example .env  # Configure as needed
   ```

2. **Start services**:
   ```bash
   just up
   ```
   This starts PostgreSQL, pgBouncer, pgWeb (UI), Mailpit, and runs migrations.

3. **Seed test data** (optional):
   ```bash
   go run . seed
   ```

4. **Start development servers**:
   ```bash
   mprocs  # Runs API, worker, and frontend UI concurrently
   ```
   - API: http://localhost:8080
   - Frontend: http://localhost:5173
   - pgWeb (database browser): http://localhost:8081
   - Mailpit (email testing): http://localhost:8025

### Manual PostgreSQL Setup

If not using Docker:

1. Create databases:
   ```bash
   psql -c "CREATE DATABASE apiary;"
   psql -c "CREATE DATABASE apiary_test;"
   psql -c "CREATE DATABASE atlas_dev;"
   ```

2. Set environment variables in `.env`:
   ```bash
   DATABASE_URL=postgres://user:password@localhost:5432/apiary
   TEST_DATABASE_URL=postgres://user:password@localhost:5432/apiary_test
   ENCRYPTION_MASTER_KEY=<base64-encoded-32-byte-key>
   ```

3. Run migrations:
   ```bash
   just migrate up
   ```

## Development Workflow

### Common Commands

```bash
# Run tests (requires test database)
just test

# Check coverage with function-level breakdown
just cover

# Assess code quality
just quality

# Generate database migrations
just generate add_column_name

# Generate OpenAPI spec and SDKs
just sdk

# Open database shell
go run . psql

# Manage migrations
just migrate up
just migrate down
just migrate status

# Watch configuration
go run . show-config --output json
```

### Database Migrations

Migrations are managed with Goose and Atlas:

- **Control DB** (`internal/migrations/control`): Platform schema (users, orgs, policies)
- **Tenant DB** (`internal/migrations/tenant`): Per-organization schema (credentials, estates)

Generate new migrations:
```bash
just generate my_migration_name
```

This creates migrations for both control and tenant databases. Migrations use Atlas for automatic schema diffing from GORM models.

### Frontend Development

```bash
cd ui
yarn dev        # Start Vite dev server
yarn build      # Production build
yarn test       # Run tests
yarn lint       # Run oxlint
yarn format     # Auto-format with Prettier
```

### Code Generation

```bash
# Generate OpenAPI spec and SDKs
just sdk

# Generate Terraform provider
just terraform
```

Generated outputs:
- `docs/openapi.json` - OpenAPI 3.0 specification
- `ui/src/sdk/` - TypeScript SDK (embedded in frontend)
- `sdk/ts/` - Standalone TypeScript/Node.js SDK
- `sdk/go/` - Standalone Go SDK
- `terraform-provider-apiary/` - Terraform provider

### Environment Variables

Key configuration (see `.env.example`):

```bash
# Database
DATABASE_URL=postgres://...
ENCRYPTION_MASTER_KEY=<base64-32-bytes>

# JWT
JWT_SECRET=<random-secret>
JWT_ISSUER=apiary
JWT_ACCESS_TTL_MIN=60
JWT_REFRESH_TTL_HOUR=720

# Email/SMTP
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=noreply@apiary.local

# Organization Admission Control
ORGS_MAX_TOTAL=1000
ORGS_MAX_NEW_PER_HOUR=10
ADMISSION_UNPROVEN_ORGS_PER_DOMAIN=5

# Feature Flags
AUTH_SEED_PUBLIC_DOMAIN_RULES=true
FRONTEND_MODE=proxy  # or 'spa', 'none'
```

## Architecture Overview

### Database Design

**Multi-Database Architecture**:
- **Control Database**: Single shared database for all users and organizations
  - Users, organizations, memberships, roles
  - Authentication state (sessions, tokens, audit events)
  - Policies and access rules
  - Job queue (River)
- **Tenant Databases**: One per organization
  - Credentials (encrypted)
  - Estates (compute infrastructure)
  - Organization-specific data

**Data Protection**:
- Sensitive fields encrypted with AES-256-GCM at application layer
- Master key rotation support via `apiary encrypt rotate-master`
- AAD (additional authenticated data) binds encrypted values to specific rows

### API Architecture

The API is built with Huma/v2 (OpenAPI-native framework):
- **Public Routes**: Authentication, invitation acceptance, password reset
- **Protected Routes**: Guarded by JWT middleware
- **Role-Based Access**: Fine-grained permission checks via middleware

**Handler Groups**:
- `auth`: Login, signup, OAuth, passkey management
- `estate`: Infrastructure management (credentials, estates)
- `me`: User profile and settings
- `org`: Organization management
- `invitation`: Team invitations
- `admin`: Platform administration

See `docs/openapi.json` for complete API specification.

### Job Processing

River-based async job queue for:
- Email delivery (with encryption for sensitive content)
- Tenant database provisioning and lifecycle
- Session cleanup
- Organization domain verification
- Cohort analysis for admission control

Workers run separately via `apiary worker`.

### Authentication Flow

1. **Password Registration**:
   - User creates account with email + password
   - Email verification link sent (encrypted JWT token)
   - EmailVerifiedAt timestamp recorded on confirmation

2. **OAuth/Social Login**:
   - OIDC provider integration
   - Account linking
   - EmailVerified status from IdP

3. **Multi-Factor Authentication**:
   - TOTP (Time-based One-Time Password)
   - WebAuthn/Passkeys (passwordless)
   - Configurable per-organization via policies

### Tenant Provisioning

Organizations automatically get dedicated PostgreSQL databases:
1. Tenant database created and initialized with schema
2. Connection string encrypted with per-org key (derived from master key)
3. Tenant database migrations applied via Goose
4. Available for application use

## API Documentation

Full OpenAPI specification available at `/api/v1/docs` and `/api/v1/openapi.json`.

### Key Endpoints

**Authentication**
- `POST /api/v1/auth/register` - Create account
- `POST /api/v1/auth/login` - Sign in
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Sign out

**Organizations**
- `POST /api/v1/orgs` - Create organization
- `GET /api/v1/orgs` - List user's organizations
- `GET /api/v1/orgs/{id}` - Get organization details
- `PATCH /api/v1/orgs/{id}` - Update organization

**Teams & Members**
- `GET /api/v1/orgs/{id}/members` - List members
- `POST /api/v1/orgs/{id}/invitations` - Invite member
- `PATCH /api/v1/orgs/{id}/members/{user_id}` - Update member role

**Credentials & Estates**
- `POST /api/v1/orgs/{id}/credentials` - Create credential
- `GET /api/v1/orgs/{id}/credentials` - List credentials
- `POST /api/v1/orgs/{id}/estates` - Create estate
- `GET /api/v1/orgs/{id}/estates` - List estates

**Security**
- `POST /api/v1/me/passkeys` - Register passkey
- `POST /api/v1/me/totp/setup` - Enable TOTP
- `GET /api/v1/me/sessions` - Active sessions
- `DELETE /api/v1/me/sessions/{id}` - Revoke session

See OpenAPI docs for complete endpoint reference and request/response schemas.

## Deployment

### Building Releases

```bash
# Snapshot build (for testing)
goreleaser build --snapshot --clean

# Full release (requires git tag)
git tag v1.2.3
git push --tags
```

Releases are automated via GitHub Actions and produce:
- Linux and macOS binaries (amd64, arm64)
- Checksums and GPG signatures
- Published to GitHub Releases

### Environment Configuration

**Production Checklist**:
- [ ] Set `ENCRYPTION_MASTER_KEY` to a cryptographically random 32-byte value
- [ ] Use strong `JWT_SECRET` (minimum 32 characters)
- [ ] Configure external SMTP server (not Mailpit)
- [ ] Set `ENV=production`
- [ ] Enable HTTPS/TLS on all external connections
- [ ] Configure database backups
- [ ] Set up monitoring and alerting
- [ ] Review and adjust admission control limits
- [ ] Enable MFA policies for sensitive roles

**Process Architecture**:
- Run `apiary serve` in load-balanced cluster (stateless API)
- Run `apiary worker` in worker pool (processes async jobs)
- Both connect to same PostgreSQL and access encrypted tenant connections

### Database Backup & Recovery

Backup entire control DB:
```bash
pg_dump $DATABASE_URL > control_backup.sql
```

Backup individual tenant:
```bash
go run . migrate --scope tenants --org organization-id status
# Then backup that specific tenant database
```

## Contributing

### Code Style

- Go: Follow `go fmt` and `golangci-lint` rules
- TypeScript: Use `prettier` and `oxlint`
- SQL: Migrations only via Atlas/Goose

### Testing

All code changes must include tests:

```bash
# Run full test suite
just test

# View coverage
just cover

# Run specific test
go test ./internal/service -run TestTokenService
```

Test database must be configured (via `TEST_DATABASE_URL`).

### Pull Request Process

1. Create feature branch from `main`
2. Make changes with tests
3. Ensure `just quality` passes
4. Push branch and open PR
5. Merge requires DCO sign-off: `git commit -s`
6. Squash merge to `main` with `Signed-off-by` trailer
7. Tag release on merged commit (never before merge)

### Security Considerations

- **Encryption**: All sensitive data encrypted at application layer with AES-256-GCM
- **Secrets**: Never committed to repo; use environment variables
- **CORS**: Configured per environment; restrict in production
- **Rate Limiting**: HTTP rate limiting on public endpoints
- **Audit Logging**: All auth events and sensitive operations logged
- **Email Tokens**: Hashed and short-lived (15 minutes default)
- **Session Management**: Refresh token rotation, automatic cleanup of expired sessions

### Adding a New Provider

1. Define credential format in `internal/models/tenant/credential.go`
2. Implement provider client in new package (e.g., `internal/providers/hetzner`)
3. Add estate provisioning logic
4. Extend OpenAPI schema with provider-specific parameters
5. Add tests for credential validation and operations

## License

Proprietary - GlueOps, Inc.

---

**Questions?** Check `docs/openapi.json` for API details or run `apiary --help` for CLI options.

Repository settings are managed by github-org-manager.
