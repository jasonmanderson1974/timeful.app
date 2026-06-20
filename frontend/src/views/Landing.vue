<template>
  <div class="tw-bg-transparent">
    <div
      class="tw-relative tw-m-auto tw-mb-12 tw-flex tw-max-w-6xl tw-flex-col tw-px-4 sm:tw-mb-20"
    >
      <!-- Header -->
      <div class="tw-mb-12 sm:tw-mb-20">
        <div class="tw-flex tw-items-center tw-pt-5">
          <router-link
            :to="{ name: 'landing' }"
            class="tw-flex tw-items-center tw-gap-3 tw-no-underline"
          >
            <SirThomasFoolery :size="44" />
            <span
              class="tw-font-display tw-text-lg tw-font-bold tw-tracking-[0.16em] tw-text-parchment sm:tw-text-xl"
              >THE FELLOWSHIP</span
            >
          </router-link>

          <v-spacer />

          <LandingPageHeader>
            <v-btn
              text
              class="tw-font-display tw-tracking-widest tw-text-brass"
              @click="openHowItWorksDialog"
              >The Manner of It</v-btn
            >
            <v-btn
              text
              class="tw-font-display tw-tracking-widest tw-text-brass"
              href="/blog"
              >The Chronicle</v-btn
            >
            <div v-if="authUser" class="tw-ml-2">
              <AuthUserMenu />
            </div>
            <v-btn
              v-else
              text
              class="tw-font-display tw-tracking-widest tw-text-brass"
              :to="{ name: 'sign-in' }"
              >Enter</v-btn
            >
          </LandingPageHeader>
        </div>
      </div>

      <div class="tw-flex tw-flex-col tw-items-center">
        <header class="tw-mb-8 tw-flex tw-flex-col tw-items-center tw-text-center">
          <SirThomasFoolery :size="104" class="tw-mb-4" />
          <p class="flw-eyebrow tw-mb-2">The Fellowship Presents</p>
          <h1
            id="header"
            class="flw-title tw-text-5xl sm:tw-text-6xl xl:tw-text-7xl"
          >
            The Gathering
          </h1>
          <div class="flw-rule tw-mt-4"><span>&#9670;</span></div>
          <p class="flw-sub tw-mt-5 tw-text-xl sm:tw-text-2xl">
            Cast thy vote, good sir &mdash; when shall we convene?
          </p>
        </header>

        <div class="tw-mb-12 tw-flex tw-flex-col tw-items-center tw-gap-2">
          <button
            class="flw-btn tw-text-sm sm:tw-text-base"
            @click="authUser ? openDashboard() : (newDialog = true)"
          >
            {{ authUser ? "To the Club Room" : "Call a Gathering" }}
          </button>
          <div
            v-if="!authUser"
            class="tw-text-center tw-text-sm tw-text-parchment-dim"
          >
            No dues &middot; no login required.
          </div>
        </div>
        <div class="tw-relative tw-w-full">
          <!-- Felt-green spotlight behind the club portrait -->
          <div
            class="tw-absolute -tw-bottom-12 tw-left-1/2 tw-h-[85%] tw-w-screen -tw-translate-x-1/2 tw-bg-green-felt sm:-tw-bottom-20"
          ></div>

          <!-- Hero portrait, framed in brass -->
          <div
            class="flw-panel tw-relative tw-z-20 tw-w-full tw-p-2 sm:tw-p-3 md:tw-mx-auto md:tw-w-fit"
          >
            <div
              class="tw-relative tw-mx-4 tw-aspect-square md:tw-size-[700px] lg:tw-size-[800px]"
            >
              <v-img
                class="tw-absolute tw-left-0 tw-top-0 tw-z-20 tw-size-full tw-transition-opacity tw-duration-300"
                :class="{ 'tw-opacity-0': isVideoPlaying }"
                src="@/assets/img/hero.jpg"
                transition="fade-transition"
                contain
              />
              <vue-vimeo-player
                video-url="https://player.vimeo.com/video/1083205305?h=d58bef862a"
                :player-width="800"
                :player-height="800"
                :options="{
                  muted: true,
                  playsinline: true,
                  responsive: true,
                }"
                :controls="false"
                :autoplay="true"
                :loop="true"
                @play="onPlay"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- How a Gathering comes to pass -->
    <div
      id="how-it-works"
      class="tw-grid tw-place-content-center tw-px-4 tw-pt-16"
    >
      <div class="tw-mb-8 tw-text-center">
        <p class="flw-eyebrow tw-mb-2">The Order of Things</p>
        <h2 class="flw-title tw-text-3xl sm:tw-text-4xl">
          How a Gathering Comes to Pass
        </h2>
        <div class="flw-rule tw-mt-4"><span>&#9670;</span></div>
      </div>
      <div class="tw-mx-auto tw-flex tw-max-w-xl tw-flex-col tw-gap-4">
        <div
          v-for="(step, i) in howItWorksSteps"
          :key="i"
          class="tw-flex tw-items-center tw-gap-3"
        >
          <NumberBullet>{{ i + 1 }}</NumberBullet>
          <div class="tw-font-head tw-text-lg tw-text-parchment md:tw-text-xl">
            <div v-html="step"></div>
          </div>
        </div>
      </div>
      <div
        class="flw-sub tw-mb-6 tw-mt-12 tw-text-center tw-text-3xl tw-text-brass md:tw-mb-12 md:tw-mt-16 md:tw-text-5xl"
      >
        And so, &rsquo;tis done.
      </div>
      <SirThomasFoolery
        :size="isPhone ? 150 : 210"
        class="tw-mx-auto -tw-mb-12"
      />
    </div>

    <!-- The proceedings, in motion -->
    <div
      class="tw-flex tw-justify-center tw-bg-green-felt tw-px-4 tw-pb-12 tw-pt-24 md:tw-pb-16"
    >
      <div class="flw-panel tw-max-w-3xl tw-flex-1 tw-p-2 sm:tw-p-3">
        <div class="tw-h-[300px] sm:tw-h-[400px] md:tw-h-[450px]">
          <iframe
            class="tw-h-full tw-w-full tw-rounded"
            src="https://www.youtube.com/embed/vFkBC8BrkOk?si=pF64JAIyDhom_1do"
            title="The Fellowship demo"
            frameborder="0"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerpolicy="strict-origin-when-cross-origin"
            allowfullscreen
          ></iframe>
        </div>
      </div>
    </div>

    <!-- FAQ -->
    <div class="tw-flex tw-justify-center tw-pt-12">
      <div class="tw-mx-4 tw-mb-12 tw-max-w-3xl tw-flex-1 sm:tw-mx-16">
        <div id="faq-section" class="tw-text-center lg:tw-pt-3">
          <p class="flw-eyebrow tw-mb-2">Should You Wonder</p>
          <h2 class="flw-title tw-text-3xl sm:tw-text-4xl">
            Matters of Procedure
          </h2>
          <div class="flw-rule tw-mb-8 tw-mt-4"><span>&#9670;</span></div>
          <div
            class="tw-grid tw-grid-cols-1 tw-gap-3 tw-text-left sm:tw-text-xl lg:tw-text-2xl"
          >
            <FAQ
              v-for="faq in faqs"
              :key="faq.question"
              @signIn="signIn"
              v-bind="faq"
            />
          </div>
        </div>
      </div>
    </div>

    <Footer />

    <!-- Sign in dialog -->
    <SignInDialog
      v-model="signInDialog"
      @signIn="_signIn"
      @emailSignIn="_emailSignIn"
    />

    <!-- New event dialog -->
    <NewDialog v-model="newDialog" no-tabs @signIn="signIn" />

    <!-- Add the dialog component -->
    <HowItWorksDialog
      v-if="showHowItWorksDialog"
      v-model="showHowItWorksDialog"
    />
  </div>
