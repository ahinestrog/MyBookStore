import os
import logging
from typing import Optional


class Config:    
    def __init__(self):
        self.grpc_port: str = self._env("USER_GRPC_PORT", "50055")
        self.db_path: str = self._env("USER_DB_PATH", "/data/user.db")
        self.rabbit_url: str = self._env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
        self.service_env: str = self._env("SERVICE_ENV", "dev")
        
        # Log the loaded configuration
        logging.info(f"[user] config loaded: {self.__dict__}")
    
    def _env(self, key: str, default: str) -> str:
        value = os.getenv(key)
        return value if value is not None else default


def load_config() -> Config:
    return Config()