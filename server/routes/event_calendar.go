package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sirtom/server/db"
	"sirtom/server/errs"
	"sirtom/server/logger"
	"sirtom/server/models"
	"sirtom/server/responses"
	"sirtom/server/services/calendar"
	"sirtom/server/utils"
)

// @Summary Return a map mapping user id to their calendar events that they have enabled for the given time range
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param timeMin query string true "Lower bound for event's start time to filter by"
// @Param timeMax query string true "Upper bound for event's end time to filter by"
// @Success 200 {object} map[string]map[string]calendar.CalendarEventsWithError
// @Router /events/{eventId}/calendar-availabilities [get]
func getCalendarAvailabilities(c *gin.Context) {
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

	// Get calendar events for each response that has calendar availability enabled
	numCalendarEventsRequests := 0
	calendarEventsChan := make(chan struct {
		UserId string
		Events map[string]calendar.CalendarEventsWithError
	})

	eventResponses, eventResponsesErr := db.GetEventResponses(event.Id.Hex())
	if eventResponsesErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	for _, eventResponse := range eventResponses {
		if utils.Coalesce(eventResponse.Response.UseCalendarAvailability) {
			user, userErr := db.GetUserById(eventResponse.UserId)
			if userErr != nil {
				logger.StdErr.Println(userErr)
				continue
			}
			if user != nil {
				numCalendarEventsRequests++

				// Construct enabled accounts set
				enabledAccounts := make([]string, 0)
				for calendarAccountKey := range utils.Coalesce(eventResponse.Response.EnabledCalendars) {
					enabledAccounts = append(enabledAccounts, calendarAccountKey)
				}

				// Fetch calendar events
				go func(userId string) {
					// Recover from panics
					defer func() {
						if err := recover(); err != nil {
							logger.StdErr.Println(err)
						}
					}()

					calendarEvents, _ := calendar.GetUsersCalendarEvents(user, utils.ArrayToSet(enabledAccounts), payload.TimeMin, payload.TimeMax)
					calendarEventsChan <- struct {
						UserId string
						Events map[string]calendar.CalendarEventsWithError
					}{
						UserId: userId,
						Events: calendarEvents,
					}
				}(eventResponse.UserId)
			}
		}
	}

	// Create a map mapping user id to the calendar events of that user
	userIdToCalendarEvents := make(map[string][]models.CalendarEvent)
	for i := 0; i < numCalendarEventsRequests; i++ {
		calendarEvents := <-calendarEventsChan
		userIdToCalendarEvents[calendarEvents.UserId] = make([]models.CalendarEvent, 0)
		for _, events := range calendarEvents.Events {
			userIdToCalendarEvents[calendarEvents.UserId] = append(userIdToCalendarEvents[calendarEvents.UserId], events.CalendarEvents...)
		}
	}

	// Filter and format calendar events
	authUser := utils.GetAuthUser(c)
	for userId, calendarEvents := range userIdToCalendarEvents {
		// Find the corresponding response
		_, eventResponse := findResponse(eventResponses, userId)
		if eventResponse == nil {
			continue
		}

		// Construct enabled calendar ids set
		enabledCalendarIdsArr := make([]string, 0)
		for _, calendarIds := range utils.Coalesce(eventResponse.EnabledCalendars) {
			enabledCalendarIdsArr = append(enabledCalendarIdsArr, calendarIds...)
		}
		enabledCalendarIds := utils.ArrayToSet(enabledCalendarIdsArr)

		// Update calendar events
		updatedCalendarEvents := make([]models.CalendarEvent, 0)
		for _, calendarEvent := range calendarEvents {
			// Get rid of events on sub calendars that aren't enabled
			if _, ok := enabledCalendarIds[calendarEvent.CalendarId]; !ok {
				continue
			}

			// Redact event names of other users
			if authUser.Id.Hex() != userId {
				calendarEvent.Summary = "BUSY"
			}

			updatedCalendarEvents = append(updatedCalendarEvents, calendarEvent)
		}
		userIdToCalendarEvents[userId] = updatedCalendarEvents
	}

	c.JSON(http.StatusOK, userIdToCalendarEvents)
}
