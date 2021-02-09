<?php

require_once('vendor/autoload.php');

use GuzzleHttp\Client;
use GuzzleHttp\Psr7\Response;
use GuzzleHttp\RequestOptions;

use function GuzzleHttp\Promise\settle;

{
    $fh = STDIN;
    $requestCount = (int)fgets($fh);

    $start = microtime(true);

    $pdo = new PDO('mysql:dbname=meetup_db;host=db', 'deployer', 'deployer');
}


$errorCount = 0;
$resultList = [];

// simulate process
{
    $promisePool = [];
    $client = new Client();
    $responseList = [];

    for (; $requestCount >= 0; $requestCount--) {
        $promisePool[] = $client->getAsync('http://worldclockapi.com/api/json/est/now', [
            RequestOptions::HEADERS => [
                'Connection' => 'Close'
            ]
        ]);

        if ($requestCount%400 === 0){
            /** @var Response[] $responseList */
            $responseList = settle($promisePool)->wait();

            foreach ($responseList as $response) {
                if (isset($response['value'])) {
                    // usleep(500000);

                    /** @var Response $resp */
                    $resp = $response['value'];

                    $result = json_decode($resp->getBody()->getContents(), true);

                    $resultList[] = $result['currentDateTime'];

                    continue;
                }

                $errorCount++;
            }

            $responseList = [];
        }
    }
}

// process db
{
    if (array_filter($resultList)) {
        $sql = 'INSERT INTO Parsing (time) VALUES ';

        foreach ($resultList as $r) {
            $sql .= sprintf("('%s'),", (new DateTime($r))->format('Y-m-d H:i:s'));
        }

        $sql = trim($sql, ',');

        $ok = $pdo->exec($sql);
        if ($ok === false) {
            echo 'error occurred';
            die;
        }
    } else {
        echo 'empty row list !!!!!';
    }
}


$end = microtime(true);

echo sprintf("execution time = %s \n", ($end - $start));
echo sprintf("Sys memory = %s \n", (memory_get_peak_usage(true) / (1024 * 1024)));
echo sprintf("error count = %s \n", $errorCount);
