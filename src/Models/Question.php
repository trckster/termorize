<?php

namespace Termorize\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property int $id
 * @property int $chat_id
 * @property int $message_id
 * @property int $vocabulary_item_id
 * @property bool $is_original True if original word was sent, false if translated word was sent
 * @property bool $is_answered
 * @property Carbon $updated_at
 * @property Carbon $created_at
 *
 * @property-read VocabularyItem $vocabularyItem
 */
class Question extends Model
{
    protected $fillable = [
        'chat_id',
        'message_id',
        'vocabulary_item_id',
        'is_original',
        'is_answered',
    ];

    public function vocabularyItem(): BelongsTo
    {
        return $this->belongsTo(VocabularyItem::class);
    }
}
