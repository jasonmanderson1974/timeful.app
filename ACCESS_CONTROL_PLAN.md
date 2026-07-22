# The Fellowship — Invite-Only Access Control Plan

Multi-day, phased plan to make the self-hosted instance **invite-only** (no open sign-ups).
Companion to `REDESIGN_PLAN.md`. Memory: `project-fellowship-access-control`.

> Resume tip: read **§1 Decisions** and **§5 Status** first. The OTP login flow already
> exists in the codebase — most of this is *wiring + gating*, not building from scratch.

---

## 1. Decisions (finalized 2026-06-20 with user)

- **Scale:** ~30–40 users total (≈12 men + wives). Many — especially wives — likely have **no
  Google account**, so the universal path must not require Google.
- **Primary auth = passwordless EMAIL CODE (OTP), gated by an allowlist.**
  Enter email → server checks the **`allowlist`** → on it: email a 6-digit code → verify → in.
  Not on it: show *"The Fellowship is by invitation only"*, **no account created, no code sent**.
- **Google OAuth = optional second door**, checked against the **same allowlist** (kept because it
  powers calendar autofill for those who have Google). Can be dropped later if user wants email-only.
- **Email transport = plain Gmail account via SMTP + app password** (`smtp.gmail.com`). Low volume
  (login codes for ~40 people) is well within Gmail's ~500/day limit. "From" = the gmail address.
- **Guest (no-login) event responses: LEFT OPEN.** Members share Gathering links internally; we are
  gating *account sign-in*, not link responses.
- **Member management:**
  - **`canInvite` role** on users — designated members (the user + any they pick) can add emails to
    the allowlist ("invite"). Regular members cannot.
  - **Self-service contact info** — Settings page gains **email + phone** editing (name already there).
  - **Email change auto-adds to the allowlist** so a member who changes their email keeps access.
    Subtlety (accepted): this lets a member allowlist any address by setting it as their own — fine
    for a trusted ~40-person club; the explicit **Invite** action is the intended way to add others.
- **Branching:** build on a feature branch (off `redesign/fellowship-phase1`, the current deployed
  branch; or off `main` once the redesign merges). Backend = Go ⇒ compiles on the VM only
  (dev box has no Go/Docker) — watch the Docker Hub `golang` pull rate limit (see REDESIGN_PLAN §5).

## 2. What already exists (don't rebuild)

- **OTP login flow — frontend:** `frontend/src/views/SignIn.vue` (enter email → check-email →
  onboarding name for new users → send code → verify). Already themed.
- **OTP login flow — backend:** `server/routes/auth.go` handlers `checkEmail` (`/auth/otp/check-email`),
  `sendOtp` (`/auth/otp/send`), `verifyOtp` (`/auth/otp/verify`); `generateOtpCode()`;
  `models.OtpCode`; `db.OtpCodesCollection`.
- **Google OAuth:** working (`signInHelper` in `auth.go`; CLIENT_ID/SECRET configured). Calendar autofill.
- **Gap to fix:** `sendOtp` currently sends the code via **Listmonk**
  (`listmonk.SendEmailAddSubscriberIfNotExist`, `LISTMONK_OTP_EMAIL_TEMPLATE_ID`) — NOT configured on
  this instance. Swap to Gmail SMTP. (`server/utils/mail_utils.go` exists — check before adding net/smtp.)

## 3. Data model

- **`allowlist` collection** (new): `{ email (lowercased, unique), addedBy, addedAt }`. The gate.
- **`User`** (`server/models/user.go`): add **`canInvite bool`**; ensure email + phone are user-editable.
- Matching is by **email, lowercased/trimmed**, exact. (Google's verified email must equal the
  allowlisted email — the key cross-check; see §6 gotchas.)

## 4. Build phases

### Phase A — Allowlist + the gate  *(DONE 2026-06-20, deployed `5853a06`, verified live)*
- [x] `allowlist` collection (unique email index, `db/init.go`) + `db/allowlist.go` accessors
      (IsAllowlisted, **IsAccessAllowed** [fail-open when empty], Add/Remove/GetAllowlist).
- [x] `errs.NotInvited` + sentinel `errs.ErrNotInvited`.
- [x] Gate `checkEmail` → `{invited:false}` (200) when blocked; `sendOtp` → 403; OAuth
      `signInHelper` returns `ErrNotInvited` before account creation; `signIn`/`signInMobile`
      callers return 403 `not-invited`.
