<?php

namespace Termorize\Commands;

class DefaultCommand extends AbstractCommand
{
    public function process(): void
    {
        $message = <<<'MESSAGE'
<b>Доступные команды</b>:

<code>/set_language de</code>
Сменить язык.

<code>/add cut to the chase:перейти к делу</code>
Добавить перевод для изучения самостоятельно (слово и его перевод разделить двоеточием).

<code>/delete перейти к делу</code>
Удалить перевод из словарного запаса.

<code>/list</code>
Список из словарного запаса.

<code>/toggle_questions</code>
Включить/выключить отправку ежедневных вопросов.

<code>/set_questions 5</code>
Установить кол-во вопросов в день.

<code>/set_schedule 08:00-23:45</code>
Установить отрезок времени <b>по UTC</b>, в который вы хотите получать вопросы.

<code>Anything else (not starting with the slash)</code>
Любое другое сообщение будет переведено и автоматически добавлено в список изучаемых слов.
MESSAGE;

        $this->reply($message, ['parse_mode' => 'HTML']);
    }
}
