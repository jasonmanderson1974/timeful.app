import { post, _delete } from "../fetch_utils"

export const archiveEvent = (eventId, archive) => {
  return post(`/events/${eventId}/archive`, {
    archive: archive,
  })
}

/**
 * Confirm (or cancel) a gathering's locked-in time and arm the pre-gathering
 * reminder email. Pass { scheduled: false } to cancel a previously-set gathering.
 * @param {string} eventId
 * @param {{scheduled: boolean, startDate?: string, endDate?: string, summary?: string, timezone?: string, reminderEnabled?: boolean, reminderLeadTimeHours?: number}} payload
 */
export const setScheduledEvent = (eventId, payload) => {
  return post(`/events/${eventId}/schedule`, payload)
}

/**
 * RSVP to a confirmed gathering.
 * @param {string} eventId
 * @param {{status: "going"|"maybe"|"no", guest: boolean, name?: string, email?: string}} payload
 */
export const setRsvp = (eventId, payload) => {
  return post(`/events/${eventId}/rsvp`, payload)
}

/** Remove the caller's RSVP. */
export const clearRsvp = (eventId, payload) => {
  return _delete(`/events/${eventId}/rsvp`, payload)
}
