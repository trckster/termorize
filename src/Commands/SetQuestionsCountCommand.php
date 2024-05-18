<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

class SetQuestionsCountCommand extends AbstractCommand
{
    public function process(): void
    {
        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => 'Команда ещё в разработке',
        ]);
    }
}