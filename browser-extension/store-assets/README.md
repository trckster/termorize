# Store assets

These files are ready for the Chrome Web Store listing:

- `screenshot-review-1280x800.png` — the actual extension review dialog rendered with production `content.js` and `content.css`.
- `screenshot-saved-1280x800.png` — the actual success notification rendered with production `content.js` and `content.css`.
- `promo-small-440x280.png` — optional small promotional tile.

The screenshots are generated from `source/screenshot-demo.html`, which stubs only the browser messaging response. The extension UI itself comes from the production scripts and styles. The promotional illustration source and its generation prompt are retained under `source/` for provenance.

The extension icon PNGs live under `../icons/` because Chromium loads them at runtime. Their vector source is the shared Termorize mark at `../../frontend/public/favicon.svg`.

## TermoClip landing-page screenshots

The landing page uses `frontend/public/images/termoclip/selection.png` and `popup-saved.png` (paths from the repository root). They show the production on-page editor beside a sample article and the production toolbar popup after saving. Only browser messaging is stubbed, with sample account and translation data; no live API or account is used.

Regenerate both from the repository root with Playwright available to Node:

```sh
npm install --prefix /tmp/termoclip-screenshots playwright
NODE_PATH=/tmp/termoclip-screenshots/node_modules CHROME_PATH=/path/to/chrome \
    node browser-extension/store-assets/source/capture-termoclip.cjs
```

Alternatively, install Playwright's Chromium with `/tmp/termoclip-screenshots/node_modules/.bin/playwright install chromium` and omit `CHROME_PATH`. The source article is `source/reading-demo.html`. Captures use a light theme and reduced motion for repeatability. The on-page screenshot is 1120 × 800; the popup is 780 × 1280 (390 × 640 at 2× resolution).
