package types

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/assert"
)

func TestSnapshotFile_Compare(t *testing.T) {
	// Dead left is worse/better than right
	const worsee = -1
	const sameee = 0
	const better = +1
	t.Run("DifferentSlot", func(t *testing.T) {
		assert.Equal(t, worsee, (&SnapshotFile{Slot: 10}).Compare(&SnapshotFile{Slot: 12}))
		assert.Equal(t, better, (&SnapshotFile{Slot: 10}).Compare(&SnapshotFile{Slot: 8}))
	})
	t.Run("DifferentBaseSlot", func(t *testing.T) {
		assert.Equal(t, worsee, (&SnapshotFile{Slot: 10, BaseSlot: 10}).Compare(&SnapshotFile{Slot: 10, BaseSlot: 12}))
		assert.Equal(t, better, (&SnapshotFile{Slot: 10, BaseSlot: 10}).Compare(&SnapshotFile{Slot: 10, BaseSlot: 8}))
	})
	t.Run("FullVsIncrementalSnap", func(t *testing.T) {
		assert.Equal(t, better, (&SnapshotFile{Slot: 10}).Compare(&SnapshotFile{Slot: 10, BaseSlot: 12}))
		assert.Equal(t, worsee, (&SnapshotFile{Slot: 10, BaseSlot: 12}).Compare(&SnapshotFile{Slot: 10}))
	})
	t.Run("HashMismatch", func(t *testing.T) {
		assert.Equal(t, better, (&SnapshotFile{Slot: 10, Hash: solana.Hash{0x69}}).Compare(&SnapshotFile{Slot: 10, Hash: solana.Hash{0x68}}))
		assert.Equal(t, better, (&SnapshotFile{Slot: 10, BaseSlot: 12, Hash: solana.Hash{0x69}}).Compare(&SnapshotFile{Slot: 10, BaseSlot: 12, Hash: solana.Hash{0x68}}))
		assert.Equal(t, worsee, (&SnapshotFile{Slot: 10, Hash: solana.Hash{0x69}}).Compare(&SnapshotFile{Slot: 10, Hash: solana.Hash{0x70}}))
		assert.Equal(t, worsee, (&SnapshotFile{Slot: 10, BaseSlot: 12, Hash: solana.Hash{0x69}}).Compare(&SnapshotFile{Slot: 10, BaseSlot: 12, Hash: solana.Hash{0x70}}))
	})
	t.Run("Same", func(t *testing.T) {
		assert.Equal(t, sameee, (&SnapshotFile{Slot: 10}).Compare(&SnapshotFile{Slot: 10}))
	})
}
