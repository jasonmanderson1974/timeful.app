package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EventType string

const (
	SPECIFIC_DATES EventType = "specific_dates"
	DOW            EventType = "dow"
	GROUP          EventType = "group"
)

// Object containing information associated with the remindee
type Remindee struct {
	Email     string   `json:"email" bson:"email,omitempty"`
	TaskIds   []string `json:"-" bson:"taskIds,omitempty"` // Task IDs of the scheduled emails
	Responded *bool    `json:"responded" bson:"responded,omitempty"`
}

// Configuration + bookkeeping for the pre-gathering reminder email that fires
// once, LeadTimeHours before a confirmed gathering's start (see ScheduledEvent).
// The in-process reminder scheduler (services/reminders) reads these fields.
type GatheringReminder struct {
	Enabled       bool                `json:"enabled" bson:"enabled"`
	LeadTimeHours int                 `json:"leadTimeHours" bson:"leadTimeHours,omitempty"`
	Timezone      string              `json:"timezone" bson:"timezone,omitempty"` // IANA tz for formatting the email time (e.g. "America/Los_Angeles")
	SentAt        *primitive.DateTime `json:"sentAt" bson:"sentAt,omitempty"`     // nil = not yet sent
}

// RSVP to a confirmed gathering (paired with ScheduledEvent). Stored on the
// Event as a map keyed by guest name or signed-in user id, mirroring
// SignUpResponses.
type RsvpStatus string

const (
	RsvpGoing RsvpStatus = "going"
	RsvpMaybe RsvpStatus = "maybe"
	RsvpNo    RsvpStatus = "no"
)

type Rsvp struct {
	Status RsvpStatus `json:"status" bson:"status"`
	// GuestCount is the number of ADDITIONAL people this responder is bringing
	// (a spouse/plus-one), i.e. the headcount for this RSVP is 1 + GuestCount.
	// Only meaningful for going/maybe.
	GuestCount  int                `json:"guestCount" bson:"guestCount,omitempty"`
	Name        string             `json:"name" bson:"name,omitempty"`
	Email       string             `json:"email" bson:"email,omitempty"`
	UserId      primitive.ObjectID `json:"userId" bson:"userId,omitempty"`
	RespondedAt primitive.DateTime `json:"respondedAt" bson:"respondedAt,omitempty"`
}

type SignUpBlock struct {
	Id        primitive.ObjectID  `json:"_id" bson:"_id,omitempty"`
	Name      string              `json:"name" bson:"name,omitempty"`
	Capacity  *int                `json:"capacity" bson:"capacity,omitempty"`
	StartDate *primitive.DateTime `json:"startDate" bson:"startDate,omitempty"`
	EndDate   *primitive.DateTime `json:"endDate" bson:"endDate,omitempty"`
}

type SignUpResponse struct {
	// The IDs of the sign up blocks the user is CONFIRMED for (within capacity)
	SignUpBlockIds []primitive.ObjectID `json:"signUpBlockIds" bson:"signUpBlockIds,omitempty"`

	// The IDs of the sign up blocks the user is WAITLISTED for (block was full).
	// Assigned server-side by capacity; see assignSignUpBlocks (C9).
	WaitlistBlockIds []primitive.ObjectID `json:"waitlistBlockIds" bson:"waitlistBlockIds,omitempty"`

	// Guest information
	Name  string `json:"name" bson:"name,omitempty"`
	Email string `json:"email" bson:"email,omitempty"`

	// User information
	UserId primitive.ObjectID `json:"userId" bson:"userId,omitempty"`
	User   *User              `json:"user" bson:",omitempty"`
}

// Representation of an Event in the mongoDB database
type Event struct {
	Id          primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	ShortId     *string            `json:"shortId" bson:"shortId,omitempty"`
	OwnerId     primitive.ObjectID `json:"ownerId" bson:"ownerId,omitempty"`
	Name        string             `json:"name" bson:"name,omitempty"`
	Description *string            `json:"description" bson:"description,omitempty"`
	IsArchived  *bool              `json:"isArchived" bson:"isArchived,omitempty"`
	IsDeleted   *bool              `json:"isDeleted" bson:"isDeleted,omitempty"`

	Duration                 *float32             `json:"duration" bson:"duration,omitempty"`
	Dates                    []primitive.DateTime `json:"dates" bson:"dates,omitempty"`
	NotificationsEnabled     *bool                `json:"notificationsEnabled" bson:"notificationsEnabled,omitempty"`
	SendEmailAfterXResponses *int                 `json:"sendEmailAfterXResponses" bson:"sendEmailAfterXResponses,omitempty"`
	When2meetHref            *string              `json:"when2meetHref" bson:"when2meetHref,omitempty"`
	CollectEmails            *bool                `json:"collectEmails" bson:"collectEmails,omitempty"`
	TimeIncrement            *int                 `json:"timeIncrement" bson:"timeIncrement,omitempty"`

	// Used for specific times for specific dates feature
	HasSpecificTimes *bool                `json:"hasSpecificTimes" bson:"hasSpecificTimes,omitempty"`
	Times            []primitive.DateTime `json:"times" bson:"times,omitempty"`

	Type EventType `json:"type" bson:"type,omitempty"`

	// Sign up form details
	IsSignUpForm    *bool                      `json:"isSignUpForm" bson:"isSignUpForm,omitempty"`
	SignUpBlocks    *[]SignUpBlock             `json:"signUpBlocks" bson:"signUpBlocks,omitempty"`
	SignUpResponses map[string]*SignUpResponse `json:"signUpResponses" bson:"signUpResponses"`

	// Whether to start the event on Monday (as opposed to Sunday, used for DOW events)
	StartOnMonday *bool `json:"startOnMonday" bson:"startOnMonday,omitempty"`

	// Whether to enable blind availability
	BlindAvailabilityEnabled *bool `json:"blindAvailabilityEnabled" bson:"blindAvailabilityEnabled,omitempty"`

	// Whether to only poll for days, not times
	DaysOnly *bool `json:"daysOnly" bson:"daysOnly,omitempty"`

	// Availability responses - old format for backward compatibility (fetched from eventResponses collection)
	ResponsesMap map[string]*Response `json:"responses" bson:"-"`

	// Used to store the number of responses for the event
	NumResponses *int `json:"numResponses" bson:"numResponses,omitempty"`

	// Scheduled event (the confirmed gathering time, once the owner locks it in)
	ScheduledEvent  *CalendarEvent `json:"scheduledEvent" bson:"scheduledEvent,omitempty"`
	CalendarEventId string         `json:"calendarEventId" bson:"calendarEventId,omitempty"`

	// Pre-gathering reminder email config/state (paired with ScheduledEvent)
	GatheringReminder *GatheringReminder `json:"gatheringReminder" bson:"gatheringReminder,omitempty"`

	// RSVPs to the confirmed gathering, keyed by guest name / signed-in user id
	Rsvps map[string]*Rsvp `json:"rsvps" bson:"rsvps,omitempty"`

	// Remindees
	Remindees *[]Remindee `json:"remindees" bson:"remindees,omitempty"`

	// Attendees for an availability group (fetched from Attendees collection)
	Attendees *[]Attendee `json:"attendees" bson:"-"`

	// Whether the user has responded to the availability group (fetched based on whether user is in Attendees)
	HasResponded *bool `json:"hasResponded" bson:"-"`
}

func (e *Event) GetId() string {
	if e.ShortId != nil {
		return *e.ShortId
	}

	return e.Id.Hex()
}
