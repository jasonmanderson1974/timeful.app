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
  - **Sign-up-block child split DONE 2026-07-22 (Tier 2):** the per-day grid overlay (dragged block +
    saved blocks + blocks-to-add) → `SignUpBlocksOverlay.vue` (presentational; state stays in the
    parent since dragGridMixin shares it; parent handles `block-click`). Runtime-verified end-to-end
    in headless Chromium: created a sign-up event, dragged a slot out (dragged branch renders live),
    saved, and as guest clicked the block → Join-slot dialog. **A5 is now DONE** — remaining
    ScheduleOverlap size (~3.1k lines) is inherent grid complexity; further splits optional.
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

- [x] **A7 · Consolidate date libraries (drop `moment`, ideally `spacetime`).** `S` — **DONE
  2026-07-22.** Both removed from package.json + lockfile (`npm ci` verified). By removal time
  **neither** had any import left in the frontend — moment was always dead, and spacetime's single
  TimezoneSelector usage had already been rewritten to dayjs (only a stale comment remained; fixed).
  dayjs is now the sole date lib. Browser-verified: timezone selector switches Pacific → Eastern,
  grid re-renders clean.

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
  that's the backlog to work down before flipping the steps to blocking.
  - **Backlog pass 2026-07-22 (later same day):** frontend eslint **errors 34 → 0** (all 34 fixed,
    incl. a real DatePicker `!= NaN` bug and an in-place Vuex sort in Dashboard.orderedFolders;
    screens browser-verified) → **frontend `Lint` step now BLOCKING** (fails on errors; ~67 warnings
    remain and still pass — mostly `no-unused-vars`, `vue/no-unused-components`, 6
    `vue/no-mutating-props` that need real design fixes). **`go vet` now BLOCKING** (clean; also
    fixed a broken `microsoftgraph_test.go` signature it caught). **golangci-lint was silently dead**
    — the pinned v1.61.0 binary (go1.23) refused to load the Go 1.25 module and continue-on-error
    hid it; upgraded to v2.12.2 with a migrated v2 config + package-list scripts exclusion, which
    surfaced the real backend backlog: **112 issues (98 errcheck, 11 staticcheck, 2 ineffassign,
    1 govet)** — stays warnings-first until worked down. Note: staticcheck SA1019 flags the CFB
    encryption in `utils/utils.go` as deprecated — do NOT swap ciphers casually; stored data is
    encrypted with it (needs a migration plan).

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

