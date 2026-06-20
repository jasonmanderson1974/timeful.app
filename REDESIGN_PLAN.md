# The Fellowship — Redesign Plan

Living plan for rebranding/redesigning the self-hosted Timeful instance into
**The Fellowship / The Gathering** (a vintage gentleman's-club theme for a small
men's event-planning club). Branch: `redesign/fellowship-phase1`.

> Resume tip: read this top-to-bottom, then check **Current state** for exactly
> where we left off. Design source of truth: `samples/the-gathering.html` + `.css`.

---

## 1. Goal & scope (decided 2026-06-20)

- **Whole-app** re-skin — every screen, not just the landing.
- **Rebrand the UI** to The Fellowship + the Sir Thomas Foolery hare crest.
  Internal code identifiers (Go module `schej.it`, Mongo `schej-it`) STAY.
- **Full archaic "gentleman's club" voice** across all copy (buttons, labels,
  empty states, errors), while keeping things usable.
- Mascot: **Sir Thomas Foolery** = "Eric the Hare" (Eric + Eloise collection)
  with monocle + handlebar mustache.

## 2. Design system (source of truth = `samples/`)

**Palette** (now in `frontend/tailwind.config.js` as Tailwind tokens, prefix `tw-`):

| token | hex | use |
|---|---|---|
| `wood-deep` | #1c1410 | page base |
| `wood` | #241a13 | panels |
| `leather` | #2e2117 | raised surfaces |
| `green-felt` | #16261d | billiard-green accent |
| `green-deep` | #0f1c15 | |
| `brass` | #c9a44c | primary gold |
| `brass-bright` | #e3c578 | highlight |
| `brass-dim` | #8a7333 | hairlines/borders |
| `parchment` | #ede4d3 | primary text |
| `parchment-dim` | #b8ad97 | secondary text |
| `oxblood` | #6e2b2b | danger |

**Fonts** (loaded in `frontend/public/index.html`; Tailwind families):
`tw-font-display` = Cinzel (spaced ALL-CAPS titles), `tw-font-head` = Cormorant
Garamond (ornamental italic heads), `tw-font-body` = EB Garamond (running text).

**Reusable classes** (in `frontend/src/index.css`, `@layer components`):
`.flw-panel` (brass-keyline wood card), `.flw-btn` (brass CTA), `.flw-rule`
(centered ornamental rule — wrap a `<span>◆</span>`), `.flw-eyebrow` (spaced
all-caps brass label), `.flw-title` (display title), `.flw-sub` (italic lead).

**Crest component:** `frontend/src/components/general/SirThomasFoolery.vue`
(`:size` prop, unique gradient id per instance).

**Motifs:** damask diamond lattice + walnut background (global, on `.v-application`
in `index.css`), brass keyline frames, ◆ diamond rule dividers.

**Voice examples:** "Cast thy vote, good sir — when shall we convene?",
"The Manner of It" (How it works), "The Chronicle" (Blog), "Enter" (Sign in),
"Call a Gathering" (Create event), "To the Club Room" (Open dashboard),
"No dues · no login required."

## 3. Current state (as of 2026-06-20, end of session)

**DONE & deployed to live URL (WIP) for review:**
- Foundation: fonts, palette, dark Vuetify theme (`src/plugins/vuetify.js`,
  `dark: true`, brass primary), global damask CSS + `flw-*` classes, crest component.
- Landing **header** (crest + THE FELLOWSHIP wordmark, brass Cinzel nav) and
  **hero** (THE GATHERING title, ◆ rule, archaic subtitle, brass CALL A GATHERING
  button, felt-green spotlight + brass-framed hero portrait). See `src/views/Landing.vue`.
- Committed as `6a25536` on branch `redesign/fellowship-phase1` (pushed to origin).

**Deploy/branch state:**
- Branch `redesign/fellowship-phase1` is pushed to origin AND checked out on the VM.
- **VM is on the redesign branch, NOT `main`** — must `git checkout main` on the VM
  (and redeploy) when we abandon/merge, or keep deploying the branch during the redesign.
- Live at gathering.sirthomasfoolery.com (frontend rebuilt, server restarted).

**NOT yet converted (still old light Timeful style):**
- Landing below the hero: How-it-works, "It's that simple", Reddit testimonials,
  FAQ, Footer.
- Hero embedded demo media still shows the old light Timeful app screenshot/video.
- Global `App.vue` chrome (logged-in header/logo), all other pages & dialogs.

## 4. Phased task list

### Phase 1 — Foundation + Landing + global chrome  *(mostly done; deployed)*
- [x] Fonts, palette, Vuetify dark theme, global CSS, `flw-*` classes, crest.
- [x] Landing header + hero.
- [x] Landing: How-it-works section (archaic steps; crest replaces `schej_character.png`).
- [x] Landing: Reddit testimonials block — REMOVED (external SaaS social proof, N/A for club).
- [x] Landing: `FAQ.vue` (dark/brass panels) + rebranded/archaic FAQ + how-it-works copy.
- [x] `Footer.vue` → club colophon (crest, THE FELLOWSHIP, Sir Thomas Foolery, Privacy).
- [x] Global chrome in `App.vue`: crest + wordmark header, dark wood bar, archaic buttons.
- [x] Deployed to VM (commit `9c61b34`); landing reads coherently end-to-end.
- [ ] `NumberBullet.vue` still green → make brass (visible on landing how-it-works).
- [ ] Swap hero demo media (still old light Timeful screenshot/Vimeo) for something on-theme.
- [ ] `LandingPageCalendar.vue` (demo calendar) — restyle or replace (if still used).
- [ ] `HowItWorksDialog.vue` content + style.

### Phase 2 — Event / availability page (core guest screen)  *(in progress; deployed)*
- [x] `src/views/Event.vue` shell: parchment title/text, brass accents+buttons,
      leather chips; removed FormerlyKnownAs. Verified on live test event `/e/FE2dd`.
- [x] `ScheduleOverlap.vue` chrome: parchment text, brass accents, dark wood
      day-header strips, leather surfaces. The EMPTY grid reads great on dark.
- [x] Heatmap CELL colors VERIFIED OK on dark (checked in edit mode on `/e/FE2dd`):
      green `#00994C` "available" reads well on dark; red→maroon "unavailable" lands
      on-theme (oxblood-ish); single `#00994C88` visible. NO remap needed. The
      Available/If-needed toggle + legend panel already theme correctly.
- [ ] Archaic copy pass on Event.vue (buttons "Add availability"→? , "this Timeful"→
      "this Gathering", doc title `... - Timeful`→`The Fellowship`).
- [ ] `RespondentsList.vue`, `MarkAvailabilityDialog.vue`, calendar-permission dialogs,
      `GuestDialog`, `SignUpForSlotDialog`, day/time pickers, toolbar (`ToolRow.vue`).
- [ ] Test event for review: **`/e/FE2dd`** ("The Inaugural Gathering").

### Phase 3 — Auth + creation flows
- [ ] **ROOT-CAUSE FIRST — dialogs render LIGHT.** The create dialog (`NewEvent.vue`,
      a plain `<v-card>` with no explicit white bg) renders white, so Vuetify's dark
      theme isn't reaching dialog overlays / `solo` text-fields / `v-date-picker`.
      Fix this globally (likely `$vuetify`/v-app theme propagation to detached overlays,
      or a global CSS override for `.v-dialog .v-card`, `.v-text-field--solo`,
      `.v-picker`) BEFORE per-dialog work — it fixes every dialog at once. NOTE: the
      primary `color="primary"` buttons (e.g. "Create event") DO already render brass,
      and `MarkAvailabilityDialog` renders dark — so it's specific surfaces (card bg,
      solo fields, date picker) that stay light.
