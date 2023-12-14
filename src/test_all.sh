#!/bin/bash

echo "Running raft"
time go test ./raft
echo "Running kvraft"
time go test ./kvraft
echo "Running shardctrler"
time go test ./shardctrler
echo "Running shardkv"
time go test ./shardkv
wait

done


