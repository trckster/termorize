<?php

namespace Termorize\Services;

define('CYRILLIC_SYMBOLS', 'абвгдеёжзиклмнопрстуфхцчшщъыьэюя');

class LanguageIdentifier
{
    private function isCyrillic(string $symbol): bool
    {

        if (str_contains(CYRILLIC_SYMBOLS, $symbol)) {
            return true;
        } else {
            return false;
        }
    }

    public static function identify(string $text): string
    {
        $identifier = new LanguageIdentifier();

        $russian = 0;
        $english = 0;

        $textArray = str_split($text, 1);

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
