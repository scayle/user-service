package main

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type user struct {
	id           string
	isAdmin      bool
	username     string
	email        string
	passwordHash string
}

type inMemoryRepository struct {
	users       map[string]*user
	usersByName map[string]*user
}

func (r *inMemoryRepository) init(ctx context.Context) {
	if r.users != nil {
		return
	}

	r.users = make(map[string]*user)
	r.usersByName = make(map[string]*user)

	// create one admin
	_, err := r.Create(ctx, true, "admin", "admin@no-mail.com", "admin")
	if err != nil {
		log.Fatalf("could not create first user", err)
	}
}

func (r *inMemoryRepository) Count(_ context.Context) int {
	return len(r.users)
}

func (r *inMemoryRepository) Create(ctx context.Context, isAdmin bool, username string, email string, password string) (string, error) {
	r.init(ctx)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	u := user{
		id:           uuid.New().String(),
		isAdmin:      isAdmin,
		username:     username,
		email:        email,
		passwordHash: string(hash),
	}

	r.users[u.id] = &u
	r.usersByName[u.username] = &u

	return u.id, nil
}

func (r *inMemoryRepository) Get(ctx context.Context, id string) (user, error) {
	r.init(ctx)

	if u, ok := r.users[id]; ok {
		return *u, nil
	}

	return user{}, errors.New("user not found")
}

func (r *inMemoryRepository) GetByName(ctx context.Context, username string) (user, error) {
	r.init(ctx)

	if u, ok := r.usersByName[username]; ok {
		return *u, nil
	}

	return user{}, errors.New("user not found")
}
