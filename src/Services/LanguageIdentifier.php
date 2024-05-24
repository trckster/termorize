<?php

namespace Termorize\Services;

use Termorize\Enums\Language;

class LanguageIdentifier
{
    const string CYRILLIC_SYMBOLS = 'йцукенгшщзхъфывапролджэячсмитьбюёЙЦУКЕНГШЩЗХЪФЫВАПРОЛДЖЭЮЯБЧЬСТМИЁ';

    private static function isCyrillic(string $symbol): bool
    {
        return str_contains(self::CYRILLIC_SYMBOLS, $symbol);
    }

    public static function identify(string $text): Language
    {
        $russian = 0;
        $english = 0;

        $textArray = mb_str_split($text);

        foreach ($textArray as $symbol) {
            if (self::isCyrillic($symbol)) {
                $russian++;
            } else {
                $english++;
            }
        }

        return $russian > $english ? Language::ru : Language::en;
    }
}
