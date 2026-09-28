# Local UI review

Run `pnpm dev:design` in `frontend`, then open:

- http://localhost:3000/translation — the daily idiom and its one-time hint.
- http://localhost:3000/settings — the implemented centered Save pill.

This opt-in server uses sample data in memory; it does not contact the backend.
English, Italian, and Russian idioms are provided. Other actions outside this
design review return an explicit unavailable response. Restarting resets sample
account settings. The Save pill updates this local sample account.

The hint appears once at least half of the idiom card is on screen: a note tied to
the card by a leader line, beside it on wide screens and above it on phones and
tablets. A short title and explanation describe the language and Telegram options
available in Settings. Got it or Escape stores `termorize:daily-idiom-hint-dismissed`
in localStorage; delete that key to preview the hint again. There is no in-app
control to reopen it, and the dismissal applies across users and visits in the
same browser origin.

The sample API is development-only. Normal `pnpm dev` continues to use the
configured backend. The centered pill saves all edited settings in one request;
appearance preferences continue to apply immediately in this browser.
