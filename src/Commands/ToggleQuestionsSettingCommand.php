<?php

namespace Termorize\Commands;

class ToggleQuestionsSettingCommand extends AbstractCommand
{
    public function process(): void
    {
        $user = $this->loadUser();
        $userSetting = $user->getOrCreateSettings();

        $userSetting->update([
            'learns_vocabulary' => !$userSetting->learns_vocabulary
        ]);

        $answer = $userSetting->learns_vocabulary
            ? "Ежедневная отправка слов включена\n\n(начнёт действовать со следующего дня)"
            : "Ежедневная отправка слов выключена\n\n(начнёт действовать со следующего дня)";

        $this->reply($answer);
    }
}