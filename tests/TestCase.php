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

    protected function setUp(): void
    {
        parent::setUp();

        // Refresh database using your class
    }

    protected function tearDown(): void
    {
        Mockery::close();

        parent::tearDown();
    }

    protected function mockCascade(array $methods, string $class = ''): MockInterface
    {
        $methodsWithReadyMocks = [];

        foreach ($methods as $method => $returns) {
            if (is_array($returns)) {
                $readyMock = $this->mockCascade($returns);
            } else {
                $readyMock = $returns;
            }

            $methodsWithReadyMocks[$method] = $readyMock;
        }

        if (empty($class)) {
            return Mockery::mock($methodsWithReadyMocks);
        } else {
            return Mockery::mock($class, $methodsWithReadyMocks);
        }
    }
}