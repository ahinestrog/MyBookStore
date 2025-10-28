"""
Event publishing for the User service.
Combines events.go and rabbit.go functionality from the Go implementation.
"""
import json
import logging
from datetime import datetime
from typing import Any, Dict, Optional
import pika
from pika.exceptions import AMQPConnectionError, AMQPChannelError


class UserCreated:
    """Event payload for when a user is created."""
    
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
    """Event payload for when a user is updated."""
    
    def __init__(self, user_id: int, name: str):
        self.user_id = user_id
        self.name = name
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "user_id": self.user_id,
            "name": self.name
        }


class EventPublisher:
    """
    RabbitMQ event publisher.
    Equivalent to EventPublisher in rabbit.go
    """
    
    def __init__(self, rabbit_url: Optional[str] = None):
        """
        Initialize the event publisher.
        
        Args:
            rabbit_url: RabbitMQ connection URL
        """
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
        """Establish connection to RabbitMQ."""
        if not self.rabbit_url:
            return
        
        try:
            # Parse connection URL and create connection
            parameters = pika.URLParameters(self.rabbit_url)
            self.connection = pika.BlockingConnection(parameters)
            self.channel = self.connection.channel()
        except AMQPConnectionError as e:
            logging.error(f"[user] Failed to connect to RabbitMQ: {e}")
            raise
    
    def _setup_exchange(self):
        """Setup the user.events exchange."""
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
        """
        Publish an event to RabbitMQ.
        
        Args:
            event_type: Type of the event (routing key)
            payload: Event payload (should have to_dict() method or be dict)
            
        Returns:
            bool: True if published successfully, False otherwise
        """
        if not self.channel:
            logging.debug(f"[user] No RabbitMQ connection, skipping event: {event_type}")
            return False
        
        try:
            # Prepare the event message
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
            
            # Publish the message
            self.channel.basic_publish(
                exchange='user.events',
                routing_key=event_type,
                body=json.dumps(message),
                properties=pika.BasicProperties(
                    content_type='application/json',
                    delivery_mode=2  # Make message persistent
                )
            )
            
            logging.info(f"[user] publish event {event_type}")
            return True
            
        except Exception as e:
            logging.error(f"[user] Failed to publish event {event_type}: {e}")
            return False
    
    def close(self):
        """Close the RabbitMQ connection."""
        try:
            if self.channel and not self.channel.is_closed:
                self.channel.close()
            if self.connection and not self.connection.is_closed:
                self.connection.close()
            logging.info("[user] RabbitMQ connection closed")
        except Exception as e:
            logging.warning(f"[user] Error closing RabbitMQ connection: {e}")


def new_event_publisher(rabbit_url: str) -> EventPublisher:
    """
    Factory function to create an EventPublisher instance.
    Equivalent to NewEventPublisher in Go.
    
    Args:
        rabbit_url: RabbitMQ connection URL
        
    Returns:
        EventPublisher instance
    """
    return EventPublisher(rabbit_url)