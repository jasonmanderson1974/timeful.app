<template>
  <footer
    class="tw-border-t tw-border-brass-dim tw-bg-wood-deep tw-px-8 tw-py-12 tw-text-parchment-dim"
  >
    <div
      class="tw-mx-auto tw-flex tw-max-w-5xl tw-flex-col tw-items-center tw-gap-4 tw-text-center"
    >
      <SirThomasFoolery :size="60" />
      <div
        class="tw-font-display tw-text-sm tw-font-bold tw-tracking-[0.16em] tw-text-parchment"
      >
        THE FELLOWSHIP
      </div>
      <div class="flw-rule tw-my-1"><span>&#9670;</span></div>
      <p class="flw-sub tw-text-base">
        Presided over by
        <span class="tw-font-display tw-not-italic tw-text-brass"
          >Sir Thomas Foolery</span
        >, Esq.
        <br />
        Eric the Hare, of the Eric &amp; Eloise Collection
      </p>
    </div>
  </footer>
</template>

<script>
import SirThomasFoolery from "@/components/general/SirThomasFoolery.vue"

export default {
  name: "Footer",
  components: { SirThomasFoolery },
}
</script>
