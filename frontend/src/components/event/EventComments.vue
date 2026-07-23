<template>
  <div
    class="tw-mt-4 tw-rounded-md tw-border tw-border-brass-dim tw-bg-leather tw-p-3 tw-text-parchment sm:tw-p-4"
  >
    <div class="tw-mb-2 tw-text-base tw-font-medium">Discussion</div>

    <!-- Thread -->
    <div v-if="comments.length" class="tw-space-y-3">
      <div v-for="comment in comments" :key="comment._id" class="tw-flex tw-gap-2">
        <v-avatar :size="22" class="tw-mt-0.5 tw-flex-none">
          <v-icon small>mdi-account</v-icon>
        </v-avatar>
        <div class="tw-min-w-0 tw-flex-grow">
          <div class="tw-flex tw-items-baseline tw-gap-2">
            <span class="tw-text-sm tw-font-medium">{{ comment.authorName }}</span>
            <span class="tw-text-xs tw-text-parchment-dim">
              {{ formatTime(comment.createdAt) }}
              <span v-if="comment.updatedAt">· edited</span>
            </span>
          </div>

          <!-- Inline edit -->
          <div v-if="editingId === comment._id" class="tw-mt-1 tw-flex tw-items-center tw-gap-2">
            <v-textarea
              v-model="editText"
              dense
              hide-details
              auto-grow
              :rows="1"
              class="tw-flex-grow tw-text-sm"
              autofocus
            ></v-textarea>
            <v-btn icon x-small @click="cancelEdit"><v-icon small>mdi-close</v-icon></v-btn>
            <v-btn icon x-small color="primary" @click="saveEdit(comment)"
              ><v-icon small>mdi-check</v-icon></v-btn
            >
          </div>

          <div
            v-else
            class="tw-whitespace-pre-wrap tw-break-words tw-text-sm tw-text-parchment-dim"
          >
            {{ comment.text }}
          </div>

          <!-- Controls -->
          <div v-if="editingId !== comment._id" class="tw-mt-0.5 tw-flex tw-gap-3">
            <a
              v-if="canEditComment(comment)"
              class="tw-text-xs tw-text-brass"
              @click="startEdit(comment)"
              >Edit</a
            >
            <a
              v-if="canDeleteComment(comment)"
              class="tw-text-xs tw-text-red"
              @click="remove(comment)"
              >Delete</a
            >
          </div>
        </div>
      </div>
    </div>
    <div v-else class="tw-text-sm tw-text-parchment-dim">
      No messages yet — start the conversation.
    </div>

    <!-- Composer -->
    <div class="tw-mt-3 tw-border-t tw-border-brass-dim tw-pt-3">
      <v-text-field
        v-if="!authUser"
        v-model="guestName"
        label="Your name"
        dense
        hide-details
        class="tw-mb-2 tw-max-w-xs"
      />
      <div class="tw-flex tw-items-end tw-gap-2">
        <v-textarea
          v-model="newText"
          placeholder="Add a message…"
          dense
          hide-details
          auto-grow
          :rows="1"
          class="tw-flex-grow tw-text-sm"
        ></v-textarea>
        <v-btn
          small
          class="tw-bg-brass tw-text-wood-deep"
          :disabled="!canPost"
          @click="submit"
          >Post</v-btn
        >
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex"
import dayjs from "dayjs"

/**
 * Event discussion thread (C7). Presentational: reads event.comments, emits
 * add-comment / edit-comment / delete-comment for Event.vue to persist +
 * refresh. Members post directly; guests post with a name (same trust model as
 * RSVP). You can edit/delete your own; the event owner can delete any.
 */
export default {
  name: "EventComments",

  props: {
    event: { type: Object, required: true },
  },

  data: () => ({
    newText: "",
    guestName: "",
    editingId: null,
    editText: "",
  }),

  emits: ["add-comment", "edit-comment", "delete-comment"],

  computed: {
    ...mapState(["authUser"]),
    comments() {
      return this.event.comments ?? []
    },
    myName() {
      return this.guestName.trim()
    },
    isEventOwner() {
      return (
        !!this.authUser &&
        !!this.event.ownerId &&
        this.event.ownerId !== 0 &&
        this.authUser._id === this.event.ownerId
      )
    },
    canPost() {
      return this.newText.trim().length > 0 && (this.authUser || this.myName.length > 0)
    },
  },

  methods: {
    formatTime(dt) {
      return dayjs(dt).format("MMM D, h:mm A")
    },
    isMine(comment) {
      if (comment.isGuest) {
        return this.myName.length > 0 && comment.userId === this.myName
      }
      return !!this.authUser && comment.userId === this.authUser._id
    },
    canEditComment(comment) {
      return this.isMine(comment)
    },
    canDeleteComment(comment) {
      return this.isMine(comment) || this.isEventOwner
    },
    // Identity to send: act as the comment's guest author if it's my own guest
    // comment, otherwise as my signed-in self (author or owner).
    identityFor(comment) {
      if (comment.isGuest && this.isMine(comment)) {
        return { guest: true, name: comment.userId }
      }
      return { guest: false }
    },
    submit() {
      const text = this.newText.trim()
      if (!text) return
      const payload = this.authUser
        ? { text, guest: false }
        : { text, guest: true, name: this.myName }
      this.$emit("add-comment", payload)
      this.newText = ""
    },
    startEdit(comment) {
      this.editingId = comment._id
      this.editText = comment.text
    },
    cancelEdit() {
      this.editingId = null
      this.editText = ""
    },
    saveEdit(comment) {
      const text = this.editText.trim()
      if (!text) return
      this.$emit("edit-comment", {
        commentId: comment._id,
        payload: { text, ...this.identityFor(comment) },
      })
      this.cancelEdit()
    },
    remove(comment) {
      this.$emit("delete-comment", {
        commentId: comment._id,
        payload: this.identityFor(comment),
      })
    },
  },
}
</script>
