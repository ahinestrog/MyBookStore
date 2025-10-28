"""
Database repository for the User service.
Equivalent to repository.go in the Go implementation.
"""
import logging
import os
from datetime import datetime
from typing import Optional
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, Session
from sqlalchemy.exc import IntegrityError, NoResultFound
from .models import Base, User


class UserRepository:
    """
    User repository equivalent to the Go UserRepository struct.
    Handles all database operations for the User model.
    """
    
    def __init__(self, dsn: str):
        """
        Initialize the repository with database connection.
        
        Args:
            dsn: Database connection string (SQLite path)
        """
        # Ensure directory exists for SQLite database
        db_dir = os.path.dirname(dsn)
        if db_dir and not os.path.exists(db_dir):
            os.makedirs(db_dir, exist_ok=True)
        
        # Create engine with SQLite specific settings
        self.engine = create_engine(
            f"sqlite:///{dsn}",
            pool_pre_ping=True,
            connect_args={'check_same_thread': False}
        )
        
        # Create session factory
        self.SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=self.engine)
        
        # Create tables
        self._migrate()
    
    def _migrate(self):
        """
        Create database schema.
        Equivalent to the migrate() function in repository.go
        """
        try:
            Base.metadata.create_all(bind=self.engine)
            logging.info("[user] Database migration completed")
        except Exception as e:
            logging.error(f"[user] Database migration failed: {e}")
            raise
    
    def _get_db(self) -> Session:
        """Get database session."""
        return self.SessionLocal()
    
    def create(self, user: User) -> int:
        """
        Create a new user in the database.
        
        Args:
            user: User instance to create
            
        Returns:
            int: The ID of the created user
            
        Raises:
            IntegrityError: If email already exists
        """
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
        """
        Get user by ID.
        
        Args:
            user_id: User ID to search for
            
        Returns:
            User instance if found, None otherwise
        """
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
        """
        Get user by email.
        
        Args:
            email: Email to search for
            
        Returns:
            User instance if found, None otherwise
        """
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
        """
        Update user's name.
        
        Args:
            user_id: ID of the user to update
            name: New name for the user
            
        Returns:
            bool: True if update was successful, False otherwise
        """
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
    """
    Factory function to create a UserRepository instance.
    Equivalent to NewUserRepository in Go.
    
    Args:
        dsn: Database connection string
        
    Returns:
        UserRepository instance
    """
    return UserRepository(dsn)