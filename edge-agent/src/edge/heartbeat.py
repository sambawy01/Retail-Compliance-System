"""Periodic camera heartbeat reporter.

Runs in a daemon thread; on each tick, POSTs a status=online heartbeat for
every camera so the cloud can mark them online and the dashboard reflects
real-time agent health.
"""
from __future__ import annotations

import threading

from .api_client import WatchDogClient
from .logging_setup import get_logger

log = get_logger(__name__)


class HeartbeatLoop:
    def __init__(
        self,
        client: WatchDogClient,
        camera_ids: list[str],
        interval_seconds: int,
    ):
        self._client = client
        self._ids = list(camera_ids)
        self._interval = interval_seconds
        self._stop = threading.Event()
        self._thread = threading.Thread(target=self._run, daemon=True)

    def start(self) -> None:
        self._thread.start()

    def stop(self) -> None:
        self._stop.set()
        self._thread.join(timeout=5)

    def _run(self) -> None:
        while not self._stop.is_set():
            for cid in self._ids:
                try:
                    self._client.heartbeat(cid, status="online")
                except Exception as e:  # noqa: BLE001 — log and continue
                    log.warning("heartbeat_failed", camera_id=cid, error=str(e))
            self._stop.wait(self._interval)