# Chrome Web Store listing

This is the copy-and-paste source of truth for the Termorize extension listing. It matches version `1.0.0` and the disclosures in the public privacy policy.

## Product details

**Name**

Termorize for Google Translate™

**Summary**

Save Google Translate™ word pairs to your Termorize vocabulary with two keyboard shortcuts.

**Detailed description**

Move useful words from Google Translate™ into Termorize without copying and pasting.

- Press Ctrl+E to review and edit the current word pair before saving it.
- Press Ctrl+S to save the current word pair immediately.
- Keep the source language, target language, original text, and translation together in your Termorize vocabulary.
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

Save the current Google Translate word pair to the signed-in user's Termorize vocabulary, either immediately or after review.

## Permission justifications

**`cookies`**

Reads only the `auth` cookie for `termorize.daniil.online` when the user asks to save a word. The cookie authenticates the save request to the user's Termorize account. The extension does not read cookies from Google Translate or other sites.

**Host access: `https://translate.google.com/*`**

Runs the content script on Google Translate so it can read the visible source text, translation, and selected language codes after the user invokes a save shortcut.

**Host access: `https://termorize.daniil.online/*`**

Reads the Termorize authentication cookie and sends the selected word pair to the Termorize vocabulary API. It also opens the Termorize sign-in page when authentication is missing.

**Remote code**

No. All executable JavaScript and CSS is included in the uploaded extension package. Network requests send data to the Termorize API but do not download or execute code.

## Privacy questionnaire

Declare these data types:

- **Authentication information:** the Termorize session cookie is read to authenticate an explicit save request.
- **Website content:** the visible source text, translation, and selected language codes are read from Google Translate after an explicit keyboard shortcut.

Do not declare web history, user activity, location, financial information, health information, or personal communications: the extension does not collect them.

Select **App functionality** as the only use of the declared data. Confirm that the data is not sold, used for advertising, used for creditworthiness or lending, or transferred for purposes unrelated to the extension's single purpose.

## Reviewer instructions

No private test account is required.

1. Open https://termorize.daniil.online and choose **Just Try** to create a temporary guest session.
2. Open https://translate.google.com/?sl=it&tl=en&text=buongiorno&op=translate and wait for the English translation.
3. Press **Ctrl+E**. Confirm that the Termorize review dialog contains the Italian source and English translation. Edit either field if desired, then press **Shift+Enter** or click **Save to vocabulary**.
4. Press **Ctrl+S** to test immediate saving with a different word pair.
5. Return to Termorize and open **Vocabulary** to confirm the saved pair.

If either shortcut is already assigned by the review browser, trigger the command from `chrome://extensions/shortcuts` after assigning a temporary key combination.

## Upload assets

- Store icon: `icons/icon-128.png`
- Screenshot: `store-assets/screenshot-review-1280x800.png`
- Screenshot: `store-assets/screenshot-saved-1280x800.png`
- Small promotional tile: `store-assets/promo-small-440x280.png`
- Upload package: generated under `dist/` by `./package.sh`

Use the screenshots without scaling or added borders. The small promotional tile contains no text so it remains legible in all store contexts.
