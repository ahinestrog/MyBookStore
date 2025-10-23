package main

import (
	"context"
	"errors"
	"log"

	"github.com/ahinestrog/mybookstore/proto/gen/common"
	"github.com/ahinestrog/mybookstore/proto/gen/user"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userpb.UnimplementedUserServer
	repo *UserRepository
	pub  *EventPublisher
}

func NewUserService(repo *UserRepository, pub *EventPublisher) *UserService {
	return &UserService{repo: repo, pub: pub}
}

func (s *UserService) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" || req.GetName() == "" {
		return nil, errors.New("name, email and password are required")
	}
	// verifica que exista
	if u, _ := s.repo.GetByEmail(ctx, req.GetEmail()); u != nil {
		return nil, errors.New("email already registered")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &User{
		Name:         req.GetName(),
		Email:        req.GetEmail(),
		PasswordHash: string(hash),
	}
	id, err := s.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}
	_ = s.pub.Publish("user.created", UserCreated{UserID: id, Name: u.Name, Email: u.Email})
	return &userpb.RegisterResponse{UserId: id}, nil
}

func (s *UserService) Authenticate(ctx context.Context, req *userpb.AuthenticateRequest) (*userpb.AuthenticateResponse, error) {
	u, err := s.repo.GetByEmail(ctx, req.GetEmail())
	if err != nil || u == nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.GetPassword())); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &userpb.AuthenticateResponse{
		Ok:     true,
		UserId: u.ID,
	}, nil
}

func (s *UserService) GetProfile(ctx context.Context, req *commonpb.UserRef) (*userpb.UserProfile, error) {
	u, err := s.repo.GetByID(ctx, req.GetUserId())
	if err != nil || u == nil {
		return nil, errors.New("user not found")
	}
	return &userpb.UserProfile{
		UserId: u.ID,
		Name:   u.Name,
		Email:  u.Email,
	}, nil
}

func (s *UserService) UpdateName(ctx context.Context, req *userpb.UpdateNameRequest) (*userpb.UserProfile, error) {
	if req.GetUserId() == 0 || req.GetName() == "" {
		return nil, errors.New("missing fields")
	}
	if err := s.repo.UpdateName(ctx, req.GetUserId(), req.GetName()); err != nil {
		return nil, err
	}
	_ = s.pub.Publish("user.updated", UserUpdated{UserID: req.GetUserId(), Name: req.GetName()})
	u, _ := s.repo.GetByID(ctx, req.GetUserId())
	return &userpb.UserProfile{
		UserId: u.ID,
		Name:   u.Name,
		Email:  u.Email,
	}, nil
}

func (s *UserService) must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
