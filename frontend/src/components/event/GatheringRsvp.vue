<template>
  <div
    class="tw-mt-3 tw-rounded-md tw-border tw-border-brass-dim tw-bg-leather tw-p-3 tw-text-parchment sm:tw-p-4"
  >
    <div class="tw-flex tw-flex-wrap tw-items-center tw-gap-x-3 tw-gap-y-1">
      <div class="tw-text-base tw-font-medium">Who's coming?</div>
      <div class="tw-text-sm tw-text-parchment-dim">
        {{ counts.going }} going · {{ counts.maybe }} maybe ·
        {{ counts.no }} can't
      </div>
    </div>

    <!-- Guest name (only when not signed in) -->
    <v-text-field
      v-if="!authUser"
      v-model="guestName"
      label="Your name"
      dense
      hide-details
      class="tw-mt-2 tw-max-w-xs"
    />

    <!-- RSVP buttons -->
    <div class="tw-mt-3 tw-flex tw-flex-wrap tw-gap-2">
      <v-btn
        v-for="opt in options"
        :key="opt.value"
        small
        :outlined="myStatus !== opt.value"
        :disabled="!authUser && !guestName.trim()"
        :class="
          myStatus === opt.value
            ? 'tw-bg-brass tw-text-wood-deep'
            : 'tw-text-brass'
        "
        @click="choose(opt.value)"
      >
        <v-icon small left>{{ opt.icon }}</v-icon>
        {{ opt.label }}
      </v-btn>
    </div>

    <!-- Roster grouped by status -->
    <div v-if="hasAnyRsvp" class="tw-mt-3 tw-space-y-1 tw-text-sm">
      <div v-for="opt in options" :key="`roster-${opt.value}`">
        <template v-if="rosters[opt.value].length">
          <span class="tw-font-medium">{{ opt.label }}:</span>
          <span class="tw-text-parchment-dim">
            {{ rosters[opt.value].join(", ") }}
          </span>
        </template>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex"

/**
 * RSVP widget for a confirmed gathering (shown when event.scheduledEvent
 * exists). Presentational: reads event.rsvps, emits set-rsvp / clear-rsvp for
 * Event.vue to persist. Signed-in users RSVP directly; guests enter a name
 * first (same trust model as guest availability).
 */
export default {
  name: "GatheringRsvp",

  props: {
    event: { type: Object, required: true },
  },

  data: () => ({
    guestName: "",
    options: [
      { value: "going", label: "Going", icon: "mdi-check" },
      { value: "maybe", label: "Maybe", icon: "mdi-help" },
      { value: "no", label: "Can't make it", icon: "mdi-close" },
    ],
  }),

  emits: ["set-rsvp", "clear-rsvp"],

  computed: {
    ...mapState(["authUser"]),
    rsvps() {
      return this.event.rsvps ?? {}
    },
    hasAnyRsvp() {
      return Object.keys(this.rsvps).length > 0
    },
    counts() {
      const c = { going: 0, maybe: 0, no: 0 }
      for (const r of Object.values(this.rsvps)) {
        if (r && c[r.status] !== undefined) c[r.status]++
      }
      return c
    },
    rosters() {
      const r = { going: [], maybe: [], no: [] }
      for (const [key, rsvp] of Object.entries(this.rsvps)) {
        if (rsvp && r[rsvp.status]) r[rsvp.status].push(rsvp.name || key)
      }
      return r
    },
    // The map key identifying the current viewer, if we can determine one.
    myKey() {
      if (this.authUser) return this.authUser._id
      const name = this.guestName.trim()
      return name.length > 0 ? name : null
    },
    myStatus() {
      const entry = this.myKey ? this.rsvps[this.myKey] : null
      return entry?.status ?? null
    },
  },

  methods: {
    choose(status) {
      // Clicking the active choice again clears the RSVP.
      if (this.myStatus === status) {
        this.clear()
        return
      }
      if (this.authUser) {
        this.$emit("set-rsvp", { status, guest: false })
      } else {
        const name = this.guestName.trim()
        if (!name) return
        this.$emit("set-rsvp", { status, guest: true, name })
      }
    },
    clear() {
      if (this.authUser) {
        this.$emit("clear-rsvp", { guest: false })
      } else {
        const name = this.guestName.trim()
        if (!name) return
        this.$emit("clear-rsvp", { guest: true, name })
      }
    },
  },
}
</script>
