<?php

namespace Termorize\Commands;

class DefaultCommand extends AbstractCommand
{
    public function process(): void
    {
        $message = <<<MESSAGE
<b>Доступные команды</b>:

<code>/add_vocabulary cut to the chase:перейти к делу</code>
Добавить перевод для изучения самостоятельно (слово и его перевод разделить двоеточием).

<code>/delete_vocabulary vocabulary word</code>
Удалить слово из словарного запаса.

<code>/toggle_questions</code>
Включить/выключить отправку ежедневных вопросов.

<code>/set_questions 5</code>
Установить кол-во вопросов в день.

<code>Anything else (not starting with the slash)</code>
Любое другое сообщение будет переведено и автоматически добавлено в список изучаемых слов.
MESSAGE;

        $this->reply($message, ['parse_mode' => 'HTML']);
    }
}
