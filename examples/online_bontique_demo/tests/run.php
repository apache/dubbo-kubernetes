<?php

function isValid(array $m): bool {
    $nanos = $m['nanos'];
    $units = $m['units'];
    $signMatches = ($nanos === 0 || $units === 0 || ($nanos < 0) === ($units < 0));
    $validNanos = ($nanos >= -999999999 && $nanos <= 999999999);
    return $signMatches && $validNanos;
}

function resetMoney(array &$m): void {
    $m['units'] = 0;
    $m['nanos'] = 0;
}

function sumMoney(array $a, array $b): array {
    if (!isValid($a) || !isValid($b)) {
        throw new InvalidArgumentException("Invalid money value");
    }
    if ($a['currencyCode'] !== $b['currencyCode']) {
        throw new InvalidArgumentException("Mismatching currency codes");
    }

    $units = $a['units'] + $b['units'];
    $nanos = $a['nanos'] + $b['nanos'];

    if (($units >= 0 && $nanos >= 0) || ($units < 0 && $nanos <= 0)) {
        $units += intdiv($nanos, 1000000000);
        $nanos %= 1000000000;
    } else {
        if ($units > 0) {
            $units--;
            $nanos += 1000000000;
        } else {
            $units++;
            $nanos -= 1000000000;
        }
    }

    return [
        'currencyCode' => $a['currencyCode'],
        'units' => $units,
        'nanos' => $nanos,
    ];
}

function multiplySlow(array $m, int $multiplier): array {
    $result = $m;
    for ($i = 1; $i < $multiplier; $i++) {
        $result = sumMoney($result, $m);
    }
    return $result;
}

$vectorPath = __DIR__ . '/../contracts/money_vectors.json';
if (!file_exists($vectorPath)) {
    $vectorPath = __DIR__ . '/contracts/money_vectors.json';
}

$data = json_decode(file_get_contents($vectorPath), true);
if (!$data) {
    fwrite(STDERR, "Failed to load money_vectors.json\n");
    exit(1);
}

$passed = 0;
$total = 0;

// Test validity
foreach ($data['validity_vectors'] as $v) {
    $total++;
    $got = isValid($v['input']);
    if ($got !== $v['expected_valid']) {
        fwrite(STDERR, sprintf("Validity failed for branch %s: got %s, expected %s\n", $v['branch'], json_encode($got), json_encode($v['expected_valid'])));
        exit(1);
    }
    $passed++;
}

// Test sum
foreach ($data['sum_vectors'] as $v) {
    $total++;
    if (!empty($v['expect_error'])) {
        try {
            sumMoney($v['a'], $v['b']);
            fwrite(STDERR, sprintf("Sum expected error for branch %s, but succeeded\n", $v['branch']));
            exit(1);
        } catch (InvalidArgumentException $e) {
            $passed++;
        }
    } else {
        $res = sumMoney($v['a'], $v['b']);
        if ($res !== $v['expected']) {
            fwrite(STDERR, sprintf("Sum failed for branch %s: got %s, expected %s\n", $v['branch'], json_encode($res), json_encode($v['expected'])));
            exit(1);
        }
        $passed++;
    }
}

// Test multiply
foreach ($data['multiply_slow_vectors'] as $v) {
    $total++;
    $res = multiplySlow($v['input'], $v['multiplier']);
    if ($res !== $v['expected']) {
        fwrite(STDERR, sprintf("Multiply failed for branch %s: got %s, expected %s\n", $v['branch'], json_encode($res), json_encode($v['expected'])));
        exit(1);
    }
    $passed++;
}

// Test reset
foreach ($data['reset_vectors'] as $v) {
    $total++;
    $m = $v['input'];
    resetMoney($m);
    if ($m !== $v['expected']) {
        fwrite(STDERR, sprintf("Reset failed for branch %s: got %s, expected %s\n", $v['branch'], json_encode($m), json_encode($v['expected'])));
        exit(1);
    }
    $passed++;
}

echo "PHP MoneyUtils tests passed ($passed/$total)\n";
exit(0);
