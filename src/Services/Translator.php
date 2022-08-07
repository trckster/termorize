<?php

namespace Termorize\Services;

use GuzzleHttp\Client;

class Translator
{
    private Client $httpClient;

    public function __construct()
    {
        $this->httpClient = new Client(['base_uri' => 'https://translate.yandex.net/api/v1.5/tr.json/']);
    }

    public function defineLang(string $text): string
    {
        $apiKey = $_ENV["YANDEX_TRANSLATOR_API_KEY"];

        $params = [
            'key' => $apiKey,
            'text' => $text,
            'hint' => ['en', 'ru'],
        ];

        $query = '?' . http_build_query($params);

        $response = $this->httpClient->get("https://translate.yandex.net/api/v1.5/tr.json/detect$query");
        $responseContent = json_decode($response->getBody()->getContents(), true);

        return $responseContent["lang"];
    }

    public function translate(string $text): string
    {
        $apiKey = $_ENV["YANDEX_TRANSLATOR_API_KEY"];

        $originTextLang = $this->defineLang($text);

        if ($originTextLang === "ru") {
            $translationLang = "en";
        } else {
            $translationLang = "ru";
        }

        $params = [
            'key' => $apiKey,
            'text' => $text,
            'lang' => $translationLang,
        ];

        $query = '?' . http_build_query($params);

        $response = $this->httpClient->get("https://translate.yandex.net/api/v1.5/tr.json/translate$query");
        $responseContent = json_decode($response->getBody()->getContents(), true);

        $translationText = $responseContent["text"][0];

        return $translationText;
    }
}