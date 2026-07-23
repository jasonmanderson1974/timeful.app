import { post, put, _delete } from "../fetch_utils"

export const archiveEvent = (eventId, archive) => {
  return post(`/events/${eventId}/archive`, {
    archive: archive,
  })
}

/**
 * Confirm (or cancel) a gathering's locked-in time and arm the pre-gathering
 * reminder email. Pass { scheduled: false } to cancel a previously-set gathering.
 * @param {string} eventId
 * A recurrenceFrequency of "weekly" | "biweekly" | "monthly" makes it a
 * repeating gathering (C5); omit or "none" for a one-off.
 * @param {{scheduled: boolean, startDate?: string, endDate?: string, summary?: string, timezone?: string, reminderEnabled?: boolean, reminderLeadTimeHours?: number, recurrenceFrequency?: "none"|"weekly"|"biweekly"|"monthly", recurrenceUntil?: string}} payload
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

// --- Discussion thread (C7) ---

/** Post a comment. payload: {text, guest, name?} */
export const addComment = (eventId, payload) => {
  return post(`/events/${eventId}/comments`, payload)
}

/** Edit your own comment. payload: {text, guest, name?} */
export const editComment = (eventId, commentId, payload) => {
  return put(`/events/${eventId}/comments/${commentId}`, payload)
}

/** Delete a comment (own, or any if you're the owner). payload: {guest, name?} */
export const deleteComment = (eventId, commentId, payload) => {
  return _delete(`/events/${eventId}/comments/${commentId}`, payload)
}

// --- Venue / activity polls (C6) ---

/** Create a poll (owner only). payload: {title, allowMultiple?, options: string[]} */
export const createPoll = (eventId, payload) => {
  return post(`/events/${eventId}/polls`, payload)
}

/** Delete a poll (owner only). */
export const deletePoll = (eventId, pollId) => {
  return _delete(`/events/${eventId}/polls/${pollId}`)
}

/**
 * Cast/update the caller's vote in a poll (members + guests). An empty
 * optionIds clears the caller's vote. payload: {optionIds, guest, name?}
 */
export const votePoll = (eventId, pollId, payload) => {
  return post(`/events/${eventId}/polls/${pollId}/vote`, payload)
}
