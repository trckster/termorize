<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @property int $id
 * @property int $translation_id
 * @property int $user_id
 * @property int $knowledge
 *
 * @property-read Translation $translation
 * @property-read Collection $questions
 */
class VocabularyItem extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'translation_id',
        'user_id',
        'knowledge',
    ];

    public function translation(): BelongsTo
    {
        return $this->belongsTo(Translation::class);
    }

    public function questions(): HasMany
    {
        return $this->hasMany(Question::class);
    }
}
