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

- [x] **A2 · Stop panicking inside request handlers on DB errors.** `M` — **FULLY DONE 2026-07-22
  (route handlers + `db/` + `services/`).** Only intentional fail-fast panics remain (`db/init.go`
  startup, `auth.go generateOtpCode` crypto/rand — see below). Converted every handler `Panicln` to
  `logger.StdErr.Println(err)` +
  `c.JSON(500, responses.Error{Error: errs.Internal})` + `return`: `events.go` (12),
  `user.go` (16), `auth.go` (5). `signInHelper` (a helper returning `(models.User, error)`)
  propagates the error instead — `return models.User{}, err`. Two handler `Panicln`s that were
  actually *bind* errors (`toggleCalendar`, `toggleSubCalendar`) now return 400. The
  `importEvent` response loop logs + `continue`s (event already inserted). `admin.go`,
  `folders.go`, `stripe.go`, `users.go` had none.
  **Intentionally left:** `auth.go generateOtpCode()` — a `crypto/rand` failure (not a DB error),
  in a helper returning `string` with no context; converting needs a signature change and the
  failure is astronomically rare.
  **`db/` + `services/` — partially DONE 2026-07-22 (the safe, error-returning subset):**
  Converted the panics in functions that *already return an `error`* (or error-like) so the error
  now flows through the return value instead of panicking — no signature change, no caller change:
  `db/folders.go CreateFolder` (the `return …, err` after it was dead code), `services/calendar/
  google_calendar.go` `GetCalendarList`/`GetCalendarEvents` (4 sites), `services/contacts/
  contacts.go SearchContacts` (2 sites — returns `*errs.GoogleAPIError{Code: 500}` so the caller's
  `c.JSON(googleError.Code, …)` stays valid). This also fixes a **latent goroutine crash**: the
  calendar async wrappers `recover()` with `err.(error)`, but `log.Panicln` panics with a *string*,
  so that assertion would itself panic and take down the process — now moot, since these no longer
  panic.

  **Deliberately NOT refactored (assessment, not laziness):** the ~40 remaining `Panicln`s in
  value-returning/void `db/` getters (`GetUserById`, `GetEventById`, …) and `services/`
  (`GetTokensFromAuthCode`, `RefreshAccessToken`, `CallApi`, `GetUserInfo`, `CreateEmailTask`).
  Reasons: (1) these getters already `return nil`/empty on *not-found* (callers handle it) and only
  panic on an *unexpected* DB error; (2) **every** such path is already contained — request handlers
  by `gin.Recovery()` (→ 500, the correct status), and all db/service-calling **goroutines** by
  their own `defer recover()` (verified: `events.go` 1005/1053/1422, calendar + auth async wrappers).
  So there is no crash bug left to fix. A full signature refactor would touch **80+ call sites**
  (`GetUserById` alone has 20) with **no local Go compiler** to verify — high risk, low benefit.
  If desired later, do it **incrementally, one function at a time, gated by Backend CI** — not as a
  single sweep. `db/init.go` panics are startup/fail-fast and should stay.

  **Incremental refactor — batch 1 DONE 2026-07-22 (CI-green):** de-panicked the lowest-caller
  getters — `db/utils.go` (`GetDailyUserLogByDate`→`(_, error)`, `UpdateDailyUserLog`,
  `GetFriendRequestById`/`DeleteFriendRequestById` — the last two have 0 callers, i.e. **dead code**,
  candidates for deletion), `GetEventsCreatedThisMonth`→`(int, error)`, `GetUserByStripeCustomerId`
  →`(*User, error)`. **Remaining tiers (heads-up — entangled/high-caller):** the event getters
  (`GetEventById`, `GetEventByShortId`, `GetEventByEitherId`, `GenerateShortEventId`) are a **single
  cluster** — `GetEventByEitherId` (11 callers) calls the other two, so they must move together
  (~17 call sites). `GetUserById` (20 callers) and `GetUserByEmail` (8) are the other big ones, plus
  `GetEventResponses`/`GetAttendees` (slices) and the `services/` functions (`CallApi`,
  `GetTokensFromAuthCode`, `RefreshAccessToken`, `GetUserInfo`, `CreateEmailTask`). These are the
  high-effort/low-benefit end (all already recovered → 500); decide whether they're worth it.

  **Batch 2 DONE 2026-07-22 (CI-green):** the event-getter cluster —
  `GetEventById`/`GetEventByShortId`/`GetEventByEitherId` → `(*Event, error)`, ~17 call sites
  updated (11 handlers → 500; `main.go` + db-internal callers keep nil-checks). `GenerateShortEventId`
  kept its `string` signature (handles the error internally).

  **Batch 3 DONE 2026-07-22:** the user-getter cluster — `GetUserById` → `(*User, error)` (20 callers)
  and `GetUserByEmail` → `(*User, error)` (8 callers) in `db/users.go`. Call-site handling by context:
  request handlers → 500 (`middleware/auth.go`, `routes/users.go` ×2, `routes/user.go` ×3,
  `routes/auth.go` ×3, `routes/events.go` handlers); `signInHelper` propagates
  `return models.User{}, err`; async goroutines (email-send blocks in `updateEventResponse`) and
  counted loops (`getResponses` populate loops, calendar-fetch loop) log + `continue`/`return` so they
  degrade gracefully rather than aborting; pure helpers that fall back safely ignore the error
  (`db/events.go isNameBlocked`, `routes/admin.go effectiveTargetRole`/invite email-check,
  `shouldKeepGroupResponseUserEmails` → treats a fetch error as "not a member"); `routes/stripe.go`
  fulfillment helper logs + returns.

  **Batch 4 DONE 2026-07-22 (final — A2 complete):** the slice getters + `services/` tail.
  - Slice getters in `db/events.go`: `GetEventResponses` → `([]EventResponse, error)` (8 callers) and
    `GetAttendees` → `([]Attendee, error)` (3 callers). All `routes/events.go` handler callers → 500;
    the `shouldKeepGroupResponseUserEmails` helper ignores the error (safe empty-slice fallback).
  - `services/services.go CallApi` → `(*http.Response, error)` (also fixed a latent nil-`req` deref by
    checking the previously-ignored `http.NewRequest` error). Callers propagate: outlook
    `GetCalendarList`/`GetCalendarEvents` (already `…, error`), contacts `SearchContacts` (2 sites →
    `*errs.GoogleAPIError{Code:500}`), and `microsoftgraph.GetUserInfo` → `(UserInfo, error)`.
  - `services/auth/auth.go`: `GetTokensFromAuthCode` → `(TokenResponse, error)` (3 handler callers in
    `user.go`/`auth.go` → 500; the OAuth-error branch now returns an error instead of panicking on the
    marshaled body) and `RefreshAccessToken` → `(AccessTokenResponse, error)` (its only caller,
    `RefreshAccessTokenAsync`, feeds the error into the existing `RefreshAccessTokenData.Error` channel
    field). `microsoftgraph.GetUserInfo` callers: `user.go` handler → 500, `auth.go signInHelper` →
    `return models.User{}, err`.
  - `services/gcloud/tasks.go CreateEmailTask` kept its `[]string` signature (no caller changes):
    reminder-email scheduling is a best-effort side effect of event create/edit, so a failure must not
    500 the event op — env-var/template-id parse errors log + `return []string{}`, and per-task
    marshal/CreateTask errors log + `continue` (partial scheduling still succeeds).
  - **Deliberately left panicking:** `db/init.go:39` (Mongo connect at startup — fail-fast) and
    `auth.go generateOtpCode` crypto/rand (astronomically rare, helper returns bare `string`).

  **Not verified locally (no Go toolchain on this machine) — Backend CI is the gate.**

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

