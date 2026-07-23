<template>
  <div>
    <!-- Sign up block being dragged -->
    <div v-if="dragging">
      <div
        class="tw-absolute tw-w-full tw-select-none tw-p-px"
        :style="draggedBlockStyle"
        style="pointer-events: none"
      >
        <SignUpCalendarBlock :title="draggedBlockName" titleOnly unsaved />
      </div>
    </div>

    <div v-if="isSignUp">
      <!-- Sign up blocks -->
      <div v-for="block in blocks" :key="block._id">
        <div
          class="tw-absolute tw-w-full tw-select-none tw-p-px"
          :style="{
            top: `calc(${block.hoursOffset} * 4 * 1rem)`,
            height: `calc(${block.hoursLength} * 4 * 1rem)`,
          }"
          @click="$emit('block-click', block)"
        >
          <SignUpCalendarBlock :signUpBlock="block" />
        </div>
      </div>

      <!-- Sign up blocks to be added after hitting 'save' -->
      <div v-for="block in blocksToAdd" :key="block._id">
        <div
          class="tw-absolute tw-w-full tw-select-none tw-p-px"
          :style="{
            top: `calc(${block.hoursOffset} * 4 * 1rem)`,
            height: `calc(${block.hoursLength} * 4 * 1rem)`,
          }"
        >
          <SignUpCalendarBlock :title="block.name" titleOnly unsaved />
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import SignUpCalendarBlock from "@/components/sign_up_form/SignUpCalendarBlock.vue"

/**
 * Per-day-column overlay rendering sign-up blocks on the ScheduleOverlap grid:
 * the block being dragged out (while editing), saved blocks (clickable to sign
 * up), and unsaved blocks-to-add. Extracted from ScheduleOverlap.vue (TODO A5,
 * Tier 2) — purely presentational; all state stays in ScheduleOverlap, which
 * passes the current day's slices and handles `block-click`.
 */
export default {
  name: "SignUpBlocksOverlay",

  components: {
    SignUpCalendarBlock,
  },

  props: {
    /** Whether a new block is currently being dragged out in this column */
    dragging: { type: Boolean, default: false },
    draggedBlockStyle: { type: Object, default: () => ({}) },
    draggedBlockName: { type: String, default: "" },
    /** Whether this event is a sign-up form */
    isSignUp: { type: Boolean, default: false },
    /** Saved sign-up blocks for this day column */
    blocks: { type: Array, default: () => [] },
    /** Unsaved blocks-to-add for this day column */
    blocksToAdd: { type: Array, default: () => [] },
  },
}
</script>
