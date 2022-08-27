<?php

function env(string $key) : string
{
return $_ENV[$key];
}