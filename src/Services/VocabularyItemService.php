<?php

namespace Termorize\Services;

use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;

class VocabularyItemService
{
    public function save(Translation $translation, int $userId)
    {
        VocabularyItem::query()->create([
            'translation_id' => $translation->id,
            'user_id' => $userId,
            'knowledge' => 0,
        ]);
    }
}
