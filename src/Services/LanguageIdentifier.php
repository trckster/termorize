<?php

namespace Termorize\Services;

class LanguageIdentifier
{

    private function isCyrillic(string $symbol) : bool
    {

        $cyrillicSymbols = "абвгдеёжзиклмнопрстуфхцчшщъыьэюя";

        if (str_contains($cyrillicSymbols, $symbol))
        {
            return true;
        } else {
            return false;
        }
    }

    public static function identify(string $text) : string
    {
        $identifier = new LanguageIdentifier();

        $russian = 0;
        $english = 0;

        $textArray = stringToArray($text);

        foreach($textArray as $symbol)
        {
            if ($identifier->isCyrillic($symbol))
            {
                $russian++;
            } else {
                $english++;
            }
        }

        if ($russian > $english)
        {
            return "ru";
        } else {
            return "en";
        }
    }

}