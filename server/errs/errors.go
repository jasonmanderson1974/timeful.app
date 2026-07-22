package errs

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Errors enum
// TODO: make these an actual type (i.e. Errors.NotSignedIn)
const (
	NotSignedIn           string = "not-signed-in"
	UserDoesNotExist      string = "user-does-not-exist"
	EventNotFound         string = "event-not-found"
	FriendRequestNotFound string = "friend-request-not-found"
	UserNotFriends        string = "user-not-friends"
	UserNotEventOwner     string = "user-not-event-owner"
	RemindeeEmailNotFound string = "remindee-email-not-found"
	AttendeeEmailNotFound string = "attendee-email-not-found"
	EventNotGroup         string = "event-not-group"
	InvalidCredentials    string = "invalid-credentials"
	OtpExpired            string = "otp-expired"
	OtpInvalidCode        string = "otp-invalid-code"
	OtpTooManyAttempts    string = "otp-too-many-attempts"
	OtpSendFailed         string = "otp-send-failed"
	InvalidIdToken        string = "invalid-id-token"
	// NotInvited: the email is not on the invite-only allowlist
	NotInvited string = "not-invited"
	// NotAuthorized: the user is signed in but lacks permission (e.g. not an inviter)
	NotAuthorized string = "not-authorized"
	// InvalidEmail: the provided email failed validation
	InvalidEmail string = "invalid-email"
	// CannotRemoveSelf: an admin tried to remove their own access / inviter role
	CannotRemoveSelf string = "cannot-remove-self"
)

// Sentinel error returned by signInHelper when an email is not allowlisted, so
// callers can distinguish it from other sign-in failures and return NotInvited.
var ErrNotInvited = errors.New(NotInvited)

type GoogleAPIError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Status  string      `json:"status"`
	Details interface{} `json:"details"`
	Errors  interface{} `json:"errors"`
}

func (e *GoogleAPIError) Error() string {
	s, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintln("GoogleAPIError: <error parsing json>")
	}

	return fmt.Sprintln("GoogleAPIError: ", string(s))
}
