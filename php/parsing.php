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

    $pdo = new PDO('mysql:dbname=meetup_db;host=127.0.0.1', 'deployer', 'deployer');
}


$errorCount = 0;
$resultList = [];

// simulate process
{
    $promisePool = [];
    $client = new Client();
    $responseList = [];
    $resultCount = 0;
    $maxRequestCount = $requestCount;

    for (; $requestCount >= 0; $requestCount--) {
        $promisePool[] = $client->getAsync('http://sam.wake-app.net/time', [
            RequestOptions::TIMEOUT => 30,
            RequestOptions::HEADERS => [
                'Connection' => 'Keep-Alive',
            ]
        ]);

        if ($requestCount % 300 === 0) {
            /** @var Response[] $responseList */
            $responseList = settle($promisePool)->wait();

            echo sprintf("%s/%s \n", $requestCount, $maxRequestCount);

            foreach ($responseList as $response) {
                if (isset($response['value'])) {
                    /** @var Response $resp */
                    $resp = $response['value'];

                    $result = json_decode($resp->getBody()->getContents(), true);

                    $resultList[] = $result['currentDateTime'];
                    $resultCount++;

                    if ($resultCount >= 5000) {
                        if (array_filter($resultList)) {
                            $sql = 'INSERT INTO Parsing (time) VALUES ';

                            foreach ($resultList as $r) {
                                $sql .= sprintf("('%s'),",
                                    (DateTime::createFromFormat('Y-d-m\TH:iP', $r))->format('Y-m-d H:i:s')
                                );
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

                        $resultCount = 0;
                        $resultList = [];
                    }


                    continue;
                }

                $errorCount++;
            }

            $responseList = [];
            $promisePool = [];
            $client = new Client();
        }
    }
}

// process db
{
    if (array_filter($resultList)) {
        $sql = 'INSERT INTO Parsing (time) VALUES ';

        foreach ($resultList as $r) {
            $sql .= sprintf(
                "('%s'),",
                (DateTime::createFromFormat('Y-d-m\TH:iP', $r))->format('Y-m-d H:i:s')
            );
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
echo sprintf("Sys memory = %s \n", (memory_get_usage(true) / 1024 / 1024));
echo sprintf("error count = %s \n", $errorCount);
