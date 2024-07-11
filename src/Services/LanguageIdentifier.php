<?php

namespace Termorize\Services;

class LanguageIdentifier
{
    const string CYRILLIC_SYMBOLS = 'йцукенгшщзхъфывапролджэячсмитьбюёЙЦУКЕНГШЩЗХЪФЫВАПРОЛДЖЭЮЯБЧЬСТМИЁ';

    private static function isCyrillic(string $symbol): bool
    {
        return str_contains(self::CYRILLIC_SYMBOLS, $symbol);
    }

    public static function isRussian(string $text): bool
    {
        $russian = 0;

        $textArray = mb_str_split($text);

        foreach ($textArray as $symbol) {
            if (self::isCyrillic($symbol)) {
                $russian++;
            }
        }

        return $russian > count($textArray) / 2;
    }
}
