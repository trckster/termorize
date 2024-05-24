<?php

namespace Termorize\Commands;

class SetQuestionsCountCommand extends AbstractCommand
{
    public function process(): void
    {
        $user = $this->loadUser();
        $userSetting = $user->getOrCreateSettings();

        $messageParts = explode(' ', $this->update->getMessage()->getText());
        if (count($messageParts) < 2 || !is_numeric($messageParts[1])) {
            $this->reply('Формат команды: <i>/set_questions 5</i>, где 5 - это кол-во вопросов в день.', [
                'parse_mode' => 'HTML',
            ]);
            return;
        }

        $questionsCount = (int)$messageParts[1];

        if ($questionsCount < 1 || $questionsCount > 1000) {
            $this->reply('Число вопросов не может быть меньше одного и больше тысячи.');
            return;
        }

        $userSetting->update([
            'questions_count' => $questionsCount,
        ]);

        $this->reply("Теперь вы будете получать $questionsCount вопросов в день!");
    }
}