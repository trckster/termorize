<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @property string $original_text
 * @property string $translation_text
 * @property string $original_lang
 * @property string $translation_lang
 */
class Translation extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'original_text',
        'translation_text',
        'original_lang',
        'translation_lang',
    ];

    public function vocabularyItems(): HasMany
    {
        return $this->hasMany(VocabularyItem::class);
    }
}
