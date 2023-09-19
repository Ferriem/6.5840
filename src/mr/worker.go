package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		task := CallTask()
		is_failed := false
		switch task.TaskType {
		case _map:
			is_failed = WorkerMap(mapf, task)
		case _reduce:
			is_failed = WorkerReduce(reducef, task)
		case _wait:
			time.Sleep(time.Second)
		case _end:
			return
		default:
			log.Fatalln("unknown task type")
		}
		if task.TaskType == _map || task.TaskType == _reduce {
			if is_failed {
				task.TaskType = _failed
			}
			CallTaskReport(task)
		}
		is_failed = false
	}
}

func CallTask() *TaskArgs {
	args := ExampleArgs{}
	reply := TaskArgs{}
	err := call("Coordinator.AllocateTask", &args, &reply)
	if !err {
		fmt.Println("call task failed")
		reply.TaskType = _wait
	}
	return &reply
}

func CallTaskReport(task *TaskArgs) {
	reply := ExampleReply{}
	err := call("Coordinator.HandleTaskReport", &task, &reply)
	if !err {
		fmt.Println("call task report failed")
	}
}

func WorkerMap(mapf func(string, string) []KeyValue, task *TaskArgs) bool {
	intermediate := []KeyValue{}
	file, err := os.Open(task.FileName)
	if err != nil {
		log.Fatalf("cannot open %v", task.FileName)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", task.FileName)
		return true
	}
	file.Close()

	//call mapf
	kva := mapf(task.FileName, string(content))
	intermediate = append(intermediate, kva...)

	//hash into bucket
	bucket := make([][]KeyValue, task.NReduce)
	for i := range bucket {
		bucket[i] = []KeyValue{}
	}
	for _, kv := range intermediate {
		bucket[ihash(kv.Key)%task.NReduce] = append(bucket[ihash(kv.Key)%task.NReduce], kv)
	}

	//write into intermediate file
	for i := range bucket {
		oname := "mr-" + strconv.Itoa(task.MapNumber) + "-" + strconv.Itoa(i)
		ofile, _ := os.CreateTemp("", oname)
		enc := json.NewEncoder(ofile)
		for _, kv := range bucket[i] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Fatalf("cannot write %v", oname)
			}
		}
		os.Rename(ofile.Name(), oname)
		ofile.Close()
	}

	fmt.Println("[Worker] MapTask Finished")
	return false
}

func WorkerReduce(reducef func(string, []string) string, task *TaskArgs) bool {
	intermediate := []KeyValue{}
	for i := 0; i < task.NMap; i++ {
		iname := "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(task.ReduceNumber)
		//open && read intermediate file
		file, err := os.Open(iname)
		if err != nil {
			log.Fatalf("cannot open %v", iname)
			return true
		}

		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}
	//sort
	sort.Sort(ByKey(intermediate))

	//output
	oname := "mr-out-" + strconv.Itoa(task.ReduceNumber)
	ofile, err := os.CreateTemp("", oname)
	if err != nil {
		log.Fatalf("cannot open %v", oname)
		return true
	}
	for i := 0; i < len(intermediate); {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	os.Rename(ofile.Name(), oname)
	ofile.Close()

	fmt.Println("[Worker] ReduceTask Finished")
	return false
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
