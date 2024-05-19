<?php

namespace Termorize\Migrations;

interface MigrationInterface
{
    public function alreadyExecuted(): bool;

    public function migrate(): void;
}
