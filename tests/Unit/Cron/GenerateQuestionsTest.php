<?php

namespace Tests\Unit\Cron;

use Carbon\Carbon;
use Termorize\Cron\GenerateQuestions;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Tasks\SendQuestion;
use Tests\TestCase;

class GenerateQuestionsTest extends TestCase
{
    /**
     * @test
     */
    public function generateQuestionsCommandWorks()
    {
        /** @var User $user */
        $user = User::query()->create([
            'id' => 1,
            'is_bot' => false,
            'first_name' => 'Name',
            'last_name' => 'Surname',
            'username' => 'user12234567',
            'language_code' => 'ru',
            'is_premium' => false,
        ]);

        $item = $user->vocabularyItems()->create([
            'knowledge' => 0,
            'translation_id' => 5,
        ]);

        $user->settings()->create([
            'learns_vocabulary' => true,
        ]);
        $command = new GenerateQuestions();
        $command->handle();

        $this->assertEquals(1, PendingTask::query()->count());

        /** @var PendingTask $pendingTask */
        $pendingTask = PendingTask::query()->first();
        $this->assertEquals([$user->id, $item->id], $pendingTask->parameters);
        $this->assertEquals(SendQuestion::class . '::execute', $pendingTask->method);
        $this->assertEquals(PendingTaskStatus::Pending, $pendingTask->status);
        $this->assertEquals(Carbon::today(), $pendingTask->scheduled_for->startOfDay());
    }
}
