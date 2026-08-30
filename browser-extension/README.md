# Termorize for Google Translate

A Chromium Manifest V3 extension that saves the word pair currently displayed on Google Translate to the signed-in user's Termorize vocabulary.

## Install locally

1. Sign in at [termorize.daniil.online](https://termorize.daniil.online).
2. Open `chrome://extensions` in Chrome, Edge, Brave, or another Chromium browser.
3. Enable **Developer mode**.
4. Choose **Load unpacked** and select this `browser-extension` directory.
5. Open [Google Translate](https://translate.google.com) and choose explicit source and target languages.

## Shortcuts

- `Ctrl+E` — review and edit the current pair before saving. Use `Shift+Enter` in the dialog to save.
- `Ctrl+S` — save the current pair immediately.

The shortcuts are also registered as Chromium extension commands so they can take precedence over the browser's built-in omnibox and Save Page actions. If another extension already owns either shortcut, reassign it under `chrome://extensions/shortcuts`.

The extension runs only on `https://translate.google.com/*`. It supports the same ten languages as Termorize: English, Russian, Italian, German, Spanish, French, Polish, Turkish, Portuguese, and Ukrainian. If the website session is missing or expired, the extension offers to open the Termorize sign-in page.

## Tests

Run the dependency-free extension test suite from this directory:

```sh
node --test *.test.js
```
