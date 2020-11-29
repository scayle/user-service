// ToDo: extract into own repo for re-usability

package mongotypes

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrNoUUID = errors.New("no valid uuid")

// FromUUID converts a uuid to a mongodb binary field with a binary subtype of 4.
// In the current uuid.UUID implementation this will never return an error, but this may change.
// You can use MustFromUUID to avoid checking the error explicitly, but keep in mind that it may panic in the future.
func FromUUID(id uuid.UUID) (primitive.Binary, error) {
	i := primitive.Binary{}

	binary, err := id.MarshalBinary()
	if err != nil {
		return i, fmt.Errorf("error while converting uuid to mongotypes compatible uuid: %w", err)
	}

	i.Subtype = 4
	i.Data = binary
	return i, nil
}

// FromUUIDString converts a string representing an uuid to a mongodb binary field with a binary subtype of 4.
// It may return an error if the string is not a valid uuid.
func FromUUIDString(id string) (primitive.Binary, error) {
	parsedId, err := uuid.Parse(id)
	if err != nil {
		return primitive.Binary{}, fmt.Errorf("could not convert string to uuid:\n%w\n%v", ErrNoUUID, err)
	}
	return FromUUID(parsedId)
}

// MustFromUUID converts a uuid to a mongodb binary field with a binary subtype of 4.
// It panics if it cannot convert the uuid.
// In the current uuid.UUID implementation this will never fail, but this may change.
func MustFromUUID(id uuid.UUID) primitive.Binary {
	if i, err := FromUUID(id); err != nil {
		panic(err)
	} else {
		return i
	}
}

func ToUUID(id primitive.Binary) (uuid.UUID, error) {
	if id.Subtype != 4 || len(id.Data) != 16 {
		return [16]byte{}, fmt.Errorf("could not convert binary to uuid: %w", ErrNoUUID)
	}
	return uuid.FromBytes(id.Data)
}
