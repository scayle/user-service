package main

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/scayle/proto-go/user_service"

	"golang.org/x/crypto/bcrypt"
)

type user struct {
	id           string
	isAdmin      bool
	username     string
	email        string
	passwordHash string
}

var (
	ErrNoPermission = errors.New("no permissions")
)

type repository interface {
	Create(ctx context.Context, isAdmin bool, username string, email string, password string) (string, error)
	Update(ctx context.Context, id string, isAdmin *bool, username *string, email *string, passwordHash *string) (user, error)
	Get(ctx context.Context, id string) (user, error)
	GetAll(ctx context.Context) ([]user, error)
	GetByName(ctx context.Context, username string) (user, error)
	Close()
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
		return nil, fmt.Errorf("creating a new user:\n%w", ErrNoPermission)
	}

	hash, err := hash(request.GetPassword())
	if err != nil {
		return nil, fmt.Errorf("could not hash the password %w", err)
	}

	id, err := h.repo.Create(ctx, request.GetIsAdmin(), request.GetUsername(), request.GetEmail(), hash)
	if err != nil {
		return nil, fmt.Errorf("could not create user %w", err)
	}

	return &pb.CreateUserResponse{
		Id: id,
	}, nil
}

func (h handler) Update(ctx context.Context, request *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	claims := request.GetClaims()
	if claims == nil {
		return nil, fmt.Errorf("update a user:\n%w", ErrNoPermission)
	}

	// Only admins or the user itself can update.
	if !claims.IsAdmin || claims.UserId != request.Id {
		return nil, fmt.Errorf("update a user:\n%w", ErrNoPermission)
	}

	// Only admins can change the IsAdmin flag.
	if !claims.IsAdmin && request.IsAdmin != nil {
		return nil, fmt.Errorf("update a user:\n%w", ErrNoPermission)
	}

	var passwordHash *string
	if request.GetPassword() != "" {
		var err error
		newHash := ""
		newHash, err = hash(request.GetPassword())
		if err != nil {
			return nil, fmt.Errorf("could not hash the password %w", err)
		}

		passwordHash = &newHash
	}

	user, err := h.repo.Update(ctx, request.GetId(), request.IsAdmin, request.Username, request.Email, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("could not create user %w", err)
	}

	return &pb.UpdateUserResponse{
		Id:       user.id,
		IsAdmin:  user.isAdmin,
		Username: user.username,
		Email:    user.email,
	}, nil
}

func (h handler) Get(ctx context.Context, request *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if request.GetClaims() == nil {
		return nil, fmt.Errorf("getting a user:\n%w", ErrNoPermission)
	}

	user, err := h.repo.Get(ctx, request.GetId())
	if err != nil {
		return nil, fmt.Errorf("could not get user %w", err)
	}
	return &pb.GetUserResponse{
		Id:       user.id,
		Email:    user.email,
		Username: user.username,
		IsAdmin:  user.isAdmin,
	}, nil
}

func (h handler) GetAll(ctx context.Context, request *pb.GetAllUserRequest) (*pb.GetAllUserResponse, error) {
	if request.GetClaims() == nil {
		return nil, fmt.Errorf("getting all users:\n%w", ErrNoPermission)
	}

	users, err := h.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get user %w", err)
	}

	res := &pb.GetAllUserResponse{}
	for _, user := range users {
		res.Users = append(res.Users, &pb.GetUserResponse{
			Id:       user.id,
			Email:    user.email,
			Username: user.username,
			IsAdmin:  user.isAdmin,
		})
	}
	return res, nil
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
