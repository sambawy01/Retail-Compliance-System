"""Backblaze B2 clip uploader (S3-compatible API).

The edge encodes clip bytes from the rolling buffer (handled in the pipeline)
and hands them to upload_clip. The key layout is year/month/day/camera_id/
clip_id.mp4 so lifecycle rules and human spot-checks are tractable.

B2 is accessed via its S3-compatible API, so we use boto3 with a custom
endpoint URL pointing to the B2 S3-compatible endpoint for the bucket's region.
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import Optional


class B2Uploader:
    """Upload clips to Backblaze B2 via the S3-compatible API."""

    def __init__(
        self,
        bucket: str,
        region: str,
        application_key_id: Optional[str] = None,
        application_key: Optional[str] = None,
    ):
        import boto3  # lazy import — boto3 is only needed when uploading

        self._bucket = bucket
        endpoint = f"https://s3.{region}.backblazeb2.com"
        if application_key_id and application_key:
            self._client = boto3.client(
                "s3",
                region_name=region,
                endpoint_url=endpoint,
                aws_access_key_id=application_key_id,
                aws_secret_access_key=application_key,
            )
        else:
            # Fall back to environment / default credential chain
            self._client = boto3.client(
                "s3", region_name=region, endpoint_url=endpoint
            )

    def upload_clip(self, mp4_bytes: bytes, camera_id: str, clip_id: str) -> str:
        d = datetime.now(timezone.utc)
        key = (
            f"clips/{d.year:04d}/{d.month:02d}/{d.day:02d}/"
            f"{camera_id}/{clip_id}.mp4"
        )
        self._client.put_object(
            Bucket=self._bucket,
            Key=key,
            Body=mp4_bytes,
            ContentType="video/mp4",
            Metadata={"camera_id": camera_id, "clip_id": clip_id},
        )
        return key