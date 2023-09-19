## Lecture-4: Primary/backup replication

[paper](https://pdos.csail.mit.edu/6.824/papers/vm-ft.pdf)

### Overview

- State-machine replication

- Two main replication approches

  - State transfer
    - Primary executes the service
    - Primary sends state snapshots over network to a storage system
  - **Replicated state machine**
    - Clients send operations to primary
      - Primary sequences and sends to backup
    - All replicas execute all operations 
    - If same start state, operations, orderm deterministic, then same state.

  State transfer is conceptually simple, but state may be large, slow to transfer over network.

  Replicated state machine often generates less network traffic

  - Operations are often smaller than state but more complex to get right.
  - VM-FT(fault-tolerance) uses replicated state machine

- Failure

  - Replication is good for "fail-stop" failures, if there is a failure, stop computer.

  - Replication didn't go well dealing with logic bugs and configuration errors.

- Challenge
  - The distributed system can't tell the difference between a network petition and a machine fail. Avoid a split-brain system. (Use a flag in storage server and apply test-and-set).
  - Keep Primaty/backup in sync.
  
- Level of operations to replicate
  - Application state
  - Machine level

- VMFT: exploit virtualization

  - Transparent replication

  - appears to client that server is a single machine
  
    <img src="/Users/ferriem/Desktop/ferriem/6.5840/md/image/primary-backup.png" alt="primary-backup" style="zoom:50%;" />

### Shared Disk

Shared disk are connected to primary and backup, there're two configurations

- **Shared Storage**: In some cases, both the primary and backup nodes access the same storage backend. This allows for simpler synchronization of data between the nodes since they are reading and writing to the same storage location. However, it can also introduce potential points of failure if the shared storage becomes unavailable.
- **Split Storage**: Alternatively, the primary and backup nodes might have seperate storage instances that are kept in sync through replication mechanisms. This approach can provide better fault isolation. However, it can also introduce additional complexity in terms of synchronization and data consistency.

Beside the storage, the shared disk also has an arbitration server, it is used to determine which node should act as the primary node when there is a conflict or uncertainly about the state of system.

**Flag** in an arbitration server are usually used to communicate the status or availability of node participating in the replication process. Flags can take various forms, such as binary value (0 or 1) or different states (ACTIVE, PASSIVE, DOWN, etc.).

<table>
  <tr>
    <th rowspan="2">Event</th>
    <th colspan="2">Flag value</th>
  </tr>
  <tr>
    <th>Primary Node A</th>
    <th>Backup Node B</th>
  </tr>
  <tr>
    <th>Normal Operation</th>
    <th>1 (Active)</th>
    <th>0 (Passive)</th>
  </tr>
  <tr>
    <th>Primary Node Failure</th>
    <th>0 (Passive)</th>
    <th>1 (Active)</th>
  </tr>
  <tr>
    <th>Primary Node Recovery</th>
    <th>1 (Active)</th>
    <th>0 (Passive)</th>
  </tr>
  <tr>
    <th>Backup Node Failure</th>
    <th>1 (Acvtive)</th>
    <th></th>
  </tr>
  <tr>
    <th>Arbitration Server Failure</th>
    <th></th>
    <th></th>
  </tr>
</table>


In Primary node recovery, it just take a revert strategy. 

### Divergence

Divergence sources

- Non-deterministic instruction (eg. get the time)
- Input packets or timer interrupt in same point in the instruction stream
- Multi-core: the winner in the primary should be the winner of the backup.

FT  handling of timer interrupts.

- Primary:

  FT fields the timer interrupt

  FT reads instruction number from CPU

  FT sends "timer interrupt at instuction # X" on logging channel

  FT delivers interrupt to primary, and resume it.

- Backup

  ignores its own timer interrupt

  FT sees log entry *before* backup gets to instruction # X

  FT tells CPU to tansfer control to FT at instruction # X

  FT mimics a timer interrupt that backup guests sees.

FT handling of network package arrival

- Primary:

  FT configures NIC (network interface card) to write package data into FT's private "bounce buffer"

  At some point a packet arrives, NIC does DMA (direct memory access), then interrupts

  FT gets the interrupt, reads instruction # from CPU

  FT pauses the primary

  FT copies the bounce buffer into the primary's memory

  FT simulates a NIC interrupt in primary FT send the packet data and the instruction # to the backup

- Backup:

  FT gets data and instruction # from log stream

  FT tells CPU to interrupt (to FT) at instructon # X

  FT copies the data to guest memory, simulates NIC interrupt in backup

Output:

Suppose clients send "increase" request, the value begin with 10.

- The primary sends on logging channel to backup

- Primary executes, sets value to 11, send 11 reply, FT send reply
- Backup executes, sets value to 11, send 11 reply, FT discard it.

If primary crash when it send reply

- clients get the "11" reply and the logging channel discards the log entry w/ client request, primary is dead, so it won't re-send
- backup goes live but with value 10 in its memory.
- a client send another increment request but get "11" again not "12".

**Output Rule**: For this situation, before primary sends output, must wait for backup to acknowledge all previous log entries

The backup **lags behind one message**.

The examples above are used to explain how to deal with non-deterministic instruction, we have **no need to do with deterministic instruction through logging channel**.

