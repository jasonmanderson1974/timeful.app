<template>
  <div class="tw-mx-auto tw-mb-12 tw-mt-5 tw-max-w-3xl">
    <div class="tw-flex tw-flex-col tw-gap-8 tw-p-4">
      <!-- Header -->
      <div class="tw-flex tw-flex-col tw-gap-1">
        <div class="tw-font-head tw-text-2xl tw-text-parchment sm:tw-text-3xl">
          The Roll
        </div>
        <div class="tw-text-sm tw-text-parchment-dim">
          Only those on the roll may enter the Fellowship. Add a gentleman's
          email to extend an invitation; strike it to revoke.
        </div>
      </div>

      <!-- Invite form -->
      <div
        class="tw-rounded-xl tw-border tw-border-brass-dim tw-bg-leather/60 tw-p-4"
      >
        <div class="tw-mb-1 tw-text-sm tw-font-medium tw-text-parchment">
          Invite a member
        </div>
        <div class="tw-flex tw-flex-col tw-gap-2 sm:tw-flex-row">
          <v-text-field
            v-model="email"
            class="tw-flex-1"
            placeholder="name@example.com"
            type="email"
            solo
            hide-details="auto"
            :error-messages="emailError"
            :disabled="adding"
            @keydown.enter="invite"
          />
          <v-btn
            color="primary"
            :loading="adding"
            :disabled="adding"
            @click="invite"
          >
            Extend invitation
          </v-btn>
        </div>
      </div>

      <!-- Roll -->
      <div class="tw-flex tw-flex-col tw-gap-3">
        <div class="tw-flex tw-items-center tw-justify-between">
          <div class="tw-text-lg tw-font-medium tw-text-parchment">
            On the roll
            <span class="tw-text-parchment-dim">({{ members.length }})</span>
          </div>
        </div>

        <div v-if="loading" class="tw-py-8 tw-text-center tw-text-parchment-dim">
          <v-progress-circular indeterminate color="brass" size="24" />
        </div>

        <div
          v-else-if="members.length === 0"
          class="tw-rounded-xl tw-border tw-border-brass-dim tw-bg-leather/40 tw-py-8 tw-text-center tw-text-sm tw-text-parchment-dim"
        >
          The roll is empty — the gate stands open until the first member is
          added.
        </div>

        <div
          v-for="member in members"
          :key="member.email"
          class="tw-flex tw-items-center tw-gap-3 tw-rounded-xl tw-border tw-border-brass-dim tw-bg-leather/40 tw-p-3"
        >
          <div class="tw-min-w-0 tw-flex-1">
            <div class="tw-truncate tw-font-medium tw-text-parchment">
              <span v-if="member.hasAccount">
                {{ member.firstName }} {{ member.lastName }}
              </span>
              <span v-else class="tw-italic tw-text-parchment-dim">
                Awaiting first entry
              </span>
            </div>
            <div class="tw-truncate tw-text-sm tw-text-parchment-dim">
              {{ member.email }}
            </div>
          </div>

          <!-- Status badge -->
          <span
            class="tw-shrink-0 tw-rounded-full tw-border tw-px-2 tw-py-0.5 tw-text-xs"
            :class="
              member.hasAccount
                ? 'tw-border-brass tw-text-brass'
                : 'tw-border-brass-dim tw-text-parchment-dim'
            "
          >
            {{ member.hasAccount ? "Member" : "Invited" }}
          </span>

          <!-- Inviter toggle (only meaningful once they have an account) -->
          <v-tooltip bottom>
            <template v-slot:activator="{ on }">
              <div v-on="on" class="tw-flex tw-shrink-0 tw-items-center">
                <v-switch
                  :input-value="member.canInvite"
                  :disabled="!member.hasAccount || busyEmail === member.email"
                  color="brass"
                  hide-details
                  dense
                  class="tw-mt-0 tw-pt-0"
                  @change="toggleCanInvite(member, $event)"
                />
                <span class="tw-hidden tw-text-xs tw-text-parchment-dim sm:tw-inline">
                  Inviter
                </span>
              </div>
            </template>
            <span>
              {{
                member.hasAccount
                  ? "Allow this member to manage the roll"
                  : "Available once they have signed in"
              }}
            </span>
          </v-tooltip>

          <!-- Remove -->
          <v-btn
            icon
            small
            :disabled="busyEmail === member.email || member.email === selfEmail"
            :title="
              member.email === selfEmail
                ? 'You cannot strike yourself from the roll'
                : 'Strike from the roll'
            "
            @click="remove(member)"
          >
            <v-icon small color="oxblood">mdi-close</v-icon>
          </v-btn>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState, mapActions } from "vuex"
import { get, post, _delete } from "@/utils"

export default {
  name: "MemberAdmin",

  metaInfo() {
    return { title: "The Roll · The Fellowship" }
  },

  data() {
    return {
      email: "",
      emailError: "",
      members: [],
      loading: true,
      adding: false,
      busyEmail: "",
    }
  },

  computed: {
    ...mapState(["authUser"]),
    selfEmail() {
      return (this.authUser?.email || "").toLowerCase()
    },
  },

  async created() {
    // Client-side guard; the /admin endpoints enforce this server-side too.
    if (!this.authUser || !this.authUser.canInvite) {
      this.$router.replace({ name: "home" })
      return
    }
    await this.fetchAllowlist()
  },

  methods: {
    ...mapActions(["showError", "showInfo"]),
    async fetchAllowlist() {
      this.loading = true
      try {
        this.members = await get("/admin/allowlist")
      } catch (err) {
        this.showError("Could not load the roll. Please try again.")
      } finally {
        this.loading = false
      }
    },
    validateEmail() {
      const email = this.email.trim()
      if (!email) {
        this.emailError = "Please enter an email address."
        return false
      }
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
        this.emailError = "Please enter a valid email address."
        return false
      }
      if (email.includes("+")) {
        this.emailError = "Email aliases with '+' are not allowed."
        return false
      }
      return true
    },
    async invite() {
      if (this.adding) return
      this.emailError = ""
      if (!this.validateEmail()) return
      this.adding = true
      try {
        await post("/admin/allowlist", { email: this.email.trim() })
        this.email = ""
        await this.fetchAllowlist()
        this.showInfo("Invitation extended.")
      } catch (err) {
        this.emailError = "Could not add that email. Please try again."
      } finally {
        this.adding = false
      }
    },
    async remove(member) {
      if (this.busyEmail) return
      if (
        !window.confirm(
          `Strike ${member.email} from the roll? They will lose access to the Fellowship.`
        )
      ) {
        return
      }
      this.busyEmail = member.email
      try {
        await _delete("/admin/allowlist", { email: member.email })
        await this.fetchAllowlist()
      } catch (err) {
        this.showError("Could not remove that email. Please try again.")
      } finally {
        this.busyEmail = ""
      }
    },
    async toggleCanInvite(member, value) {
      if (this.busyEmail) return
      this.busyEmail = member.email
      try {
        await post("/admin/member/can-invite", {
          email: member.email,
          canInvite: value,
        })
        await this.fetchAllowlist()
      } catch (err) {
        this.showError("Could not update inviter status. Please try again.")
        // Revert the optimistic switch position by reloading.
        await this.fetchAllowlist()
      } finally {
        this.busyEmail = ""
      }
    },
  },
}
</script>
