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
            ? 'Ежедневная отправка слов включена'
            : 'Ежедневная отправка слов выключена';

        $this->reply($answer);
    }
}