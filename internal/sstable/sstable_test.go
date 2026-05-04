package sstable

import (
	"os"
	"testing"
)

func TestSSTableBasic(t *testing.T) {
	filePath := "./test_sstable.sst"
	defer os.Remove(filePath)

	// 构建 SSTable
	builder := NewBuilder()
	builder.Add("apple", []byte("red"), false)
	builder.Add("banana", []byte("yellow"), false)
	builder.Add("cherry", []byte("dark red"), false)
	builder.Add("date", []byte("brown"), false)

	sst, err := builder.Build(filePath)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	sst.Close()

	// 重新打开并读取
	sst, err = Open(filePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer sst.Close()

	// 测试 Get
	tests := []struct {
		key      string
		wantVal  string
		wantFind bool
	}{
		{"apple", "red", true},
		{"banana", "yellow", true},
		{"cherry", "dark red", true},
		{"date", "brown", true},
		{"elderberry", "", false},
	}

	for _, tt := range tests {
		val, found, err := sst.Get(tt.key)
		if err != nil {
			t.Errorf("Get(%s) error: %v", tt.key, err)
			continue
		}
		if found != tt.wantFind {
			t.Errorf("Get(%s) found = %v, want %v", tt.key, found, tt.wantFind)
		}
		if found && string(val) != tt.wantVal {
			t.Errorf("Get(%s) = %s, want %s", tt.key, val, tt.wantVal)
		}
	}
}

func TestSSTableForEach(t *testing.T) {
	filePath := "./test_sstable_foreach.sst"
	defer os.Remove(filePath)

	// 构建 SSTable（乱序添加）
	builder := NewBuilder()
	builder.Add("cherry", []byte("dark red"), false)
	builder.Add("apple", []byte("red"), false)
	builder.Add("date", []byte("brown"), false)
	builder.Add("banana", []byte("yellow"), false)

	sst, err := builder.Build(filePath)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	sst.Close()

	// 重新打开
	sst, err = Open(filePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer sst.Close()

	// 遍历应该按顺序输出
	expected := []string{"apple", "banana", "cherry", "date"}
	var actual []string

	sst.ForEach(func(key string, value []byte, deleted bool) bool {
		actual = append(actual, key)
		return true
	})

	if len(actual) != len(expected) {
		t.Errorf("ForEach count mismatch, got %d, want %d", len(actual), len(expected))
	}

	for i, v := range actual {
		if v != expected[i] {
			t.Errorf("ForEach order mismatch at %d, got %s, want %s", i, v, expected[i])
		}
	}
}

func TestSSTableDeleted(t *testing.T) {
	filePath := "./test_sstable_deleted.sst"
	defer os.Remove(filePath)

	// 构建 SSTable，包含删除标记
	builder := NewBuilder()
	builder.Add("key1", []byte("value1"), false)
	builder.Add("key2", []byte("value2"), true) // 已删除
	builder.Add("key3", []byte("value3"), false)

	sst, err := builder.Build(filePath)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	sst.Close()

	// 重新打开
	sst, err = Open(filePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer sst.Close()

	// key2 应该找不到（已删除）
	_, found, err := sst.Get("key2")
	if err != nil {
		t.Errorf("Get(key2) error: %v", err)
	}
	if found {
		t.Error("Get(key2) should not find deleted key")
	}

	// key1 和 key3 应该能找到
	val1, found1, _ := sst.Get("key1")
	val3, found3, _ := sst.Get("key3")

	if !found1 || string(val1) != "value1" {
		t.Error("Get(key1) failed")
	}
	if !found3 || string(val3) != "value3" {
		t.Error("Get(key3) failed")
	}
}