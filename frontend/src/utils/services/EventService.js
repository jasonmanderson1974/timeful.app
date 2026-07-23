import { post } from "../fetch_utils"

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
