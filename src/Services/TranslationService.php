<?php

namespace Termorize\Services;

use Termorize\Enums\Language;
use Termorize\Models\Translation;

readonly final class TranslationService
{
    private YandexTranslationApi $api;

    public function __construct()
    {
        $this->api = new YandexTranslationApi;
    }

    public function translate(string $text, Language $foreign): Translation
    {
        $text = mb_strtolower($text);

        $isOriginalRussian = LanguageIdentifier::isRussian($text);

        $originalLanguage = $isOriginalRussian ? Language::ru : $foreign;
        $translationLanguage = $isOriginalRussian ? $foreign : Language::ru;

        $translation = Translation::query()
            ->where('is_custom', false)
            ->where('original_text', $text)
            ->where('original_lang', $originalLanguage->name)
            ->where('translation_lang', $translationLanguage->name)
            ->first();

        if ($translation) {
            return $translation;
        }

        $translationText = $this->api->translate($text, $translationLanguage);

        $equalTranslation = Translation::query()
            ->where('is_custom', false)
            ->where('original_text', $translationText)
            ->where('translation_text', $text)
            ->where('original_lang', $translationLanguage->name)
            ->where('translation_lang', $originalLanguage->name)
            ->first();

        if ($equalTranslation) {
            return $equalTranslation;
        }

        return Translation::query()->create([
            'original_text' => $text,
            'translation_text' => mb_strtolower($translationText),
            'original_lang' => $originalLanguage,
            'translation_lang' => $translationLanguage,
            'is_custom' => false,
        ]);
    }

    public function saveCustomTranslation(string $original, string $translation, Language $foreign): Translation
    {
        $isOriginalRussian = LanguageIdentifier::isRussian($original);

        $originalLanguage = $isOriginalRussian ? Language::ru : $foreign;
        $translationLanguage = $isOriginalRussian ? $foreign : Language::ru;

        return Translation::query()->create([
            'original_text' => $original,
            'translation_text' => $translation,
            'original_lang' => $originalLanguage,
            'translation_lang' => $translationLanguage,
            'is_custom' => true,
        ]);
    }
}
