<?php

$uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$method = $_SERVER['REQUEST_METHOD'];

if ($uri === '/healthz' && $method === 'GET') {
    header('Content-Type: application/json');
    echo json_encode(['status' => 'ok']);
    exit(0);
}

if ($uri === '/send-order-confirmation' && $method === 'POST') {
    $input = json_decode(file_get_contents('php://input'), true) ?? [];
    $email = $input['email'] ?? '';
    $order = $input['order'] ?? '';
    error_log(sprintf("Sending order confirmation to %s for order %s\n", $email, $order));
    header('Content-Type: application/json');
    echo json_encode(['message' => 'Order confirmation sent successfully!']);
    exit(0);
}

http_response_code(404);
header('Content-Type: application/json');
echo json_encode(['error' => 'Not Found']);
