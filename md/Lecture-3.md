## Lectrue-3 (GFS)

[GFS paper](https://pdos.csail.mit.edu/6.824/papers/gfs.pdf)

To implement a good storage system is hard.

```
high performance => shared data across servers
many servers => constant faults
faults tolerance => replication
replication => potential inconsistency
strong consistenct => lower performance
```

### Ideal consistency

Behave as a single system. 

### GFS

GFS is a file system for mapreduce.

The paper touched to four major theme of 6.5840

- Performance, fault tolerance, replication, consistency

Overview: 

- 100s/1000s of clients
- 100s of chunkservers
- one coordinator

Key properties:

- Big: large data set
- Fast: automatic sharing
- Global: all apps see same system

- Fault tolerant: atomic

### Design overview

#### Chunk

See `Figure 1` in the paper. The Application spilt big file into 64MB chunks.

Each **64MB** chunk stored (replicated) on three chunkservers. Client wries are send to all of a chunk's server, a read just needs to consults one copy.

A large chunk size **reduce clients' need to interact** with the coordinator and a client is more likely to perfor many operations on a given chunk, it **reduce network overhead**. Moreover, it **reduces the size of the metadata** stored on the coordinator.

Disadvantage also exist, if a small file consists of a small number of chunks, perhaps just one. The chunkservers storing those chunk may become **hot spots**. In practice, hot spots have not been a major issue. (higher replication factor and stagger application start times can ease the problem)

#### Client

Neither client nor the chunkserver caches file data. (Clients do cache metadata however.)

Client never read and write file data through coordinator, it just asks for information and interacts with the chunkservers directly for many subsequent operations.

#### Coordinator

Coordinator maintains all file system metadata. Including namespace, access control information, the mapping from files to chunks, the current locations of chunks, chunk lease management, garbage collection of orphaned chunks, chunk migration between chunkservers. The coordinator periodically communicates with each chunkservers in *HeartBeat* message to give it instructions and collect its state.

```
file name -> array of chunkhandle(stable storage)
chunkhandle -> version# (stable storage)
							 list of chunkservers
							 primary,secondary
							 lease time
							 
log + checkpoints (sits on stable storage)
//checkpoints is a way to check whether the storage is valid, when a coordinator crash, recovery imformation are stored in log, first check the checkpoints to avoid source waste in repeat recovery.
```

#### Metadata

The coordinator stores three major types of metadata: **the file and chunk namespaces**, **the mapping from files to chunks**, **the locations of each chunk's replicas**. All metadata is kept in coordinator's memory. **The first two (namespaces and file-to-chunk mapping) are also kept persistent by logging mutations to an *operation log* stored on the coordinator's local disk** and replicated on remote machines. Using a log allows us to update the coordinator state simply, reliably, and without risking inconsistencies in the event of coordinator crash. **The coordinator doesn't store chunk location information persistently**, it **asks each chunkserver** about its chunk at coordinator startup and whenever chunkserver joins the cluster.

Recovery needs only the latest complete checkpoint and subsequent log files.

#### Consistency Model

For GFS

- File namespace mutations are **atomic**. They are handled exclusively by the coordinator.
- The state of a file region after a data mutation depends on the type of mutation.
- Data mutations may be *writes or record appends*.
- After a sequence of successful mutations, the mutated file region is guaranteed to be defined and contain the data writen by the last mutation.
- Since clients cache chunk locations, they may read from a **stale** replica before that information is refreshed.
- Once a problem surfaces, the data is restored from valid replicas as soon as possible.

For applications

- relying on appends rather than overwrites, checkpointing, and writing self-validating, self-identifying records.
- In one typical use, a writer generates a file form beginning to end. It **atomically renames** the file to a permanent name after writing all the data, or periodically chekpoints how much has beed successfully writen. Checkpoints may also include application-level checksums. Readers vertify and process only the file region up to the last checkpoint (defined state).

#### Mutation order

We use leases to maintain a consistent mutation order across replicas. The coordinator grants a chunk lease to one of the replicas, which we call the *primary*. The primary picks a serial order for all mutaitions to the chunk. All replicas follow this order when applying mutations.

A lease has an initial timeout of **60 seconds**.

- Steps to read a file

```
1.C sends filename and offset to coordinator(CO) (if not cached)
	CO has a filename -> array-of-chunkhandle table
	and a chunkhandle -> list-of-chunkservers table
2.CO finds chunk handle for that offset
3.CO replies with chunkhandle + list of chunkservers + version#
4.C caches handle + chunkserver list
5.C sends request to nearest chunksercer
	chunk handle, offset
6.chunk server check version#, reads from chunk file on disk, return to client
```

Write:

One write should send to one more servers, so the order may differ from each server. Imply primary/secondary replication to resolve it.

For each chunk, designate one server as "primary". Clients send write requests to the primary, the primary chooses the order for all client writes. Then tells the secondaries.

- Step of write (Figure 2)

  ```
  1.C asks CO about file's chunk @ offset 
  2.CO find the chunkserver holding current lease. If no one has a lease, grants one.
  3.CO tells C the primary, secondaries and verison#++
  4.C caches this data and sends data to all, waits for all replies
  5.C asks P(Primary) to write
  6.P checks that lease hasn't expired and version#
  7.P writes its own chunk file
  8.P tells each secondary to write 
  9.P waits for all secondaries to reply
  10.P tells C "ok" or "error" //error if one secondary didn't reply
  11.C retries from start if error
  ```

If a record append fails at any replica, the client retries the operation. As a result, replicas of the same chunk may contain different data possibly including duplicates. GFS only guarantee that all data is written **at least once** as an atomic unit.

#### Garbage Collection

After a file is deleted, GFS does not immediately reclaim the available physical storage. It does so only lazily during regular garbage collection at both the file and chunk levels.

- *Mechanism*

  When a file is deleted from the application, the coordinator logs thedeletion immediately just like other changes. However, instead of reclaiming resources immediately, the file is just renamed to a hidden name that includes the deletion timestamp. During the coordinator's regular scan, it removes any such hidden files if they have existed for more than three days.

  In a similar regular scan of the chunk namespace, the coordinator identifies orphaned chunks and erases the metadata for those chunks.

- *Discussion*

  We can easily identify all references to chunks: they are in the file-to-chunk mapping maintained exclusively by the coordinator.

#### Stale Replica Detection

Chunk repicas may become stale if a chunkserver fails and misses mutations to the chunk while it is down. For each chunk, the coordinator maintain a *chunk version number* to distinguish between up-to-date and stale replicas.

Whenever the amster grants a new lease on a chunk, it increases the chunk verison number and informs the up-to-dte replicas.

The master removes stale replicas in its regular garbage collection.

### High Availability

- Fast Recovery

- Chunk Replication

- Master Replication

  When a master's machine or disk fails, monitoring infrastructure outside GFS starts a new master process elsewhere with the replicated operation log.

### Data Integrity

Each chunkserver uses checksumming to detect corruption.

A chunk is broken up into 64-KB blocks, each has a corresponding 32 bit checksum. Checksums are kept in memory and stored persistently with logging.

### Recovery

- Coordinator crashed

  - Coordinator writes critical state to its disk.

    If it crashes and reboots with disk intact, re-read state, resume operations.

  - Coordinator sends each state update to a "backup coordinator", which also records it to disk; backup coordinator can take over if main coordinator cannot be restarted.

  Coordinator save to disk to recover from crashed

  - Table mapping file name -> array of chunk handles.
  - Table mapping chunk handle -> current version #

  A rebooted coordinator asks all the chunkservers what they store and wait one lease time before designating any new primaries.