package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// AllowlistEntry is one approved email on the invite-only allowlist. A sign-in
// (Google or email-code) is only permitted if the email is present here (or the
// allowlist is empty, which is the bootstrap "open" state before it's seeded).
type AllowlistEntry struct {
	Id      primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Email   string             `json:"email" bson:"email"`
	AddedBy string             `json:"addedBy,omitempty" bson:"addedBy,omitempty"`
	AddedAt primitive.DateTime `json:"addedAt,omitempty" bson:"addedAt,omitempty"`
}
