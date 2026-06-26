"""Typed config loader for the Watch Dog edge agent."""
from __future__ import annotations

from pathlib import Path
from typing import Dict, Optional

import yaml
from pydantic import BaseModel, Field, field_validator


class CameraAssignment(BaseModel):
    camera_id: str
    rtsp_url: str
    feature_flags: Dict[str, bool] = Field(default_factory=dict)


class EdgeConfig(BaseModel):
    api_url: str
    api_token: str
    agent_id: str
    b2_bucket: str
    b2_region: str
    cameras: list[CameraAssignment] = Field(default_factory=list)
    heartbeat_seconds: int = 30
    clip_buffer_seconds: int = 60   # rolling window length
    clip_extract_seconds: int = 30  # length of extracted event clip
    # WebRTC signaling — left blank to disable WebRTC.
    signaling_ws_url: str = ""
    turn_url: Optional[str] = None
    turn_user: Optional[str] = None
    turn_pass: Optional[str] = None

    @field_validator("api_token")
    @classmethod
    def _token_nonempty(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("api_token must be set")
        return v

    @classmethod
    def load(cls, path: str | Path) -> "EdgeConfig":
        data = yaml.safe_load(Path(path).read_text())
        return cls.model_validate(data)