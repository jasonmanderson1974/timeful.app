package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"schej.it/server/db"
	"schej.it/server/errs"
	"schej.it/server/logger"
	"schej.it/server/models"
	"schej.it/server/responses"
	"schej.it/server/services/gcloud"
	"schej.it/server/services/listmonk"
	"schej.it/server/utils"
)

// @Summary Gets responses for an event, filtering availability to be within the date ranges
// @Tags events
// @Produce json
// @Param eventId path string true "Event ID"
// @Param timeMin query string true "Lower bound for start time to filter availability by"
// @Param timeMax query string true "Upper bound for end time to filter availability by"
// @Success 200 {object} map[string]models.Response
// @Router /events/{eventId}/responses [get]
func getResponses(c *gin.Context) {
	// Bind query parameters
	payload := struct {
		TimeMin time.Time `form:"timeMin" binding:"required"`
		TimeMax time.Time `form:"timeMax" binding:"required"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	// Fetch event
	eventId := c.Param("eventId")
	event, eventErr := db.GetEventByEitherId(eventId)
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}

	// Convert to map format and filter availability
	eventResponses, eventResponsesErr := db.GetEventResponses(event.Id.Hex())
	if eventResponsesErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	responsesMap := getResponsesMap(eventResponses)

	// Filter availability slice based on timeMin and timeMax
	for userId, response := range responsesMap {
		subsetAvailability := make([]primitive.DateTime, 0)
		for _, timestamp := range response.Availability {
			if timestamp.Time().Compare(payload.TimeMin) >= 0 && timestamp.Time().Compare(payload.TimeMax) <= 0 {
				subsetAvailability = append(subsetAvailability, timestamp)
			}
		}
		response.Availability = subsetAvailability

		subsetIfNeeded := make([]primitive.DateTime, 0)
		for _, timestamp := range response.IfNeeded {
			if timestamp.Time().Compare(payload.TimeMin) >= 0 && timestamp.Time().Compare(payload.TimeMax) <= 0 {
				subsetIfNeeded = append(subsetIfNeeded, timestamp)
			}
		}
		response.IfNeeded = subsetIfNeeded

		subsetManualAvailability := make(map[primitive.DateTime][]primitive.DateTime)
		for timestamp := range utils.Coalesce(response.ManualAvailability) {
			if timestamp.Time().Compare(payload.TimeMin) >= 0 && timestamp.Time().Compare(payload.TimeMax) <= 0 {
				subsetManualAvailability[timestamp] = (*response.ManualAvailability)[timestamp]
			}
		}
		response.ManualAvailability = &subsetManualAvailability
		responsesMap[userId] = response
	}

	// Determine if the requester is the event owner
	ownerSesh := event.OwnerId.Hex()
	session := sessions.Default(c)
	userIdInterface := session.Get("userId")
	var userSesh string
	if userIdInterface != nil {
		userSesh = userIdInterface.(string)
	}
	guestName := c.Query("guestName")
	isOwner := userSesh != "" && ownerSesh == userSesh

	// Strip sensitive user info from all responses
	showEmails := isOwner && utils.Coalesce(event.CollectEmails)
	for userId, response := range responsesMap {
		stripSensitiveUserFields(response.User)
		if !showEmails {
			response.Email = ""
			if response.User != nil && !shouldKeepGroupResponseUserEmails(event, userSesh, isOwner) {
				response.User.Email = ""
			}
		}
		responsesMap[userId] = response
	}

	// Apply blind-availability privacy filtering, then return.
	c.JSON(http.StatusOK, filterResponsesForBlindAvailability(event, responsesMap, userSesh, guestName))
}

// filterResponsesForBlindAvailability applies the blind-availability privacy
// rule and returns the response map that should be sent to the requester. When
// blind availability is off, everyone sees every response. When it's on: the
// owner sees all; a logged-in non-owner sees only their own response; a guest
// (identified by guestName) sees only theirs; an anonymous viewer sees nothing.
func filterResponsesForBlindAvailability(event *models.Event, responsesMap map[string]*models.Response, userSesh string, guestName string) map[string]*models.Response {
	if !utils.Coalesce(event.BlindAvailabilityEnabled) {
		return responsesMap
	}

	if userSesh != "" {
		if event.OwnerId.Hex() == userSesh {
			return responsesMap
		}
		filteredMap := make(map[string]*models.Response)
		if userResponse, exists := responsesMap[userSesh]; exists {
			filteredMap[userSesh] = userResponse
		}
		return filteredMap
	}

	if guestName != "" {
		filteredMap := make(map[string]*models.Response)
		if guestResponse, exists := responsesMap[guestName]; exists {
			filteredMap[guestName] = guestResponse
		}
		return filteredMap
	}

	return make(map[string]*models.Response)
}

// @Summary Updates the current user's availability
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param payload body object{availability=[]string,ifNeeded=[]string,guest=bool,name=string,useCalendarAvailability=bool,enabledCalendars=map[string][]string,manualAvailability=map[string][]string,calendarOptions=models.CalendarOptions,signUpBlockIds=[]string} true "Object containing info about the event response to update"
// @Success 200
// @Router /events/{eventId}/response [post]
func updateEventResponse(c *gin.Context) {
	payload := struct {
		Availability []primitive.DateTime `json:"availability"`
		IfNeeded     []primitive.DateTime `json:"ifNeeded"`

		// Guest information
		Guest *bool  `json:"guest" binding:"required"`
		Name  string `json:"name"`
		Email string `json:"email"`

		// Calendar availability variables for Availability Groups feature
		UseCalendarAvailability *bool                                        `json:"useCalendarAvailability"`
		EnabledCalendars        *map[string][]string                         `json:"enabledCalendars"`
		ManualAvailability      *map[primitive.DateTime][]primitive.DateTime `json:"manualAvailability"`
		CalendarOptions         *models.CalendarOptions                      `json:"calendarOptions"`

		// Sign up form variables
		SignUpBlockIds []primitive.ObjectID `json:"signUpBlockIds"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}
	session := sessions.Default(c)
	eventId := c.Param("eventId")
	event, eventErr := db.GetEventByEitherId(eventId)
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}

	// Security check: If blindAvailabilityEnabled is true, non-owners cannot set guest availability
	//NOTE: this ONLY stops a user from setting guest availability from their account (via setSlots), somebody could still
	// go on incognito and set guest availability.
	if utils.Coalesce(event.BlindAvailabilityEnabled) {
		ownerSesh := event.OwnerId.Hex()
		userIdInterface := session.Get("userId")
		var userSesh string
		if userIdInterface != nil {
			userSesh = userIdInterface.(string)
		}

		// If user is logged in and NOT the owner, and they're trying to set guest availability, block it
		if userSesh != "" && ownerSesh != userSesh && *payload.Guest {
			c.JSON(http.StatusForbidden, responses.Error{Error: errs.UserNotEventOwner})
			c.Abort()
			return
		}
	}

	eventResponses, eventResponsesErr := db.GetEventResponses(event.Id.Hex())
	if eventResponsesErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	var userIdString string
	var userHasResponded bool
	if !utils.Coalesce(event.IsSignUpForm) {
		// Populate response differently if guest vs signed in user
		var response models.Response
		if *payload.Guest {
			userIdString = payload.Name

			response = models.Response{
				Name:         payload.Name,
				Email:        payload.Email,
				Availability: payload.Availability,
				IfNeeded:     payload.IfNeeded,
			}
		} else {
			userIdInterface := session.Get("userId")
			if userIdInterface == nil {
				c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.NotSignedIn})
				c.Abort()
				return
			}
			userIdString = userIdInterface.(string)
			userId := utils.StringToObjectID(userIdString)

			response = models.Response{
				UserId:                  userId,
				Availability:            payload.Availability,
				IfNeeded:                payload.IfNeeded,
				UseCalendarAvailability: payload.UseCalendarAvailability,
				EnabledCalendars:        payload.EnabledCalendars,
				CalendarOptions:         payload.CalendarOptions,
			}

			if event.Type == models.GROUP {
				user, userErr := db.GetUserById(userIdString)
				if userErr != nil {
					c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
					return
				}

				// Set declined to false (in case user declined group in the past)
				if user != nil {
					db.AttendeesCollection.UpdateOne(context.Background(), bson.M{
						"email":   user.Email,
						"eventId": event.Id,
					}, bson.M{
						"$set": bson.M{
							"declined": false,
						},
					})
				}

				// Update manual availability
				_, existingResponse := findResponse(eventResponses, userIdString)
				if existingResponse != nil {
					response.ManualAvailability = existingResponse.ManualAvailability
				}
				if response.ManualAvailability == nil {
					manualAvailability := make(map[primitive.DateTime][]primitive.DateTime)
					response.ManualAvailability = &manualAvailability
				}

				// Replace availability on days that already exist in manual availability map
				for day := range utils.Coalesce(response.ManualAvailability) {
					for payloadDay, availableTimes := range utils.Coalesce(payload.ManualAvailability) {
						// Check if day is between start and end times of the payload day
						endTime := payloadDay.Time().Add(time.Duration(*event.Duration) * time.Hour)
						if day.Time().Compare(payloadDay.Time()) >= 0 && day.Time().Compare(endTime) <= 0 {
							// Replace availability with updated availability
							delete(*response.ManualAvailability, day)
							(*response.ManualAvailability)[payloadDay] = availableTimes
							delete(*payload.ManualAvailability, payloadDay)
							break
						}
					}

					// Break if no more items in manual availability
					if len(utils.Coalesce(payload.ManualAvailability)) == 0 {
						break
					}
				}

				// Add the rest of manual availability that was not replaced
				for day, availableTimes := range utils.Coalesce(payload.ManualAvailability) {
					(*response.ManualAvailability)[day] = availableTimes
				}
			}
		}

		// Check if user has responded to event before (edit response) or not (new response)
		idx, _ := findResponse(eventResponses, userIdString)
		userHasResponded = idx != -1

		// Update event responses
		if userHasResponded {
			db.EventResponsesCollection.UpdateOne(context.Background(), bson.M{
				"_id": eventResponses[idx].Id,
			}, bson.M{
				"$set": bson.M{
					"response": &response,
				},
			})
		} else {
			if _, err := db.EventResponsesCollection.InsertOne(context.Background(), models.EventResponse{
				UserId:   userIdString,
				Response: &response,
				EventId:  event.Id,
			}); err != nil {
				logger.StdErr.Println(err)
			} else {
				*event.NumResponses++
			}
		}
	} else {
		var response models.SignUpResponse
		var userIdString string
		// Populate response differently if guest vs signed in user
		if *payload.Guest {
			userIdString = payload.Name

			response = models.SignUpResponse{
				SignUpBlockIds: payload.SignUpBlockIds,
				Name:           payload.Name,
				Email:          payload.Email,
			}
		} else {
			userIdInterface := session.Get("userId")
			if userIdInterface == nil {
				c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.NotSignedIn})
				c.Abort()
				return
			}
			userIdString = userIdInterface.(string)

			response = models.SignUpResponse{
				SignUpBlockIds: payload.SignUpBlockIds,
				UserId:         utils.StringToObjectID(userIdString),
			}
		}

		// Check if user has responded to event before (edit response) or not (new response)
		_, userHasResponded = event.SignUpResponses[userIdString]

		// Update event responses
		if event.SignUpResponses == nil {
			event.SignUpResponses = make(map[string]*models.SignUpResponse)
		}
		event.SignUpResponses[userIdString] = &response
	}

	// Send notification emails
	if (utils.Coalesce(event.NotificationsEnabled) || event.Type == models.GROUP) && !userHasResponded && userIdString != event.OwnerId.Hex() {
		// Send email asynchronously
		go func() {
			// Recover from panics
			defer func() {
				if err := recover(); err != nil {
					logger.StdErr.Println(err)
				}
			}()

			creator, creatorErr := db.GetUserById(event.OwnerId.Hex())
			if creatorErr != nil {
				logger.StdErr.Println(creatorErr)
				return
			}
			if creator == nil {
				return
			}

			var respondentName string
			if *payload.Guest {
				respondentName = payload.Name
			} else {
				respondent, respondentErr := db.GetUserById(userIdString)
				if respondentErr != nil {
					logger.StdErr.Println(respondentErr)
					return
				}
				respondentName = fmt.Sprintf("%s %s", respondent.FirstName, respondent.LastName)
			}

			if event.Type == models.GROUP {
				someoneRespondedEmailId := 13
				listmonk.SendEmail(creator.Email, someoneRespondedEmailId, bson.M{
					"groupName":      event.Name,
					"ownerName":      creator.FirstName,
					"respondentName": respondentName,
					"groupUrl":       fmt.Sprintf("%s/g/%s", utils.GetBaseUrl(), event.GetId()),
				})
			} else {
				someoneRespondedEmailId := 10
				listmonk.SendEmail(creator.Email, someoneRespondedEmailId, bson.M{
					"eventName":      event.Name,
					"ownerName":      creator.FirstName,
					"respondentName": respondentName,
					"eventUrl":       fmt.Sprintf("%s/e/%s", utils.GetBaseUrl(), event.GetId()),
				})
			}
		}()
	}

	// Send email after X responses
	sendEmailAfterXResponses := utils.Coalesce(event.SendEmailAfterXResponses)
	if sendEmailAfterXResponses > 0 && !userHasResponded && sendEmailAfterXResponses == len(eventResponses)+1 { // We add 1 because eventResponses is the old event responses before the current user is added
		// Set SendEmailAfterXResponses variable to -1 to prevent additional emails from being sent
		*event.SendEmailAfterXResponses = -1

		// Send email asynchronously
		go func() {
			// Recover from panics
			defer func() {
				if err := recover(); err != nil {
					logger.StdErr.Println(err)
				}
			}()

			creator, creatorErr := db.GetUserById(event.OwnerId.Hex())
			if creatorErr != nil {
				logger.StdErr.Println(creatorErr)
				return
			}
			if creator == nil {
				return
			}

			sendEmailAfterXResponsesEmailId := 14
			listmonk.SendEmail(creator.Email, sendEmailAfterXResponsesEmailId, bson.M{
				"eventName":    event.Name,
				"ownerName":    creator.FirstName,
				"eventUrl":     fmt.Sprintf("%s/e/%s", utils.GetBaseUrl(), event.GetId()),
				"numResponses": len(eventResponses) + 1, // We add 1 because eventResponses is the old event responses before the current user is added
			})
		}()
	}

	// Update event in mongodb
	_, err := db.EventsCollection.UpdateByID(
		context.Background(),
		event.Id,
		bson.M{"$set": event},
	)
	if err != nil {
		logger.StdErr.Println(err)
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

// @Summary Delete the current user's availability
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param payload body object{userId=string,guest=bool,name=string} true "Object containing info about the event response to delete"
// @Success 200
// @Router /events/{eventId}/response [delete]
func deleteEventResponse(c *gin.Context) {
	payload := struct {
		UserId string `json:"userId"`
		Guest  *bool  `json:"guest" binding:"required"`
		Name   string `json:"name"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}
	session := sessions.Default(c)
	eventId := c.Param("eventId")
	event, eventErr := db.GetEventByEitherId(eventId)
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}
	eventResponses, eventResponsesErr := db.GetEventResponses(event.Id.Hex())
	if eventResponsesErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	if *payload.Guest {
		if utils.Coalesce(event.IsSignUpForm) {
			delete(event.SignUpResponses, payload.Name)
		} else {
			// Remove response from array
			for i := range eventResponses {
				if eventResponses[i].Response.Name == payload.Name {
					db.EventResponsesCollection.DeleteOne(context.Background(), bson.M{
						"_id": eventResponses[i].Id,
					})
					*event.NumResponses--
					break
				}
			}
		}
	} else {
		userIdInterface := session.Get("userId")
		if userIdInterface == nil {
			c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.NotSignedIn})
			c.Abort()
			return
		}
		userIdString := userIdInterface.(string)

		// Don't allow user to delete availability of other users if they aren't the owner of the event
		if payload.UserId != userIdString && event.OwnerId.Hex() != userIdString {
			c.JSON(http.StatusForbidden, responses.Error{Error: errs.UserNotEventOwner})
			c.Abort()
			return
		}

		if utils.Coalesce(event.IsSignUpForm) {
			delete(event.SignUpResponses, payload.UserId)
		} else {
			// Remove response from array
			for i := range eventResponses {
				if eventResponses[i].UserId == payload.UserId {
					db.EventResponsesCollection.DeleteOne(context.Background(), bson.M{
						"_id": eventResponses[i].Id,
					})
					*event.NumResponses--
					break
				}
			}
		}

		// If this event is a Group, also make the attendee "leave the group" by setting "declined" to true
		if event.Type == models.GROUP {
			user, userErr := db.GetUserById(userIdString)
			if userErr != nil {
				c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
				return
			}
			if user != nil {
				db.AttendeesCollection.UpdateOne(context.Background(), bson.M{
					"email":   user.Email,
					"eventId": event.Id,
				}, bson.M{
					"$set": bson.M{
						"declined": true,
					},
				})
			}
		}
	}

	// Update responses in mongodb
	_, err := db.EventsCollection.UpdateByID(
		context.Background(),
		event.Id,
		bson.M{
			"$set": event,
		},
	)
	if err != nil {
		logger.StdErr.Println(err)
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

// @Summary Rename a guest response
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param payload body object{oldName=string,newName=string} true "Object containing info about the guest response to rename"
// @Success 200
// @Failure 400 {object} responses.Error "Guest name already exists"
// @Router /events/{eventId}/rename-user [post]
func renameUser(c *gin.Context) {
	payload := struct {
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}
	eventId := c.Param("eventId")
	event, eventErr := db.GetEventByEitherId(eventId)
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}

	// Check if the new name already exists (only if it's different from the old name)
	if payload.NewName != payload.OldName {
		if db.GuestNameExists(event.Id.Hex(), payload.NewName) {
			c.JSON(http.StatusBadRequest, responses.Error{Error: "A guest with this name already exists for this event"})
			return
		}
	}

	// Check if old name is a guest response
	db.UpdateGuestResponseName(event.Id.Hex(), payload.OldName, payload.NewName)

	c.JSON(http.StatusOK, gin.H{})
}

// @Summary Mark the user as having responded to this event
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param payload body object{email=string} true "Object containing the user's email"
// @Success 200
// @Router /events/{eventId}/responded [post]
func userResponded(c *gin.Context) {
	payload := struct {
		Email string `json:"email" binding:"required"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	// Fetch event
	eventId := c.Param("eventId")
	event, eventErr := db.GetEventByEitherId(eventId)
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}

	// Update responded boolean for the given email
	if event.Remindees == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.RemindeeEmailNotFound})
		return
	}
	index := utils.Find(*event.Remindees, func(r models.Remindee) bool {
		return r.Email == payload.Email
	})
	if index == -1 {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.RemindeeEmailNotFound})
		return
	}
	if *(*event.Remindees)[index].Responded {
		// If remindee has already responded, just return and don't update db
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	(*event.Remindees)[index].Responded = utils.TruePtr()

	// Delete the reminder email tasks
	for _, taskId := range (*event.Remindees)[index].TaskIds {
		gcloud.DeleteEmailTask(taskId)
	}

	// Update event in database
	db.EventsCollection.UpdateByID(context.Background(), event.Id, bson.M{
		"$set": event,
	})

	// Email owner of event if all remindees have responded
	everyoneResponded := true
	for _, remindee := range *event.Remindees {
		if !*remindee.Responded {
			everyoneResponded = false
			break
		}
	}
	if everyoneResponded {
		// Get owner
		owner, ownerErr := db.GetUserById(event.OwnerId.Hex())
		if ownerErr != nil {
			c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
			return
		}

		// Get event url
		baseUrl := utils.GetBaseUrl()
		eventUrl := fmt.Sprintf("%s/e/%s", baseUrl, eventId)

		// Send email
		everyoneRespondedEmailTemplateId := 8
		listmonk.SendEmail(owner.Email, everyoneRespondedEmailTemplateId, bson.M{
			"eventName": event.Name,
			"eventUrl":  eventUrl,
		})
	}

	c.JSON(http.StatusOK, gin.H{})
}

