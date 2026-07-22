# Timeful / The Fellowship — Improvement & Feature Backlog

> Compiled 2026-07-22 from a full-codebase review. Context that shaped priorities:
> this is a **self-hosted, invite-only fork** for a **~30–40 person club** (≈12 men + wives).
> That means: **reliability, maintainability, and small-club utility** matter far more
> than horizontal scale, multi-tenant concerns, or growth/monetization features.
> Companion docs: `REDESIGN_PLAN.md`, `ACCESS_CONTROL_PLAN.md`.

Priority legend: **P0** = do first (correctness / risk / cheap-and-high-value) ·
**P1** = high value · **P2** = moderate · **P3** = nice-to-have.
Effort: **S** ≈ <½ day · **M** ≈ 1–2 days · **L** ≈ 3+ days.

---

## PART A — Refactoring & Code-Health

### P0 — Correctness & risk (do first)

- [x] **A1 · Standardize `c.Bind` failure responses.** `S` — **DONE 2026-07-22 (events.go).**
  Correction to the original finding: the `events.go` sites use `c.Bind` (not `c.ShouldBind`), and
  Gin's `c.Bind` **already** calls `AbortWithError(400)` internally — so these returned a **400 with
  an empty body**, not a silent 200. (A true 200-on-bad-input would only occur at `ShouldBind`
  sites, which already return proper JSON.) So A1 was a *consistency* issue, not a silent-success
  bug. Fixed all **11** `events.go` bind handlers to return `c.JSON(http.StatusBadRequest,
  responses.Error{Error: err.Error()})` (removing the lone `fmt.Println` debug print and the
  bare/`c.Status`-only variants). **Follow-up:** the other route files' bind handlers already use
  either `ShouldBindJSON` with a JSON error body or `BindJSON` (also auto-400) — spot-check for
  consistency but no silent-200 bug found.

- [x] **A2 · Stop panicking inside request handlers on DB errors.** `M` — **ALL route handlers
  DONE (2026-07-22); `db/` + `services/` remain (needs a signature refactor, not a mechanical
  swap).** Converted every handler `Panicln` to `logger.StdErr.Println(err)` +
  `c.JSON(500, responses.Error{Error: errs.Internal})` + `return`: `events.go` (12),
  `user.go` (16), `auth.go` (5). `signInHelper` (a helper returning `(models.User, error)`)
  propagates the error instead — `return models.User{}, err`. Two handler `Panicln`s that were
  actually *bind* errors (`toggleCalendar`, `toggleSubCalendar`) now return 400. The
  `importEvent` response loop logs + `continue`s (event already inserted). `admin.go`,
  `folders.go`, `stripe.go`, `users.go` had none.
  **Intentionally left:** `auth.go generateOtpCode()` — a `crypto/rand` failure (not a DB error),
  in a helper returning `string` with no context; converting needs a signature change and the
  failure is astronomically rare.
  **Follow-up (separate task, still P0-class):** ~55 `Panicln`s remain in `db/` (`events.go` 9,
  `utils.go` 6, `users.go` 3, `folders.go`/`init.go` 1 each) and `services/` (`gcloud/tasks.go` 5,
  `calendar/google_calendar.go` 4, `auth/auth.go` 3, `contacts` 2, …). These live in functions
  that mostly return no `error` (e.g. `func GetUserById(id) *models.User`), so removing the panics
  means changing signatures to return errors and updating every caller — a real refactor to plan,
  not a mechanical pass.

- [x] **A3 · Unchecked writes in loops.** `S` — **DONE 2026-07-22 (the 3 listed sites).**
  `createEvent` now builds an `[]interface{}` and uses a single `InsertMany` with an error check
  (returns 500 on failure — it runs before the event is inserted, so no partial event). The
  `editEvent` added-attendees insert and the `updateEventResponse` new-response insert now capture
  and log the error (the latter only increments `NumResponses` on success). **Follow-up:** this is a
  subset of a broader pattern — many `UpdateOne`/`UpdateByID` calls across the routes ignore their
  error too (e.g. `updateEventResponse:947`); worth a dedicated unchecked-write sweep.

- [x] **A4 · Remove duplicate `refreshAuthUser` store action.** `S` — **DONE 2026-07-22.**
  Deleted the second (shadowing) definition in `frontend/src/store/index.js`; the original at the
  top of `actions` remains.

### P1 — Structural debt that slows every future change

- [ ] **A5 · Break up `ScheduleOverlap.vue` (4,638 lines, ~99 methods).** `L`
  This is the single largest maintenance liability in the repo — a god-component mixing drag-select
  grid math, availability animation, calendar-account plumbing, sign-up-block editing, and
  respondent hover/selection. Extract cohesive units: a composable/mixin for grid geometry
  (`getRowColFromXY`, `clamp*`, `normalizeXY`, drag lifecycle), one for availability
  fetch/format/animate, and split sign-up-block editing into its own child. No behavior change;
  do it incrementally behind the existing tests.

