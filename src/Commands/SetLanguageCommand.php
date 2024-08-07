<?php

namespace Termorize\Commands;

use Illuminate\Support\Str;
use Termorize\Enums\Language;

class SetLanguageCommand extends AbstractCommand
{
    public function process(): void
    {
        $language = Str::lower($this->getClearedMessage());

        if (!array_key_exists($language, Language::SUPPORTED_LANGUAGES)) {
            $this->reply("Такой язык не поддерживается.\n\n" . $this->listLanguages());

            return;
        }

        $settings = $this->loadUser()->getOrCreateSettings();
        $settings->update(['language' => $language]);

        $this->reply('Язык установлен: ' . mb_strtolower(Language::SUPPORTED_LANGUAGES[$language]));
    }

    private function listLanguages(): string
    {
        $text = "Список доступных языков:\n";

        foreach (Language::SUPPORTED_LANGUAGES as $code => $language) {
            $text .= "$code — $language\n";
        }

        return $text;
    }
}
