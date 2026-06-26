"""HTTP client to the Watch Dog cloud API.

The edge agent talks to a handful of endpoints: post detections, post clips,
heartbeat, bootstrap config (get_camera / list_zones), and face matching
(identity/match). Calls are wrapped with tenacity retries for transient
network/5xx errors.
"""
from __future__ import annotations

from typing import Any, Optional

import httpx
from tenacity import retry, stop_after_attempt, wait_exponential


class WatchDogClient:
    """Synchronous HTTP client to the Watch Dog backend."""

    def __init__(self, base_url: str, token: str, timeout_s: float = 10.0):
        self._client = httpx.Client(
            base_url=base_url.rstrip("/"),
            headers={
                "Authorization": f"Bearer {token}",
                "User-Agent": "watchdog-edge/0.1.0",
            },
            timeout=timeout_s,
        )

    def close(self) -> None:
        self._client.close()

    # -- Vision endpoints -------------------------------------------------

    @retry(stop=stop_after_attempt(3), wait=wait_exponential(min=1, max=10))
    def post_detection(self, body: dict) -> dict:
        r = self._client.post("/api/v1/vision/detections", json=body)
        r.raise_for_status()
        return r.json()

    @retry(stop=stop_after_attempt(3), wait=wait_exponential(min=1, max=10))
    def post_clip(self, body: dict) -> dict:
        r = self._client.post("/api/v1/vision/clips", json=body)
        r.raise_for_status()
        return r.json()

    @retry(stop=stop_after_attempt(3), wait=wait_exponential(min=1, max=10))
    def heartbeat(self, camera_id: str, status: str) -> None:
        r = self._client.post(
            f"/api/v1/vision/cameras/{camera_id}/heartbeat",
            json={"status": status},
        )
        r.raise_for_status()

    @retry(stop=stop_after_attempt(3), wait=wait_exponential(min=1, max=10))
    def get_camera(self, camera_id: str) -> dict:
        r = self._client.get(f"/api/v1/vision/cameras/{camera_id}")
        r.raise_for_status()
        return r.json()

    @retry(stop=stop_after_attempt(3), wait=wait_exponential(min=1, max=10))
    def list_zones(self, camera_id: str) -> list[dict]:
        r = self._client.get(
            "/api/v1/vision/zones", params={"camera_id": camera_id}
        )
        r.raise_for_status()
        return r.json()

    # -- Identity endpoint ------------------------------------------------

    @retry(stop=stop_after_attempt(2), wait=wait_exponential(min=0.5, max=2))
    def match_face(
        self, embedding: list[float], threshold: float = 0.6
    ) -> Optional[dict[str, Any]]:
        """Send a 512-d embedding to the cloud for face matching.

        Returns the match payload or ``None`` if no match was found.
        """
        r = self._client.post(
            "/api/v1/identity/match",
            json={"embedding": embedding, "threshold": threshold},
        )
        r.raise_for_status()
        body = r.json()
        if not body.get("matched"):
            return None
        return body