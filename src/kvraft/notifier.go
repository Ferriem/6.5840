package kvraft

import (
	"sync"
	"time"
)

const waitTime = 500 * time.Millisecond

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Notifier struct {
	done            sync.Cond
	maxRegisteredId int
}

func (kv *KVServer) makeNotifier(op Op) {
	kv.getNotifier(op, true)
	go func() {
		<-time.After(waitTime)
		kv.mu.Lock()
		defer kv.mu.Unlock()
		kv.notify(op)
	}()
}

func (kv *KVServer) getNotifier(op Op, force bool) *Notifier {
	if notifier, ok := kv.NotifierOfClerk[op.ClerkId]; ok {
		notifier.maxRegisteredId = max(notifier.maxRegisteredId, op.OpId)
		return notifier
	}

	if !force {
		return nil
	}

	notifier := new(Notifier)
	notifier.done = *sync.NewCond(&kv.mu)
	notifier.maxRegisteredId = op.OpId
	kv.NotifierOfClerk[op.ClerkId] = notifier

	return notifier
}

func (kv *KVServer) wait(op Op) {
	for !kv.killed() {
		if notifier := kv.getNotifier(op, false); notifier != nil {
			notifier.done.Wait()
		} else {
			break
		}
	}
}

func (kv *KVServer) notify(op Op) {
	if notifier := kv.getNotifier(op, false); notifier != nil {
		if op.OpId == notifier.maxRegisteredId {
			delete(kv.NotifierOfClerk, op.ClerkId)
			notifier.done.Broadcast()
		}
	}
}
