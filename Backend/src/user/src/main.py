import logging
import signal
import sys
import os
from concurrent import futures
import grpc
from grpc_reflection.v1alpha import reflection

import sys
import os

proto_path = os.path.join(os.path.dirname(__file__), '..', '..', '..', '..', 'proto', 'gen')
if not os.path.exists(proto_path):
    proto_path = '/app/proto/gen'
sys.path.insert(0, proto_path)

sys.path.insert(0, os.path.join(proto_path, 'user'))
sys.path.insert(0, os.path.join(proto_path, 'common'))

import user_pb2_grpc

from .config import load_config
from .repository import new_user_repository
from .events import new_event_publisher
from .server import new_user_service


def setup_logging():
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(sys.stdout)
        ]
    )


def signal_handler(signum, frame):
    logging.info("[user] Received shutdown signal, stopping server...")
    sys.exit(0)


def main():
    setup_logging()
    logging.info("[user] Starting User service...")
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    try:
        cfg = load_config()
        
        repo = new_user_repository(cfg.db_path)
        logging.info(f"[user] Database initialized: {cfg.db_path}")
        
        pub = new_event_publisher(cfg.rabbit_url)
        if pub.connection:
            logging.info("[user] RabbitMQ connection established")
        else:
            logging.warning("[user] RabbitMQ not available, continuing without events")
        
        svc = new_user_service(repo, pub)
        
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        
        user_pb2_grpc.add_UserServicer_to_server(svc, server)
        
        service_names = [
            'user.User',
        ]
        reflection.enable_server_reflection(service_names, server)
        
        listen_addr = f"[::]:{cfg.grpc_port}"
        server.add_insecure_port(listen_addr)
        server.start()
        
        logging.info(f"[user] gRPC listening on :{cfg.grpc_port}")
        
        try:
            server.wait_for_termination()
        except KeyboardInterrupt:
            logging.info("[user] Received interrupt signal")
        finally:
            pub.close()
            server.stop(grace=5)
            logging.info("[user] Server stopped")
    
    except Exception as e:
        logging.error(f"[user] Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()