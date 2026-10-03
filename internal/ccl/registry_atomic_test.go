package ccl

import (
	"reflect"
	"sync"
	"testing"
)

// #259: a caller's registration of an aggregate or sequence function writes the
// function and the mark that says its arithmetic is the caller's. Done under two
// locks, a reader in between saw the caller's function with no mark, so a
// streaming form would compute the built-in arithmetic under the caller's name.
// Both have to change together.
func TestUserRegistrationIsOneStep(t *testing.T) {
	builtinAgg := AggFunc(func(args ...[]any) (any, error) { return 0, nil })
	userAgg := AggFunc(func(args ...[]any) (any, error) { return 1, nil })
	builtinSeq := SeqFunc(func(args ...[]any) ([]any, error) { return nil, nil })
	userSeq := SeqFunc(func(args ...[]any) ([]any, error) { return []any{}, nil })
	userAggPtr := reflect.ValueOf(userAgg).Pointer()
	userSeqPtr := reflect.ValueOf(userSeq).Pointer()

	const name = "ZZATOMIC_EN1"
	defer func() {
		registryMu.Lock()
		delete(aggregateFunctions, name)
		delete(userAggregates, name)
		delete(sequenceFunctions, name)
		delete(userSequences, name)
		registryMu.Unlock()
	}()

	const rounds = 20000
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			registerAggregateFunction(name, builtinAgg)
			RegisterAggregateFunction(name, userAgg)
			registerSequenceFunction(name, builtinSeq)
			RegisterSequenceFunction(name, userSeq)
		}
		close(stop)
	}()

	torn := 0
	for done := false; !done; {
		select {
		case <-stop:
			done = true
		default:
		}
		registryMu.RLock()
		agg, aggOK := aggregateFunctions[name]
		aggMarked := userAggregates[name]
		seq, seqOK := sequenceFunctions[name]
		seqMarked := userSequences[name]
		registryMu.RUnlock()
		if aggOK && reflect.ValueOf(agg).Pointer() == userAggPtr && !aggMarked {
			torn++
		}
		if seqOK && reflect.ValueOf(seq).Pointer() == userSeqPtr && !seqMarked {
			torn++
		}
	}
	wg.Wait()
	if torn > 0 {
		t.Fatalf("saw a caller's function without its mark %d times", torn)
	}
}
