<?php

namespace Termorize\Helpers;

class KeyboardHelper
{
    public static function makeButton(string $buttonText, string $callbackName, array $callbackData): string
    {
        $keyboard = [
            'inline_keyboard' => [
                [
                    [
                        'text' => $buttonText,
                        'callback_data' => self::makeCallback($callbackName, $callbackData)
                    ]
                ]
            ]
        ];

        return json_encode($keyboard);
    }

    public static function makeCallback(string $callbackName, array $callbackData): string
    {
        $callback = [
            'callback' => $callbackName,
            'data' => $callbackData,
        ];

        return json_encode($callback);
    }
}