- [ ] `SignIn.vue` / `SignUp.vue` / `Auth.vue`, `SignInDialog.vue`.
- [ ] `NewEvent.vue` / `NewGroup.vue` / `NewSignUp.vue` create dialogs: dark card, brass
      tabs ("Dates and times/Dates only"), archaic header ("New event"→"Call a Gathering"),
      `tw-text-black`→parchment.
- [ ] Google OAuth button styling (keep Google brand req's on the button itself).

### Phase 4 — Logged-in app
- [ ] `Home.vue` dashboard + `home/` components, `EventItem.vue`, `EventType.vue`.
- [ ] Groups (`groups/`), Friends, `Settings` + `settings/` components.
- [ ] `AuthUserMenu.vue`, snackbars, global dialogs.

### Phase 5 — Misc & cross-cutting
- [ ] `PrivacyPolicy.vue`, error/empty states, `AutoSnackbar`, tooltips.
- [ ] Favicon, PWA manifest (`public/`), OG image (`/img/ogImage.png`) → Fellowship crest.
- [ ] Meta titles/descriptions (also the Go `NoRoute` OG-tag injector in `server/main.go`
      sets per-route titles — update those strings for the rebrand).
- [ ] Full archaic-voice copy pass once layouts are settled.
- [ ] Accessibility/contrast audit on dark; mobile pass at 390px.

### Phase 6 — Finalize
- [ ] Replace remaining `schej_*`/`timeful_*` image assets with Fellowship art.
- [ ] Decide hero media (custom illustration / crest animation?).
- [ ] Merge `redesign/fellowship-phase1` → `main`; switch VM back to `main`; deploy.

## 5. Conventions & gotchas

- **Deploy = commit + push + deploy** (standing rule). VM repo at
  `/home/jasonmanderson/docker/timeful.app`, `ssh jasonmanderson@192.168.54.68`.
- **Docker Hub pull rate limit (429):** `docker compose up -d --build` rebuilds the
  Go server too, which pulls `golang:1.25-alpine` and can hit the anonymous pull limit.
  When ONLY the frontend changed, deploy with:
  `docker compose up -d --build frontend && docker compose restart server`
  (the server must restart to re-read the new `dist` — it registers static routes by
  walking `frontend/dist` at startup). Longer-term fix: `docker login` on the VM.
- **Dual-purpose Tailwind tokens:** legacy tokens like `white`/`light-gray` are used as
  BOTH backgrounds and text in different places — do NOT global-remap them; convert each
  screen to the new `flw`/Fellowship tokens explicitly. Legacy tokens retained for now.
- **No local Go/Docker on the dev box** — backend can't be compiled locally; rely on the
  VM build (safe failure: bad build keeps old container running).
- **Verifying a live route:** Go `NoRoute` serves index.html (200 text/html) for any
  unregistered path — check content-type, not status, to tell removed vs live API routes.
- **Review/screenshots:** Playwright+Chromium installed at `/tmp/shot` on the dev box;
  serve a local build with `python3 -m http.server 8099` from `frontend/dist`.
- Build check: `cd frontend && npm run build` (no backend needed; API calls just fail).

## 6. Open questions / decisions to revisit

- Hero demo media: keep a product demo, or replace with bespoke club art?
- Footer: keep Blog/Articles/Reddit columns, or replace with club-appropriate links?
- How literal should the archaic voice be on functional errors (usability vs theme)?
- Light-mode toggle, or dark-only? (Currently dark-only.)
