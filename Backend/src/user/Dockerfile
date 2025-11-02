FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY Backend/src/user/src/ ./src/

COPY proto/gen/ ./proto/gen/

ENV PYTHONPATH=/app:/app/proto/gen

RUN mkdir -p /data

EXPOSE 50055
ENV USER_GRPC_PORT=50055
ENV USER_DB_PATH=/data/user.db
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
ENV SERVICE_ENV=production

CMD ["python", "-m", "src.main"]