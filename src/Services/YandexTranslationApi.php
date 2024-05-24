<?php

namespace Termorize\Services;

use GuzzleHttp\Client;
use Termorize\Enums\Language;

class YandexTranslationApi
{
    private Client $httpClient;
    private string $apiKey;

    public function __construct()
    {
        $this->httpClient = new Client(['base_uri' => 'https://translate.yandex.net/api/v1.5/tr.json/']);
        $this->apiKey = env('YANDEX_TRANSLATOR_API_KEY');
    }

    public function translate(string $word, Language $to): string
    {
        $query = '?' . http_build_query([
                'key' => $this->apiKey,
                'text' => $word,
                'lang' => $to->name,
            ]);

        $response = $this->httpClient->get("translate$query");
        $responseContent = json_decode($response->getBody()->getContents(), true);

        return $responseContent['text'][0];
    }
}