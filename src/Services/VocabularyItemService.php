<?php

namespace Termorize\Services;

use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;

class VocabularyItemService
{
    public function save(Translation $translation, int $userId): void
    {
        $translation->vocabularyItems()->create([
            'user_id' => $userId,
            'knowledge' => 0,
        ]);

    }
}
