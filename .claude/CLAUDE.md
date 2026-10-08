# Upstore Project - Claude Workspace Documentation

**Last Updated:** 2026-10-08  
**Project:** Upstore Backend Services  
**Email:** simankov.personal@gmail.com

---

## Product Overview

Upstore is an e-commerce platform for creating, running and growing your own online store, built for merchants in Belarus and Russia. It exists because sanctions have cut much of the region off from global platforms like Shopify, leaving local sellers without a modern, reliable way to sell online. Upstore fills that gap with an all-in-one admin: products and variants, orders and returns, customers and reviews, discounts, and a customisable storefront on your own domain.

It is designed around local reality from day one:
- **Currencies:** prices in BYN and RUB
- **Payments:** cards, ERIP, and cash on delivery
- **Delivery:** local couriers and pickup points
- **Languages:** the interface is in Russian and English

Merchants see their money like a wallet, with balance, upcoming payouts and every transaction in one place. Analytics show what sells, when, and where shoppers drop off. A guided launch checklist takes a new seller from sign-up to first sale in minutes.

**Visual direction:** calm and precise. Warm graphite-and-paper tones, soft rounded shapes, and colour used only when it means something. It draws on Shopify's practicality, shadcn's restraint, and Phantom's friendliness.

**Goal:** give independent sellers in the region a store platform as polished as the global ones, without depending on them.

---

## Technical Overview

Upstore is a microservices-based backend platform. The project uses:
- **Language:** Go
- **Architecture:** Microservices (accounts and profiles services, etc.)
- **Database:** PostgreSQL, with goose migrations and sqlc for typed queries
- **Message Queue:** RabbitMQ
- **Cache:** Redis
- **Git Flow:** Branch-based with master as main branch

---

## 🎯 Recent Work: Validation Library v2

### What Was Completed

#### 1. **Replaced v1 Validation Library**
- ❌ Removed: `shared/validate/string.go` (v1 - unsafe, called os.Exit(1))
- ✅ Replaced with: Safe v2 implementation  
- **Impact:** Config validation now returns errors instead of crashing the process
- **Status:** Ready for production use

#### 2. **Structure Reorganization**
```
shared/validate/
├── Implementation Files (6 files - ALL FULLY DOCUMENTED):
│   ├── errors.go          # 4 functions - error types & helpers
│   ├── schema.go          # 3 functions/types - base interface
│   ├── string.go          # 20 functions - string validators
│   ├── number.go          # 34 functions - int & float validators
│   ├── array.go           # 11 functions - array validators
│   └── object.go          # 2 functions - struct composition
│
└── Test Files (4 files - 42 tests):
    ├── string_test.go     # 11 test functions
    ├── number_test.go     # 13 test functions
    ├── array_test.go      # 8 test functions
    └── errors_test.go     # 11 test functions
```

#### 3. **Comprehensive Go Documentation**
- ✅ Every public function documented with:
  - Purpose and behavior  
  - Usage examples
  - Parameter descriptions
  - Return value documentation
- **Functions Documented:** 69+ functions across all files
- **Standard:** Follows Go conventions (godoc) - accessible via `go doc` command
- **Location:** Inline in source files (errors.go, schema.go, string.go, number.go, array.go, object.go)

#### 4. **Full Test Coverage**
- **Total Tests:** 42 test functions
- **All Passing:** ✅
- **Test Organization:** By implementation file (string_test.go, number_test.go, etc.)
- **Coverage:**
  - String validators: Required, Optional, Min, Max, Email, URL, Regex, OneOf, Refine
  - Number validators: Min, Max, Positive, Negative, OneOf, ParseString, Refine
  - Array validators: Size constraints, item validation, uniqueness, custom validation
  - Error handling: Error creation, collection, filtering, interface compliance

#### 5. **Single Source of Truth - Code Documentation**
- ✅ All documentation is in Go doc comments within source files
- ✅ Removed separate documentation files for maintainability
- ✅ Every function is documented with examples accessible via `go doc`

---

## 📦 Validation Library API

### Core Types (All Documented)

**Errors:**
- `ValidationError` - Single error
- `ValidationErrors` - Error collection
- `NewValidationError()` - Constructor

**Schemas:**
- `String()` - String validation
- `Int()` - Integer validation
- `Float()` - Float validation
- `Array[T]()` - Generic array validation
- `ValidateStruct()` - Compose field validators

