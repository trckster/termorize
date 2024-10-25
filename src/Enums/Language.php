<?php

namespace Termorize\Enums;

enum Language
{
    public const array SUPPORTED_LANGUAGES = [
        'en' => 'Английский',
        'de' => 'Немецкий',
        'fr' => 'Французский',
        'es' => 'Испанский',
        'it' => 'Итальянский',
        'pl' => 'Польский',
        'tr' => 'Турецкий',
    ];

    case ru;
    case en;
    case de;
    case fr;
    case es;
    case it;
    case pl;
    case tr;

    public function getName(): string
    {
        return self::SUPPORTED_LANGUAGES[$this->name];
    }
}
