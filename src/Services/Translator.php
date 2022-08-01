<?php

require_once __DIR__ . "/../../vendor/autoload.php";

use GuzzleHttp\Client;
use Psr\Http\Message\ResponseInterface;


$dotenv = Dotenv\Dotenv::createImmutable(__DIR__ . "/../../");
$dotenv->load();

define("TRANSLATOR_KEY", $_ENV["YANDEX_TRANSLATOR_API_KEY"]);

class Translator
{
    public static function defineLang(string $text)
    {
        $client = new GuzzleHttp\Client(['base_uri' => 'https://foo.com/api/']);
        $api_key = TRANSLATOR_KEY;

        $response = $client->get("https://translate.yandex.net/api/v1.5/tr.json/detect?key=$api_key&text=$text&hint=en,ru");
        $response_content = json_decode($response->getBody()->getContents(), true);

        return $response_content["lang"];
    }

    public static function translate(string $text)
    {
        $client = new GuzzleHttp\Client(['base_uri' => 'https://foo.com/api/']);
        $api_key = TRANSLATOR_KEY;

        $origin_text_lang = self::defineLang($text);
        $translation_lang = '';

        if ($origin_text_lang == "ru") {
            $translation_lang = "en";
        } else {
            $translation_lang = "ru";
        }

        $response = $client->get("https://translate.yandex.net/api/v1.5/tr.json/translate?key=$api_key&text=%$text&lang=$translation_lang");
        $response_content = json_decode($response->getBody()->getContents(), true);

        $translation_text = $response_content["text"][0];

        $strings = explode("%", $translation_text);
        var_dump($strings);

        return $strings[1];
    }
}