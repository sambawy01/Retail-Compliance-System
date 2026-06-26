# Watch Dog Edge Agent — Python 3.12-slim + OpenCV + insightface deps.
# Build:  docker build -t watchdog-edge:dev .
# Run:    docker run -v /path/to/config.yaml:/etc/watchdog/config.yaml watchdog-edge:dev
FROM python:3.12-slim

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg libgl1 libglib2.0-0 ca-certificates \
      libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy project metadata and install deps first so layers cache between builds.
COPY pyproject.toml ./
RUN pip install --upgrade pip && pip install -e .

COPY src/ ./src/
# Weights are baked in OR bind-mounted at runtime; see README.
RUN mkdir -p ./weights

ENV PYTHONPATH="/app/src"

RUN useradd -m -u 10001 app && \
    chown -R app:app /app
USER app

ENTRYPOINT ["python", "-m", "edge.agent"]
CMD ["/etc/watchdog/config.yaml"]