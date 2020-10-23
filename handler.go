package main

import (
	"context"
	"fmt"

	pb "github.com/scayle/proto/go/user_service"

	"golang.org/x/crypto/bcrypt"
)

type repository interface {
	Create(ctx context.Context, isAdmin bool, username string, email string, password string) (string, error)
	Get(ctx context.Context, id string) (user, error)
	GetByName(ctx context.Context, username string) (user, error)
	Count(_ context.Context) int
}

type authenticator interface {
	NewToken(ctx context.Context, userId string, isAdmin bool) (string, error)
	Validate(ctx context.Context, tokenString string) (JwtClaims, error)
}

type handler struct {
	pb.UnimplementedUserServiceServer
	repo repository
	auth authenticator
}

func (h handler) Create(ctx context.Context, request *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if request.GetClaims() == nil {
		return nil, fmt.Errorf("no permissions to create a new user")
	}

	id, err := h.repo.Create(ctx, request.GetIsAdmin(), request.GetUsername(), request.GetEmail(), request.GetPassword())
	if err != nil {
		return nil, fmt.Errorf("could not create user %w", err)
	}

	return &pb.CreateUserResponse{
		Id: id,
	}, nil
}

func (h handler) Get(ctx context.Context, request *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := h.repo.Get(ctx, request.GetId())
	if err != nil {
		return nil, fmt.Errorf("could not get user %w", err)
	}
	return &pb.GetUserResponse{
		Id:       user.id,
		Email:    user.email,
		Username: user.username,
	}, nil
}

func (h handler) Auth(ctx context.Context, request *pb.AuthRequest) (*pb.AuthResponse, error) {
	user, err := h.repo.GetByName(ctx, request.GetUsername())
	if err != nil {
		return nil, fmt.Errorf("could not get name\n%w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.passwordHash), []byte(request.GetPassword()))
	if err != nil {
		return nil, fmt.Errorf("could not validate password\n%w", err)
	}

	token, err := h.auth.NewToken(ctx, user.id, user.isAdmin)
	if err != nil {
		return nil, fmt.Errorf("could not generate token\n%w", err)
	}

	return &pb.AuthResponse{
		Id:    user.id,
		Token: token,
	}, nil
}

func (h handler) ValidateToken(ctx context.Context, request *pb.ValidateTokenRequest) (*pb.TokenClaims, error) {
	claims, err := h.auth.Validate(ctx, request.GetToken())

	if err != nil {
		return nil, fmt.Errorf("invalid token\n%w", err)
	}

	return &pb.TokenClaims{
		IsAdmin: claims.IsAdmin,
		UserId:  claims.UserId,
		Expires: claims.Expires,
	}, nil
}
