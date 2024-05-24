<?php

namespace Termorize\Services;

use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;

class VocabularyItemService
{
    public function save(Translation $translation, int $userId): ?VocabularyItem
    {
        $itemAlreadyExists = $translation->vocabularyItems()->where('user_id', $userId)->exists();

        if ($itemAlreadyExists) {
            return null;
        }

        return $translation->vocabularyItems()->create([
            'user_id' => $userId,
            'knowledge' => 0,
        ]);
    }
}
