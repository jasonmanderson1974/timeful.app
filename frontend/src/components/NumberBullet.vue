<!-- Displays a number within a brass medallion -->
<template>
  <div
    class="tw-grid tw-h-7 tw-w-7 tw-shrink-0 tw-place-content-center tw-rounded-full tw-bg-brass tw-font-display tw-text-sm tw-font-semibold tw-text-wood-deep"
  >
    <slot></slot>
  </div>
</template>

<script>
export default {
  name: "NumberBullet",
}
</script>
