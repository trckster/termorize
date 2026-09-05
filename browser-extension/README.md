# TermoClip

TermoClip is Termorize’s Chromium Manifest V3 extension. It translates selected text on any ordinary web page and saves useful word pairs to the signed-in user's Termorize vocabulary. It also keeps its direct Google Translate workflow.

## Install locally

1. Sign in at [termorize.daniil.online](https://termorize.daniil.online).
2. Open `chrome://extensions` in Chrome, Edge, Brave, or another Chromium browser.
3. Enable **Developer mode**.
4. Choose **Load unpacked** and select this `browser-extension` directory.
5. Pin the TermoClip icon so the translation popup is always one click away.

For Chrome Web Store publication materials, see [STORE_LISTING.md](STORE_LISTING.md). The published extension privacy policy lives at [termorize.daniil.online/extension-privacy.html](https://termorize.daniil.online/extension-privacy.html).

## Shortcuts

- `Alt+T` — translate the selected text on the current page and open a compact editor beside it.
- `Ctrl+E` — review and edit the current pair before saving. Use `Shift+Enter` in the dialog to save.
- `Ctrl+S` — save the current pair immediately.

`Ctrl+E` and `Ctrl+S` apply on Google Translate and are shown in a compact hint on that page. The handlers use physical key codes, so they continue to work with non-Latin keyboard layouts. If Vivaldi or another extension already owns a shortcut, reassign it under `vivaldi://extensions/shortcuts` or `chrome://extensions/shortcuts`.

For text inside an embedded cross-origin frame, use the page's context menu and choose **Translate selection with TermoClip**. Browser security prevents `Alt+T` from reading those frames without permanent access to every site, so the context-menu action passes the selection to the same editor without adding that permission.

Clicking the icon opens a popup that reads the current selection, detects its source language, and translates it to the account's target language. Changing the target in the popup updates the Termorize account setting and retranslates immediately. The extension supports the same ten languages as Termorize: English, Russian, Italian, German, Spanish, French, Polish, Turkish, Portuguese, and Ukrainian. If the website session is missing or expired, the popup offers to open the Termorize sign-in page.

The extension uses temporary `activeTab` access after an icon click, `Alt+T`, or its selection context-menu action; it does not request permanent access to every website. Chromium blocks extensions on internal browser pages and other protected surfaces, so selection translation is available on ordinary web pages rather than `chrome://` or `vivaldi://` pages.

## Tests

Run the dependency-free extension test suite from this directory:

```sh
node --test *.test.js
```

## Build the store upload

From the repository root:

```sh
./browser-extension/package.sh
```

The versioned ZIP is written to `browser-extension/dist/` with `manifest.json` at the archive root. Tests, source artwork, screenshots, and store copy are intentionally excluded.

Google Translate is a trademark of Google LLC. This extension is independent and is not affiliated with, sponsored by, or endorsed by Google LLC.
