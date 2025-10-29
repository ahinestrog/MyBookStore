import logging
import bcrypt
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

import user_pb2, user_pb2_grpc
import common_pb2
from .repository import UserRepository
from .events import EventPublisher, UserCreated, UserUpdated
from .models import User


class UserService(user_pb2_grpc.UserServicer):
    
    def __init__(self, repo: UserRepository, pub: EventPublisher):
        self.repo = repo
        self.pub = pub
    
    def Register(self, request, context):
        if not request.email or not request.password or not request.name:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("name, email and password are required")
            return user_pb2.RegisterResponse()
        
        existing_user = self.repo.get_by_email(request.email)
        if existing_user:
            context.set_code(grpc.StatusCode.ALREADY_EXISTS)
            context.set_details("email already registered")
            return user_pb2.RegisterResponse()
        
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
        
        # Crear usuario
        user = User(
            name=request.name,
            email=request.email,
            password_hash=password_hash
        )
        
        try:
            user_id = self.repo.create(user)
            
            self.pub.publish("user.created", UserCreated(user_id, user.name, user.email))
            
            logging.info(f"[user] User registered successfully: {user_id}")
            return user_pb2.RegisterResponse(user_id=user_id)
            
        except Exception as e:
            logging.error(f"[user] User registration failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("user creation failed")
            return user_pb2.RegisterResponse()
    
    def Authenticate(self, request, context):
        user = self.repo.get_by_email(request.email)
        if not user:
            context.set_code(grpc.StatusCode.UNAUTHENTICATED)
            context.set_details("invalid credentials")
            return user_pb2.AuthenticateResponse()
        
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
        if not request.user_id or not request.name:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("missing fields")
            return user_pb2.UserProfile()
        
        success = self.repo.update_name(request.user_id, request.name)
        if not success:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("user not found")
            return user_pb2.UserProfile()
        
        self.pub.publish("user.updated", UserUpdated(request.user_id, request.name))
        
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
    return UserService(repo, pub)