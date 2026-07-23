// Package reminders runs an in-process scheduler that emails a one-time
// reminder to every availability respondent (with an email) a configurable
// number of hours before a confirmed gathering's start time.
//
// It deliberately avoids the legacy Cloud Tasks + listmonk path (unconfigured
// on this fork) and sends via Gmail SMTP (utils.SendEmail), the same transport
// used for OTP + invite emails. Single-VM assumption: no distributed locking.
package reminders

import (
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"schej.it/server/db"
	"schej.it/server/logger"
	"schej.it/server/models"
	"schej.it/server/utils"
)

// SendFunc sends a single email. Matches utils.SendEmail's signature so it can
// be injected in tests (and defaulted to utils.SendEmail in production).
type SendFunc func(toEmail, subject, body, contentType string) error

const defaultInterval = 5 * time.Minute

// StartReminderScheduler launches the background ticker and returns a stop
// function. If Gmail SMTP creds are absent it logs and no-ops (mirrors
// gcloud.InitTasks), so the server still boots fine without email configured.
func StartReminderScheduler() func() {
	if os.Getenv("GMAIL_APP_PASSWORD") == "" || os.Getenv("SCHEJ_EMAIL_ADDRESS") == "" {
		logger.StdOut.Println("GMAIL creds not set, pre-gathering reminder scheduler disabled")
		return func() {}
	}

	interval := defaultInterval
	if v := os.Getenv("REMINDER_SCHEDULER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		} else {
			logger.StdErr.Println("invalid REMINDER_SCHEDULER_INTERVAL, using default:", err)
		}
	}

	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		// Run once at startup so a reminder that came due while the server was
		// down doesn't wait a full interval.
		processDueReminders(time.Now(), utils.SendEmail)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				processDueReminders(time.Now(), utils.SendEmail)
			}
		}
	}()

	logger.StdOut.Println("Pre-gathering reminder scheduler started, interval:", interval)
	return func() {
		ticker.Stop()
		close(done)
	}
}

// processDueReminders sends reminders for every gathering that has entered its
// lead-time window and hasn't been reminded yet, then marks each as sent so it
// is never re-sent (even if some individual sends failed).
func processDueReminders(now time.Time, send SendFunc) {
	events, err := db.GetEventsWithPendingReminders()
	if err != nil {
		return // already logged
	}

	for i := range events {
		event := events[i]
		if event.ScheduledEvent == nil || event.GatheringReminder == nil {
			continue
		}
		start := event.ScheduledEvent.StartDate.Time()
		if !isReminderDue(now, start, event.GatheringReminder.LeadTimeHours) {
			continue
		}

		lookupEmail := func(userId string) string {
			user, err := db.GetUserById(userId)
			if err != nil || user == nil {
				return ""
			}
			return user.Email
		}

		// Prefer RSVPs once anyone has responded (remind going + maybe, skip
		// decliners). Before any RSVP exists, fall back to all availability
		// respondents so reminders keep working.
		var emails []string
		if len(event.Rsvps) > 0 {
			emails = collectRsvpRecipientEmails(&event, lookupEmail)
		} else {
			responses, respErr := db.GetEventResponses(event.Id.Hex())
			if respErr != nil {
				// Transient DB error — skip (don't mark sent) so we retry next tick.
				continue
			}
			emails = collectRecipientEmails(responses, lookupEmail)
		}

		subject, body := buildReminderEmail(&event, start)
		for _, email := range emails {
			if err := send(email, subject, body, "text/html"); err != nil {
				logger.StdErr.Println("reminder email failed for", email, ":", err)
			}
		}

		// Mark sent regardless of per-recipient failures to avoid resend loops.
		db.MarkGatheringReminderSent(event.Id, primitive.NewDateTimeFromTime(now))
	}
}

// isReminderDue reports whether a reminder should fire now: the gathering is
// still in the future and we've reached (start - leadTimeHours).
func isReminderDue(now, start time.Time, leadTimeHours int) bool {
	if !start.After(now) {
		return false // gathering already started/past — don't send a late reminder
	}
	dueAt := start.Add(-time.Duration(leadTimeHours) * time.Hour)
	return !now.Before(dueAt) // now >= dueAt
}

// collectRecipientEmails returns the deduped set of email addresses to remind:
// a guest's supplied email, or a signed-in responder's account email resolved
// via lookupEmail (userId hex -> email, "" if none). Order is stable by first
// appearance so output is deterministic (helps tests).
func collectRecipientEmails(responses []models.EventResponse, lookupEmail func(userId string) string) []string {
	seen := make(map[string]bool)
	emails := make([]string, 0)

	add := func(email string) {
		if email == "" || seen[email] {
			return
		}
		seen[email] = true
		emails = append(emails, email)
	}

	for _, resp := range responses {
		if resp.Response != nil && resp.Response.Email != "" {
			add(resp.Response.Email) // guest-supplied email
			continue
		}
		if _, err := primitive.ObjectIDFromHex(resp.UserId); err == nil {
			add(lookupEmail(resp.UserId)) // signed-in responder
		}
	}

	return emails
}