- [x] `SignIn.vue` shows "by invitation only" on the email path + via `?notInvited=1` query;
      `Auth.vue` bounces a not-invited Google login to `/sign-in?notInvited=1`.
- [x] VERIFIED live: empty list ⇒ fail-open (no lockout); add 1 email ⇒ enforces (listed=invited,
      others=invited:false); cleared ⇒ dormant. **Allowlist is currently EMPTY (dormant).**

### Phase B — Gmail SMTP for the codes  *(DONE 2026-07-21, code complete; awaits VM env + deploy)*
- [x] Replaced the Listmonk send in `sendOtp` with **`utils.SendEmail`** (existing Gmail SMTP helper,
      `smtp.gmail.com:587` via gomail). **Reused the existing env vars** `GMAIL_APP_PASSWORD` +
      `SCHEJ_EMAIL_ADDRESS` instead of inventing new `SMTP_*` names — the helper already existed.
- [x] Fellowship-themed HTML code email (`buildOtpEmailBody` in `routes/auth.go`): dark leather/brass,
      inline styles, 10-min-expiry copy. Subject: "NNNNNN is your Fellowship sign-in code".
- [x] Dropped `strconv` + `LISTMONK_OTP_EMAIL_TEMPLATE_ID` from the OTP path; `.env.template` updated
      (GMAIL vars now marked *required for email OTP auth*; Listmonk OTP template marked no-longer-used).
- [ ] **Manual (user, on VM):** set `GMAIL_APP_PASSWORD` + `SCHEJ_EMAIL_ADDRESS` in `server/.env`, then
      `docker compose up -d --build server`. Send a test code to confirm delivery (check spam).

### Phase C — Inviter role + admin UI
- [ ] `User.canInvite` flag; protect invite/admin endpoints by it.
- [ ] Endpoints: list allowlist, add email (invite), remove email, toggle `canInvite` on a user.
- [ ] A simple **member-admin page** (Fellowship-themed) visible only to `canInvite` users.

### Phase D — Self-service contact info
- [ ] Settings page: edit **email + phone** (name already editable).
- [ ] On email change: update `user.email` + **add new email to the allowlist** (keep access).
      Decide whether to also remove the old email (likely keep, or replace — TBD when building).

### Phase E — Seed
- [ ] Seed script (`server/scripts/` dated-folder convention) to load the initial ~30–40 emails
      and mark the initial `canInvite` admins.

## 5. Status / progress  (update each session)

- 2026-06-20: Design finalized + documented.
- 2026-06-20: **Phase A DONE & deployed** (`5853a06`) — allowlist + sign-in gate, fail-open while
  empty, verified live. Allowlist currently EMPTY ⇒ gate dormant, site behaves as before.
- 2026-07-21: **Phase B code DONE** — OTP codes now sent via `utils.SendEmail` (Gmail SMTP), themed
  HTML email, Listmonk OTP dependency removed. Reused `GMAIL_APP_PASSWORD`/`SCHEJ_EMAIL_ADDRESS`.
  Awaits: user sets those two vars in `server/.env` on the VM + rebuild/deploy + delivery test.
- Phases C–E: ☐ not started. **Next after B ships = Phase C** (inviter role + admin UI).

## 6. Needs-from-user / manual steps

- [ ] **Gmail app password:** on a plain @gmail.com account, enable 2-Step Verification, then create an
      **App Password** (Google Account → Security → App passwords). Add `SMTP_USER`(the gmail address) +
      `SMTP_PASS`(the 16-char app password) + `SMTP_FROM` to `server/.env` on the VM (like CLIENT_SECRET —
      never paste the password in chat). I'll give exact var names when Phase B lands.
- [ ] **Initial allowlist:** the ~30–40 emails (email required; name optional).
- [ ] **Initial admins:** which emails get `canInvite` (the user + anyone else).

## 7. Gotchas / decisions to revisit

- **Email-match risk:** the allowlisted email must equal the email the person actually signs in with
  (their Google email, or the exact address they enter for the code). If someone's Google email differs
  from what we have, they use the email-code path with their allowlisted address.
- **Gmail SMTP:** needs 2FA + app password; "less secure apps" is gone. Watch the daily send cap (~500;
  fine here). Deliverability from a plain gmail is usually OK at this volume but codes could land in spam
  — tell members to check spam on first login.
- **Backend deploys:** Go changes compile on the VM; if `docker compose up --build` hits the Docker Hub
  `golang` 429 pull limit, build only what changed / `docker login` on the VM (see REDESIGN_PLAN §5).
- Old Listmonk OTP env/template references can be removed once Gmail SMTP is in.