### String Validators (20 Functions)
- `.Required()` / `.Optional()` - Field optionality
- `.Min()` / `.Max()` / `.Length()` - Length constraints
- `.Email()` - RFC 5322 email validation
- `.URL()` - URL validation with scheme & host
- `.PostgresURL()` - PostgreSQL connection validation
- `.RedisURL()` - Redis connection validation  
- `.RabbitMQURL()` - RabbitMQ/AMQP connection validation
- `.OneOf()` - Enum-like validation
- `.Regex()` - Pattern matching
- `.Refine()` - Custom validation functions
- `.Validate()` / `.ValidateAll()` - Execute validation

### Number Validators (34 Functions)

**Int (17 functions):**
- Range: `.Min()`, `.Max()`, `.Positive()`, `.Negative()`
- Enum: `.OneOf()`
- Custom: `.Refine()`
- Conversion: `.ParseString()`
- Execution: `.Validate()`, `.ValidateAll()`

**Float (17 functions):**
- Same as Int with floating-point semantics

### Array Validators (11 Functions)
- Size: `.Min()`, `.Max()`, `.Length()`
- Items: `.Items()` for per-element validation
- Uniqueness: `.Unique()`
- Custom: `.Refine()`
- Execution: `.Validate()`, `.ValidateAll()`

### Struct Validators (2 Functions)
- `ValidateStruct()` - Compose multiple field validators
- `MustValidate()` - Panic on error (for initialization)

---

## 🔧 Current Services Using Validation

### accounts service
**Config File:** `services/accounts/internal/config/config.go`
**Currently Using:** v1 validation (needs migration to v2)
**Fields:**
- HTTPPort (int, 1-65535)
- GRPCPort (int, 1-65535)
- LogLevel (enum: debug/info/warn/error)
- RedisURL (Redis connection)
- Environment (enum: dev/staging/prod)
- RabbitMQURL (AMQP connection)
- DatabaseURL (PostgreSQL connection)

**Migration Status:** 🔄 Pending - Next step when refactoring config

---

## 🚀 Using the Validation Library

### Basic Usage Pattern

```go
import "github.com/nikita-simankov/upstore/shared/validate"

// Single field validation
if err := validate.String("email").Email().Required().Validate(email); err != nil {
    log.Fatal(err)
}

// Structured config validation
errors := validate.ValidateStruct(
    func() validate.ValidationErrors {
        return validate.Int("port").Min(1).Max(65535).ValidateAll(cfg.Port)
    },
    func() validate.ValidationErrors {
        return validate.String("db_url").PostgresURL().Required().ValidateAll(cfg.DBUrl)
    },
)
if errors.HasErrors() {
    for _, e := range errors {
        log.Printf("%s: %s", e.Field, e.Message)
    }
}
```

### Documentation via Go Doc

For detailed function signatures and examples:
```bash
go doc validate.String
go doc validate.Int
go doc validate.Array
```

### Key Differences from v1

| Feature | v1 | v2 |
|---------|----|----|
| Error Handling | os.Exit(1) ❌ | Returns error ✅ |
| Port Type | String | Int ✅ |
| Testing | Crashes ❌ | Full coverage ✅ |
| Multiple Errors | Single only ❌ | All collected ✅ |
| Documentation | None | Comprehensive ✅ |

---

## 📋 Documentation

### Code Documentation (Go Doc Comments)

All 69+ public functions include comprehensive Go doc comments:
- **Location:** Inline in source files
- **Access:** Use `go doc github.com/nikita-simankov/upstore/shared/validate` or read source files
- **Coverage:** Every function has examples, parameter descriptions, and return value documentation
- **Format:** Follows Go conventions for IDE tooltips and documentation generation

### Quick Reference via Go Doc

```bash
# View function documentation
go doc validate.String
go doc validate.Int
go doc validate.Array

# View all functions
go doc -all validate
```

---

## 🧪 Running Tests

```bash
# Run all validation tests
go test ./shared/validate -v

# Run specific test function
go test -run TestStringEmail ./shared/validate -v

# Run with coverage
go test ./shared/validate -cover
```

**Current Status:** ✅ All 42 tests passing

---

## 📊 Project Statistics

| Metric | Count |
|--------|-------|
| Implementation Files | 6 |
| Test Files | 4 |
| Public Functions Documented | 69+ |
| Test Functions (All Passing) | 42 ✅ |
| Total Code Lines | ~1,500+ |
| Validators: String | 20 functions |
| Validators: Int/Float | 34 functions |
| Validators: Array | 11 functions |
| Error Handling | 4 functions |

---

## 📝 Project Standards