</template>

<style scoped>
@media screen and (min-width: 375px) and (max-width: 640px) {
  #header {
    font-size: 1.875rem !important; /* 30px */
    line-height: 2.25rem !important; /* 36px */
  }
}
</style>
<style>
.rdt-h {
  @apply tw-rounded tw-bg-light-green/20 tw-px-px tw-text-black;
}
</style>

<script>
import LandingPageCalendar from "@/components/landing/LandingPageCalendar.vue"
import { isPhone, signInGoogle, signInOutlook } from "@/utils"
import FAQ from "@/components/FAQ.vue"
import Header from "@/components/Header.vue"
import NumberBullet from "@/components/NumberBullet.vue"
import NewEvent from "@/components/NewEvent.vue"
import NewDialog from "@/components/NewDialog.vue"
import LandingPageHeader from "@/components/landing/LandingPageHeader.vue"
import Logo from "@/components/Logo.vue"
import GithubButton from "vue-github-button"
import SignInDialog from "@/components/SignInDialog.vue"
import { calendarTypes } from "@/constants"
import HowItWorksDialog from "@/components/HowItWorksDialog.vue"
import { vueVimeoPlayer } from "vue-vimeo-player"
import Footer from "@/components/Footer.vue"
import PronunciationMenu from "@/components/PronunciationMenu.vue"
import { mapState, mapMutations } from "vuex"
import AuthUserMenu from "@/components/AuthUserMenu.vue"
import FormerlyKnownAs from "@/components/FormerlyKnownAs.vue"
import SirThomasFoolery from "@/components/general/SirThomasFoolery.vue"

