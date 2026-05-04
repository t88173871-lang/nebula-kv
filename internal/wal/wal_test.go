package wal

import (
	"os"
	"testing"
)

func TestWALBasic(t *testing.T) {
	dir := "./test_wal_data"
	defer os.RemoveAll(dir)

	// 打开 WAL
	wal, err := Open(dir)
	if err != nil {
		t.Fatalf("Open WAL failed: %v", err)
	}
	defer wal.Close()

	// 写入日志
	entries := []*Entry{
		{Key: "name", Value: []byte("张三")},
		{Key: "age", Value: []byte("25")},
		{Key: "city", Value: []byte("北京")},
	}

	for _, entry := range entries {
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// 读取日志
	recovered, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// 验证
	if len(recovered) != len(entries) {
		t.Errorf("Entry count mismatch, got %d, want %d", len(recovered), len(entries))
	}

	for i, entry := range recovered {
		if entry.Key != entries[i].Key {
			t.Errorf("Key mismatch at %d, got %s, want %s", i, entry.Key, entries[i].Key)
		}
		if string(entry.Value) != string(entries[i].Value) {
			t.Errorf("Value mismatch at %d, got %s, want %s", i, entry.Value, entries[i].Value)
		}
	}
}

func TestWALRecovery(t *testing.T) {
	dir := "./test_wal_recovery"
	defer os.RemoveAll(dir)

	// 第一次：写入数据
	wal1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open WAL failed: %v", err)
	}

	wal1.Append(&Entry{Key: "key1", Value: []byte("value1")})
	wal1.Append(&Entry{Key: "key2", Value: []byte("value2")})
	wal1.Close() // 模拟进程关闭

	// 第二次：重新打开，验证恢复
	wal2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open WAL failed: %v", err)
	}
	defer wal2.Close()

	recovered, err := wal2.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(recovered) != 2 {
		t.Errorf("Recovery failed, got %d entries, want 2", len(recovered))
	}
}

func TestWALTruncate(t *testing.T) {
	dir := "./test_wal_truncate"
	defer os.RemoveAll(dir)

	wal, err := Open(dir)
	if err != nil {
		t.Fatalf("Open WAL failed: %v", err)
	}
	defer wal.Close()

	// 写入数据
	wal.Append(&Entry{Key: "key1", Value: []byte("value1")})

	// 清空日志（模拟 MemTable 刷盘后）
	if err := wal.Truncate(); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// 验证清空
	recovered, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(recovered) != 0 {
		t.Errorf("Truncate failed, got %d entries, want 0", len(recovered))
	}
}