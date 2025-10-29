import json
import logging
from datetime import datetime
from typing import Any, Dict, Optional
import pika
from pika.exceptions import AMQPConnectionError, AMQPChannelError


class UserCreated:
    def __init__(self, user_id: int, name: str, email: str):
        self.user_id = user_id
        self.name = name
        self.email = email
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "user_id": self.user_id,
            "name": self.name,
            "email": self.email
        }


class UserUpdated:
    def __init__(self, user_id: int, name: str):
        self.user_id = user_id
        self.name = name
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "user_id": self.user_id,
            "name": self.name
        }


class EventPublisher:
    def __init__(self, rabbit_url: Optional[str] = None):
        self.rabbit_url = rabbit_url
        self.connection: Optional[pika.BlockingConnection] = None
        self.channel: Optional[pika.channel.Channel] = None
        
        if rabbit_url:
            try:
                self._connect()
                self._setup_exchange()
                logging.info("[user] RabbitMQ event publisher initialized")
            except Exception as e:
                logging.warning(f"[user] WARN: RabbitMQ not available ({e}). Continuing without events.")
                self.connection = None
                self.channel = None
    
    def _connect(self):
        if not self.rabbit_url:
            return
        
        try:
            parameters = pika.URLParameters(self.rabbit_url)
            self.connection = pika.BlockingConnection(parameters)
            self.channel = self.connection.channel()
        except AMQPConnectionError as e:
            logging.error(f"[user] Failed to connect to RabbitMQ: {e}")
            raise
    
    def _setup_exchange(self):
        if self.channel:
            try:
                self.channel.exchange_declare(
                    exchange='user.events',
                    exchange_type='topic',
                    durable=True
                )
            except AMQPChannelError as e:
                logging.error(f"[user] Failed to declare exchange: {e}")
                raise
    
    def publish(self, event_type: str, payload: Any) -> bool:
        if not self.channel:
            logging.debug(f"[user] No RabbitMQ connection, skipping event: {event_type}")
            return False
        
        try:
            if hasattr(payload, 'to_dict'):
                payload_dict = payload.to_dict()
            elif isinstance(payload, dict):
                payload_dict = payload
            else:
                payload_dict = payload.__dict__
            
            message = {
                "type": event_type,
                "timestamp": datetime.utcnow().isoformat() + "Z",
                "payload": payload_dict
            }
            
            self.channel.basic_publish(
                exchange='user.events',
                routing_key=event_type,
                body=json.dumps(message),
                properties=pika.BasicProperties(
                    content_type='application/json',
                    delivery_mode=2
                )
            )
            
            logging.info(f"[user] publish event {event_type}")
            return True
            
        except Exception as e:
            logging.error(f"[user] Failed to publish event {event_type}: {e}")
            return False
    
    def close(self):
        try:
            if self.channel and not self.channel.is_closed:
                self.channel.close()
            if self.connection and not self.connection.is_closed:
                self.connection.close()
            logging.info("[user] RabbitMQ connection closed")
        except Exception as e:
            logging.warning(f"[user] Error closing RabbitMQ connection: {e}")


def new_event_publisher(rabbit_url: str) -> EventPublisher:
    return EventPublisher(rabbit_url)