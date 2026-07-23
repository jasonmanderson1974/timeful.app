package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Comment is a message in an event's discussion thread (C7). Stored in its own
// `comments` collection (keyed by EventId), mirroring EventResponse.
type Comment struct {
	Id      primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	EventId primitive.ObjectID `json:"eventId" bson:"eventId"`

	// UserId is the guest's name OR a signed-in user's id hex — it authorizes
	// edit / delete-own. IsGuest disambiguates the two.
	UserId  string `json:"userId" bson:"userId"`
	IsGuest bool   `json:"isGuest" bson:"isGuest"`

	// Denormalized author display name (guest name, or the account's "First Last").
	AuthorName string `json:"authorName" bson:"authorName"`

	Text      string              `json:"text" bson:"text"`
	CreatedAt primitive.DateTime  `json:"createdAt" bson:"createdAt"`
	UpdatedAt *primitive.DateTime `json:"updatedAt" bson:"updatedAt,omitempty"` // set when edited
}
