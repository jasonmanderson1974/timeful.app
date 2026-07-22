package db

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"schej.it/server/models"
)

func normalizeAllowlistEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IsAllowlisted returns true if the email is explicitly on the allowlist.
func IsAllowlisted(email string) bool {
	count, err := AllowlistCollection.CountDocuments(
		context.Background(),
		bson.M{"email": normalizeAllowlistEmail(email)},
	)
	return err == nil && count > 0
}

// IsAccessAllowed is the invite-only sign-in gate. It returns true if the
// allowlist is EMPTY (bootstrap "open" state, before the list has been seeded —
// prevents locking everyone out) OR the email is explicitly allowlisted.
func IsAccessAllowed(email string) bool {
	total, err := AllowlistCollection.CountDocuments(context.Background(), bson.M{})
	if err == nil && total == 0 {
		return true
	}
	return IsAllowlisted(email)
}

// AddToAllowlist adds an email to the allowlist with the given role (idempotent
// upsert). If the email is already listed, its role is left unchanged — use
// SetAllowlistRole to change an existing entry's role.
func AddToAllowlist(email string, addedBy string, role models.Role) error {
	e := normalizeAllowlistEmail(email)
	_, err := AllowlistCollection.UpdateOne(
		context.Background(),
		bson.M{"email": e},
		bson.M{"$setOnInsert": bson.M{
			"email":   e,
			"addedBy": addedBy,
			"addedAt": primitive.NewDateTimeFromTime(time.Now()),
			"role":    models.NormalizeRole(role),
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

// GetAllowlistRole returns the role recorded on the allowlist for the given
// email, or "" if the email is not listed. Used to seed a new user's role at
// first sign-in.
func GetAllowlistRole(email string) models.Role {
	var entry models.AllowlistEntry
	err := AllowlistCollection.FindOne(
		context.Background(),
		bson.M{"email": normalizeAllowlistEmail(email)},
	).Decode(&entry)
	if err != nil {
		return ""
	}
	return entry.Role
}

// SetAllowlistRole updates the role on an existing allowlist entry.
func SetAllowlistRole(email string, role models.Role) error {
	_, err := AllowlistCollection.UpdateOne(
		context.Background(),
		bson.M{"email": normalizeAllowlistEmail(email)},
		bson.M{"$set": bson.M{"role": models.NormalizeRole(role)}},
	)
	return err
}

// RemoveFromAllowlist removes an email from the allowlist.
func RemoveFromAllowlist(email string) error {
	_, err := AllowlistCollection.DeleteOne(
		context.Background(),
		bson.M{"email": normalizeAllowlistEmail(email)},
	)
	return err
}

// GetAllowlist returns all allowlist entries.
func GetAllowlist() []models.AllowlistEntry {
	entries := make([]models.AllowlistEntry, 0)
	cursor, err := AllowlistCollection.Find(context.Background(), bson.M{})
	if err != nil {
		return entries
	}
	cursor.All(context.Background(), &entries)
	return entries
}
