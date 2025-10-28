FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements from project root and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code from the correct path relative to build context
COPY Backend/src/user/src/ ./src/

# Copy protobuf generated files
COPY proto/gen/ ./proto/gen/

# Set PYTHONPATH to find the generated protobuf files
ENV PYTHONPATH=/app:/app/proto/gen

# Create data directory for SQLite database
RUN mkdir -p /data

# Expose gRPC port
EXPOSE 50055
ENV USER_GRPC_PORT=50055
ENV USER_DB_PATH=/data/user.db
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
ENV SERVICE_ENV=production

# Run the service
CMD ["python", "-m", "src.main"]