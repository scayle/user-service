package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/google/uuid"
	"github.com/scayle/common-go"
	"github.com/scayle/user-service/mongotypes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrUserNotFound = errors.New("user not found")
var ErrCouldNotDecode = errors.New("could not decode document")

type MongoUser struct {
	Id           primitive.Binary `bson:"_id"`
	IsAdmin      bool             `bson:"is_admin"`
	Username     string           `bson:"username"`
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
	_, err = r.GetByName(ctx, "admin")
	if err != nil && errors.Is(err, ErrUserNotFound) {
		// ToDo: we have to make sure the admin gets never deleted...
		//       else it would be a security risk to re-create the admin with the fixed password.
		hash, err := hash("admin")
		if err != nil {
			log.Fatal(err)
		}
		_, err = r.Create(ctx, true, "admin", "admin@no-mail.com", hash)
		if err != nil {
			log.Fatalf("could not create first user %v", err)
		}
	} else if err != nil {
		log.Fatalf("could not load the admin %v", err)
	}

	return &r
}

func (r *mongoRepository) users() *mongo.Collection {
	return r.client.Database("user-service").Collection("users")
}

func (r *mongoRepository) Create(ctx context.Context, isAdmin bool, username string, email string, passwordHash string) (string, error) {
	users := r.users()

	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	_, err = users.InsertOne(ctx, MongoUser{
		Id:           mongotypes.MustFromUUID(id),
		IsAdmin:      isAdmin,
		Username:     username,
		PasswordHash: passwordHash,
		Email:        email,
	})
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (r *mongoRepository) Update(ctx context.Context, id string, isAdmin *bool, username *string, email *string, passwordHash *string) (user, error) {
	users := r.users()

	updateFields := bson.D{}
	if isAdmin != nil {
		updateFields.Map()["is_admin"] = *isAdmin
	}

	if username != nil {
		updateFields.Map()["username"] = *username
	}

	if email != nil {
		updateFields.Map()["email"] = *email
	}

	if passwordHash != nil {
		updateFields.Map()["password_hash"] = *passwordHash
	}

	_, err := users.UpdateOne(ctx, bson.D{{"id", id}}, updateFields)
	if err != nil {
		return user{}, err
	}

	return r.Get(ctx, id)
}

func (r *mongoRepository) decodeUser(res interface{ Decode(interface{}) error }) (user, error) {
	foundUser := new(MongoUser)
	err := res.Decode(foundUser)
	if err != nil {
		return user{}, fmt.Errorf("%w:\n", ErrCouldNotDecode)
	}

	id, err := mongotypes.ToUUID(foundUser.Id)
	if err != nil {
		return user{}, fmt.Errorf("could not convert to UUID: %v\n%w", foundUser.Id, err)
	}

	return user{
		id:           id.String(),
		isAdmin:      foundUser.IsAdmin,
		username:     foundUser.Username,
		email:        foundUser.Email,
		passwordHash: foundUser.PasswordHash,
	}, nil
}

func (r *mongoRepository) queryUser(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) (user, error) {
	users := r.users()

	res := users.FindOne(ctx, filter, opts...)
	if res.Err() != nil && errors.Is(mongo.ErrNoDocuments, res.Err()) {
		return user{}, fmt.Errorf("%w:\n", ErrUserNotFound)
	} else if res.Err() != nil {
		return user{}, fmt.Errorf("error while searching for user:\n%w", res.Err())
	}

	return r.decodeUser(res)
}

func (r *mongoRepository) queryUsers(ctx context.Context, filter interface{}, opts ...*options.FindOptions) ([]user, error) {
	users := r.users()

	res, err := users.Find(ctx, filter, opts...)
	if err != nil {
		return []user{}, fmt.Errorf("error while searching for users:\n%w", err)
	}

	foundUsers := make([]user, 0)

	for ; res.RemainingBatchLength() > 0; res.Next(ctx) {
		decodedUser, err := r.decodeUser(res)
		if err != nil {
			return []user{}, nil
		}

		foundUsers = append(foundUsers, decodedUser)
	}

	return foundUsers, nil
}

func (r *mongoRepository) Get(ctx context.Context, id string) (user, error) {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return user{}, fmt.Errorf("could not parse id %v:\n%w", id, err)
	}
	return r.queryUser(ctx, bson.D{{"_id", mongotypes.MustFromUUID(parsedUUID)}})
}

func (r *mongoRepository) GetByName(ctx context.Context, username string) (user, error) {
	return r.queryUser(ctx, bson.D{{"username", username}})
}

func (r *mongoRepository) GetAll(ctx context.Context) ([]user, error) {
	return r.queryUsers(ctx, bson.D{})
}

func (r *mongoRepository) Close() {
	if r.client != nil {
		err := r.client.Disconnect(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}
}
