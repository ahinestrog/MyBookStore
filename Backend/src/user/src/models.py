"""
Database models for the User service.
Equivalent to models.go in the Go implementation.
"""
from datetime import datetime
from typing import Optional
from sqlalchemy import Column, Integer, String, DateTime, create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker

Base = declarative_base()


class User(Base):
    """
    User model equivalent to the Go User struct.
    
    Go struct fields:
    - ID           int64     `db:"id"`
    - Name         string    `db:"name"`
    - Email        string    `db:"email"`
    - PasswordHash string    `db:"password_hash"`
    - CreatedAt    time.Time `db:"created_at"`
    - UpdatedAt    time.Time `db:"updated_at"`
    """
    __tablename__ = "users"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    name = Column(String, nullable=False)
    email = Column(String, nullable=False, unique=True)
    password_hash = Column(String, nullable=False)
    created_at = Column(DateTime, nullable=False, default=datetime.utcnow)
    updated_at = Column(DateTime, nullable=False, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    def __repr__(self):
        return f"<User(id={self.id}, name='{self.name}', email='{self.email}')>"
    
    def to_dict(self):
        """Convert User instance to dictionary."""
        return {
            'id': self.id,
            'name': self.name,
            'email': self.email,
            'password_hash': self.password_hash,
            'created_at': self.created_at,
            'updated_at': self.updated_at
        }