- [ ] **A6 · Split `server/routes/events.go` (1,925 lines).** `M`
  One file owns event CRUD, responses, calendar availability, and import, with several 200–300-line
  handlers (`updateEventResponse` ~290, `getEvent` ~180, `importEvent` ~190). Split by concern —
  `events.go` (CRUD), `event_responses.go`, `event_calendar.go`, `event_import.go` — keeping the
  `InitEvents` registration in one place. Pure reorg; keeps diffs reviewable going forward.

- [ ] **A7 · Consolidate date libraries (drop `moment`, ideally `spacetime`).** `S`
  `package.json` ships **three** date libs. `moment` has **0** references in `src/` — it's pure dead
  weight (deprecated upstream too); remove it. `spacetime` is used in exactly **1** file vs `dayjs`
  in **9** — migrate that one usage and drop `spacetime` as well. Shrinks the bundle and removes a
  "which lib do I reach for?" decision from every date change. (`date_utils.test.js` guards the
  behavior.)

- [ ] **A8 · Add linting to CI (nothing lints today).** `S`
  No ESLint in the frontend (only `prettier`), no `golangci-lint` / `go vet` on the backend. CI runs
  tests + build only. Add `go vet ./...` + `golangci-lint` to `backend-ci.yml` and an ESLint step to
  `frontend-ci.yml`. This is the cheapest way to stop A1/A3-class issues (ignored errors, unused
  vars) from recurring. Introduce as **warnings first** to avoid blocking on a large existing backlog.

### P2 — Cleanup & smaller components

- [ ] **A9 · Delete dead code.** `S`
  - `server/main.go:254` `splitPath()` — defined, never called.
  - `createEvent` (`events.go:170-238`) carries large commented-out blocks (group-owner response
    seeding, slackbot messaging). Either restore intentionally or delete.
  - `isPremiumUser` getter/util is kept "inert" after the paywall removal (per `REDESIGN_PLAN`) —
    confirm nothing depends on it and remove.
  - `pricingPageConversion` experiment state in the store — leftover A/B infra with no paywall.

- [ ] **A10 · Normalize `fetch_utils.js` error handling.** `S`
  `frontend/src/utils/fetch_utils.js` has inconsistent style (mixed semicolons/indentation from
  line 60) and no shared timeout/abort or centralized snackbar-on-error. Standardize the error
  shape and consider a single interceptor so every call site doesn't re-implement
  `try/catch → dispatch("showError")` (see the repetition in `store/index.js` actions).

- [ ] **A11 · Trim remaining large components.** `M`
  After A5: `Event.vue` (1,815), `NewEvent.vue` (1,011), `RespondentsList.vue` (844),
  `NewSignUp.vue` (827). Each is a candidate for extracting presentational children and moving
  pure helpers into `utils/`. Lower urgency than A5 but same class of problem.

- [ ] **A12 · Remove stray `console.log` (6) and backend `fmt.Println` debug prints (2).** `S`
  Route through the existing `logger` on the backend; drop or gate the frontend logs.

### P3 — Housekeeping

- [ ] **A13 · Align Go toolchain version.** `S`
  `server/go.mod` declares `go 1.20`; CI builds with **Go 1.25**. Bump the `go` directive so local
  and CI agree (and to unlock newer stdlib). Low risk, avoids "works in CI, not locally" surprises.

- [ ] **A14 · Prune legacy CORS origins.** `S`
  `main.go:105` still defaults to `schej.it` / `www.schej.it` origins. Harmless, but for an
  invite-only Fellowship instance the default allowlist should reflect the real deployed domain(s).

- [ ] **A15 · Clean up / document migration scripts.** `S`
  `server/scripts/*` reference outdated models and intentionally don't compile (noted in
  `backend-ci.yml`). Fine to keep as history, but a one-line README per dated folder stating
  "run once on <date>, superseded" would prevent someone re-running a destructive migration.

---

## PART B — Test Coverage (its own track — currently thin)

- [ ] **B1 · Cover the core `events.go` handlers.** `M` · **P1**
  The 1,925-line heart of the app has **no test file**. Total Go test code is ~428 lines; JS ~320.
  Prioritize `updateEventResponse`, `getResponses`/`getResponsesMap`, and the
  blind-availability / email-stripping logic (`shouldKeepGroupResponseUserEmails`,
  `stripSensitiveUserFields`) — these encode privacy rules that are easy to regress. Do this
  *after* A6 so the split files are testable in isolation.

