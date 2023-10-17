## Lecture-9: Consistency and Linearizability

### Correctness

Consider a simple key-value store, similar to [etcd](https://github.com/etcd-io/etcd), that maps strings to strings and supports two operations: `Put(key, value)` and `Get(key, value)`. First, we consider how it behaves in the **sequential case**.

#### Sequential Specifications

`Get` operation must replect the result of applying all previous `Put` operation. 

```python
class KVStore:
    def __init__(self):
        self._data = {}

    def put(self, key, value):
        self._data[key] = value

    def get(self, key):
        return self._data.get(key, "")
```

#### Linearizability

Next, we consider how our key-value store can behave under concurrent operation.

![img](https://anishathalye.com/_next/static/files/2b0e0147b19476508c9d4fe9aa8cfbeb/operation-1.svg)

It's not immediately obvious what value the `Get("x")` operation should be allowed to return. It can be "y", " ", "z".

We formally specify correctness for concurrent operations based on a sequential specification using a consistency model known as [linearizability](https://cs.brown.edu/~mph/HerlihyW90/p463-herlihy.pdf). In a linearizable system, **every operation appears to execute atomically and instantaneously at some point between the invocation and response.**

Consider an example history with invocations and return values of operations on a key-value store: ![img](https://anishathalye.com/_next/static/files/dcad052c9ca58fa8ed15a71baf4a51b7/history-1.svg)

![img](https://anishathalye.com/_next/static/files/c6a0419af74449cfeebe1d8fea316d81/history-1-linearization.svg)

If we can assign linearization points to operations in the history, we can call it linearizable.

**GFS** is not linearizable: client can read a value from a replica that hasn't yet been updated. If we wanted GFS to be linearizable, client reads would have to go through the primary too, and wait for outstanding writes to complete.

Similarly, for Lab 3, clients can only **read from the leader**, not the follower.

Linearizability **forbids** mant situations:

- Split brain
- Forgetting committed writes after reboot
- Reading from lagging replicas.

When network error occurs,  client may re-send request. To ensure linearizability, **duplicate requests from retransmissions must be suppressed**. 

### Testing

With a solid definition of correctness, we can think about how to test distributed systems. The general approach is to test for correct operation while randomly injecting faults such as machine failures and network partictions.

#### Ad-hoc testing

```text
for client_id = 0..10 {
    spawn thread {
        for i = 0..1000 {
            value = rand()
            kvstore.put(client_id, value)
            assert(kvstore.get(client_id) == value)
        }
    }
}
wait for threads
```

If the above test fails, then the key-value store is not linearizable. However, this test is not thjat thorough: there are non-linearizable key-value stores that would always pass this test.

#### Linearizability

A better test would be to have parallel clients run **completely random operations**: repeatedly calling `kvstore.put(rand(), rand())` and `kvstore.get(rand())`.

While every operation has more than one right answer. So we have to take an alternative approach: we can test for correctness by recording an entire history of operations on the system and then checking if the history is linearizable with respect to the sequential specification.

- Linearizability Checking 

  A linearizability checker takes as input a sequential specification and a concurrent history, and it runs a decision procedure to check whether the history is linearizable with respect to the spec.

- NP-Completeness

  Unfortunately, linearizability checking is [NP-complete](https://en.wikipedia.org/wiki/NP-completeness). 

- Implementation

  Even though linearizability is NP-complete, in practice, it can work pretty well on small histories. There are existing linearizability checkers like [Knossos](https://github.com/jepsen-io/knossos). And [Porcupine](https://github.com/anishathalye/porcupine), a fast linearizability checker implemented in Go.

### Effectiveness

The ad-hoc tests caught some of the most egregious bugs, but the tests were incapable of catching the more subtle bugs. In contrast, cannot introduce a single correctness bug that the lineariability test couldn't catch.

- Formal methods can provide strong guarantees about the correctness of distributed systems. 
- Ideally, all production systems would have formal specifications. Some systems that are being used in the real world today do have formal specs: eg. Raft.