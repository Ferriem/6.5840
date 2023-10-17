num_count=10;
for((i=1; i<=num_count; i++)) do
    echo "Running iteration $i"
    time go test -run 3A -race
    done
wait

