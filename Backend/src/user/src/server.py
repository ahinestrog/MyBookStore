"""
gRPC server implementation for the User service.
Equivalent to server.go in the Go implementation.
"""
import logging
import bcrypt
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

import user_pb2, user_pb2_grpc
import common_pb2
from .repository import UserRepository
from .events import EventPublisher, UserCreated, UserUpdated
from .models import User


class UserService(user_pb2_grpc.UserServicer):
    """
    User service implementation.
    Equivalent to UserService in server.go
    """
    
    def __init__(self, repo: UserRepository, pub: EventPublisher):
        """
        Initialize the User service.
        
        Args:
            repo: User repository instance
            pub: Event publisher instance
        """
        self.repo = repo
        self.pub = pub
    
    def Register(self, request, context):
        """
        Register a new user.
        Equivalent to Register method in Go.
        """
        # Validate required fields
        if not request.email or not request.password or not request.name:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("name, email and password are required")
            return user_pb2.RegisterResponse()
        
        # Check if user already exists
        existing_user = self.repo.get_by_email(request.email)
        if existing_user:
            context.set_code(grpc.StatusCode.ALREADY_EXISTS)
            context.set_details("email already registered")
            return user_pb2.RegisterResponse()
        
        # Hash password
        try:
            password_hash = bcrypt.hashpw(
                request.password.encode('utf-8'), 
                bcrypt.gensalt()
            ).decode('utf-8')
        except Exception as e:
            logging.error(f"[user] Password hashing failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("password processing failed")
            return user_pb2.RegisterResponse()
        
        # Create user
        user = User(
            name=request.name,
            email=request.email,
            password_hash=password_hash
        )
        
        try:
            user_id = self.repo.create(user)
            
            # Publish event
            self.pub.publish("user.created", UserCreated(user_id, user.name, user.email))
            
            logging.info(f"[user] User registered successfully: {user_id}")
            return user_pb2.RegisterResponse(user_id=user_id)
            
        except Exception as e:
            logging.error(f"[user] User registration failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("user creation failed")
            return user_pb2.RegisterResponse()
    
    def Authenticate(self, request, context):
        """
        Authenticate a user.
        Equivalent to Authenticate method in Go.
        """
        # Get user by email
        user = self.repo.get_by_email(request.email)
        if not user:
            context.set_code(grpc.StatusCode.UNAUTHENTICATED)
            context.set_details("invalid credentials")
            return user_pb2.AuthenticateResponse()
        
        # Verify password
        try:
            password_valid = bcrypt.checkpw(
                request.password.encode('utf-8'),
                user.password_hash.encode('utf-8')
            )
            
            if not password_valid:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details("invalid credentials")
                return user_pb2.AuthenticateResponse()
            
            logging.info(f"[user] User authenticated successfully: {user.id}")
            return user_pb2.AuthenticateResponse(ok=True, user_id=user.id)
            
        except Exception as e:
            logging.error(f"[user] Authentication error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("authentication failed")
            return user_pb2.AuthenticateResponse()
    
    def GetProfile(self, request, context):
        """
        Get user profile.
        Equivalent to GetProfile method in Go.
        """
        user = self.repo.get_by_id(request.user_id)
        if not user:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("user not found")
            return user_pb2.UserProfile()
        
        logging.debug(f"[user] Profile retrieved for user: {user.id}")
        return user_pb2.UserProfile(
            user_id=user.id,
            name=user.name,
            email=user.email
        )
    
    def UpdateName(self, request, context):
        """
        Update user name.
        Equivalent to UpdateName method in Go.
        """
        if not request.user_id or not request.name:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("missing fields")
            return user_pb2.UserProfile()
        
        # Update the name
        success = self.repo.update_name(request.user_id, request.name)
        if not success:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("user not found")
            return user_pb2.UserProfile()
        
        # Publish event
        self.pub.publish("user.updated", UserUpdated(request.user_id, request.name))
        
        # Get updated user
        user = self.repo.get_by_id(request.user_id)
        if not user:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("failed to retrieve updated user")
            return user_pb2.UserProfile()
        
        logging.info(f"[user] Name updated for user: {user.id}")
        return user_pb2.UserProfile(
            user_id=user.id,
            name=user.name,
            email=user.email
        )


def new_user_service(repo: UserRepository, pub: EventPublisher) -> UserService:
    """
    Factory function to create a UserService instance.
    Equivalent to NewUserService in Go.
    
    Args:
        repo: User repository instance
        pub: Event publisher instance
        
    Returns:
        UserService instance
    """
    return UserService(repo, pub)