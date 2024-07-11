<?php

function getBasePath(string $subDir = ''): string
{
    return __DIR__ . '/../../' . $subDir;
}

function mb_levenshtein(string $s1, string $s2): int
{
    $charMap = [];
    $s1 = utf8_to_extended_ascii($s1, $charMap);
    $s2 = utf8_to_extended_ascii($s2, $charMap);

    return levenshtein($s1, $s2);
}

function utf8_to_extended_ascii($str, &$map)
{
    $matches = [];
    if (!preg_match_all('/[\xC0-\xF7][\x80-\xBF]+/', $str, $matches)) {
        return $str;
    }

    foreach ($matches[0] as $mbc) {
        if (!isset($map[$mbc])) {
            $map[$mbc] = chr(128 + count($map));
        }
    }

    return strtr($str, $map);
}
