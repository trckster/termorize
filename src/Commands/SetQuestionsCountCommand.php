<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

class SetQuestionsCountCommand extends AbstractCommand
{
    public function process(): void
    {
        $user = $this->loadUser();
        $userSetting = $user->getOrCreateSettings();

        $messageParts = explode(' ', $this->update->getMessage()->getText());
        if (count($messageParts) < 2 || !is_numeric($messageParts[1])) {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'parse_mode' => 'HTML',
                'text' => 'Формат команды: <i>/set_questions 5</i>, где 5 - это кол-во вопросов в день.',
            ]);
            return;
        }

        $questionsCount = (int)$messageParts[1];

        if ($questionsCount < 1 || $questionsCount > 1000) {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Число вопросов не может быть меньше одного и больше тысячи.',
            ]);
            return;
        }

        $userSetting->update([
            'questions_count' => $questionsCount,
        ]);

        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => "Теперь вы будете получать $questionsCount вопросов в день!",
        ]);
    }
}