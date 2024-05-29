<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Termorize\Enums\Language;

/**
 * @property int $id
 * @property string $original_text
 * @property string $translation_text
 * @property Language $original_lang
 * @property Language $translation_lang
 * @property bool $is_custom
 */
class Translation extends Model
{
    public const null CREATED_AT = null;
    public const null UPDATED_AT = null;

    protected $fillable = [
        'original_text',
        'translation_text',
        'original_lang',
        'translation_lang',
        'is_custom',
    ];

    protected $casts = [
        'original_lang' => Language::class,
        'translation_lang' => Language::class,
    ];

    public function vocabularyItems(): HasMany
    {
        return $this->hasMany(VocabularyItem::class);
    }
}
