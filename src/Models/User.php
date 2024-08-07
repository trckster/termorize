<?php

namespace Termorize\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\HasManyThrough;
use Illuminate\Database\Eloquent\Relations\HasOne;

/**
 * @property int $id
 * @property bool $is_bot
 * @property string $first_name
 * @property string $last_name
 * @property string $username
 * @property string $language_code
 * @property bool $is_premium
 * @property bool $added_to_attachment_menu
 * @property Carbon $created_at
 * @property Carbon $updated_at
 *
 * @property-read UserSetting $settings
 * @property-read Collection|VocabularyItem[] $vocabularyItems
 * @property-read Collection|Question[] $questions
 */
class User extends Model
{
    protected $fillable = [
        'id',
        'is_bot',
        'first_name',
        'last_name',
        'username',
        'language_code',
        'is_premium',
        'added_to_attachment_menu',
        'created_at',
        'updated_at',
    ];

    protected $table = 'user';
    public $incrementing = false;

    public function settings(): HasOne
    {
        return $this->hasOne(UserSetting::class, 'user_id', 'id');
    }

    public function getOrCreateSettings(): UserSetting
    {
        if (!$this->settings) {
            $this->settings = UserSetting::createDefaultSetting($this);
        }

        return $this->settings;
    }

    public function vocabularyItems(): HasMany
    {
        return $this->hasMany(VocabularyItem::class);
    }

    public function questions(): HasManyThrough
    {
        return $this->hasManyThrough(Question::class, VocabularyItem::class);
    }
}