- [ ] **B2 · Cover access-control end to end.** `S` · **P1**
  `models/roles.go` has unit tests, but the middleware wiring (`middleware/auth.go` allowlist
  enforcement + `CanInviteRequired`) is the actual gate for an invite-only app. Add handler-level
  tests that a struck-off member's session is rejected and a guest cannot create events.

- [ ] **B3 · Frontend: cover the grid/availability math extracted in A5.** `M` · **P2**
  The geometry helpers are pure functions once extracted — high-value, easy-to-test, and currently
  the riskiest untested logic on the frontend.

---

## PART C — New Features (fit for a ~40-person invite-only club)

### P1 — High value, leverages infrastructure already present

- [ ] **C1 · RSVP / attendance tracking for a *confirmed* gathering.** `M`
  Today the app finds the best time and can schedule a calendar event, but there's no
  yes/no/maybe headcount once a time is locked. For a club that actually meets, "who's coming to
  Saturday's gathering" is the natural next step after "when works." Reuses the `Attendee` model
  and the `Response`→scheduled-event flow.

- [ ] **C2 · Automated pre-gathering reminder emails.** `S–M`
  The Cloud Tasks + email plumbing already exists (`services/gcloud/CreateEmailTask`,
  `Remindee` scheduling in `createEvent`). Extend it from "remind people to respond" to "remind
  confirmed attendees N hours before the event." Very high utility, low new infrastructure.

- [ ] **C3 · "Add to calendar" / `.ics` export for confirmed gatherings.** `S`
  Many club members (esp. spouses) have no Google account (per `ACCESS_CONTROL_PLAN`), so the
  Google-calendar scheduling path doesn't serve them. A universal `.ics` download / "add to
  calendar" link works for everyone and needs no OAuth. The server already parses ICS
  (`services/calendar/ics_calendar.go`) — generation is the mirror of that.

- [ ] **C4 · Plus-one / guest handling on responses.** `S–M`
  The club is "≈12 men + wives." Let a respondent indicate they're bringing a spouse/guest so
  headcounts (C1) are accurate without every spouse needing an account. Small model addition on the
  response/attendee.

### P2 — Strong quality-of-life

- [ ] **C5 · Recurring gatherings.** `M`
  A club that meets regularly benefits from "repeat monthly" rather than recreating an event each
  time. Builds on existing event duplication (`duplicateEvent`, `events.go:1553`).

- [ ] **C6 · Venue / activity poll (not just time).** `M`
  Extend the availability-poll concept to "where / what" — a lightweight multiple-choice poll so the
  club can vote on venue or activity. Overlaps with the sign-up-block UI already built.

- [ ] **C7 · Per-gathering discussion thread / comments.** `M`
  A place to coordinate details ("I'll bring cigars," "parking is out back") attached to the event.
  Fits the club's social nature; keeps coordination off scattered group texts.

- [ ] **C8 · Web push notifications for "new gathering" / "you were invited."** `M`
  A service worker is **already registered** (`register-service-worker`, `kill-sw.js` kill switch),
  so the client half is partly there. Push for invitations and "the time is set" closes the loop
  without relying only on email.

- [ ] **C9 · Sign-up-block capacity + waitlist.** `S`
  The `SignUpBlock` model **already has a `Capacity *int` field** that isn't fully enforced/surfaced.
  Enforce capacity and add a simple waitlist when full — useful for limited-seat gatherings (dinners,
  outings).

### P3 — Nice-to-have / thematic

- [ ] **C10 · Members-only gathering archive ("The Chronicle").** `M`
  The redesign removed the public blog, but an internal, roll-gated history of past gatherings
  (date, attendees, a photo or two) fits the "gentleman's club" theme and gives the Fellowship a
  sense of continuity. Reuses the existing role-gated Fellowship directory work.

- [ ] **C11 · Printable / exportable roster of the Fellowship directory.** `S`
  `Fellowship.vue` / `MemberAdmin.vue` already render the roll; a print-friendly or PDF/CSV export
  is a small addition members would use.

- [ ] **C12 · Map / venue location on an event.** `S–M`
  A `Location` model already exists (`models/location.go`, `location_utils.js`). Surfacing a venue
  with a static map/address on the gathering page is a natural, mostly-wiring feature.

---

## Suggested sequencing

1. **P0 correctness** (A1–A4) — small, safe, removes silent-failure and crash-on-error footguns.
2. **A8 lint-as-warnings + A13** — cheap guardrails so the rest of the cleanup stays clean.
3. **A6 (split events.go) → B1/B2 (tests)** — reorganize the backend core, then lock it with tests.
4. **A5 (split ScheduleOverlap.vue) → B3** — the biggest frontend win; tackle in slices.
5. **Feature track in parallel:** C2 → C3 → C1 (reminder infra → universal calendar → RSVP) are the
   highest-leverage, lowest-new-infrastructure wins for an active club.
