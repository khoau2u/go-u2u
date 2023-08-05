package compactdb

import (
	"bytes"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/unicornultrafoundation/go-hashgraph/u2udb"
	"github.com/unicornultrafoundation/go-hashgraph/u2udb/table"
)

func isEmptyDB(db u2udb.Iteratee) bool {
	it := db.NewIterator(nil, nil)
	defer it.Release()
	return !it.Next()
}

func firstKey(db u2udb.Store) []byte {
	it := db.NewIterator(nil, nil)
	defer it.Release()
	if !it.Next() {
		return nil
	}
	return it.Key()
}

func lastKey(db u2udb.Store) []byte {
	var start []byte
	for {
		for b := 0xff; b >= 0; b-- {
			if !isEmptyDB(table.New(db, append(start, byte(b)))) {
				start = append(start, byte(b))
				break
			}
			if b == 0 {
				return start
			}
		}
	}
}

func addToPrefix(prefix *big.Int, diff *big.Int, size int) []byte {
	endBn := new(big.Int).Set(prefix)
	endBn.Add(endBn, diff)
	if len(endBn.Bytes()) > size {
		// overflow
		return bytes.Repeat([]byte{0xff}, size)
	}
	end := endBn.Bytes()
	res := make([]byte, size-len(end), size)
	return append(res, end...)
}

func Compact(unprefixedDB u2udb.Store, loggingName string) error {
	lastLog := time.Time{}
	compact := func(db u2udb.Store, b int, start, end []byte) error {
		if len(loggingName) != 0 && time.Since(lastLog) > time.Second*16 {
			log.Info("Compacting DB", "name", loggingName, "until", hexutils.BytesToHex(append([]byte{byte(b)}, end...)))
			lastLog = time.Now()
		}
		return db.Compact(start, end)
	}

	for b := 0; b < 256; b++ {
		prefixed := table.New(unprefixedDB, []byte{byte(b)})
		first := firstKey(prefixed)
		if first == nil {
			continue
		}
		last := lastKey(prefixed)
		if last == nil {
			continue
		}
		keySize := len(last)
		if keySize < len(first) {
			keySize = len(first)
		}
		first = common.RightPadBytes(first, keySize-len(first))
		last = common.RightPadBytes(last, keySize-len(last))
		firstBn := new(big.Int).SetBytes(first)
		lastBn := new(big.Int).SetBytes(last)
		diff := new(big.Int).Sub(lastBn, firstBn)
		if diff.Cmp(big.NewInt(10000)) < 0 {
			// short circuit if too few keys
			err := compact(prefixed, b, nil, nil)
			if err != nil {
				return err
			}
			continue
		}
		var prev []byte
		for i := 32; i >= 1; i-- {
			until := addToPrefix(firstBn, new(big.Int).Div(diff, big.NewInt(int64(i))), keySize)
			err := compact(prefixed, b, prev, until)
			if err != nil {
				return err
			}
			prev = common.CopyBytes(until)
		}
	}
	return nil
}