- [ ] **A5 · Break up `ScheduleOverlap.vue` (was 4,638 lines, ~99 methods).** `L` — **SAFE PART DONE
  2026-07-22 (3 mixin slices, CI-green; component 4,638 → 3,744 lines, ~19% smaller). Only the
  riskier child-component split remains — see caveat.**
  This is the single largest maintenance liability in the repo — a god-component mixing drag-select
  grid math, availability animation, calendar-account plumbing, sign-up-block editing, and
  respondent hover/selection.
  - **Step 1 DONE:** grid geometry + drag lifecycle (`normalizeXY`, `clampRow`, `clampCol`,
    `getRowColFromXY`, `endDrag`, `inDragRange`, `moveDrag`, `startDrag`) →
    `schedule_overlap/dragGridMixin.js`. 4,638 → 4,303.
  - **Step 2 DONE:** "Aggregate user availability" (`fetchResponses`, `getResponsesFormatted`,
    `getRespondentsForHoursOffset`, `showAvailability`) → `availabilityMixin.js`. 4,303 → 4,125.
  - **Step 3 DONE:** "Current user availability" incl. the animate cluster (`refreshAuthUser`,
    `resetCurUserAvailability`, `populateUserAvailability`, `getIsTimeBlockInFirstSplit`,
    `getTimeBlockStyle`, `getAvailabilityFromCalendarEvents`, `setAvailabilityAutomatically`,
    `animateAvailability`, `stopAvailabilityAnim`) → `currentAvailabilityMixin.js`. 4,125 → 3,744.
  - All three are verbatim Vue 2 mixin moves (behavior-preserving: methods run on the same instance
    `this`; template bindings and cross-`this.*` calls resolve unchanged). Verified per step via
    `npm build`, **eslint `no-undef`** (the real gate — it caught a `dayjs` free-reference in step 3
    that `npm build` bundled silently), and unit tests (23/23).
  - **Steps 4–6 DONE 2026-07-22 (Tier 1 slices, runtime-verified via headless Chromium against the
    local stack):** respondent hover/selection → `respondentSelectionMixin.js`; the whole Timeslot
    region (484 lines: sizing, class/style maps, von handlers, valid-time-ranges) →
    `timeslotStylingMixin.js`; Options-panel handlers → `optionsMixin.js`. Component now **3,166**
    lines (was 4,638 pre-A5). Verified in-browser: grid renders, respondent hover/click switches
    single/subset availability views, best-times toggle re-renders + persists.
  - **Remaining — do NOT do blind:** split **sign-up-block editing into a child component** (template
    + props + events = runtime-only behavior `npm build` can't verify — Tier 2 of the runtime-refactor
    plan; browser-verify locally like the Tier 1 slices).
    B3 (grid-math tests) can extract the pure bits of the geometry logic for real coverage.

