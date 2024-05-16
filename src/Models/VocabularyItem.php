<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property int $id
 * @property int $translation_id
 * @property int $user_id
 * @property int $knowledge
 *
 * @property-read Translation $translation
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
}
