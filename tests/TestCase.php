<?php

namespace Tests;

use Mockery;
use Mockery\MockInterface;
use PHPUnit\Framework\TestCase as BaseTestCase;
use Termorize\Services\Kernel;
use Tests\Utils\DatabaseRefresher;
use ReflectionClass;

class TestCase extends BaseTestCase
{
    protected function mockPrivateProperty($object, string $propertyName, $value)
    {
        $reflection = new ReflectionClass($object);
        $property = $reflection->getProperty($propertyName);
        $property->setAccessible(true);
        $property->setValue($object, $value);

    }
    protected function connectDatabase(): void
    {
        $kernel = new Kernel();
        $kernel->connectDatabase();
    }

    protected function makeAlias(string $class): MockInterface
    {
        return Mockery::mock("alias:$class");
    }

    protected function setUp(): void
    {
        parent::setUp();
        $this->connectDatabase();
        //DatabaseRefresher::clearDatabase();
    }

    protected function tearDown(): void
    {
        Mockery::close();

        parent::tearDown();
    }

    protected function mockCascade(array $methods): MockInterface
    {
        $class = null;

        $methodsWithReadyMocks = [];

        foreach ($methods as $method => $returns) {
            if ($method === '__class') {
                $class = $returns;

                continue;
            }

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