- [x] **A6 · Split `server/routes/events.go` (1,925 lines).** `M` — **DONE 2026-07-22 (CI-green,
  in 3 incremental commits).** Pure reorg, no behavior change; `InitEvents` (route registration)
  stays in `events.go`. All handlers/helpers moved within `package routes`, so cross-file references
  resolve without changes. Final layout:
  - `events.go` (946 lines) — CRUD/read: `InitEvents`, `createEvent`, `editEvent`, `getEventIds`,
    `getEvent`, `deleteEvent`, `duplicateEvent`, `archiveEvent`.
  - `event_responses.go` (837) — `getResponses`, `updateEventResponse`, `deleteEventResponse`,
    `renameUser`, `userResponded`, `declineInvite` + helpers `findResponse`,
    `shouldKeepGroupResponseUserEmails`, `stripSensitiveUserFields`, `getResponsesMap`.
  - `event_import.go` (226) — `importEvent`.
  - `event_calendar.go` (150) — `getCalendarAvailabilities`.
  Each file's import block was hand-curated (no local Go — verified per-commit by Backend CI, since
  Go errors on unused imports). No route/comment content changed, so Swagger `docs/` need no regen.
  **Now testable in isolation → unblocks B1.**

- [ ] **A7 · Consolidate date libraries (drop `moment`, ideally `spacetime`).** `S`
  `package.json` ships **three** date libs. `moment` has **0** references in `src/` — it's pure dead
  weight (deprecated upstream too); remove it. `spacetime` is used in exactly **1** file vs `dayjs`
  in **9** — migrate that one usage and drop `spacetime` as well. Shrinks the bundle and removes a
  "which lib do I reach for?" decision from every date change. (`date_utils.test.js` guards the
  behavior.)

- [x] **A8 · Add linting to CI (nothing lints today).** `S` — **DONE 2026-07-22 (warnings-first).**
  All lint steps use `continue-on-error: true`, so findings surface in the CI log without blocking
  merges on the existing backlog. **Backend** (`backend-ci.yml`): added `go vet` (scoped
  `go vet $(go list ./... | grep -v '/scripts')` so it skips the non-compiling migration scripts) and
  `golangci-lint` via `golangci/golangci-lint-action@v6` **pinned to v1.61.0** (v2 changed the config
  schema) with a v1-format `server/.golangci.yml` (default linter set; `skip-dirs: [scripts]`).
  **Frontend** (`frontend-ci.yml`): added `eslint@^8.57` + `eslint-plugin-vue@^9.27` as devDeps
  (lockfile regenerated; `npm ci` verified green), a `.eslintrc.cjs` (Vue 2 preset
  `plugin:vue/essential` + `eslint:recommended`; `vue/multi-word-component-names` off since view
  components are intentionally single-word; noisiest rules set to `warn`), `.eslintignore`, a `lint`
  npm script, and a non-blocking `Lint` CI step. Baseline: **102 problems (41 errors, 61 warnings)** —
  that's the backlog to work down before flipping the steps to blocking. **Next:** once the backlog is
  cleared, drop `continue-on-error` to make lint a real gate.

