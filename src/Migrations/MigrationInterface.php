<?php

namespace Termorize\Migrations;

interface MigrationInterface
{
    public function getTable(): string;

    public function migrate(): void;
}
