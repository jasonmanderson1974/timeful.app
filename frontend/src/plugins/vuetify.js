import Vue from "vue"
import Vuetify from "vuetify/lib"
import tailwind from "../../tailwind.config"

Vue.use(Vuetify)

const c = tailwind.theme.colors

export default new Vuetify({
  // The Fellowship — dark vintage gentleman's-club theme
  theme: {
    dark: true,
    options: { customProperties: true },
    themes: {
      dark: {
        primary: c.brass,
        secondary: c["green-felt"],
        accent: c["brass-bright"],
        error: c.oxblood,
        info: c.brass,
        success: c.brass,
        warning: c.brass,
        background: c["wood-deep"],
        surface: c.leather,
      },
      light: {
        primary: c.brass,
        error: c.oxblood,
      },
    },
  },
  breakpoint: {
    thresholds: {
      xs: 640,
      sm: 768,
      md: 1024,
      lg: 1280,
    },
    scrollBarWidth: 0,
  },
})