### P2 — Cleanup & smaller components

- [x] **A9 · Delete dead code.** `S` — **DONE 2026-07-22 (CI-green).**
  - ✅ `server/main.go` `splitPath()` — removed (recursive helper, no external callers).
  - ✅ `createEvent` commented-out "add owner to group by default" block — removed (referenced the
    long-gone `event.Responses` field).
  - ✅ `pricingPageConversion` A/B state — removed (write-only Vuex state; its only mutation caller was
    already commented out and the value is never read; also dropped the mutation + mapMutations reg).
  - **Left intentionally:** `isPremiumUser` is NOT dead — the store getter is still wired via
    `mapGetters` in `ScheduleOverlap.vue`, `Event.vue`, `ToolRow.vue`. Removing it needs confirming
    those template/logic uses are truly inert, which can't be verified without running the app; folded
    into the [A11]/paywall-cleanup consideration rather than removed blind.

- [x] **A10 · Normalize `fetch_utils.js` error handling.** `S` — **CORE DONE 2026-07-22 (CI-green);
  timeout/interceptor deferred.** Fixed the inconsistent style (the stray semicolons/indentation from
  line 60) and **standardized the error shape** — which also fixed a live regression: the Aug-2025
  "better debug logs" change had rewritten `throw returnValue` into a wrapped Error exposing only
  `.parsed`, silently breaking the `err.error` contract 6 call sites still use (`switch (err.error)`,
  `err.error?.code`, `err.error === …`), while 2 sites had migrated to `err.parsed?.error`. The
  thrown error now exposes **both** `err.error` (server code, or raw body if not an object) and
  `err.parsed` (full body), plus `err.status`/message; dropped the unused `.url`/`.responseBody`/
  `.headers`. Locked with `fetch_utils.test.js` (6 tests mocking `fetch`; suite 32 → 38).
  **Deferred (behavior change, needs app-run verification — not done blind):** the shared
  timeout/abort (a default timeout could kill legitimately-slow calls like calendar fetches) and the
  centralized snackbar-on-error interceptor (auto-dispatching `showError` in the client would
  double-show or override the ~58 call sites that handle errors themselves).

- [ ] **A11 · Trim remaining large components.** `M` — **STARTED 2026-07-22 (dead-code slice done;
  larger extractions need app-run verification — see caveat).**
  After A5: `Event.vue` (now 1,776), `NewEvent.vue` (1,010), `RespondentsList.vue` (844),
  `NewSignUp.vue` (827). Candidates for extracting presentational children and moving pure helpers
  into `utils/`.
  - **Done:** removed `Event.vue`'s dead `interceptPluginResponses` debug method (listener was
    commented out) → 1,815 → 1,776.
  - **Done 2026-07-22:** `pluginMessagesMixin` extracted (`components/event/pluginMessagesMixin.js`
    — `handleMessage`/`setSlots`/`getSlots`, 567 lines, verbatim; orphaned + pre-existing unused
    imports pruned) → Event.vue **1,175**. Plugin API runtime-verified via headless Chromium:
    get-slots/set-slots round-trip on a real event (guest write + readback + UI), no console errors.
  - **⚠️ Verification caveat (learned the hard way here):** `Event.vue` is mostly `this`-coupled
    action handlers, not the pure/geometry code A5 had. The only large "method" appeared to be ~595
    lines but was actually THREE methods — `interceptPluginResponses` (dead) **plus the active
    `setSlots`/`getSlots` plugin handlers** (an `async`-method detection gap hid them). `npm build`
    and eslint do NOT catch deleting an actively-`this`-called method, so a wrong boundary silently
    breaks runtime (here: the plugin API). Remaining A11 extractions (a `pluginMessagesMixin` for
    `handleMessage`/`setSlots`/`getSlots`, or child-component splits) should be done **with the app
    running to smoke-test** — do not do them blind. Pre-existing unused imports in `Event.vue` (~7)
    are separate baseline cruft, safe to prune later.

- [x] **A12 · Remove stray `console.log` and backend `fmt.Println` debug prints.** `S` — **DONE
  2026-07-22 (CI-green).** Dropped the stray frontend logs: `SignUpForSlotDialog` (logged the block
  on submit), `FeatureNotReadyDialog` (empty-feedback else that only logged — removed the branch),
  `NewEvent` edit-error catch (kept user-facing `showError`, dropped unused `err`). **Left
  intentionally:** the structured `[PLUGIN RESPONSE]` logging in `Event.vue` (deliberate plugin-API
  dev tooling). Backend: the only remaining `fmt.Println` is `utils.PrintJson`, a named debug utility
  whose print IS its purpose (only called from a non-compiling script) — not a stray print; the stray
  handler prints were already removed back in A1/A3.

