<?php

function getBasePath(string $subDir = ''): string
{
    return __DIR__ . '/../../' . $subDir;
}
