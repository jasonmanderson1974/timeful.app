<template>
  <div
    class="tw-flex tw-h-full tw-flex-col tw-items-center tw-justify-center tw-p-2"
  >
    <SirThomasFoolery :size="160" class="tw-mb-6" />
    <div class="tw-mb-8 tw-max-w-[30rem]">
      <h1 class="tw-mb-2 tw-text-center tw-text-2xl tw-font-medium">
        You need access
      </h1>
      <div class="tw-text-center tw-text-parchment-dim">
        Ask the group owner to share this group with your email: <br />
        <span class="tw-font-medium tw-text-brass">{{ authUser.email }}</span>
      </div>
    </div>
    <v-btn href="/home" color="primary">Back to home</v-btn>
  </div>
</template>

<script>
import { mapState } from "vuex"
import SirThomasFoolery from "@/components/general/SirThomasFoolery.vue"

export default {
  name: "AccessDenied",

  components: { SirThomasFoolery },

  computed: {
    ...mapState(["authUser"]),
  },
}
</script>
