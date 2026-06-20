<template>
  <!-- Sir Thomas Foolery — the monocled, mustachioed hare crest of The Fellowship -->
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 120 120"
    role="img"
    aria-label="Sir Thomas Foolery, the monocled hare"
    class="flw-crest"
  >
    <defs>
      <radialGradient :id="gid" cx="50%" cy="40%" r="65%">
        <stop offset="0%" stop-color="#2e2117" />
        <stop offset="100%" stop-color="#160f0a" />
      </radialGradient>
    </defs>
    <circle
      cx="60"
      cy="60"
      r="56"
      :fill="`url(#${gid})`"
      stroke="#c9a44c"
      stroke-width="2"
    />
    <circle cx="60" cy="60" r="50" fill="none" stroke="#8a7333" stroke-width="1" />
    <!-- ears -->
    <g fill="none" stroke="#e3c578" stroke-width="2.4" stroke-linecap="round">
      <path d="M50 44 Q44 18 49 12" />
      <path d="M70 44 Q76 18 71 12" />
    </g>
    <!-- head -->
    <ellipse
      cx="60"
      cy="62"
      rx="20"
      ry="22"
      fill="none"
      stroke="#e3c578"
      stroke-width="2.4"
    />
    <!-- monocle (right eye) -->
    <circle cx="68" cy="58" r="7.5" fill="none" stroke="#e3c578" stroke-width="2" />
    <line x1="74" y1="63" x2="79" y2="74" stroke="#e3c578" stroke-width="1.5" />
    <!-- eyes -->
    <circle cx="52" cy="58" r="1.8" fill="#e3c578" />
    <circle cx="68" cy="58" r="1.8" fill="#e3c578" />
    <!-- handlebar mustache -->
    <path
      d="M60 70 Q52 72 46 68 Q42 65 44 71 Q47 76 54 73 Q58 71 60 73 Q62 71 66 73 Q73 76 76 71 Q78 65 74 68 Q68 72 60 70 Z"
      fill="#e3c578"
    />
    <!-- nose -->
    <circle cx="60" cy="67" r="1.6" fill="#e3c578" />
  </svg>
</template>

<script>
let _uid = 0

export default {
  name: "SirThomasFoolery",

  props: {
    size: { type: [Number, String], default: 116 },
  },

  data() {
    // unique gradient id so multiple crests on a page don't collide
    return { gid: `stf-seal-${++_uid}` }
  },
}
</script>

<style scoped>
.flw-crest {
  display: block;
  filter: drop-shadow(0 6px 14px rgba(0, 0, 0, 0.6));
}
</style>
