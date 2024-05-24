<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

class DefaultCommand extends AbstractCommand
{
    public function process(): void
    {
        $message = <<<MESSAGE
<b>Доступные команды</b>:

<i>/toggle_questions</i> => Включить/выключить отправку ежедневных вопросов.

<i>/set_questions 5</i> => Установить кол-во вопросов в день.

<i>Enormous mansion</i> => Любое другое сообщение будет переведено и автоматически добавлено в список изучаемых слов.
MESSAGE;

        $this->reply($message, ['parse_mode' => 'HTML']);
    }
}
