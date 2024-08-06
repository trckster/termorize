<?php

namespace Termorize\Commands;

class StatCommand extends AbstractCommand
{
    public function process(): void
    {
        $user = $this->loadUser(['vocabularyItems', 'questions']);

        $all = $user->vocabularyItems->count();
        $done = $user->vocabularyItems->where('knowledge', 100)->count();
        $inProgress = $user->vocabularyItems->whereBetween('knowledge', [1, 99])->count();
        $notYet = $user->vocabularyItems->where('knowledge', 0)->count();

        $questionsCount = $user->questions->where('is_answered', true)->count();

        $message = <<<'MESSAGE'
<b>Статистика за всё время</b>

*️⃣ Размер словарной базы: %d
🟢 Слов выучено: %d
🟡 Слов в изучении: %d
🔴 Слов предстоит изучить: %d

❔ На вопросов отвечено: %d
MESSAGE;

        $message = sprintf($message, $all, $done, $inProgress, $notYet, $questionsCount);

        $this->reply($message, ['parse_mode' => 'HTML']);
    }
}
