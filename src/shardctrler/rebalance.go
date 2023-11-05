package shardctrler

import "sort"

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (sc *ShardCtrler) rebalance(Shards [NShards]int, Group map[int][]string, isJoin bool) [NShards]int {
	DPrintf("Start rebalance")

	DPrintf("Group: %v", Group)

	GidToShards := make(map[int][]int)

	for shard, gid := range Shards {
		if _, ok := GidToShards[gid]; !ok {
			GidToShards[gid] = make([]int, 0)
		}
		GidToShards[gid] = append(GidToShards[gid], shard)
	}

	for gid := range Group {
		if _, ok := GidToShards[gid]; !ok {
			GidToShards[gid] = make([]int, 0)
		}
	}

	gidAndShardsArray := make([]GidAndShards, 0)
	for gid, shards := range GidToShards {
		gidAndShardsArray = append(gidAndShardsArray, GidAndShards{gid: gid, shards: shards})
	}

	sort.SliceStable(gidAndShardsArray, func(i, j int) bool { return len(gidAndShardsArray[i].shards) > len(gidAndShardsArray[j].shards) })

	groupsNum := len(Group)
	if groupsNum == 0 {
		i := 0
		for i < NShards {
			Shards[i] = 0
			i++
		}
		return Shards
	}
	DPrintf("groupsNum %v", groupsNum)
	length := NShards / groupsNum
	remain := NShards % groupsNum

	expectShardsNumOfGroup := make(map[int]int)
	for _, gidAndShards := range gidAndShardsArray {

		_, ok := Group[gidAndShards.gid]

		if !isJoin && !ok {
			expectShardsNumOfGroup[gidAndShards.gid] = 0
			continue
		}

		expectShardsNum := length
		if remain > 0 {
			expectShardsNum++
			remain--
		}
		expectShardsNumOfGroup[gidAndShards.gid] = expectShardsNum
	}

	from := make([]GidAndShards, 0)
	to := make([]GidAndShards, 0)

	for i, gidAndShards := range gidAndShardsArray {

		gid := gidAndShards.gid
		curShards := gidAndShards.shards
		expectShardsNum := expectShardsNumOfGroup[gid]
		if len(curShards) > expectShardsNum {
			//give out shards
			// from
			remainShards := curShards[:expectShardsNum]
			gaveOutShards := curShards[expectShardsNum:]
			from = append(from, GidAndShards{gid: gid, shards: gaveOutShards})
			gidAndShardsArray[i].shards = remainShards
		} else if len(curShards) < expectShardsNum {
			//get shards
			// to
			to = append(to, GidAndShards{gid: gid, shards: make([]int, expectShardsNum-len(curShards))})
		}
	}

	// from -> to
	movements := make([]Movement, 0)
	for i, toGidAndShards := range to {
		shardsNum := len(toGidAndShards.shards)
		tmp := 0

		for j, fromGidAndShards := range from {
			neededShardsNum := shardsNum - tmp
			gaveOutShardsNum := min(len(fromGidAndShards.shards), neededShardsNum)
			if gaveOutShardsNum <= 0 {
				//all shards are gave out
				continue
			}
			gaveOutShards := fromGidAndShards.shards[:gaveOutShardsNum]
			if len(fromGidAndShards.shards) <= gaveOutShardsNum {
				//all shards will be gave out, clear from
				from[j].shards = make([]int, 0)
			} else {
				//some shards will be gave out, modify from
				from[j].shards = fromGidAndShards.shards[gaveOutShardsNum:]
			}

			for k := 0; k < gaveOutShardsNum; k++ {
				toGidAndShards.shards[tmp+k] = gaveOutShards[k]
			}
			tmp += gaveOutShardsNum

			movements = append(movements, Movement{from: fromGidAndShards.gid, to: toGidAndShards.gid, shards: gaveOutShards})
		}
		to[i] = toGidAndShards
	}

	DPrintf("Movements: %v", movements)

	for _, movement := range movements {
		for _, shard := range movement.shards {
			Shards[shard] = movement.to
		}
		DPrintf("G%v -> G%v: %v", movement.from, movement.to, movement.shards)
	}
	return Shards
}