### Code Documentation
- Every public function has Go doc comments
- Doc comments include:
  - Brief description (first line)
  - Detailed explanation
  - Usage example(s)
  - Parameter descriptions
  - Return value documentation

### Testing
- Test files organized by implementation file (*_test.go)
- Test functions prefixed with Test
- Table-driven tests for multiple cases
- Edge case coverage
- Error message validation

### Validation
- Type-safe schemas (generics for Array[T])
- No side effects in library code
- Composable validators (chain operations)
- Structured error types (field + code + message)
- UTF-8 aware string operations

---

## 🏗️ Current Work: Accounts and Profiles Services

The `users` service was split into two services, and `users` was renamed to `accounts`.

### Service boundaries
- **accounts** (`services/accounts`): credentials, sessions, security. It owns the `accounts`, `auth_identities`, `sessions`, `email_verifications`, and `outbox_events` tables. It publishes `account.created` through the outbox.
- **profiles** (`services/profiles`): account type (seller or shopper) and profile fields. It consumes `account.created` from `profiles.account-created` and creates the profile idempotently.
- **shared/events**: the exchange name (`upstore.events`), the routing keys, the account types, and the event payloads. Both services import it.
- **gateway** (`services/gateway`): the public entry point on host port 9090. It verifies access tokens with the public key only, forwards `/v1/auth/*` to accounts without a token, and forwards protected routes (registered with `Protect`) only when the token is valid. It strips any client-sent `X-Account-ID` and sets its own from the verified token.
- **shared/authtoken**: token verification (EdDSA, issuer, expiry) and public key parsing. Accounts signs with it, and the gateway verifies with it.

### Status
| Step | Area | State |
|---|---|---|
| 1 | Rename users to accounts | done |
| 2 | Accounts schema (status, bans, auth identities) | done |
| 3 | sqlc queries | done |
| 4 | Outbox, relay, RabbitMQ publisher | done |
| 5 | Profiles service and account.created handler | done |
| 6 | Wiring: relay and consumer in `main`, per-service goose tables | done |
| 7 | Password hashing (Argon2id) and login logic | done |
| 8 | Security fixes from audit (enumeration, rate limits, tampered hashes, logging) | done |
| 9 | Register and login HTTP endpoints with rate limits | done |
| 10 | Sessions: Ed25519 access tokens, refresh rotation, logout | done |
| 11 | Email verification: single-use links, resend, Resend mailer | done |
| 12 | Gateway: token verification, auth passthrough, protected routes, host port 9090 | done (staged, not committed) |

### Accounts HTTP API (`/v1/auth/`)
- `POST register`: 201 with a pending account, sends a verification link. 409 if the email is taken.
- `POST login`: 200 with tokens. 401 for any wrong credential. 403 for a banned or pending account.
- `POST refresh`: rotates the refresh token. Reusing a rotated token revokes every session of the account.
- `POST logout`: 204, whether or not the token was valid.
- `POST verify-email`: consumes a link and activates the account. 400 for an invalid or expired link.
- `POST resend-verification`: always 202, so it does not reveal whether an email is registered.
- `GET /health`: database ping.

### Security decisions
- Access tokens are EdDSA JWTs, 15 minutes. Verification accepts only EdDSA and checks the issuer.
- Refresh and verification tokens are random and stored only as SHA-256 hashes.
- Locked, wrong-password, and unknown-email logins return the same error and do the same hashing work.
- Passwords are Argon2id with 64 MiB per hash. Concurrent hashing is capped at 8. Stored hashes with other parameters are refused.
- Rate limits are per client IP and per email. They use the TCP peer address only. Behind a proxy, the trusted proxy address must be configured first.
- Logs contain account IDs only, never emails, passwords, or tokens.
- Verification mail goes through Resend (`https://resend.com`) outside `development`. Startup fails without `RESEND_API_KEY` and `MAIL_FROM`.
- In `development` only, verification links are printed to the log by `DevLogMailer`. A link is a working credential, so never run development mode in a shared environment.
- `ACCESS_SIGNING_KEY` has no default. The compose key is dev-only.

### Environment (accounts)
`DATABASE_URL`, `RABBITMQ_URL`, `HTTP_PORT`, `GRPC_PORT`, `ENVIRONMENT`, `ACCESS_SIGNING_KEY` (base64 Ed25519 seed, required), `PUBLIC_APP_URL` (verification links point at `/verify-email`), `RESEND_API_KEY` and `MAIL_FROM` (required outside `development`; the key is a secret and is never logged).

