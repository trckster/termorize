# Store assets

These files are ready for the Chrome Web Store listing:

- `screenshot-review-1280x800.png` — the actual extension review dialog rendered with production `content.js` and `content.css`.
- `screenshot-saved-1280x800.png` — the actual success notification rendered with production `content.js` and `content.css`.
- `promo-small-440x280.png` — optional small promotional tile.

The screenshots are generated from `source/screenshot-demo.html`, which stubs only the browser messaging response. The extension UI itself comes from the production scripts and styles. The promotional illustration source and its generation prompt are retained under `source/` for provenance.

The extension icon PNGs live under `../icons/` because Chromium loads them at runtime. Their vector source is the shared Termorize mark at `../../frontend/public/favicon.svg`.
