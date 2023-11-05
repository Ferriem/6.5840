package shardctrler

import (
	"log"
	"sort"
)

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func (sc *ShardCtrler) PrintGidToShards(config *Config) {
	if !Debug {
		return
	}
	gidToShards := make(map[int][]int)
	for shard, gid := range config.Shards {
		if _, ok := gidToShards[gid]; !ok {
			gidToShards[gid] = make([]int, 0)
		}
		gidToShards[gid] = append(gidToShards[gid], shard)
	}

	for gid := range config.Groups {
		if _, ok := gidToShards[gid]; !ok {
			gidToShards[gid] = make([]int, 0)
		}
	}

	gidAndShardArray := make([]GidAndShards, 0)

	for gid, shards := range gidToShards {
		gidAndShardArray = append(gidAndShardArray, GidAndShards{gid: gid, shards: shards})
	}

	sort.Slice(gidAndShardArray, func(i, j int) bool { return gidAndShardArray[i].gid < gidAndShardArray[j].gid })

	for _, gidAndShards := range gidAndShardArray {
		DPrintf("G%v: %v\n", gidAndShards.gid, gidAndShards.shards)
	}
}
