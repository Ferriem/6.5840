# 6.5840

I create this repository after finishing log-replication. 

The git branch information was provided below.

```mermaid
gitGraph
	commit id: "update"
  branch leader-election
  commit id: "leader-election"
  branch log-replication
  commit id: "log-replication"
  branch persist
  commit id: "persist"
  branch log-compaction
  commit id: "log-compaction"
  branch kv-without-snapshot
  commit id: "kv-without-snapshot"
  branch kv-with-snapshot
  commit id: "kv-with-snapshot"
  branch ShardCtrler
  commit id: "ShardCtrler"
```



### Lab 1: MapReduce

Introduction: [Lecture](md/Lecture-1.md)

RPC and Threads: [Lecture](md/Lecture-2.md)

[Lab](md/Lab-1.md)

### Lab 2: Raft

![raft-diagram](./md/image/raft-diagram.png)

GFS: [Lecture](md/Lecture-3.md)

Primary-Backup Replication: [Lecture](md/Lecture-4.md)

Raft: [Lecture](md/Lecture-5.md) 

[Lab](md/Lab-2.md) 

### Lab 3:KV Service

![kvserver](./md/image/kvserver.png)

[Linearizability](md/Lecture-9.md)

[ZooKeeper](md/Lecture-10.md)

[Chain Replication](md/Lecture-11.md)

[Disrtibuted Transactions](md/Lecture-12.md)

[Frangipani](md/Lecture-13.md)

[Lab](md/Lab-3.md)

### Lab 4:Sharded Key/Value Service

[Spanner](md/Lecture-14.md)

[FaRM](md/Lecture-15.md)

[Lab](md/Lab-4.md)
