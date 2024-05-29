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

    public function translate(string $text): Translation
    {
        $text = mb_strtolower($text);

        $translation = Translation::query()
            ->where('is_custom', false)
            ->where('original_text', $text)
            ->first();
        if ($translation) {
            return $translation;
        }

        $originalLanguage = LanguageIdentifier::identify($text);
        $translationLanguage = $originalLanguage === Language::ru ? Language::en : Language::ru;

        $translationText = $this->api->translate($text, $translationLanguage);

        return Translation::query()->create([
            'original_text' => $text,
            'translation_text' => mb_strtolower($translationText),
            'original_lang' => $originalLanguage,
            'translation_lang' => $translationLanguage,
            'is_custom' => false,
        ]);
    }

    public function saveCustomTranslation(string $original, string $translation): Translation
    {
        $originalLanguage = LanguageIdentifier::identify($original);
        $translationLanguage = $originalLanguage === Language::ru ? Language::en : Language::ru;

        return Translation::query()->create([
            'original_text' => $original,
            'translation_text' => $translation,
            'original_lang' => $originalLanguage,
            'translation_lang' => $translationLanguage,
            'is_custom' => true,
        ]);
    }
}
