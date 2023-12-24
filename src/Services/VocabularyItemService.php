<?php

namespace Termorize\Services;

use Termorize\Models\Translation;

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