// @Summary Decline the current user's invite to the event
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Success 200
// @Router /events/{eventId}/decline [post]
func declineInvite(c *gin.Context) {
	// Fetch event
	eventId := c.Param("eventId")
	event, eventErr := db.GetEventById(eventId)
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}

	// Ensure that event is a group
	if event.Type != models.GROUP {
		c.JSON(http.StatusBadRequest, responses.Error{Error: errs.EventNotGroup})
		return
	}

	// Get current user
	userInterface, _ := c.Get("authUser")
	user := userInterface.(*models.User)

	// Check if user is in attendees array
	attendee := db.AttendeesCollection.FindOne(context.Background(), bson.M{
		"email":   user.Email,
		"eventId": event.Id,
	})
	if attendee == nil {
		// User not in attendees array
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.AttendeeEmailNotFound})
		return
	}

	// Decline invite
	db.AttendeesCollection.UpdateOne(context.Background(), bson.M{
		"email":   user.Email,
		"eventId": event.Id,
	}, bson.M{
		"$set": bson.M{
			"declined": true,
		},
	})

	c.JSON(http.StatusOK, gin.H{})
}

// Helper function to find a response by userId
func findResponse(responses []models.EventResponse, userId string) (int, *models.Response) {
	for i, resp := range responses {
		if resp.UserId == userId {
			return i, resp.Response
		}
	}
	return -1, nil
}

