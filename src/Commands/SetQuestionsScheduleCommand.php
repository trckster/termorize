<?php

namespace Termorize\Commands;

use Exception;
use Throwable;

class SetQuestionsScheduleCommand extends AbstractCommand
{
    public function process(): void
    {
        $timeRange = $this->getClearedMessage();

        try {
            [$from, $to] = $this->parseTimes($timeRange);
        } catch (Throwable) {
            $this->reply('Укажите время в нужном формате');

            return;
        }

        $user = $this->loadUser();

        $userSetting = $user->getOrCreateSettings();
        $userSetting->questions_schedule = [
            'from' => $from,
            'to' => $to,
        ];
        $userSetting->save();

        $this->reply('Отрезок времени для отправки вопросов сохранён!');
    }

    private function parseTimes(string $timeRange): array
    {
        [$from, $to] = explode('-', $timeRange);

        $from = $this->inMinutes($from);
        $to = $this->inMinutes($to);

        if ($from < 0 || $to > 1439 || $from >= $to) {
            throw new Exception;
        }

        return [
            $from,
            $to,
        ];
    }

    private function inMinutes(string $time): int
    {
        [$hours, $minutes] = explode(':', $time);

        return (int) $hours * 60 + (int) $minutes;
    }
}
