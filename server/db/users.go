package db

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"schej.it/server/logger"
	"schej.it/server/models"
)

// Returns a user based on their _id
func GetUserById(userId string) *models.User {
	objectId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		// userId is malformatted
		return nil
	}
	result := UsersCollection.FindOne(context.Background(), bson.M{
		"_id": objectId,
	})
	if result.Err() == mongo.ErrNoDocuments {
		// User does not exist!
		return nil
	}

	// Decode result
	var user models.User
	if err := result.Decode(&user); err != nil {
		logger.StdErr.Panicln(err)
	}

	return &user
}

func GetUserByStripeCustomerId(stripeCustomerId string) (*models.User, error) {
	result := UsersCollection.FindOne(context.Background(), bson.M{
		"stripeCustomerId": stripeCustomerId,
	})

	if result.Err() == mongo.ErrNoDocuments {
		// User does not exist!
		return nil, nil
	}

	// Decode result
	var user models.User
	if err := result.Decode(&user); err != nil {
		logger.StdErr.Println(err)
		return nil, err
	}

	return &user, nil
}

// SetUserRole sets the role on the user with the given email (case-insensitive).
// Returns the number of users matched (0 if no account exists for that email yet).
func SetUserRole(email string, role models.Role) (int64, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return 0, nil
	}
	opts := options.Update().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // case-insensitive match on email
	})
	res, err := UsersCollection.UpdateOne(
		context.Background(),
		bson.M{"email": e},
		bson.M{"$set": bson.M{"role": models.NormalizeRole(role)}},
		opts,
	)
	if err != nil {
		return 0, err
	}
	return res.MatchedCount, nil
}

// GetUsersByEmails fetches users for the given emails in a single query and
// returns them keyed by lowercased email (case-insensitive match). Avoids N+1
// lookups when enriching a list of allowlist entries.
func GetUsersByEmails(emails []string) map[string]models.User {
	result := make(map[string]models.User)
	if len(emails) == 0 {
		return result
	}
	lowered := make([]string, 0, len(emails))
	for _, e := range emails {
		lowered = append(lowered, strings.ToLower(strings.TrimSpace(e)))
	}

	opts := options.Find().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // case-insensitive match on email
	})
	cursor, err := UsersCollection.Find(context.Background(), bson.M{
		"email": bson.M{"$in": lowered},
	}, opts)
	if err != nil {
		logger.StdErr.Println(err)
		return result
	}

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		logger.StdErr.Println(err)
		return result
	}
	for _, u := range users {
		result[strings.ToLower(strings.TrimSpace(u.Email))] = u
	}
	return result
}

func GetUserByEmail(email string) *models.User {
	emailQuery := strings.TrimSpace(email)
	if emailQuery == "" {
		return nil
	}
	opts := options.FindOne().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // case-insensitive match on email
	})
	result := UsersCollection.FindOne(context.Background(), bson.M{
		"email": emailQuery,
	}, opts)
	if result.Err() == mongo.ErrNoDocuments {
		// User does not exist!
		return nil
	}

	// Decode result
	var user models.User
	if err := result.Decode(&user); err != nil {
		logger.StdErr.Panicln(err)
	}

	return &user
}
