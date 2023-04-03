<?php

namespace Tests\Unit\Services;

use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;
use Termorize\Services\VocabularyItemService;
use Tests\TestCase;

class VocabularyItemServiceTest extends TestCase
{
    /**
     * @test
     */
    public function canCreateVocabularyItemsTest()
    {
       /** @var Translation $translation */
       $translation = Translation::query()->create([
           'original_text' => 'ьшщагши',
           'translation_text' => 'iofsdfg',
           'original_lang' => 'ru',
           'translation_lang' => 'en',
       ]);

       $service = new VocabularyItemService();
       $service->save($translation, 1);
       /** @var VocabularyItem $item */
       $item = VocabularyItem::query()->first();
       $this->assertEquals(1, $item->user_id);
       $this->assertEquals(1, $item->translation_id);
       $this->assertEquals(0, $item->knowledge);

    }
}
