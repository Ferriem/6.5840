## Lecture-12: Distributed transaction

Distributed transactions = concurrency control + atomic commit

### Concurrenct control

Isolated/serializable execution of concurrent transactions

Two classes of concurrency control for transaction:

- Pessimistic: 

  - lock records before use
  - conflicts cause  delays (waiting for locks)

- optimistic

  - use records without locking
  - commit checks if reads/writes were serializable (linearizable)
  - conflict causes abort + retry
  - called Optimistic Concurrency Control (OCC)

  pessimistic is faster if conflicts are frequent. 

#### Two-phase locking

Two-phase locking is one way to implement serializability.

2PL rules: 

- a transaction must acquire a record's lock **before using it**
- a transaction must hold its locks until ***after* commit or abort**

details

- each database record has a lock
- an executing transaction acquires locks as needed, at the first use `add()` and `get()` implicitly acquires record's lock `END-X()` releases all locks
- all locks are exclusive the full name is "strong strict two-phase locking" related to thread locking but:
  - programmer doesn't explicitly lock
  - programer supplies BEGIN-X/END-X
  - DB locks automatically, at transaction end
  - DB may automatically abort to cure deadlock

2PL rules can cause dead lock, below are some solutions

- Timeouts:  If a transaction takes too long to acquire all the necessary locks, it may be rolled back, allowing other transaction to proceed.
- Deadlock Detection: Periodically, the system checks for circular wait conditions. If a deadlock is detected, one or more transactions involved in the deadlock are rolled back to break the circular dependency.
- Wait-Die and Wound-Wait Schemes: Techniques used in database systems to handle deadlocks. In the Wait-Die scheme, older transactions wait for younger ones to release locks, while in Wound-Wait scheme, younger transactions are forced to wait or be terminated if they request a lock held by an transaction.
- Transaction Priority: Assigning priorities to transactions can help resolve deadlocks.
- Resource Allocation Graphs: Constructing a graph where transactions are nodes, and edges represent the resources they are waiting for. Deadlock can be detected by finding cycles in this graph.

2PL may forbid a correct serializable execution.

### Distributed transactions cope with failures

#### Atomic commit

A protocol called "**two-phase commit** used by distributed database for multi-server transactions.

- Data is sharded among multiple servers
- Transactions run on "transaction **coordinators**" (TCs)
- For each read/write TC sends RPC to relevant shard server
  - Each is a "**participant**"
  - Each participant manages **locks** for its shard of the data
- There may be many concurrent transactions, many TCs
  - TC assigns unique transaction ID (TID) to each transaction
  - Every message, every piece of xaction state **tagged with TID**
  - To avoid confusion

Two-phase commit without failures:

```
TC sends put(), get(), &c RPCs to A, B
A and B lock records
Modifications are tentative, only installed if commit
TC gets to the end of the transaction
TC send PREPARE messages to A and B
If A is able to commit
	A reponse Tes
	then A is in "prepared" state
otherwise, A responds No
Same for B

If both A and B said YES, TC sends COMMIT messages to A and B
If eitehr A or B said No, TC sends ABORT messages.
A/B commit if they get a COMMIT message from the TC
	they copy tentative records to the real DB,
	And release the transaction's locks on their records
A/B acknowledge COMMIT message.
```

Participants must **remember on disk before saying YES**, includinf modified data.

- If participants reboots, and disk says YES but didn't receive COMMIT from TC, it must ask TC, or wait for TC to re-send.
- Meanwhile, participant must continue to hold the transaction's locks.
- If TC says COMMIT, participant copies modified data to real data.

TC must remember if it has sent COMMIT before crash.

- TC must **write COMMIT to disk before sending** COMMIT msgs.
- Repeat COMMIT if it crashes and reboots.
- Participants must filter out duplicate COMMITs (using TID)

Raft and two-phase commit solve different problems:

- Use Raft to get high availablity by replicating (all server do the *same* thing)
- Use 2PC when each participant does something different (*all* of them must di their part)

High availability and atomic commit

- The TC and servers should each be replicated with Raft
- Run two-phase commit among the replicated services
