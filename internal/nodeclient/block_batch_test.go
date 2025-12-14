package nodeclient

import "testing"

func TestBlockBatchSize_DefaultAdaptive(t *testing.T) {
	t.Setenv("SYNC_BLOCK_BATCH_SIZE", "")
	t.Setenv("SYNC_BLOCK_BATCH_TARGET_BYTES", "")

	if got := blockBatchSize(128 << 10); got != 256 { // 32MiB / 128KiB = 256
		t.Fatalf("expected 256, got %d", got)
	}
	if got := blockBatchSize(4 << 20); got != 8 { // 32MiB / 4MiB = 8
		t.Fatalf("expected 8, got %d", got)
	}
}

func TestBlockBatchSize_Override(t *testing.T) {
	t.Setenv("SYNC_BLOCK_BATCH_SIZE", "123")
	if got := blockBatchSize(128 << 10); got != 123 {
		t.Fatalf("expected 123, got %d", got)
	}
}