export default {
  name: "Landing",

  metaInfo: {
    title: "The Fellowship · The Gathering",
  },

  components: {
    SirThomasFoolery,
    LandingPageCalendar,
    FAQ,
    Header,
    NumberBullet,
    NewEvent,
    NewDialog,
    LandingPageHeader,
    GithubButton,
    Logo,
    SignInDialog,
    HowItWorksDialog,
    vueVimeoPlayer,
    Footer,
    PronunciationMenu,
    AuthUserMenu,
    FormerlyKnownAs,
  },

  data: () => ({
    signInDialog: false,
    newDialog: false,
    githubSnackbar: true,
    howItWorksSteps: [
      "Call a Gathering and propose the candidate evenings",
      "Circulate the summons, that each man may mark his availability",
      "Behold where the whole Order's free hours align &mdash; and set the date",
    ],
    faqs: [
      {
        question: "Does the Fellowship account for timezones?",
        answer:
          "Indeed, good sir. All hours are rendered in each viewer's own timezone automatically. Should you wish to set it yourself, a timezone selector awaits at the foot of every Gathering.",
      },
      {
        question: "How many men may answer a summons?",
        answer:
          "As many as you please — we have tested Gatherings with upwards of 500 respondents, and it holds firm.",
      },
      {
        question: "Which calendars may I consult?",
        answer:
          "You may draw your availability from Google Calendar, Outlook, Apple Calendar, or an ICS feed URL. Further provisions are forthcoming.",
      },
      {
        question: "Must I grant calendar access to take part?",
        answer:
          "Not at all. You may enter your availability by hand — though we heartily recommend granting access, that you may consult your engagements as you mark them.",
      },
      {
        question: "Will others spy upon my calendar?",
        answer:
          "Never. Your fellows see only the availability you choose to enter for a given Gathering — nothing more.",
      },
      {
        question: "How do I amend my availability?",
        answer:
          'If you are signed in, simply press the "Edit availability" button. Should you have answered as a guest, hover upon your name and press the pencil beside it.',
      },
      {
        question: "What sets the Fellowship apart from lesser tools?",
        points: [
          "A far more handsome interface, on desk and in pocket alike",
          "Seamless, dependable calendar integration",
          "A great many further conveniences too numerous to set down here",
        ],
      },
      {
        question: `I should like only myself to see the replies.`,
        answer: `But of course — tick "Only show responses to event creator" under Advanced Options when calling your Gathering, and your fellows shall see neither names nor availability of one another.`,
        authRequired: true,
      },
      {
        question: `May I be notified when a man answers?`,
        answer: `Most certainly. Tick "Email me each time someone joins my event" when calling a Gathering. <br><br>For word only after a certain number (X) of replies, tick "Email me after X responses" under Advanced Options.`,
        authRequired: true,
      },
      {
        question: `How do I prompt the laggards to reply?`,
        answer: `Open the "Email Reminders" section when calling your Gathering and enter each man's address. Reminders are dispatched on the day of creation, the day following, and three days hence. <br><br>You shall also be notified once the whole Order has answered.`,
        authRequired: true,
      },
    ],
    redditComments: [
      {
        text: "Genuinely the <span class='rdt-h'>best lightweight version of this kind of website</span> that I've come across so far, exceptional.",
        author: "u/voipClock",
        link: "https://www.reddit.com/r/opensource/comments/1klu471/comment/mt4l2ab",
        picture:
          "https://www.redditstatic.com/avatars/defaults/v2/avatar_default_1.png",
      },
      {
        text: "It's almost <span class='rdt-h'>comically easy</span> to schedule meetings with Timeful.",
        author: "u/stuffingmybrain",
        link: "https://www.reddit.com/r/schej/comments/1drs26z/comment/lb8rvty",
        picture:
          "https://styles.redditmedia.com/t5_qqojf/styles/profileIcon_snooa54a8eae-bc7f-406f-9778-b3b9dfb818e5-headshot.png?width=64&height=64&frame=1&auto=webp&crop=&s=a0a91575ff7cfc3b6698cac69da6c012c7deb8d6",
      },
      {
        text: "Timeful is everything I've ever wanted and more. On top of that, <span class='rdt-h'>community support is the best I've seen</span> of any app or software, ever.",
        author: "u/DMODD",
        link: "https://www.reddit.com/r/schej/comments/1drs26z/comment/lb8udud",
        picture:
          "https://www.redditstatic.com/avatars/defaults/v2/avatar_default_6.png",
      },
      {
        text: "With Timeful, <span class='rdt-h'>I'm very quickly able to figure out the optimal time</span> to schedule online extra help sessions before an exam.",
        author: "u/crackwurst",
        link: "https://www.reddit.com/r/schej/comments/1drs26z/comment/lb9dmbe",
        picture:
          "https://www.redditstatic.com/avatars/defaults/v2/avatar_default_3.png",
      },
      {
        text: "Exactly what I was looking for! Clear and clean interface, also on mobile (<span class='rdt-h'>Doodle is a disaster</span>).",
        author: "u/Willem1976",
        link: "https://www.reddit.com/r/opensource/comments/1dlol7r/comment/lkn7sle",
        picture:
          "https://styles.redditmedia.com/t5_c0qtc/styles/profileIcon_snooa9d429ce-e3d9-458a-be9e-1b6dd157a209-headshot.png?width=64&height=64&frame=1&auto=webp&crop=&s=7eba44ea268928b969bcf73ee8667357412132ca",
      },
      // {
      //   text: "Thank you very much! My workplace cannot seem to pick between when2meet and Doodle and I feel like this brings the best of each into one.\n\nWell done <3",
      //   author: "u/jadiepants",
      //   link: "https://www.reddit.com/r/opensource/comments/1dlol7r/comment/m6bf3li",
      //   picture:
      //     "https://styles.redditmedia.com/t5_d7myp/styles/profileIcon_snoof50f1128-f439-433b-a6b2-8e987630e506-headshot.png?width=64&height=64&frame=1&auto=webp&crop=&s=94077bf80603c2855747f1bfc0b9dd1539fae75c",
      // },
    ],
    rive: null,
    showSchejy: false,
    showHowItWorksDialog: false,
    isVideoPlaying: false,
  }),

  computed: {
    ...mapState(["authUser"]),
    isPhone() {
      return isPhone(this.$vuetify)
    },
  },

  methods: {
    ...mapMutations(["setAuthUser"]),
    loadRiveAnimation() {
      // if (!this.rive) {
      //   this.rive = new Rive({
      //     src: "/rive/schej.riv",
      //     canvas: document.querySelector("canvas"),
      //     autoplay: false,
      //     stateMachines: "wave",
      //     onLoad: () => {
      //       // r.resizeDrawingSurfaceToCanvas()
      //     },
      //   })
      //   setTimeout(() => {
      //     this.showSchejy = true
      //     setTimeout(() => {
      //       this.rive.play("wave")
      //     }, 1000)
      //   }, 4000)
      // } else {
      //   this.rive.play("wave")
      // }
    },
    _signIn(calendarType) {
      if (calendarType === calendarTypes.GOOGLE) {
        signInGoogle({ state: null, selectAccount: true })
      } else if (calendarType === calendarTypes.OUTLOOK) {
        // NOTE: selectAccount is not supported implemented yet for Outlook, maybe add it later
        signInOutlook({ state: null, selectAccount: true })
      }
    },
    _emailSignIn(user) {
      this.setAuthUser(user)
      this.$posthog?.identify(user._id, {
        email: user.email,
        firstName: user.firstName,
        lastName: user.lastName,
      })
      this.$router.replace({ name: "home" })
    },
    signIn() {
      this.$router.push({ name: "sign-in" })
    },
    openHowItWorksDialog() {
      this.showHowItWorksDialog = true
      this.$posthog.capture("how_it_works_clicked")
    },
    onPlay() {
      setTimeout(() => {
        this.isVideoPlaying = true
      }, 1000)
    },
    openDashboard() {
      this.$router.push({ name: "home" })
    },
  },

  beforeDestroy() {
    this.rive?.cleanup()
  },

  watch: {
    [`$vuetify.breakpoint.name`]: {
      immediate: true,
      handler() {
        if (this.$vuetify.breakpoint.mdAndUp) {
          setTimeout(() => {
            this.loadRiveAnimation()
          }, 0)
        }
      },
    },
  },
}
</script>
