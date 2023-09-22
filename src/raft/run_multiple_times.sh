#!/bin/bash

num_instance=1 # Change this to the desired number of instances

count=10  # Change this to the desired number of times to run the program
for ((i=1; i<=num_instance; i++)); do
    (
    for((j=1; j<=count; j++)); do
        echo "Running iteration $j for instance $i"
        time go test -run 2C -race
    done
    wait
    ) &
done


