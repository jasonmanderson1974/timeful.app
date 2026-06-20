// PostHog analytics has been removed from this self-hosted instance.
// This stub keeps `this.$posthog` defined so the existing call sites
// (capture/identify/get_distinct_id/reset/etc.) become harmless no-ops
// and nothing is sent to any analytics backend.
const noop = () => {}
const posthogStub = new Proxy(
  {},
  {
    get() {
      return noop
    },
  }
)

export default {
  install(Vue) {
    Vue.prototype.$posthog = posthogStub
  },
}
