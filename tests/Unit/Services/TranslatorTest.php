<?php

namespace Tests\Unit\Services;

use Termorize\Services\Kernel;
use Termorize\Services\Translator;
use Termorize\Models\Translation;
use Tests\TestCase;

class TranslatorTest extends TestCase
{
    public function canSaveTranslationInDatabase()
    {
        // Ask translation service to translate a word

        // Assert that database has this translation
    }

    public function canUseCacheWhenRequestingTheSameTranslation()
    {
        // Ask translation service to translate a word

        // Ask translation service to translate the same word

        // Assert that database was used and no requests were sent
    }
}