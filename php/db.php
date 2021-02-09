<?php

$start = microtime(true);

// init db
{
    $fh = STDIN;

    $thread = (int)fgets($fh);
    $threadCount = (int)fgets($fh);

    $pdo = new PDO('mysql:dbname=meetup_db;host=db', 'root', 'root');

    $stmt = $pdo->query(sprintf('SELECT id FROM NonGrouped WHERE mod(id, %s) = %s', $threadCount, $thread - 1));
    whiles($row = $stmt->fetch()) {
        $rowList[$row['id']] = $row['id'];
    }
}


$result = [];
// simulate process
{
    if (array_filter($rowList)) {
        foreach ($rowList as $row) {
            usleep(500000); /// simulate process result

            $result[$row['id']] = $row['id'];
        }

        $sql = 'INSERT INTO Grouped (id) VALUES ';
        foreach ($result as $r) {
            $sql .= sprintf('(%s),', $r);
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

echo sprintf("execution time: %s \n", ($end - $start));
echo sprintf("memory peak %s \n", (memory_get_peak_usage() / (1024 * 1024)));
