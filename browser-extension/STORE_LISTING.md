# Chrome Web Store listing

This is the copy-and-paste source of truth for the Termorize extension listing. It matches version `1.1.0` and the disclosures in the public privacy policy.

## Product details

**Name**

Termorize for Google Translate™

**Summary**

Translate selected text anywhere, then edit or save it to your Termorize vocabulary.

**Detailed description**

Move useful words from any page into Termorize without copying and pasting.

- Select text anywhere and press Alt+T to translate it in a compact on-page editor.
- Translate text inside a cross-origin embedded frame from the selection context menu.
- Click the Termorize icon to translate the current selection in the extension popup.
- Change the target language and get an updated translation immediately.
- Edit either side before saving, or save the generated translation unchanged.
- Press Ctrl+E to review and edit the current word pair before saving it.
- Press Ctrl+S to save the current word pair immediately.
- See the Ctrl+E and Ctrl+S shortcut hints directly on Google Translate™.
- Continue practicing the saved words on the Termorize website or through its Telegram bot.

The extension supports English, Russian, Italian, German, Spanish, French, Polish, Turkish, Portuguese, and Ukrainian. Choose explicit source and target languages in Google Translate before saving.

A Termorize session is required. You can sign in through Telegram or use a temporary guest account from the Termorize home page. If a shortcut conflicts with another browser command, it can be reassigned at `chrome://extensions/shortcuts`.

Google Translate is a trademark of Google LLC. Termorize for Google Translate™ is an independent extension and is not affiliated with, sponsored by, or endorsed by Google LLC.

**Category**

Productivity

**Language**

English

**Homepage URL**

https://termorize.daniil.online

**Support URL**

https://github.com/trckster/termorize/issues

**Privacy policy URL**

https://termorize.daniil.online/extension-privacy.html

## Single purpose

Translate text selected by the user and optionally save the resulting word pair to the signed-in user's Termorize vocabulary.

## Permission justifications

**`cookies`**

Reads only the `auth` cookie for `termorize.daniil.online` when the user opens or invokes Termorize. The cookie authenticates account settings, translation, and save requests. The extension does not read cookies from other sites.

**`activeTab`**

Temporarily accesses only the active tab after the user clicks the extension icon, invokes Alt+T, or chooses the Termorize selection context-menu action. This access is used to read or display the text the user selected.

**`contextMenus`**

Adds a single selection-only action that translates text inside cross-origin embedded frames without requesting permanent access to every website.

**`scripting`**

Runs the packaged selection reader and translation overlay in the active tab after an explicit icon click or shortcut. It does not inject remote code or run continuously across browsing sessions.

**Host access: `https://translate.google.com/*`**

Runs the packaged Google Translate integration so it can show shortcut hints and read the visible source text, translation, and selected language codes after the user invokes a save shortcut.

**Host access: `https://termorize.daniil.online/*`**

Reads the Termorize authentication cookie, loads and updates the account's target language, sends selected text to the translation API, and saves an approved word pair. It also opens the Termorize sign-in page when authentication is missing.

**Remote code**

No. All executable JavaScript and CSS is included in the uploaded extension package. Network requests send data to the Termorize API but do not download or execute code.

## Privacy questionnaire

Declare these data types:

- **Authentication information:** the Termorize session cookie is read to authenticate an explicit save request.
- **Website content:** text explicitly selected by the user, plus the visible source text, translation, and selected language codes on Google Translate after an explicit command.

Do not declare web history, general user activity, location, financial information, health information, or advertising identifiers: the extension does not collect them. Selected text may contain personal content, but the extension handles it only after the user explicitly invokes translation.

Select **App functionality** as the only use of the declared data. Confirm that the data is not sold, used for advertising, used for creditworthiness or lending, or transferred for purposes unrelated to the extension's single purpose.

## Reviewer instructions

No private test account is required.

1. Open https://termorize.daniil.online and choose **Just Try** to create a temporary guest session.
2. Select a phrase on any ordinary HTTPS page and press **Alt+T**. Confirm that the on-page panel detects its language and displays a translation.
3. Select text inside a cross-origin embedded frame, right-click it, and choose **Translate selection with Termorize**. Confirm that the same editor opens over the main page.
4. Change the target language and confirm the translation updates. Edit either field and save it.
5. Open https://translate.google.com/?sl=it&tl=en&text=buongiorno&op=translate and confirm that the Ctrl+E and Ctrl+S hints are visible.
6. Press **Ctrl+E** to review the pair, then test **Ctrl+S** with a different word.
7. Return to Termorize and open **Vocabulary** to confirm the saved pairs.

If either shortcut is already assigned by the review browser, trigger the command from `chrome://extensions/shortcuts` after assigning a temporary key combination.

## Upload assets

- Store icon: `icons/icon-128.png`
- Screenshot: `store-assets/screenshot-review-1280x800.png`
- Screenshot: `store-assets/screenshot-saved-1280x800.png`
- Small promotional tile: `store-assets/promo-small-440x280.png`
- Upload package: generated under `dist/` by `./package.sh`

Use the screenshots without scaling or added borders. The small promotional tile contains no text so it remains legible in all store contexts.
