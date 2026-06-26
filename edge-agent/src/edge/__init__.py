"""Watch Dog edge agent.

Runs on a retail store edge device (Pi 5 or Jetson). Pulls RTSP frames from
each registered IP camera, executes the open-source CV stack (YOLO + ArcFace),
and publishes retail compliance vision events to the Watch Dog cloud API.
"""

__version__ = "0.1.0"