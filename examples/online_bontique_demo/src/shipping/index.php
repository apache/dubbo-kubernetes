<?php

$uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$method = $_SERVER['REQUEST_METHOD'];

if ($uri === '/healthz' && $method === 'GET') {
    header('Content-Type: application/json');
    echo json_encode(['status' => 'ok']);
    exit(0);
}

function getRandomLetterCode(): string {
    return chr(65 + rand(0, 25));
}

function getRandomNumber(int $digits): string {
    $str = '';
    for ($i = 0; $i < $digits; $i++) {
        $str .= (string)rand(0, 9);
    }
    return $str;
}

function createTrackingId(string $salt): string {
    return sprintf(
        '%s%s-%d%s-%d%s',
        getRandomLetterCode(),
        getRandomLetterCode(),
        strlen($salt),
        getRandomNumber(3),
        intdiv(strlen($salt), 2),
        getRandomNumber(7)
    );
}

if ($uri === '/quote' && $method === 'POST') {
    header('Content-Type: application/json');
    echo json_encode([
        'costUsd' => [
            'currencyCode' => 'USD',
            'units' => 8,
            'nanos' => 99000000,
        ],
    ]);
    exit(0);
}

if ($uri === '/ship' && $method === 'POST') {
    $input = json_decode(file_get_contents('php://input'), true) ?? [];
    $addr = $input['address'] ?? [];
    $street = $addr['streetAddress'] ?? '';
    $city = $addr['city'] ?? '';
    $state = $addr['state'] ?? '';
    $baseAddress = sprintf('%s, %s, %s', $street, $city, $state);
    $trackingId = createTrackingId($baseAddress);

    header('Content-Type: application/json');
    echo json_encode(['trackingId' => $trackingId]);
    exit(0);
}

http_response_code(404);
header('Content-Type: application/json');
echo json_encode(['error' => 'Not Found']);
