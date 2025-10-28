"""
Main entry point for the User service.
Equivalent to main.go in the Go implementation.
"""
import logging
import signal
import sys
import os
from concurrent import futures
import grpc
from grpc_reflection.v1alpha import reflection

# Import generated protobuf classes
import sys
import os

# Add proto/gen to Python path (works both in dev and container)
proto_path = os.path.join(os.path.dirname(__file__), '..', '..', '..', '..', 'proto', 'gen')
if not os.path.exists(proto_path):
    # In container, proto files are at /app/proto/gen
    proto_path = '/app/proto/gen'
sys.path.insert(0, proto_path)

# Direct imports to avoid module conflicts
sys.path.insert(0, os.path.join(proto_path, 'user'))
sys.path.insert(0, os.path.join(proto_path, 'common'))

import user_pb2_grpc

from .config import load_config
from .repository import new_user_repository
from .events import new_event_publisher
from .server import new_user_service


def setup_logging():
    """Setup logging configuration."""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(sys.stdout)
        ]
    )


def signal_handler(signum, frame):
    """Handle shutdown signals gracefully."""
    logging.info("[user] Received shutdown signal, stopping server...")
    sys.exit(0)


def main():
    """Main function to start the User service."""
    # Setup logging
    setup_logging()
    logging.info("[user] Starting User service...")
    
    # Setup signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    try:
        # Load configuration
        cfg = load_config()
        
        # Initialize repository
        repo = new_user_repository(cfg.db_path)
        logging.info(f"[user] Database initialized: {cfg.db_path}")
        
        # Initialize event publisher
        pub = new_event_publisher(cfg.rabbit_url)
        if pub.connection:
            logging.info("[user] RabbitMQ connection established")
        else:
            logging.warning("[user] RabbitMQ not available, continuing without events")
        
        # Initialize service
        svc = new_user_service(repo, pub)
        
        # Create gRPC server
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        
        # Register the service with gRPC server
        user_pb2_grpc.add_UserServicer_to_server(svc, server)
        
        # Enable reflection for debugging
        service_names = [
            'user.User',
        ]
        reflection.enable_server_reflection(service_names, server)
        
        # Start listening
        listen_addr = f"[::]:{cfg.grpc_port}"
        server.add_insecure_port(listen_addr)
        server.start()
        
        logging.info(f"[user] gRPC listening on :{cfg.grpc_port}")
        
        # Keep the server running
        try:
            server.wait_for_termination()
        except KeyboardInterrupt:
            logging.info("[user] Received interrupt signal")
        finally:
            # Cleanup
            pub.close()
            server.stop(grace=5)
            logging.info("[user] Server stopped")
    
    except Exception as e:
        logging.error(f"[user] Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()