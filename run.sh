#!/bin/bash
trap "rm server;kill 0" EXIT # trap命令用于在shell脚本退出时，删除临时文件，结束子进程

go build -o server

./server -port=8002 -api=1 &
./server -port=8003  &
./server -port=8001  &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
sleep 2
curl "http://localhost:9999/api?key=Jack" &
sleep 2
curl "http://localhost:9999/api?key=Sam" &

wait