import logging
import os
from datetime import datetime
from typing import Optional
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, Session
from sqlalchemy.exc import IntegrityError, NoResultFound
from .models import Base, User


class UserRepository:
    def __init__(self, dsn: str):
        db_dir = os.path.dirname(dsn)
        if db_dir and not os.path.exists(db_dir):
            os.makedirs(db_dir, exist_ok=True)
        
        self.engine = create_engine(
            f"sqlite:///{dsn}",
            pool_pre_ping=True,
            connect_args={'check_same_thread': False}
        )
        
        self.SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=self.engine)
        
        self._migrate()
    
    def _migrate(self):
        try:
            Base.metadata.create_all(bind=self.engine)
            logging.info("[user] Database migration completed")
        except Exception as e:
            logging.error(f"[user] Database migration failed: {e}")
            raise
    
    def _get_db(self) -> Session:
        return self.SessionLocal()
    
    def create(self, user: User) -> int:
        db = self._get_db()
        try:
            # Set timestamps
            now = datetime.utcnow()
            user.created_at = now
            user.updated_at = now
            
            db.add(user)
            db.commit()
            db.refresh(user)
            
            logging.info(f"[user] Created user with ID: {user.id}")
            return user.id
            
        except IntegrityError as e:
            db.rollback()
            logging.error(f"[user] Failed to create user: {e}")
            raise
        finally:
            db.close()
    
    def get_by_id(self, user_id: int) -> Optional[User]:
        db = self._get_db()
        try:
            user = db.query(User).filter(User.id == user_id).first()
            if user:
                logging.debug(f"[user] Found user by ID: {user_id}")
            else:
                logging.debug(f"[user] User not found by ID: {user_id}")
            return user
        finally:
            db.close()
    
    def get_by_email(self, email: str) -> Optional[User]:
        db = self._get_db()
        try:
            user = db.query(User).filter(User.email == email).first()
            if user:
                logging.debug(f"[user] Found user by email: {email}")
            else:
                logging.debug(f"[user] User not found by email: {email}")
            return user
        finally:
            db.close()
    
    def update_name(self, user_id: int, name: str) -> bool:
        db = self._get_db()
        try:
            user = db.query(User).filter(User.id == user_id).first()
            if not user:
                logging.warning(f"[user] User not found for update: {user_id}")
                return False
            
            user.name = name
            user.updated_at = datetime.utcnow()
            
            db.commit()
            logging.info(f"[user] Updated name for user ID: {user_id}")
            return True
            
        except Exception as e:
            db.rollback()
            logging.error(f"[user] Failed to update user name: {e}")
            return False
        finally:
            db.close()


def new_user_repository(dsn: str) -> UserRepository:
    return UserRepository(dsn)