### P3 — Housekeeping

- [x] **A13 · Align Go toolchain version.** `S` — **DONE 2026-07-22.**
  Bumped `server/go.mod` `go 1.20` → `go 1.25` to match the CI toolchain (`setup-go` with
  `go-version: "1.25"` in `backend-ci.yml`). Verified green by CI (no local Go toolchain).

- [ ] **A14 · Prune legacy CORS origins.** `S` — **DEFERRED (needs the real deployed domain).**
  `main.go` still defaults to `schej.it` / `www.schej.it` (+ `timeful.app`, localhost). This is only
  the *fallback* default — prod sets `CORS_ORIGINS` — but pruning it means knowing the real Fellowship
  domain, which is an open rebranding decision (**[D0]/[D2]**). Do it as part of the domain rebrand
  rather than guessing a prod origin here.

- [x] **A15 · Clean up / document migration scripts.** `S` — **DONE 2026-07-22.**
  Added `server/scripts/README.md`: explains each dated folder is a run-once manual migration (kept
  for history), warns against re-running destructive ones, and documents that they intentionally
  don't compile / are excluded from CI. Used a single README listing the folders in date order rather
  than fabricating per-folder run-date/status (each `main.go` is its own record).

---

## PART B — Test Coverage (its own track — currently thin)

- [x] **B1 · Cover the core `events.go` handlers.** `M` · **P1** — **DONE 2026-07-22 (CI-green,
  3 incremental commits).** Went from zero event tests to 20, split into DB-free unit tests and
  DB-backed integration tests.
  **Pure logic** (`routes/event_responses_test.go`, 17 tests, no Mongo): the easy-to-regress privacy
  rules — `getResponsesMap` (keying/empty/duplicate-id last-wins), `findResponse`,
  `stripSensitiveUserFields` (clears calendar/billing, preserves identity, nil-safe),
  `shouldKeepGroupResponseUserEmails` (DB-free guard branches), and **blind-availability filtering**:
  extracted `getResponses`' inline logic into a pure helper `filterResponsesForBlindAvailability`
  (behavior-preserving) and covered the full matrix (blind off → all; blind on → owner sees all,
  non-owner only own, guest only theirs, anon nothing).
  **DB-backed handlers** (option (a): `routes/event_responses_db_test.go`, 3 tests): `TestMain` calls
  `db.Init()` when `MONGODB_URI` is set (`mongo.Connect` is lazy, so safe); tests gate on that via
  `requireDB(t)` so `go test ./routes/` still passes without Mongo (they skip) and run in CI (Mongo
  service). Drive the handlers through a real gin engine + session middleware: `getResponses` 404 on
  missing event and blind-off happy path (returns all); `updateEventResponse` guest POST persists a
  new `EventResponse`. Fixtures inserted under a fresh ObjectID, cleaned up per-test.
  **Optional follow-ups (not blocking):** the per-response email-visibility loop (`showEmails` +
  `User.Email` stripping, entangled with the `shouldKeep` DB call) and `updateEventResponse`'s
  signed-in / GROUP / sign-up-form branches are still uncovered.

