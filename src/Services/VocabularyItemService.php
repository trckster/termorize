<?php

namespace Termorize\Services;

use Exception;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Str;
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

    public function deleteItem(int $uid, string $word): void
    {
        $word = Str::lower($word);

        $vocabularyItems = VocabularyItem::query()
            ->where('user_id', $uid)
            ->whereHas('translation', function (Builder $query) use ($word) {
                $query->where('original_text', $word)
                    ->orWhere('translation_text', $word);
            })
            ->get();

        if ($vocabularyItems->isEmpty()) {
            throw new Exception('Nothing to delete');
        }

        $vocabularyItems->each(function (VocabularyItem $item) {
            $item->questions()->delete();
            $item->delete();
        });
    }
}