// shouldKeepGroupResponseUserEmails is true for signed-in group owners and invitees
// so clients can match pending attendees to respondents when collectEmails is off.
func shouldKeepGroupResponseUserEmails(event *models.Event, userSesh string, isOwner bool) bool {
	if event.Type != models.GROUP || userSesh == "" {
		return false
	}
	if isOwner {
		return true
	}
	user, _ := db.GetUserById(userSesh)
	if user == nil {
		return false
	}
	viewerEmail := utils.NormalizeEmail(user.Email)
	if viewerEmail == "" {
		return false
	}
	var attendees []models.Attendee
	if event.Attendees != nil {
		attendees = *event.Attendees
	} else {
		attendees, _ = db.GetAttendees(event.Id.Hex())
	}
	for _, a := range attendees {
		if utils.NormalizeEmail(a.Email) == viewerEmail {
			return true
		}
	}
	return false
}

// stripSensitiveUserFields removes fields from a User that should never be
// exposed in the event page API response (calendar accounts, billing info, etc.).
// Email is NOT stripped here as callers handle email visibility separately based
// on the collectEmails setting and owner status.
func stripSensitiveUserFields(user *models.User) {
	if user == nil {
		return
	}
	user.CalendarAccounts = nil
	user.CalendarOptions = nil
	user.StripeCustomerId = nil
	user.PrimaryAccountKey = nil
}

// Helper function to get all responses as a map (for backward compatibility)
func getResponsesMap(responses []models.EventResponse) map[string]*models.Response {
	result := make(map[string]*models.Response)
	for _, resp := range responses {
		result[resp.UserId] = resp.Response
	}
	return result
}