- [x] **B2 · Cover access-control end to end.** `S` · **P1** — **DONE 2026-07-22 (CI-green).**
  New `routes/access_control_test.go` exercises the invite-only gate end to end (not just the
  `models/roles.go` unit level). `AuthRequired` is driven through a real gin engine + session
  middleware, with the session cookie minted via a `test-login` helper route (the way a real sign-in
  would): not-signed-in → 401; **struck-off member** (valid session, email no longer allowlisted) →
  401 (a sentinel allowlist entry keeps the list non-empty so `IsAccessAllowed` can't fail open);
  allowlisted member → 200. `CanInviteRequired` (pure): guest → 403 + aborted, member → passes.
  Handler role check: a **guest POSTing `/events` → 403** before any event is built. DB-backed cases
  gate on `requireDB` (skip without Mongo; run in CI).
  *(Fixed one CI-caught bug: `primitive.DateTime` binds from an RFC3339 string, not epoch-ms.)*

- [x] **B3 · Frontend: cover the grid/availability math extracted in A5.** `M` · **P2** — **DONE
  2026-07-22 (CI-green).** The A5 mixin still held the geometry as `this`-dependent methods, so first
  extracted the pure computational core into `schedule_overlap/gridGeometry.js` (`clampRow`,
  `clampCol`, `getRowColFromXY` as pure functions of their inputs) and made `dragGridMixin` delegate
  to it (exact transcription → behavior unchanged; dropped the now-unused `clamp` import). Added
  `gridGeometry.test.js` — 9 vitest cases covering row/col clamping in both daysOnly and time-grid
  views, `columnOffsets`-based column derivation, past-last-offset clamping, and the split-gap row
  adjustment. Frontend suite 23 → 32 tests. **Remaining frontend test gap:** the availability
  fetch/format/animate logic (now in `availabilityMixin`/`currentAvailabilityMixin`) is still
  `this`-dependent and would need the same extract-pure-core treatment to be unit-testable.

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

---

## PART D — Rebranding (remove all `schej-it` / `schej.it` and `timeful.app` references)

> **Supersedes the CLAUDE.md caveat** ("internal identifiers … still use the old name — leave them
> alone unless rebranding is the explicit task"). This item *is* that explicit task. Scope from a
> 2026-07-22 survey: **~290 `schej*` matches across ~50 files** (234 `schej.it`, 44 `schej-it`, plus
> bare `schej`) and **~69 `timeful*` matches**. Split into a safe/mechanical tier and a
> dangerous/infra tier — **do NOT treat this as one find-replace.**

- [ ] **D0 · Decide the target name(s) first.** `S` · **P3** — **prerequisite / open decision.**
  Nothing below can start until we pick concrete replacements for each identifier class, because they
  have *different* constraints: the **Go module path** (`schej.it/server`), a **public domain** (for
  CORS/nginx/email links), the **Mongo DB name**, the **GCP project id**, and the **brand string**
  ("Schej.it"/"Timeful" shown to users). The fork is "The Fellowship" — but e.g. a Go module path and a
  GCP project id can't contain spaces/caps, so each needs its own decided value. Record them here once
  chosen.

- [ ] **D1 · Safe code/brand renames (mechanical, CI-gated).** `M` · **P3**
  - **Go module path** `schej.it/server` → new path: edit `server/go.mod` `module` directive and the
    `schej.it/server/...` import prefix in **59 `.go` files**. Purely mechanical but touches nearly
    every backend file — **no local Go toolchain, so gate strictly on Backend CI** (do it as one
    dedicated commit so a red build is easy to bisect/revert). `swag init` will also regenerate
    `docs/` with the new path.
  - **User-facing brand strings**: "Timeful"/"Schej.it" in the frontend (`frontend/` — titles, OG
    meta in `main.go`'s NoRoute handler, `package.json` name) and in **email templates / listmonk**
    copy. Cosmetic; low risk once the brand name is decided.
  - `kill-sw.js`, `maintenance_page/`, `server/README.md`, `.env.template` comments — string swaps.

- [ ] **D2 · Dangerous / infra-coupled references (NOT a code-only change).** `L` · **P3**
  These are tied to live infrastructure and data — changing the string in code without the matching
  infra change will break prod. Each needs a coordinated migration, ideally done by the human with VM
  access:
  - **Mongo DB name `schej-it`** (`db/init.go`, `mongodump/mongorestore` commands in docs): renaming
    the database is a **data migration** (`mongodump` old → `mongorestore` into new name → cutover),
    not a code edit. Sequence with a deploy window.
  - **GCP Cloud Tasks project `schej-it`** (`services/gcloud/tasks.go`:
    `projects/schej-it/locations/us-central1/queues/SendReminderEmail`): this is a real **GCP project
    id**. It can only change if the project itself is renamed/recreated — coordinate with whoever owns
    the GCP project or leave as-is.
  - **Domains/CORS/nginx**: `main.go`'s default CORS origins (see also A14), `deploy_scripts/
    nginx_configs/*`, `deploy_scripts/reboot_server_if_down.sh` — must match the **real deployed
    domain**; only change alongside the actual DNS/hosting cutover.

- [ ] **D3 · Historical migration scripts — leave or annotate, don't rename.** `S` · **P3**
  `server/scripts/*` account for ~13 of the `schej` matches but intentionally **don't compile** (they
  reference outdated models — noted in `backend-ci.yml`) and are run-once history. Renaming identifiers
  there is pointless and risks implying they're live. Overlaps with A15 (document the dated folders);
  handle there, not as part of the rename.
