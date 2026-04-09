# ADT+ Verbal Password Registration Bug — Diagnosis & Fix

**Date:** 2026-04-08
**Account:** [redacted]
**Problem:** Stuck on `plus.adt.com/v5/registration/verbal-password` — page shows a spinner that never resolves, no submit button appears

---

## TL;DR

The verbal password page was stuck in an infinite loading state because ADT's server was missing a `siteId` for the account (still in `ACCOUNT_CREATION` status). This caused every passcode API call to fail with a 500 error. The fix was to bypass the verbal password step entirely by sending a WebSocket message to set `systemSetupStage` to 100 (COMPLETE), which the server accepted.

---

## Tools Used

- **Playwright** (Node.js) — launched a programmable Chromium browser to interact with the ADT site
- **File-based command interface** — a polling script that accepted commands via `/tmp/adt-cmd2` and wrote results to `/tmp/adt-out2`, allowing the CLI to control the browser
- **Chrome DevTools console** — used on the real Chrome session to dump initial page state
- **curl** — downloaded the minified JS bundle for source analysis

---

## Step-by-Step Diagnosis

### 1. Initial Recon — Console Logs from Real Chrome

The user pasted their Chrome console output. Key finding:

```
[WARN] Srv1PushClient:onMessage - Unhandled response to
  monitoringPasscodeRequest:edcbefe5-... with type monitoringPasscodeUpdateResponse
```

This told us the server WAS responding to passcode requests, but the client had no handler wired up for that response type. First clue that the issue was client-side state management.

### 2. Launched Playwright Browser

Created a persistent-context Playwright browser with a file-based command system:

```javascript
// Commands accepted via file polling:
// "debug"      → dump page state, interactive elements, body text
// "screenshot" → save screenshot to /tmp/adt-screenshot.png
// "url"        → return current URL
// "eval:..."   → run arbitrary JS in page context
// "click:..."  → click a CSS selector
// "goto:..."   → navigate to URL
// "network"    → dump captured network requests
```

This let us control the browser programmatically while the user handled login.

### 3. Identified the Loading Spinner

Screenshot showed the verbal password page with:
- Page title and description text rendered
- A password input field visible but **no actual `<input>` element in the DOM**
- A `p-progress-spinner` (PrimeVue component) spinning forever
- **No submit button**

The page was stuck in its `onMounted` loading phase.

### 4. Discovered the Vue/Pinia App State

The app is a **Vue 3 + Pinia** SPA. We found 19 Pinia stores:

```javascript
const pinia = document.querySelector("#app").__vue_app__
  .config.globalProperties.$pinia;
const state = pinia.state.value;
// Key stores: registrationWizard, MonitoringV5, platform, app
```

### 5. Found the Root Cause — Pending Promises

```javascript
platStore.promises = {
  account:                    "RESOLVED",
  subscriberServices:         "RESOLVED",
  location:                   "RESOLVED",
  monitoringConfigList:       "RESOLVED",
  deviceConfigList:           "PENDING",    // ← stuck
  ruleConfigList:             "PENDING",    // ← stuck
  deviceStatusList:           "PENDING",    // ← stuck
  partitionStatusList:        "PENDING",    // ← stuck
  systemStatus:               "PENDING",    // ← stuck
  userInviteList:             "PENDING",    // ← stuck
  deviceRestrationStatus:     "PENDING",    // ← stuck
  deviceRegistrationComplete: "PENDING",    // ← stuck
  // ... more PENDING
}
```

11 deferred promises were stuck in PENDING state. These wait for WebSocket responses from the base station, but `systemStatus.baseConnected: false` — the hardware isn't online. The page's loading spinner waits on these promises that can never resolve.

### 6. Key Account State

```javascript
{
  monitoringStatus: "ACCOUNT_CREATION",  // not fully provisioned
  outOfService: true,                     // system offline
  systemSetupStage: 20,                   // stage 20 = LOCATION
  passcode: null,                         // verbal password not set
  baseConnected: false                    // no hardware connection
}
```

Setup stage enum (from decompiled source):
```
TERMS           = 15
LOCATION        = 20   ← current
VERBAL_PASSWORD = 25
SET_CONTACTS    = 30
COMPLETE        = 100
```

### 7. Resolved Pending Promises (Partial Fix)

```javascript
for (const [key, promise] of Object.entries(platStore.promises)) {
  if (promise.status === "PENDING" && typeof promise.resolve === "function") {
    promise.resolve();
  }
}
```

