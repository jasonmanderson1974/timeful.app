package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// ChronicleAttendee is one person recorded as having attended a past gathering
// (captured from the gathering's RSVPs at completion time).
type ChronicleAttendee struct {
	Name       string     `json:"name" bson:"name,omitempty"`
	Status     RsvpStatus `json:"status" bson:"status,omitempty"` // going / maybe
	GuestCount int        `json:"guestCount" bson:"guestCount,omitempty"`
}

// ChronicleEntry is an auto-captured snapshot of a gathering that has taken
// place — the club's history ("The Chronicle", C10). Written by the gathering
// scheduler when a confirmed gathering's time passes: for a one-off event once,
// and for a recurring gathering once per occurrence (captured before the
// occurrence rolls forward and its RSVPs are cleared). Stored in its own
// collection so history survives even if the source event is later deleted.
type ChronicleEntry struct {
	Id      primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	EventId primitive.ObjectID `json:"eventId" bson:"eventId,omitempty"`
	ShortId *string            `json:"shortId" bson:"shortId,omitempty"` // to link back to the event

	Name        string  `json:"name" bson:"name,omitempty"`
	Description *string `json:"description" bson:"description,omitempty"`
	Location    *string `json:"location" bson:"location,omitempty"`

	StartDate primitive.DateTime `json:"startDate" bson:"startDate,omitempty"`
	EndDate   primitive.DateTime `json:"endDate" bson:"endDate,omitempty"`

	// Attendees recorded as going/maybe, plus the total headcount
	// (sum of 1 + guestCount over those attendees).
	Attendees []ChronicleAttendee `json:"attendees" bson:"attendees,omitempty"`
	HeadCount int                 `json:"headCount" bson:"headCount,omitempty"`

	CapturedAt primitive.DateTime `json:"capturedAt" bson:"capturedAt,omitempty"`
}
