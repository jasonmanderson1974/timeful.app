package calendar

import (
	"bytes"
	"fmt"
	"net/url"
	"time"

	"github.com/emersion/go-ical"
	"schej.it/server/models"
	"schej.it/server/utils"
)

// GenerateEventICS builds a downloadable .ics (a VCALENDAR with one VEVENT) for
// an event's confirmed gathering time. This is the universal "add to calendar"
// path — it needs no OAuth, so it works for everyone, including members without
// a Google/Outlook account. Mirrors the parsing in ics_calendar.go.
//
// Returns an error if the event has no confirmed gathering (ScheduledEvent nil).
func GenerateEventICS(event *models.Event) ([]byte, error) {
	if event == nil || event.ScheduledEvent == nil {
		return nil, fmt.Errorf("event has no confirmed gathering time")
	}

	start := event.ScheduledEvent.StartDate.Time().UTC()
	end := event.ScheduledEvent.EndDate.Time().UTC()

	summary := event.ScheduledEvent.Summary
	if summary == "" {
		summary = event.Name
	}

	eventUrl := fmt.Sprintf("%s/e/%s", utils.GetBaseUrl(), event.GetId())
	description := eventUrl
	if event.Description != nil && *event.Description != "" {
		description = fmt.Sprintf("%s\n\n%s", *event.Description, eventUrl)
	}

	vevent := ical.NewEvent()
	// Stable UID so re-downloading updates the same calendar entry rather than
	// creating a duplicate.
	vevent.Props.SetText(ical.PropUID, fmt.Sprintf("%s@timeful.app", event.Id.Hex()))
	vevent.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	vevent.Props.SetDateTime(ical.PropDateTimeStart, start)
	vevent.Props.SetDateTime(ical.PropDateTimeEnd, end)
	vevent.Props.SetText(ical.PropSummary, summary)
	vevent.Props.SetText(ical.PropDescription, description)
	if u, err := url.Parse(eventUrl); err == nil {
		vevent.Props.SetURI(ical.PropURL, u)
	}
	vevent.SetStatus(ical.EventConfirmed)

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//The Fellowship//Timeful//EN")
	cal.Props.SetText(ical.PropMethod, "PUBLISH")
	cal.Children = append(cal.Children, vevent.Component)

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
