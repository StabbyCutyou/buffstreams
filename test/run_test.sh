#!/bin/bash
for i in `seq 1 ${COUNT:=10}`
do
        GOMAXPROCS=32 go run client/test_client.go 2> log$i.csv &
 done  