### Environment (gateway)
`HTTP_PORT` (default 9090), `ACCESS_PUBLIC_KEY` (base64 Ed25519 public key, required; the gateway never holds the signing key), `ACCOUNTS_URL` (default `http://accounts:9090`).

### Tests
- Unit: `go test ./...` in `services/accounts` and `services/profiles`. Integration tests skip without their variables.
- Database: `ACCOUNTS_TEST_DATABASE_URL` and `PROFILES_TEST_DATABASE_URL`. Apply `migrations/00001_*.sql` first. Run with `-p 1`, because packages share the database.
- Broker: `ACCOUNTS_TEST_AMQP_URL` and `PROFILES_TEST_AMQP_URL`.

### Next steps
1. **Protected routes:** no upstream sits behind `Protect` yet. Register `/v1/profiles/` once profiles has an HTTP API. The accounts service is still published on host port 9091, so direct access bypasses the gateway. That is acceptable for auth routes, but protected services must never be published to the host.
2. **OAuth sign-in:** Google, Apple, and Telegram callbacks that create or link `auth_identities`.
3. **Resend setup:** verify the sending domain in Resend, then set `RESEND_API_KEY` and `MAIL_FROM` in each shared environment. The Resend mailer is tested against a local server only, so the first real send is still unverified.
4. **Profiles API:** read and update profiles.
5. **Consumer hardening:** a retry limit and a dead-letter queue.
6. **Full compose run:** `docker compose up` for the whole stack, which has not been run.
7. **Open decisions:** whether `POST register` should stop revealing taken emails (409), and whether `ENVIRONMENT` uses `development/production/staging` or `dev/staging/prod`.

### Security audit
Run `/security-code-audit` before each release of the auth code. Last audit: accounts auth package, with the findings fixed above. The open findings are those in the HTTP layer, now addressed, and the register enumeration decision.

---

## 🧾 Commit Conventions

- Every commit message is a single line.
- Prefix the message with exactly one of four categories, followed by a colon and a space:
  - `fix:` bug fixes
  - `chore:` maintenance, tooling, config, dependencies
  - `feature:` new functionality
  - `misc:` anything that fits none of the above
- Describe the change in past tense, matching the existing history, e.g.:
  `feature: implemented validate package for data validation`
- Do not add `Co-Authored-By` or other attribution trailers to commits.

---

## 🛠️ Troubleshooting

### Tests Failing
- Ensure all implementation files are in `shared/validate/`
- Check that test files reference the validate package correctly
- Run `go clean -testcache && go test ./shared/validate -v`

### Import Errors
- Path should be: `github.com/nikita-simankov/upstore/shared/validate`
- Previous path `github.com/nikita-simankov/upstore/shared/validate/v2` is no longer valid

### Configuration Not Validating
- Use `.ValidateAll()` to collect all errors
- Use `.Validate()` for fail-fast (first error only)
- Always check `errs.HasErrors()` before proceeding

### Viewing Documentation
- Use `go doc validate.<FunctionName>` to view doc comments
- Open source files in IDE for inline documentation with examples
- Each function has doc comments with usage examples

---

## 📞 Contact & Attribution

**Project Author:** Никита Simankov (simankov.personal@gmail.com)  
**Claude Contributor:** Claude Haiku 4.5  
**Last Updated:** 2026-10-08

---

## Changelog

### 2026-10-08 - Phase 2: Cleanup & Finalization
- ✅ Removed separate documentation files (examples.go, integration_example.go, INTEGRATION_GUIDE.md, DOCUMENTATION.md)
- ✅ Kept only code and Go doc comments (single source of truth)
- ✅ Updated CLAUDE.md to reflect new structure
- ✅ Streamlined project to essential files only

### 2026-10-08 - Phase 1: Initial Implementation
- ✅ Created v2 validation library from scratch
- ✅ Removed v1 unsafe code (os.Exit calls)
- ✅ Added comprehensive Go documentation (69+ functions)
- ✅ Organized implementation into logical modules:
  - errors.go (4 functions)
  - schema.go (3 functions/types)
  - string.go (20 functions)
  - number.go (34 functions)
  - array.go (11 functions)
  - object.go (2 functions)
- ✅ Created 42 comprehensive test functions (all passing)
- ✅ Organized tests by implementation file (*_test.go)
- ✅ Reorganized directory structure:
  - Implementation files → shared/validate/
  - Tests → shared/validate/*_test.go
- ✅ Created CLAUDE.md workspace file
