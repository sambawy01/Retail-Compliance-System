# Watch Dog — Retail Compliance System

Enterprise-grade CCTV compliance monitoring for retail stores. Live WebRTC
streaming, face ID with PDP Law compliance, 16 retail event types, bilingual
dashboard (EN/AR).

## Architecture

```
IP Cameras (Hikvision/Dahua)
  ↓ RTSP
Edge Agent (Python, Pi 5/Jetson)
  ↓ detections + WebRTC stream + heartbeat
Go Backend (Railway)
  ↓ REST API + WebSocket
React Dashboard (Vercel) — bilingual EN/AR
  ↓ alerts
Telegram Bot → management
```

## Stack

| Component | Tech |
|---|---|
| Backend | Go 1.25, chi router, pgx/v5 |
| Database | Postgres 16 + pgvector (Neon) |
| Cache | Redis 7 |
| Edge agent | Python (YOLO, ArcFace, aiortc) |
| Frontend | React + Vite + Tailwind |
| Clip storage | Backblaze B2 (S3-compatible) |
| Live streaming | WebRTC (aiortc ↔ browser) |
| Deploy | Railway (backend), Vercel (frontend) |

## Event types (16)

**Critical:** slip\_fall, cash\_drawer, after\_hours, buddy\_punch, blocked\_exit
**Warning:** uniform\_violation, hygiene\_violation, phone\_usage, cleanliness\_alert, checkout\_bottleneck, stockroom\_anomaly, loitering, camera\_degraded
**Info:** loyalty\_recognized, occupancy\_update, activity\_update

## Zones (8)

checkout, aisles, stockroom, back\_office, entrance, restroom, restricted, privacy\_mask

## Face ID

Full identity stack with Egypt PDP Law (No. 151/2020) compliance:
- Encrypted face embeddings (AES-256-GCM + pgvector)
- Consent management (bilingual EN/AR)
- Access audit trail
- Erasure log
- Per-tenant data encryption keys

## Development

```bash
docker compose up -d     # start Postgres + Redis
go run ./cmd/server      # start backend
```

## License

Proprietary. © 2026 Hany Sadek.