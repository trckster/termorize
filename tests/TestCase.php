<?php

namespace Tests;

use Mockery;
use Mockery\MockInterface;
use PHPUnit\Framework\TestCase as BaseTestCase;

class TestCase extends BaseTestCase
{
    protected function makeAlias(string $class): MockInterface
    {
        return Mockery::mock("alias:$class");
    }

    protected function tearDown(): void
    {
        Mockery::close();

        parent::tearDown();
    }
}