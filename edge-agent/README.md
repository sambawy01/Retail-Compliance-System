# Watch Dog Edge Agent

The edge agent runs on a retail store's local edge device (Raspberry Pi 5 or
NVIDIA Jetson). It pulls RTSP frames from each registered IP camera, runs the
open-source CV stack (YOLO for person/object detection + ArcFace for face ID),
and publishes retail compliance vision events to the Watch Dog cloud API.

## Hardware requirements

- Raspberry Pi 5 (8 GB) or NVIDIA Jetson Orin Nano
- 4 GB RAM minimum (8 GB recommended)
- 64 GB SSD/SD (rolling clip buffer + model weights)
- Gigabit Ethernet
- Connected IP cameras with RTSP streams

## Retail compliance checks

The agent performs these checks per camera, controlled by feature flags:

| Check | Flag | Description |
|-------|------|-------------|
| phone_usage | `phone_usage` | Employee using phone in checkout/aisles |
| cleanliness_alert | `cleanliness` | Spill/mess detected (placeholder) |
| checkout_bottleneck | `checkout_bottleneck` | Too many people at checkout |
| loitering | `loitering` | Person stationary in entrance zone |
| after_hours | `after_hours` | Person detected outside business hours |
| buddy_punch | `buddy_punch` | Face match fails during clock-in |
| slip_fall | `slip_fall` | Person on ground (pose/aspect-ratio based) |
| cash_drawer | `cash_drawer` | Person at register with no transaction (placeholder) |
| loyalty_recognized | `loyalty` | Face match succeeds |
| occupancy_update | `occupancy` | Person count per zone |

## Setup

1. Copy `config.example.yaml` to `config.yaml` and fill in:
   - `api_url` — the Watch Dog API base, e.g.
     `https://watchdog-api.example.com`
   - `api_token` — service token issued by your tenant admin
   - `agent_id` — a short unique ID for this box, e.g. `agent-easymart-01`
   - `b2_bucket` / `b2_region` — Backblaze B2 bucket for clip storage
   - `cameras[]` — one entry per camera (camera_id from Watch Dog, RTSP URL)
2. Drop model weights into `weights/` (YOLO weights, optional custom object
   model). Person detector defaults to `yolov8n.pt` (auto-downloaded by
   ultralytics on first run).
3. `docker build -t watchdog-edge:dev .`
4. `docker run -v ./config.yaml:/etc/watchdog/config.yaml watchdog-edge:dev`
5. Verify: check `docker logs -f` and look for `agent_start` JSON log.
   Check the Watch Dog UI for `status: online` on each camera.

## Config schema

```yaml
api_url: https://watchdog-api.example.com
api_token: SERVICE_TOKEN_HERE
agent_id: agent-easymart-01
b2_bucket: watchdog-vision
b2_region: us-west-002
heartbeat_seconds: 30
clip_buffer_seconds: 60
clip_extract_seconds: 30
cameras:
  - camera_id: 11111111-1111-1111-1111-111111111111
    rtsp_url: rtsp://10.0.0.10/stream1
    feature_flags:
      phone_usage: true
      checkout_bottleneck: true
      loitering: true
      after_hours: true
      occupancy: true
# WebRTC fields (optional):
# signaling_ws_url: wss://watchdog-api.example.com
# turn_url: turn:turn.watchdog.example:3478
# turn_user: watchdog
# turn_pass: ...
```

## Lightweight import (no ML deps)

The agent is designed to be importable without ML libraries installed, so
config validation and unit testing work in CI:

```bash
pip install pydantic httpx pyyaml tenacity numpy
PYTHONPATH=src python -c "import edge; print('OK')"
```

Heavy ML imports (`ultralytics`, `insightface`, `aiortc`) are lazy — they
only load inside `__init__` or methods, never at module import time.

## Tests

The full ML stack is heavy and assumes a GPU. CI runs a lighter subset:

```bash
pip install -e ".[ci]"
PYTHONPATH=src pytest -m "not gpu and not rtsp"
```

This exercises config loader, API client, zone geometry, clip buffer, event
builders, heartbeat loop, and the pipeline orchestrator with stub detectors.

GPU-bound tests (real YOLO/insightface) and RTSP-source tests are marked with
`@pytest.mark.gpu` / `@pytest.mark.rtsp` and skipped on CI.

## Source map

```
src/edge/
  agent.py             entrypoint
  config.py            pydantic + YAML config loader
  api_client.py        HTTP to Watch Dog cloud (httpx + tenacity)
  rtsp.py              OpenCV frame source w/ auto-reconnect
  pipeline.py          per-camera retail compliance orchestrator
  clip_buffer.py       rolling N-second frame buffer
  heartbeat.py         periodic camera health reporter
  b2_uploader.py       Backblaze B2 clip uploader (S3-compatible API)
  signaling_client.py  WebSocket signaling client for WebRTC
  webrtc_publisher.py  WebRTC stream publisher via aiortc
  logging_setup.py     structured JSON logging
  detectors/
    __init__.py        Detection + Detector protocol
    person_yolo.py     YOLOv8 person detector
    objects_yolo.py    YOLO retail object detector (phone, cup, bag, etc.)
    face_arcface.py    ArcFace 512-d face embedding extractor
    zone_geometry.py   normalized polygon hit-testing
  events/
    __init__.py        16 retail event subject constants (mirrors Go events.go)
    compliance.py      phone_usage, uniform_violation, hygiene_violation,
                       cleanliness_alert, blocked_exit
    safety.py          slip_fall
    security.py        cash_drawer, after_hours, loitering, buddy_punch
    operations.py      checkout_bottleneck, stockroom_anomaly, camera_degraded
    customer.py        loyalty_recognized, occupancy_update, activity_update
```