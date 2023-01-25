<?php

namespace Termorize\Services;

use GuzzleHttp\Client;
use Termorize\Models\Translation;

class Translator
{
    private Client $httpClient;

    public function __construct()
    {
        $this->httpClient = new Client(['base_uri' => 'https://translate.yandex.net/api/v1.5/tr.json/']);
    }

    public function translate(string $text): string
    {
        if (Translation::query()->where('original_text', $text)->exists()) {
            return Translation::query()->where('original_text', $text)->value('translation_text');
        }

        $apiKey = env('YANDEX_TRANSLATOR_API_KEY');

        $originTextLang = LanguageIdentifier::identify($text);

        if ($originTextLang === 'ru') {
            $translationLang = 'en';
        } else {
            $translationLang = 'ru';
        }

        $params = [
            'key' => $apiKey,
            'text' => $text,
            'lang' => $translationLang,
        ];

        $query = '?' . http_build_query($params);

        $response = $this->httpClient->get("translate$query");
        $responseContent = json_decode($response->getBody()->getContents(), true);

        $translationText = $responseContent['text'][0];

        Translation::query()->create([
            'original_text' => $text,
            'translation_text' => $translationText,
            'original_lang' => $originTextLang,
            'translation_lang' => $translationLang
        ]);

        return $translationText;
    }
}
