<?php

namespace Termorize\Services;

class LanguageIdentifier
{
    public const CYRILLIC_SYMBOLS = "йцукенгшщзхъфывапролджэячсмитьбюёЙЦУКЕНГШЩЗХЪФЫВАПРОЛДЖЭЮЯБЧЬСТМИЁ";

    private function isCyrillic(string $symbol): bool
    {
        return str_contains(self::CYRILLIC_SYMBOLS, $symbol);
    }

    public static function identify(string $text): string
    {
        $identifier = new LanguageIdentifier();

        $russian = 0;
        $english = 0;

        $textArray = str_split($text);

        foreach ($textArray as $symbol) {
            if ($identifier->isCyrillic($symbol)) {
                $russian++;
            } else {
                $english++;
            }
        }

        if ($russian > $english) {
            return 'ru';
        } else {
            return 'en';
        }
    }
}
