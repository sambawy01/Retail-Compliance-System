# Watch Dog — Retail Compliance System

AI-powered retail compliance monitoring with computer vision, face recognition,
staff performance tracking, and real-time event alerts.

## Architecture

- **Backend**: Go 1.25, chi router, pgx/v5, JWT RS256 auth, Postgres with Row-Level Security
- **Frontend**: React 18, Vite 5, TailwindCSS, i18n (EN/AR), RTL support
- **Edge Agent**: Python, YOLO detection, ArcFace recognition, RTSP ingestion, WebRTC publishing
- **Database**: Supabase Postgres (Paris) with pgvector for biometric embeddings
- **Deploy**: Railway (backend), Vercel (frontend)

## Quick Start

### Prerequisites
- Go 1.25+
- Node.js 20+
- PostgreSQL with pgvector extension

### Backend
```bash
# Set required environment variables
export DB_HOST=your-db-host
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your-password
export JWT_PRIVATE_KEY_B64=your-base64-private-key
export JWT_PUBLIC_KEY_B64=your-base64-public-key
export ENV=production
export ALLOWED_ORIGINS=https://your-frontend.vercel.app

# Run (migrations auto-apply on startup)
go run ./cmd/server
```

### Frontend
```bash
cd web
npm install
npm run dev  # dev server at localhost:5173
npm run build  # production build
npm run test  # run vitest
```

## API

Base URL: `/api/v1/`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (no auth) |
| `/auth/login` | POST | Login with email/password |
| `/auth/me` | GET | Current user info |
| `/auth/refresh` | POST | Refresh JWT token |
| `/auth/logout` | POST | Clear session |
| `/vision/cameras` | GET/POST | List/create cameras |
| `/vision/cameras/{id}` | GET/PATCH/DELETE | Camera CRUD |
| `/vision/zones` | GET/POST | Zone management |
| `/vision/detections` | GET/POST | Detection events |
| `/vision/clips` | GET/POST | Video clips |
| `/identity/persons` | GET/POST | Staff enrollment |
| `/identity/persons/{id}` | GET/DELETE | Person details/revoke |
| `/identity/persons/{id}/consent` | GET/POST | Consent records |
| `/identity/persons/{id}/templates` | POST | Biometric embeddings |
| `/identity/match` | POST | Face matching |
| `/identity/audit` | GET | Access audit log |
| `/staff/` | GET | Staff performance profiles |
| `/staff/{id}` | GET | Individual staff profile |
| `/staff/{id}/report` | GET | AI-generated performance report |
| `/staff/{id}/attendance` | GET/POST | Attendance records |
| `/staff/{id}/holidays` | GET/POST | Leave management |
| `/notifications/rules` | GET/POST/PATCH/DELETE | Notification rules |
| `/webrtc/offer` | POST | WebRTC signaling |
| `/webrtc/turn` | POST | TURN credentials |

## Database Migrations

Migrations are embedded in the binary and run automatically on startup.
Files are in `internal/migrations/versions/*.sql` and tracked in `schema_migrations` table.

To add a new migration:
1. Create `internal/migrations/versions/002_description.sql`
2. Use `IF NOT EXISTS` for idempotency
3. The binary auto-applies on next deploy

## Environment Variables

### Backend
| Variable | Required | Description |
|----------|----------|-------------|
| `DB_HOST` | Yes* | Database host (or use DATABASE_URL) |
| `DB_PORT` | No | Database port (default: 6543) |
| `DB_USER` | No | Database user (default: postgres) |
| `DB_PASSWORD` | Yes* | Database password |
| `DATABASE_URL` | Yes* | Full connection string (alternative to DB_*) |
| `JWT_PRIVATE_KEY_B64` | Yes | Base64-encoded RSA private key |
| `JWT_PUBLIC_KEY_B64` | Yes | Base64-encoded RSA public key |
| `ENV` | No | Environment (development/production) |
| `ALLOWED_ORIGINS` | No | Comma-separated CORS origins (default: *) |
| `PORT` | No | HTTP port (default: 8080) |
| `LOG_LEVEL` | No | Log level (default: info) |
| `TELEGRAM_BOT_TOKEN` | No | Telegram alerts bot token |
| `TELEGRAM_CHAT_ID` | No | Telegram alerts chat ID |

### Frontend
| Variable | Required | Description |
|----------|----------|-------------|
| `VITE_API_URL` | Yes | Backend API base URL |

## Testing

```bash
# Backend
go test -v -race ./...

# Frontend
cd web && npm run test
```

## License

Proprietary. All rights reserved.