- [x] **A11 · Trim remaining large components.** `M` — **DONE 2026-07-22 (all slices browser-verified
  via the headless-Chromium loop; see per-item notes).**
  After A5: `Event.vue` (now 1,776), `NewEvent.vue` (1,010), `RespondentsList.vue` (844),
  `NewSignUp.vue` (827). Candidates for extracting presentational children and moving pure helpers
  into `utils/`.
  - **Done:** removed `Event.vue`'s dead `interceptPluginResponses` debug method (listener was
    commented out) → 1,815 → 1,776.
  - **Done 2026-07-22:** `pluginMessagesMixin` extracted (`components/event/pluginMessagesMixin.js`
    — `handleMessage`/`setSlots`/`getSlots`, 567 lines, verbatim; orphaned + pre-existing unused
    imports pruned) → Event.vue **1,175**. Plugin API runtime-verified via headless Chromium:
    get-slots/set-slots round-trip on a real event (guest write + readback + UI), no console errors.
  - **Done 2026-07-22 (Tier 2 child splits, both browser-verified):** `EventHeader.vue` (title/chips/
    date + 8-event action-button block; helpDialog moves in) and `EventBottomBar.vue` (phone action
    bar + mobile button-text computeds; 7 events) → Event.vue **1,006** (was 1,815 pre-A11).
  - **Done 2026-07-22 (final Tier 2 splits, both browser-verified):** `NewEventAdvancedOptions.vue`
    (Advanced-options panel content, 6 `.sync`-bound fields; verified by setting every field through
    the UI and confirming the created event's API payload) → NewEvent.vue **887** (was 1,010); and
    `ExportCsvMenu.vue` (kebab menu + dialog + whole CSV build/download feature; both export formats
    verified by downloading and checking content) → RespondentsList.vue **677** (was 844).
    **A11 complete** — remaining component sizes are inherent feature complexity; further splits
    would be churn, not payoff.
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

- [x] **A14 · Prune legacy CORS origins.** `S` — **DONE 2026-07-23 (folded into D1).** Once D0 settled
  the domain, `main.go`'s fallback default became
  `https://gathering.sirthomasfoolery.com,http://localhost:8080` (was `schej.it`/`www.schej.it`/
  `timeful.app` set). Prod still sets `CORS_ORIGINS` explicitly; this is just the sane fallback now.

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

- [x] **C1 · RSVP / attendance tracking for a *confirmed* gathering.** `M` — **DONE 2026-07-23
  (CI-green; backend build/vet/tests + frontend build/lint/tests pass; RSVP endpoints +
  RSVP→reminder pipeline verified live against local Mongo).** Adds a 3-state RSVP
  (Going / Maybe / Can't make it) to any event with a confirmed gathering (**[C2]**'s
  `scheduledEvent`), a live headcount + roster, and wires the result into C2's reminder targeting.
  - **Storage** (`models/event.go`): a new `Rsvps map[string]*Rsvp{status,name,email,userId,
    respondedAt}` on the Event doc, keyed by guest-name / user-id — mirrors `SignUpResponses`.
    **Not** the `Attendee` model (that's group-invite-specific: Email + Declined only). No new
    collection / migration.
  - **Endpoints** (`routes/event_responses.go`): `POST /events/:eventId/rsvp` (status +
    guest/signed-in identity, keyed like `updateEventResponse`; signed-in RSVPs backfill
    name/email from the account) and `DELETE …/rsvp` to un-RSVP. Requires a confirmed gathering
    (400 `gathering-not-scheduled`) and validates the status enum.
  - **C2 integration** (`services/reminders`): `processDueReminders` now prefers RSVPs — new
    `collectRsvpRecipientEmails` reminds `going`+`maybe` (decliners excluded); if **no** RSVP
    exists yet it falls back to all availability respondents, so reminders keep working before
    anyone RSVPs.
  - **Frontend**: `GatheringRsvp.vue` (shown when `event.scheduledEvent` exists) — headcount
    ("N going · M maybe · K can't"), a roster grouped by status (visible to all — it's a club),
    and 3 RSVP buttons highlighting the viewer's choice (re-click to clear). Signed-in users
    RSVP directly; guests enter a name first (same trust model as guest availability). Mounted in
    `Event.vue` between the description and the calendar; `EventService.setRsvp/clearRsvp` persist
    then `refreshEvent`.
  - **Tests**: `collectRsvpRecipientEmails` unit test (going/maybe in, no out, dedupe, signed-in
    lookup) + a DB-gated integration test (RSVPs present → only going+maybe emailed, availability
    responders ignored). Live-verified the full endpoint flow (pre-schedule 400 → RSVP →
    change → un-RSVP).
  - **Independently re-verified live on prod 2026-07-23 (this machine, via headless Chromium
    against gathering.sirthomasfoolery.com):** BOTH UI paths end-to-end on a throwaway
    scheduled event — **guest** (name field → Going, roster "Going: <name>") and **signed-in**
    (no name field, identity backfilled → "Going: Jason Anderson"), plus the [C4] plus-one
    stepper and the decliner-forces-0-guests rule, status changes, and clear-on-re-click. All
    assertions green, no console errors; test events deleted afterward. (See PART E for two
    incidental findings this surfaced.)
  - **Non-goal:** guest plus-ones/spouse headcount is **[C4]** — the `Rsvp` struct leaves room
    for a `GuestCount` without migration.

- [x] **C2 · Automated pre-gathering reminder emails.** `S–M` — **DONE 2026-07-23 (CI-green;
  backend build/vet/tests + frontend build/lint/tests all pass; API round-trip + DB-backed
  pipeline verified against a local Mongo).**
  The TODO's premise turned out stale: (1) **no confirmed time was ever persisted** — the
  "Schedule event" button only opened a Google/Outlook *template URL* and wrote nothing back;
  (2) the **Cloud Tasks + listmonk path is dead on this fork** (all `# optional`, points at
  the upstream's GCP project `schej-it`; OTP already moved to Gmail SMTP). So instead of
  extending that path, the feature persists the locked-in time and sends via the fork's real
  mail transport (Gmail SMTP), on a **self-contained in-process scheduler** — no GCP/listmonk.
  - **Persist the gathering** (`POST /events/:eventId/schedule`, owner-gated like `editEvent`):
    reuses the existing (previously-unwritten) `Event.ScheduledEvent *CalendarEvent` for the
    time, plus a new `GatheringReminder{Enabled, LeadTimeHours, Timezone, SentAt}` struct
    (`models/event.go`). `scheduled:false` cancels (unsets both). Lead time clamped 1..168h,
    default 24; `SentAt` reset to nil on every (re)schedule so it re-arms.
  - **Scheduler** (`services/reminders/`): a ticker goroutine started in `main.go`
    (`REMINDER_SCHEDULER_INTERVAL`, default 5m) that no-ops with a log if the Gmail vars are
    unset (mirrors `gcloud.InitTasks`). Each tick: `db.GetEventsWithPendingReminders()` →
    Go-side lead-time window (`isReminderDue`) → recipients = all availability respondents with
    an email (`collectRecipientEmails`: guest `Response.Email`, else signed-in via
    `db.GetUserById`, deduped) → inline-HTML email (Fellowship style, time formatted in the
    saved tz) → **mark `SentAt` regardless of per-send failures** so it never loops. Sender is
    injected (`SendFunc`) for testability.
  - **Frontend**: `EventService.setScheduledEvent`; `ScheduleOverlap.confirmScheduleEvent` now
    persists (keeps opening the organizer's own calendar URL) + `cancelGathering`; `ToolRow`
    gains a reminder toggle + lead-time select in the Schedule menu and a "Gathering set"
    indicator (shows time + reminder summary) with Reschedule / Cancel actions. Mobile
    (`EventBottomBar`) uses the defaults (reminder on, 24h).
  - **Tests**: `services/reminders` pure unit tests (`isReminderDue`, `collectRecipientEmails`
    guest/signed-in/dedupe, `buildReminderEmail` tz + UTC fallback) + a DB-gated integration
    test (`requireDB`/`TestMain`) driving the whole pipeline with a mock `SendFunc`.
  - **Notes / non-goals:** single-VM scheduler (no distributed lock — fine for this fork);
    recipients = respondents-with-email until **C1 (RSVP)** lands, then swap
    `collectRecipientEmails` for the confirmed-attendee list. **Swagger `docs/` regenerated
    2026-07-23** (the dedicated docs-regen pass): resynced with the D1 `@title` ("The Fellowship
    API") + all the new routes (rsvp/ics/comments/schedule/…). Requires
    `swag init --parseDependency --parseInternal` — a bare `swag init` aborts on the allowlist
    models' `primitive.DateTime`; the flags resolve it (guidance in CLAUDE.md).

- [x] **C3 · "Add to calendar" / `.ics` export for confirmed gatherings.** `S` — **DONE
  2026-07-23 (CI-green; backend build/vet/tests + frontend build/lint/tests pass; live .ics
  download verified against local Mongo).** Builds directly on **[C2]**'s persisted
  `scheduledEvent`: a universal, no-OAuth "add to calendar" that works for everyone (incl.
  spouses without a Google account).
  - **Generation** (`services/calendar/ics_generate.go` — the mirror of `ics_calendar.go`'s
    parsing, same `emersion/go-ical` lib): `GenerateEventICS(event)` builds a `VCALENDAR` +
    one `VEVENT` from `event.ScheduledEvent` — stable `UID` (`{id}@timeful.app`), UTC
    DTSTART/DTEND, SUMMARY, DESCRIPTION (+ event link), URL, `STATUS:CONFIRMED`,
    `METHOD:PUBLISH`. Errors if the event has no confirmed gathering.
  - **Endpoint** (`routes/events.go`): public `GET /events/:eventId/ics` — no auth so any
    invitee can add it; returns `text/calendar` with `Content-Disposition: attachment;
    filename="<slug>.ics"`. 404 (`gathering-not-scheduled`, new `errs` code) until a time is
    locked in.
  - **Reminder email** (`services/reminders`): the C2 pre-gathering email now carries an
    **"Add to calendar"** button pointing at `/api/events/{id}/ics` — closes the loop for the
    no-Google-account members right in the reminder.
  - **Frontend**: an "Add to calendar" chip on `EventHeader.vue` (visible to **everyone** who
    opens the event once it's scheduled) + an item in the owner's `ToolRow` "Gathering set"
    menu; both `:href` the `.ics` URL (`serverURL` + `/events/{id}/ics`), a plain download —
    no JS, works for guests.
  - **Tests**: `ics_generate_test.go` (structure/UTC formatting/escaping + no-gathering error).
    Live-verified end to end: 404 before scheduling → 200 with correct headers + a valid,
    comma-escaped `VEVENT` after. **Note:** `createEvent` still doesn't accept a `description`
    (only `editEvent` does) — pre-existing, unrelated; the generator includes it when present.

- [x] **C4 · Plus-one / guest handling on responses.** `S–M` — **DONE 2026-07-23 (CI-green;
  backend build/vet/tests + frontend build/lint/tests pass; plus-one persist + clamp verified
  live).** A small extension of **[C1]**: a respondent can indicate how many extra people
  (spouse/guests) they're bringing, so the headcount is accurate for the "≈12 men + wives" reality
  without every spouse needing an account.
  - **Model** (`models/event.go`): added `GuestCount int` to `Rsvp` — the number of *additional*
    people (headcount for an RSVP = 1 + GuestCount). The room the C1 struct left, now filled; no
    migration.
  - **Endpoint** (`routes/event_responses.go`): `rsvpToEvent` accepts `guestCount`, clamped by
    `clampGuestCount` (0..20; forced to 0 for `no` — decliners can't bring guests).
  - **Frontend** (`GatheringRsvp.vue`): a "Bringing guests: [− N +]" stepper that appears once
    you're going/maybe and re-submits the RSVP on change; the headcount now reads
    "N going (+G) · M maybe (+g) · K can't" and the roster shows "Alice (+2)".
  - **Tests**: `clampGuestCount` unit test (negative→0, over-max→20, decliner→0). Live-verified:
    going +2 / maybe +1 persist; `no +5`→0; `going +999`→20.

### P2 — Strong quality-of-life

- [x] **C5 · Recurring gatherings.** `M` — **DONE 2026-07-23 (CI-green; backend build/vet/tests +
  frontend build/lint/tests pass; schedule-with-recurrence + .ics RRULE verified live against a
  local server + Mongo; auto-advance covered by DB-gated integration tests).** Rather than a
  spawn-a-copy model (the TODO's `duplicateEvent` hint), this makes a *single* confirmed gathering
  repeat — one permanent event link + calendar series — which fits "First-Friday poker night"
  better than minting a new event/URL each cycle. Builds on **[C2]**'s `scheduledEvent` +
  reminder pipeline and **[C3]**'s `.ics`.
  - **Model** (`models/event.go`): `GatheringRecurrence{Frequency, Until}` on the Event (paired with
    `ScheduledEvent`; nil = one-off). Frequencies weekly / biweekly / monthly. No migration. Pure
    methods live with the type (unit-tested): `IsRecurring`, `Step` (monthly uses `addMonthsClamped`
    so Jan 31 → Feb 28/29, not Mar 3), `NextOccurrenceAfter` (skips a long outage — jumps to the next
    occurrence after `now`, doesn't replay missed ones), and `RRULE` (RFC 5545 string).
  - **Schedule endpoint** (`routes/events.go`): `scheduleEvent` accepts `recurrenceFrequency`
    (+ optional `recurrenceUntil`), validates the enum (bad value → 400), stores/clears
    `gatheringRecurrence`; `none` and cancel both unset it.
  - **.ics** (`services/calendar/ics_generate.go`): emits an `RRULE` (`FREQ=WEEKLY` /
    `FREQ=WEEKLY;INTERVAL=2` / `FREQ=MONTHLY`, + `UNTIL`) via `Props.Set` (not `SetText`, which would
    tag it `VALUE=TEXT` — wrong for a RECUR value), so a single "Add to calendar" covers the whole
    series in members' calendars.
  - **Auto-advance** (`services/reminders`): the scheduler now rolls a recurring gathering forward to
    its next future occurrence once the current one has *ended* — clearing that cycle's RSVPs (fresh
    headcount) and re-arming the reminder (`sentAt` unset). Guarded by a conditional `AdvanceGathering`
    update (keyed on the expected current start) so concurrent ticks can't double-advance; stops once
    the next occurrence would fall after `Until`. **Decoupled from email**: `StartReminderScheduler`
    now always runs the advance pass (only the *send* is gated on Gmail creds), so the event page shows
    the next occurrence even on an email-less instance.
  - **Frontend**: recurrence selector ("Does not repeat / Weekly / Every 2 weeks / Monthly") in the
    `ToolRow` Schedule menu (`.sync` through `ScheduleOverlap.confirmScheduleEvent`), a "Repeats …"
    line in the owner's "Gathering set" summary, and a **"Repeats …" chip on `EventHeader`** so
    *everyone* (not just the owner) sees the cadence next to "Add to calendar".
  - **Tests**: `models/event_recurrence_test.go` (Step, monthly clamp incl. leap-year + year boundary,
    skip-outage, RRULE incl. UNTIL), `ics_generate_test.go` (RRULE present for recurring / absent for
    one-off), and DB-gated `recurrence_integration_test.go` (rolls forward + clears RSVPs + re-arms +
    idempotent; respects `Until`; skips a still-ongoing occurrence).
  - **Swagger `docs/` regenerated** (pinned `swag@v1.16.1 --parseDependency --parseInternal`):
    documents the new schedule params + `GatheringRecurrence` model. The regen also normalized some
    pre-existing `primitive.DateTime` string/integer drift in the committed baseline (it was already
    internally inconsistent) — expected, not from this change.
  - **Known limitations (v1, documented in code):** monthly recurrence on day 29–31 clamps to the
    month's last day and can compound across short months (fine for a club — meetings fall on normal
    days), which may diverge slightly from a strict RRULE reader; no "nth-weekday" (e.g. "2nd
    Saturday") rule; comments accumulate across occurrences (single rolling event). Non-goal: per-
    occurrence history — that's **[C10]** (The Chronicle).

- [ ] **C6 · Venue / activity poll (not just time).** `M`
  Extend the availability-poll concept to "where / what" — a lightweight multiple-choice poll so the
  club can vote on venue or activity. Overlaps with the sign-up-block UI already built.

- [x] **C7 · Per-gathering discussion thread / comments.** `M` — **DONE 2026-07-23 (CI-green;
  backend build/vet/tests + frontend build/lint/tests pass; post/edit/delete verified live).** A
  discussion thread on every event for coordinating details ("I'll bring cigars", "parking's out
  back"), keeping it off scattered group texts.
  - **Decisions (confirmed):** members **and** guests (by name, same trust model as RSVP/
    availability — guest posting stays open on enforced instances); **full** management (edit +
    delete-own, owner deletes any).
  - **Storage:** a dedicated `comments` collection (mirrors `eventResponses` — many-per-event,
    append-heavy), keyed by `eventId`; `models/comment.go` + `db/comments.go` + registered in
    `db/init.go`. `getEvent` attaches `event.Comments` (like it does `ResponsesMap`/`Attendees`),
    so the existing `refreshEvent()` surfaces them with no extra fetch.
  - **Endpoints** (`routes/comments.go`, registered in `InitEvents`): `POST …/comments`,
    `PUT …/comments/:id` (own-only), `DELETE …/comments/:id` (own OR event owner). Text trimmed +
    capped at 2000; empty → 400. Reused the guest/signed-in key helper — renamed `rsvpKey` →
    `responderKey` (generic) and shared it.
  - **Frontend:** `EventComments.vue` — thread with author/time/"edited", inline edit + delete
    controls on your own (delete also on any when you're the owner), and a composer (members post
    directly; guests enter a name first, like `GatheringRsvp`). Mounted below the calendar in
    `Event.vue`; `EventService.addComment/editComment/deleteComment` persist then `refreshEvent`.
  - **Tests:** `sanitizeCommentText` unit test + DB-gated integration (guest post→appears in
    getEvent; edit sets `updatedAt`; other-guest delete→403; owner deletes another's). Live-verified
    the full post/edit/delete/authz flow.
  - **Non-goal (v1):** no new-comment notifications (email/web-push) — follow-up tying into
    **[C2]**/**[C8]**. Optional later polish: enrich member comments with account avatars at read time.

- [ ] **C8 · Web push notifications for "new gathering" / "you were invited."** `M` —
  **DEFERRED 2026-07-23 (premise was wrong — reassess value before picking up).**
  **Correction:** the original note ("a service worker is already registered … client half partly
  there") is **false**. `git log` shows `f857320 "remove pwa"` (the SW/PWA was *deliberately
  removed*) then `e8deeee "Create kill-sw.js"` (a kill switch that *unregisters* the SW from clients).
  `main.js` registers nothing; `register-service-worker` is an unused dependency. So there is **no
  active service worker** — C8 would **reintroduce** one, reversing a deliberate decision (a bad SW
  can brick the app for all members — the likely reason it was pulled).
  - **If revived, do it safely:** a **push-only SW** (`push` + `notificationclick` handlers **only**,
    NO fetch interception / caching) to avoid the caching footgun that got the PWA removed.
  - **Needs infra:** a VAPID key pair — private key on the VM (`server/.env`, like
    `GMAIL_APP_PASSWORD`), public key baked into the frontend build. A Go webpush lib
    (e.g. `SherClockHolmes/webpush-go`) + a `pushSubscriptions` store + subscribe/unsubscribe routes.
  - **iOS gap:** Safari delivers web push only to home-screen-installed PWAs (iOS 16.4+) — most of the
    club's iPhone users won't get pushes unless they install the site. **[C2]'s email reminders already
    cover the "gathering is set" need for everyone incl. iOS**, which is why value here is now
    questionable. Reassess whether it's worth the SW risk before building.

- [x] **C9 · Sign-up-block capacity + waitlist.** `S` — **DONE 2026-07-23 (CI-green; backend
  build/vet/tests + frontend build/lint/tests pass; capacity+waitlist verified live).** The
  `SignUpBlock.Capacity` field was only *displayed* (client hid the join link when full) and **not
  enforced server-side**, and there was no waitlist. Now capacity is authoritative and overflow is
  waitlisted.
  - **Model** (`models/event.go`): added `WaitlistBlockIds []ObjectID` to `SignUpResponse`
    (`SignUpBlockIds` = confirmed, within capacity; `WaitlistBlockIds` = waitlisted). No migration.
  - **Enforcement** (`routes/event_responses.go`): new `assignSignUpBlocks(event, user, requested)`
    splits requested blocks into confirmed/waitlisted by each block's `Capacity` (nil = unlimited),
    excluding the user's own prior signup from the count and **preserving an already-confirmed spot
    on re-submit**. The sign-up branch of `updateEventResponse` now routes through it — a direct API
    call can no longer overfill a slot.
  - **Frontend**: `resetSignUpForm` (`ScheduleOverlap.vue`) populates a per-block `waitlist`;
    `handleSignUpBlockClick` now lets a **full** block be clicked (→ server waitlists);
    `SignUpBlock.vue` shows a "Waitlist" roster and the join link reads **"+ Join waitlist"** when
    full instead of disappearing. (The compact calendar-tile variant is joinable via the same
    handler; detailed waitlist lives in the list view.)
  - **Tests**: `TestAssignSignUpBlocks` (full→waitlist, unlimited→confirmed, already-confirmed keeps
    spot, fresh user on full block→waitlist). Live-verified: 3 guests → capacity-2 block → first two
    confirmed, third waitlisted.
  - **Follow-up (not v1):** auto-promotion — when a confirmed signup is removed, the earliest
    waitlisted user isn't auto-promoted (they get confirmed on their next re-submit, since a spot is
    now free). Proper promotion needs a signup timestamp/order on `SignUpResponse`; deferred.

### P3 — Nice-to-have / thematic

- [ ] **C10 · Members-only gathering archive ("The Chronicle").** `M`
  The redesign removed the public blog, but an internal, roll-gated history of past gatherings
  (date, attendees, a photo or two) fits the "gentleman's club" theme and gives the Fellowship a
  sense of continuity. Reuses the existing role-gated Fellowship directory work.

- [x] **C11 · Printable / exportable roster of the Fellowship directory.** `S` — **DONE 2026-07-23
  (browser-verified; frontend-only).** Added an **Export** menu to `Fellowship.vue` with **Print /
  PDF** (opens a clean light serif print document — name/role/email/telephone + count + date — in a
  separate window so it doesn't fight the dark app theme; Save-as-PDF from the print dialog) and
  **Download CSV** (`The Fellowship Roster.csv`, quoted/escaped). Both operate on the
  currently-filtered roll (search + Show-guests) — export what you see. No backend change (reuses
  `GET /admin/allowlist`). `MemberAdmin.vue` left as-is (the same roll, admin-managed).

- [x] **C12 · Venue / location on an event.** `S–M` — **DONE 2026-07-23 (CI-green; backend
  build/vet/tests + frontend build/lint/tests pass; venue create/edit + .ics LOCATION verified
  live).** **Correction to the finding:** the existing `models/location.go` / `location_utils.js`
  are **IP-geolocation of the user** (country/city/lat-long), not a venue — and the Event had no
  location field. So this added a real venue field, not just wiring.
  - **Model** (`models/event.go`): `Location *string` on Event (free-text venue/address).
  - **Endpoints** (`routes/events.go`): `location` accepted in `createEvent` + `editEvent`
    (persists via the existing `$set: event` path).
  - **Surfaced everywhere a gathering appears:** `services/calendar/ics_generate.go` sets the .ics
    `LOCATION`; the C2 reminder email shows a "📍 venue" line linking to Google Maps; the schedule
    Google/Outlook calendar URLs pass `&location=`.
  - **Frontend**: new `EventLocation.vue` — inline-editable venue on the event page (mirrors
    `EventDescription`); everyone sees the venue + an **"open in Google Maps"** link, the owner can
    add/edit it. Mounted under the description in `Event.vue`.
  - **Design choice:** keyless — a free-text address + a plain `google.com/maps/search` link (no
    maps-provider API key, which this fork doesn't have). Tests: `.ics` `LOCATION` assertion.
    Live-verified: set at create → persists; edit via PUT → updates; `.ics` carries `LOCATION`.
  - **Follow-up (not v1):** an embedded **static-map image** needs a maps-provider API key
    (Google Static Maps / Mapbox) + config; add if a key is ever provisioned.

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

- [x] **D0 · Decide the target name(s) first.** `S` · **P3 — DECIDED 2026-07-23.**
  - **Go module path:** `schej.it/server` → **`sirtom/server`**.
  - **Public domain:** **`gathering.sirthomasfoolery.com`** (final; for CORS/nginx/email links/ICS UIDs).
  - **Brand string** ("Schej.it"/"Timeful" shown to users) → **"The Fellowship"** (org) / **"The
    Gathering"** (the scheduling/event concept). Stray "Timeful(s)" in code comments → "gathering/event".
  - **`timeful.app` refs:** technical/URL (e.g. ICS UID `@timeful.app`) → `@gathering.sirthomasfoolery.com`;
    brand → The Fellowship/The Gathering.
  - **Contact email:** `contact@timeful.app` → **`sirthomasfoolery24@gmail.com`**.
  - **Remove** the upstream author's cal.com link in `TeamsNotReadyDialog.vue`.
  - **Mongo DB name `schej-it`** → **LEAVE as-is** (internal/invisible; renaming is a risky live-data
    migration for no user benefit). **GCP project id `schej-it`** → **LEAVE** (that Cloud Tasks path is
    dead on this fork). Both are D2 and intentionally not changed.

- [x] **D1 · Safe code/brand renames (mechanical, CI-gated).** `M` · **P3 — DONE 2026-07-23 (two
  commits; verified locally with Go 1.26.5 + eslint/build, both blocking-clean).**
  - **Go module path** `schej.it/server` → **`sirtom/server`**: `go.mod` module directive + the import
    prefix in **74 `.go` files** (the survey's "59" undercounted; the other machine had since added
    comments/waitlist/location routes). Isolated commit. `docs/` doesn't import the module path, so no
    swag regen was needed. (The `no local Go toolchain` caveat is stale — dev box now has Go 1.26.5, so
    this was `go build`/`vet`/`test`-verified locally before push, not just on CI.)
  - **User-facing brand + domain/URL**: OG event title, Swagger title, CORS default origins, email/
    event `baseUrl`, slackbot urls, ICS UID/ProductID, the Settings contact email
    (`sirthomasfoolery24@gmail.com`), removed the upstream cal.com link, dropped dead commented Timeful
    OG block in index.html, `package.json` name, maintenance page, stray code comments, and factual
    doc fixes (CLAUDE.md/DEVELOPMENT.md now say `sirtom/server`).
  - **Follow-ups since done:** the unused upstream `deploy_scripts/` + `deploy.yml` were **deleted**
    (see D2), and the root `README.md` was **rebranded** to The Fellowship (+ orphaned Timeful
    `hero.jpg`/`logo.svg` assets removed), both 2026-07-23.
  - **Intentionally LEFT (see D0/D2):** Mongo DB name `schej-it`, `SCHEJ_EMAIL_ADDRESS` env var, GCP
    project id in the dead Cloud Tasks code, Discord channel names, dead Stripe/paywall log strings.
    The only remaining `schej`/`timeful` hits are exactly these leaves + historical plan docs
    (REDESIGN_PLAN/ACCESS_CONTROL_PLAN, kept as history).

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
  - **Domains/CORS/nginx**: `main.go`'s default CORS origins — **DONE via A14/D1**
    (→ `gathering.sirthomasfoolery.com`). The `deploy_scripts/` + `.github/workflows/deploy.yml`
    ("Deploy Schej") were the **upstream's** screen-based auto-deploy to the old schej.it AWS VPS
    (`workflow_dispatch`-only, targets `secrets.SSH_HOST` this fork doesn't have) — **DELETED
    2026-07-23** rather than rewritten, since this fork deploys via `./deploy.sh` (docker compose +
    Caddy) on the VM. So D2's domain/nginx tail is resolved. **Still open (intentionally):** Mongo DB
    name `schej-it` (data migration) + GCP project id (dead code) — both LEFT per D0.

- [ ] **D3 · Historical migration scripts — leave or annotate, don't rename.** `S` · **P3**
  `server/scripts/*` account for ~13 of the `schej` matches but intentionally **don't compile** (they
  reference outdated models — noted in `backend-ci.yml`) and are run-once history. Renaming identifiers
  there is pointless and risks implying they're live. Overlaps with A15 (document the dated folders);
  handle there, not as part of the rename.

---

## PART E — Security & Access-Control follow-ups

> Companion doc: `ACCESS_CONTROL_PLAN.md`. These came out of the 2026-07-23 live RSVP
> verification (see [C1]); none is a regression in the RSVP feature itself.

- [x] **E1 · Gate `createEvent` / `scheduleEvent` behind auth on enforced invite-only instances.**
  `S–M` · **P2 · DONE 2026-07-23 (decision: gate via the existing `INVITE_ONLY_ENFORCED` flag).**
  Closes the anonymous-write surface without a new config knob or breaking guest flows:
  - `db.AccessControlEnforced()` exported (was private `accessControlEnforced`).
  - New `middleware.AuthRequiredIfInviteOnly()` — delegates to `AuthRequired` when
    `INVITE_ONLY_ENFORCED` is on, else passes through. Applied to `POST /events` (createEvent).
  - `scheduleEvent`: the owner-less-event branch now requires a signed-in caller when enforced
    (the existing owner-check already covers owned events).
  - **Verified live (enforced=true):** anonymous `POST /events` → 401; anonymous schedule of an
    owner-less event → 401; guest availability/RSVP endpoints still reachable (404, not 401).
    Not-enforced (dev/open) preserves the guest create/schedule path. Tests:
    `TestAuthRequiredIfInviteOnly_{NotEnforcedPassesAnonymous,EnforcedBlocksAnonymous}` (DB-free)
    + existing `TestCreateEvent_GuestForbidden` (signed-in guest → 403, unchanged). `.env.template`
    documents the expanded flag semantics.
  - **Left open (intentional, per decision):** RSVP `POST/DELETE …/rsvp` stay guest-open by design;
    if abuse becomes a concern, prefer rate-limiting / a per-event toggle over blanket auth.

  <details><summary>Original finding (for history)</summary>

  · **P2 · OPEN — needs discussion before any change.**
  **Finding:** the invite-only allowlist is enforced *inside* `middleware.AuthRequired()`, which is
  applied **per-route**. `POST /events` (create), `POST /events/:id/schedule`, and the RSVP endpoints
  are **not** behind it, so they're reachable by an **unauthenticated** caller who can hit the API. In
  the 2026-07-23 verification this was used deliberately (the guest path is *supposed* to be open — it
  mirrors guest availability responses), but it also means an anonymous party who reaches the API can
  **create and schedule arbitrary events**. Not exploitable for data disclosure (no member data is
  exposed), but it is an unauthenticated write surface.
  - **Why it's not a simple flip:** guest, no-account interaction is a genuine product requirement
    for this club (guests RSVP and respond to availability by name). Locking *all* of these behind
    `AuthRequired` would break the guest flows. The real question is which **writes** should require a
    member session vs. which must stay open, e.g.:
    - `createEvent` — does an anonymous visitor ever legitimately create an event on this private
      instance? If not, gate it (members create; guests only *respond* to existing events). Watch the
      existing **guest-created event** path (`ownerId == 0`) — some flows rely on it; confirm none are
      user-facing on this fork before gating.
    - `scheduleEvent` — already **owner-gated when the event has an owner**; the gap is
      **owner-less (guest-created) events**, where anyone can schedule. Likely fine to require auth
      unconditionally (only an owner should lock in a time), but confirm against the guest-event UX.
    - RSVP `POST/DELETE …/rsvp` — intentionally open (guest RSVP by name). Leave open; if abuse is a
      concern, prefer rate-limiting / a per-event toggle over blanket auth.
  - **Decide & discuss:** whether to (a) leave as-is (guest-open by design), (b) gate `createEvent`
    +`scheduleEvent` for owner-less events behind `AuthRequired`, or (c) add a config flag
    (`GUEST_EVENT_CREATION_ENABLED`) defaulting to off for invite-only instances. No code change until
    this is settled.

  *(Resolved with option (b), scoped to enforced instances via the existing flag — see above.)*
  </details>

- [x] **E2 · `deleteEvent` only accepts the Mongo `_id`, not the short id.** `S` · **P3 — DONE
  2026-07-23 (backend build/vet + full suite green).** `deleteEvent` now resolves via
  `db.GetEventByEitherId` up front and drives every DB op (responses lookup + the ownerId-scoped
  delete/soft-delete + folder cleanup) off the resolved `_id`; unknown id now **404**s instead of
  400/500. DB-gated tests added (`events_delete_db_test.go`): delete-by-short-id → 200 + gone;
  unknown id → 404.
  <details><summary>original finding</summary>

  Pre-existing, not RSVP-related: `DELETE /events/:eventId` called `primitive.ObjectIDFromHex(eventId)`
  directly, so a **short id** returned **400** (every other event route uses `db.GetEventByEitherId`).
  The real UI always deletes by `_id`, so no user-facing bug — but an inconsistency / sharp edge for
  API scripting. (Surfaced when API-cleaning up an RSVP test event by short id fell back to a direct
  Mongo delete.)
  </details>
