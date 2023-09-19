## Lecture-2: Thread and RPC

### Thread

In package `sync`, `sync.Mutex` implement a lock to `Lock()` and `Unlock()`

### Channel

```go
c := make(chan, type)
c <- x
y := <-c
for y := range c
```

Channel implicit imply lock. A receiver waits until some grouting sends.

Sender block until the receiver receive.  If receiver close while sender still sending, it will waste resources.

```go
var mu sync.Mutex
cond := sync.Cond(&mu)

cond.Wait()
cond.Broadcast()
```

### RPC (Remote Procedure Call)

Use package `net/rpc`,

```c
Software structure
  client app        handler fns
   stub fns         dispatcher
   RPC lib           RPC lib
     net  ------------ net
```

```go
err := client.Call("KV.Put", &args, &reply)
func (kv *KV) Put(args *PutArgs, reply *PutReply) error
```

Failures

- "Best-effort RPC"

- "At-most-once"(Go RPC)
  - Open TCP connection
  - Write request to TCP connection
  - Go RPC never re-sends a request
  - Go RPC code returns an error if doesn't get a reply
    - timeout
    - server didn't see request
    - server/net failed

