<?php

namespace Termorize\Commands;

class SettingsCommand extends AbstractCommand
{
    public function process(): void
    {
        $settings = $this->loadUser()->getOrCreateSettings();

        $message = <<<'MESSAGE'
<b>Настройки</b>

🈂️ Язык: <b>%s</b>
%s Получение ежедневных вопросов: <b>%s</b>
🔢 Количество ежедневных вопросов: <b>%d</b>
🕑 Расписание (UTC): <b>%s</b>
MESSAGE;

        $message = sprintf(
            $message,
            $settings->language->getName(),
            $settings->learns_vocabulary ? '✅' : '❌',
            $settings->learns_vocabulary ? 'Включено' : 'Выключено',
            $settings->questions_count,
            $settings->getHumanSchedule(),
        );

        $this->reply($message, ['parse_mode' => 'HTML']);
    }
}
