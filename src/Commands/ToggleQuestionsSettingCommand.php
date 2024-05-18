<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

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
            : 'Ежедневная отпрака слов выключена';

        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => $answer,
        ]);
    }
}