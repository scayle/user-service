package main

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"

	"github.com/google/uuid"
	"github.com/scayle/common-go"
	"github.com/scayle/user-service/mongotypes"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserNotFound = errors.New("user not found")

type MongoUser struct {
	Id           primitive.Binary `bson:"_id"`
	IsAdmin      bool             `bson:"is_admin"`
	Username     string           `bson:"is_username"`
	PasswordHash string           `bson:"password_hash"`
	Email        string           `bson:"email,omitempty"`
}

type mongoRepository struct {
	client *mongo.Client
}

func NewMongoRepository(ctx context.Context) *mongoRepository {
	r := mongoRepository{}

	mongodbEntry := common.GetRandomServiceWithConsul("mongodb")
	uri := "mongodb://" + net.JoinHostPort(mongodbEntry.Service.Address, strconv.Itoa(mongodbEntry.Service.Port))
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	r.client = client

	// create one admin if it doesn't exist yet
	//	_, err = r.GetByName(ctx, "admin")
	//	if err != nil && errors.Is(err, ErrUserNotFound) {
	// ToDo: we have to make sure the admin gets never deleted...
	//       else it would be a security risk to re-create the admin with the fixed password.
	_, err = r.Create(ctx, true, "admin", "admin@no-mail.com", "admin")
	if err != nil {
		log.Fatalf("could not create first user %v", err)
	}
	//	} else if err != nil {
	//		log.Fatalf("could not load the admin %v", err)
	//  }

	return &r
}

func (r *mongoRepository) users() *mongo.Collection {
	return r.client.Database("user-service").Collection("users")
}

func (r *mongoRepository) Create(ctx context.Context, isAdmin bool, username string, email string, password string) (string, error) {
	users := r.users()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	_, err = users.InsertOne(ctx, MongoUser{
		Id:           mongotypes.MustFromUUID(id),
		IsAdmin:      isAdmin,
		Username:     username,
		PasswordHash: string(hash),
		Email:        email,
	})
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (r *mongoRepository) Get(ctx context.Context, id string) (user, error) {
	panic("implement me")
}

func (r *mongoRepository) GetByName(ctx context.Context, username string) (user, error) {
	panic("implement me")
}

func (r *mongoRepository) Count(ctx context.Context) int {
	panic("implement me")
}

func (r *mongoRepository) Close() {
	if r.client != nil {
		r.client.Disconnect(context.Background())
	}
}
