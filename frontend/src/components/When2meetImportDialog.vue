<template>
  <v-dialog v-model="dialog" max-width="600px" content-class="tw-m-0">
    <v-card>
      <v-card-title>
        <span class="tw-text-xl tw-font-medium"
          >Convert When2meet to a Gathering</span
        >
        <v-spacer />
        <v-btn
          absolute
          @click="$emit('input', false)"
          icon
          class="tw-right-0 tw-mr-2 tw-self-center"
        >
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </v-card-title>
      <v-card-text class="tw-text-parchment-dim">
        <p class="tw-mb-4">
          If someone accidentally sends you a When2meet link instead of a
          Gathering, you can convert it by following these steps:
        </p>
        <ol class="tw-mb-6 tw-list-decimal tw-pl-5">
          <li>Go to your When2meet event link.</li>
          <li>Add "better" to the beginning of the URL.</li>
          <li>🪄💫✨</li>
          <li>Send link back to the group chat.</li>
        </ol>
        <p>
          That's it! The date range, time range, and ALL THE EXISTING RESPONSES
          will be copied over as well.
        </p>

      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" text @click="dialog = false"> Got it! </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
export default {
  name: "When2meetImportDialog",
  props: {
    value: Boolean, // v-model
  },
  computed: {
    dialog: {
      get() {
        return this.value
      },
      set(val) {
        this.$emit("input", val)
      },
    },
  },
}
</script>

<style scoped>
.video-container {
  position: relative;
  padding-bottom: 56.25%; /* 16:9 aspect ratio */
  height: 0;
  overflow: hidden;
}

.video-container iframe {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}
</style>