// collectRsvpRecipientEmails returns the deduped emails of everyone who RSVP'd
// going or maybe (decliners excluded). Uses the RSVP's stored email, else
// resolves a signed-in RSVP's account email via lookupEmail. Keyed like
// collectRecipientEmails so both share the reminder pipeline. `key` is the map
// key (guest name or user-id hex); it's the userId lookup source when a
// signed-in RSVP has no stored email.
func collectRsvpRecipientEmails(event *models.Event, lookupEmail func(userId string) string) []string {
	seen := make(map[string]bool)
	emails := make([]string, 0)

	add := func(email string) {
		if email == "" || seen[email] {
			return
		}
		seen[email] = true
		emails = append(emails, email)
	}

	for key, rsvp := range event.Rsvps {
		if rsvp == nil {
			continue
		}
		if rsvp.Status != models.RsvpGoing && rsvp.Status != models.RsvpMaybe {
			continue // skip decliners
		}
		if rsvp.Email != "" {
			add(rsvp.Email)
			continue
		}
		// Signed-in RSVP with no stored email: the map key is the user-id hex.
		if _, err := primitive.ObjectIDFromHex(key); err == nil {
			add(lookupEmail(key))
		}
	}

	return emails
}

// buildReminderEmail renders the subject + inline-HTML body for a gathering
// reminder, formatting the start time in the event's saved timezone (falling
// back to UTC). Styled to match the Fellowship OTP/invite emails.
func buildReminderEmail(event *models.Event, start time.Time) (subject, body string) {
	loc := time.UTC
	if event.GatheringReminder != nil && event.GatheringReminder.Timezone != "" {
		if l, err := time.LoadLocation(event.GatheringReminder.Timezone); err == nil {
			loc = l
		}
	}
	when := start.In(loc).Format("Monday, January 2 at 3:04 PM MST")

	subject = fmt.Sprintf("Reminder: %s", event.Name)
	eventUrl := fmt.Sprintf("%s/e/%s", utils.GetBaseUrl(), event.GetId())
	// Universal "add to calendar" (.ics) — works for members without a Google
	// account. Same-origin /api path (prod), served by getEventIcs.
	icsUrl := fmt.Sprintf("%s/api/events/%s/ics", utils.GetBaseUrl(), event.GetId())

	descriptionRow := ""
	if event.Description != nil && *event.Description != "" {
		descriptionRow = fmt.Sprintf(
			`<div style="font-size:14px;color:#b8ad97;line-height:1.6;margin-bottom:24px;">%s</div>`,
			*event.Description,
		)
	}

	body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="margin:0;padding:0;background-color:#1c1410;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#1c1410;">
    <tr>
      <td align="center" style="padding:40px 16px;">
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:440px;background-color:#241a13;border:1px solid #8a7333;border-radius:14px;">
          <tr>
            <td style="padding:32px 36px;font-family:Georgia,'Times New Roman',serif;color:#ede4d3;">
              <div style="font-size:13px;font-weight:bold;letter-spacing:0.16em;color:#c9a44c;text-transform:uppercase;">The Fellowship</div>
              <div style="height:1px;background-color:#8a7333;margin:18px 0 24px;"></div>
              <div style="font-size:22px;color:#ede4d3;margin-bottom:10px;">A gathering approaches</div>
              <div style="font-size:16px;color:#ede4d3;margin-bottom:6px;"><strong>%s</strong></div>
              <div style="font-size:14px;color:#e3c578;margin-bottom:24px;">%s</div>
              %s
              <div style="text-align:center;margin-bottom:16px;">
                <a href="%s" style="display:inline-block;background-color:#c9a44c;color:#1c1410;font-weight:bold;text-decoration:none;padding:12px 28px;border-radius:8px;letter-spacing:0.04em;">View the Gathering</a>
              </div>
              <div style="text-align:center;margin-bottom:24px;">
                <a href="%s" style="display:inline-block;color:#e3c578;text-decoration:none;font-size:13px;border:1px solid #8a7333;padding:9px 22px;border-radius:8px;">Add to calendar</a>
              </div>
              <div style="font-size:12px;color:#b8ad97;line-height:1.5;">
                Or visit: <span style="color:#e3c578;">%s</span>
              </div>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, event.Name, when, descriptionRow, eventUrl, icsUrl, eventUrl)

	return subject, body
}
