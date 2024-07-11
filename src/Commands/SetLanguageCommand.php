<?php

namespace Termorize\Commands;

use Illuminate\Support\Str;

class SetLanguageCommand extends AbstractCommand
{
    public const array SUPPORTED_LANGUAGES = [
        'en' => 'Английский',
        'de' => 'Немецкий',
        'fr' => 'Французский',
        'es' => 'Испанский',
        'it' => 'Итальянский',
        'pl' => 'Польский',
    ];

    public function process(): void
    {
        $language = Str::lower($this->getClearedMessage());

        if (!array_key_exists($language, self::SUPPORTED_LANGUAGES)) {
            $this->reply("Такой язык не поддерживается.\n\n" . $this->listLanguages());

            return;
        }

        $settings = $this->loadUser()->getOrCreateSettings();
        $settings->update(['language' => $language]);

        $this->reply('Язык установлен: ' . mb_strtolower(self::SUPPORTED_LANGUAGES[$language]));
    }

    private function listLanguages(): string
    {
        $text = "Список доступных языков:\n";

        foreach (self::SUPPORTED_LANGUAGES as $code => $language) {
            $text .= "$code — $language\n";
        }

        return $text;
    }
}
