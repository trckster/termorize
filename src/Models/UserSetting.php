<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;
use Termorize\Enums\Language;

/**
 * @property int $user_id
 * @property bool $learns_vocabulary
 * @property int $questions_count
 * @property null|array $questions_schedule
 * @property Language $language
 */
class UserSetting extends Model
{
    protected $table = 'users_settings';
    protected $primaryKey = 'user_id';
    public $incrementing = false;

    public const array DEFAULT_SCHEDULE = [
        'from' => 0,  // 00:00
        'to' => 1439, // 23:59
    ];

    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'user_id',
        'learns_vocabulary',
        'questions_count',
        'questions_schedule',
        'language',
    ];

    protected $casts = [
        'questions_schedule' => 'array',
        'language' => Language::class,
    ];

    public function getQuestionsScheduleFrom(): int
    {
        return $this->questions_schedule['from'] ?? self::DEFAULT_SCHEDULE['from'];
    }

    public function getQuestionsScheduleTo(): int
    {
        return $this->questions_schedule['to'] ?? self::DEFAULT_SCHEDULE['to'];
    }

    public static function createDefaultSetting(User $user): self
    {
        return self::query()
            ->create([
                'user_id' => $user->id,
                'learns_vocabulary' => true,
                'questions_count' => 1,
                'questions_schedule' => self::DEFAULT_SCHEDULE,
                'language' => Language::en,
            ]);
    }
}