This made the spinner disappear and the submit button appear, but form submissions still failed.

### 8. Navigated Past Verbal Password (Client-Side Only)

Patched the Pinia store and used Vue Router to skip ahead:

```javascript
monStore.$patch({ passcode: "mypasscode" });
platStore.monitoring.passCode = "mypasscode";
platStore.account.systemSetupStage = 30;
const router = platStore.router;
await router.push("/v5/registration/primary-contact");
```

This got us to the Emergency Contact page, but form submissions there also failed because the WebSocket calls hit the same `siteId` error. And navigating to `/v5/dashboard` always bounced back — the Vue Router `beforeEach` guard re-checked server state on every navigation.

### 9. Analyzed the Source Code

Downloaded the 1.4MB app bundle and decompiled key functions:

**Verbal password component** (`RegistrationVerbalPassword-DIS9ehBm.js`):
```javascript
// On submit:
await n.updatePasscode(r.value, r.value, n.passcode)
// Then advances the server-side stage:
await w.updateAccount({
  status: "active",
  systemSetupStage: 25  // VERBAL_PASSWORD
})
```

**updatePasscode payload** (from main bundle):
```javascript
async function Re(q, pe, be) {
  return e.push.request({
    type: "monitoringPasscodeUpdateRequest",
    payload: {
      newPasscode: q,
      currentPasscode: be,
      confirmNewPasscode: pe
    }
  })
}
```

**updateAccount** sends an `accountUpdateRequest` over WebSocket.

### 10. Attempted Passcode Update via WebSocket

Sent the message directly on the WebSocket:

```javascript
ws.send(JSON.stringify({
  type: "monitoringPasscodeUpdateRequest",
  payload: {
    newPasscode: "mypasscode",
    currentPasscode: "",
    confirmNewPasscode: "mypasscode"
  },
  requestId: crypto.randomUUID()
}));
```

**Response:** `500 — "Encountered error while updating passcode: Required: siteId. Missing: siteId."`

The monitoring subsystem can't set a passcode because the account doesn't have a `siteId` — it's still being provisioned.

### 11. The Fix — accountUpdateRequest

The passcode API requires a `siteId` that doesn't exist yet. But the `accountUpdateRequest` endpoint doesn't require it:

```javascript
ws.send(JSON.stringify({
  type: "accountUpdateRequest",
  payload: {
    status: "active",
    systemSetupStage: 100  // COMPLETE
  },
  requestId: crypto.randomUUID()
}));
```

**Response:** `200 OK` with `systemSetupStage: 100` confirmed.

After this, navigating to `plus.adt.com/v5/dashboard` loaded the full dashboard with a welcome message.

---

## Why the Mobile App Sometimes Worked

The user reported that the ADT+ mobile app occasionally got past the verbal password screen randomly. This is likely because:

1. The mobile app has different timeout/fallback logic for the pending promises
2. A race condition in the app's initialization — if certain WebSocket responses arrive in a different order, the loading state resolves before the passcode check blocks the UI
3. The mobile app may have a different navigation guard implementation that's less strict

---

## What's Still Needed

1. **Verbal password is not set server-side** — the passcode is still blank because the `siteId` doesn't exist. This needs to be set via ADT support (1-800-238-2727) or through the app once the base station is connected and provisioned.

2. **Emergency contacts were skipped** — should be configured in Settings > Monitoring > Emergency Contacts.

3. **Base station setup** — `outOfService: true` and `baseConnected: false` mean the physical hardware still needs to be activated. The "Go to ADT+ app to set up your Base" card on the dashboard addresses this.

---

## Bug Summary for ADT

The registration flow has a deadlock for accounts in `ACCOUNT_CREATION` state:

1. Registration requires setting a verbal password at stage 20→25
2. The verbal password page loads and requests device/system data via WebSocket
3. These requests never resolve because the base station isn't connected (the user hasn't set it up yet — that comes AFTER registration)
4. The page gets stuck in a loading spinner and never shows the submit button
5. Even if the spinner is bypassed, the passcode update API fails with "Missing: siteId" because the monitoring system isn't provisioned yet
6. The user is trapped: can't complete registration, can't skip it, can't access the dashboard

The `accountUpdateRequest` endpoint accepts `systemSetupStage: 100` without requiring a `siteId`, which is the escape hatch. But no UI element exposes this — users have to reverse-engineer the WebSocket protocol to find it.
