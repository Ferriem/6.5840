package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type Coordinator struct {
	files          []string
	NReduce        int
	NMap           int
	MapTaskLog     []int //0 unused, 1 assigned, 2 finished
	ReduceTaskLog  []int
	MapFinished    int
	ReduceFinished int
	lock           sync.Mutex
}

func (c *Coordinator) AllocateTask(args *ExampleArgs, reply *TaskArgs) error {
	c.lock.Lock()
	if c.MapFinished < c.NMap {
		//allocate map task
		c.AssignMapTask(reply)
		fmt.Println("[Cooridinator] AllocateMapTask")
	} else if c.MapFinished == c.NMap && c.ReduceFinished < c.NReduce {
		c.AssignReduceTask(reply)
		fmt.Println("[Cooridinator] AllocateReduceTask")
	} else {
		reply.TaskType = _end
		c.lock.Unlock()
	}
	return nil
}

func (c *Coordinator) HandleTaskReport(task *TaskArgs, reply *ExampleReply) error {
	c.lock.Lock()
	if task.TaskType == _map {
		c.MapTaskLog[task.MapNumber] = 2
		c.MapFinished++
	}
	if task.TaskType == _reduce {
		c.ReduceTaskLog[task.ReduceNumber] = 2
		c.ReduceFinished++
	}
	c.lock.Unlock()
	return nil
}

func (c *Coordinator) AssignMapTask(reply *TaskArgs) {
	allocate := -1
	for i := 0; i < c.NMap; i++ {
		if c.MapTaskLog[i] == 0 {
			allocate = i
			break
		}
	}
	if allocate == -1 {
		reply.TaskType = _wait
		c.lock.Unlock()
	} else {
		reply.NReduce = c.NReduce
		reply.TaskType = _map
		reply.FileName = c.files[allocate]
		reply.MapNumber = allocate
		c.MapTaskLog[allocate] = 1
		c.lock.Unlock()
		go func() {
			time.Sleep(10 * time.Second)
			c.lock.Lock()
			if c.MapTaskLog[allocate] == 1 {
				c.MapTaskLog[allocate] = 0
			}
			c.lock.Unlock()
		}()
	}
}

func (c *Coordinator) AssignReduceTask(reply *TaskArgs) {
	allocate := -1
	for i := 0; i < c.NReduce; i++ {
		if c.ReduceTaskLog[i] == 0 {
			allocate = i
			break
		}
	}
	if allocate == -1 {
		reply.TaskType = _wait
		c.lock.Unlock()
	} else {
		reply.NMap = c.NMap
		reply.TaskType = _reduce
		reply.ReduceNumber = allocate
		c.ReduceTaskLog[allocate] = 1
		c.lock.Unlock()
		go func() {
			time.Sleep(10 * time.Second)
			c.lock.Lock()
			if c.ReduceTaskLog[allocate] == 1 {
				c.ReduceTaskLog[allocate] = 0
			}
			c.lock.Unlock()
		}()
	}
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := c.ReduceFinished == c.NReduce

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	c.files = files
	c.NMap = len(files)
	c.NReduce = nReduce
	c.MapTaskLog = make([]int, c.NMap)
	c.ReduceTaskLog = make([]int, c.NReduce)
	c.MapFinished = 0
	c.ReduceFinished = 0
	c.server()
	fmt.Println("Coordinator initialized")
	return &c
}
