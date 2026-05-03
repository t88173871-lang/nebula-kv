package memtable

import (
	"fmt"
	"testing"
)

func TestSkipListBasic(t *testing.T) {
	sl := NewSkipList()

	// 测试插入
	sl.Put("name", []byte("张三"))
	sl.Put("age", []byte("25"))
	sl.Put("city", []byte("北京"))

	// 测试获取
	if val, ok := sl.Get("name"); !ok || string(val) != "张三" {
		t.Errorf("Get(name) failed, got %s", val)
	}

	if val, ok := sl.Get("age"); !ok || string(val) != "25" {
		t.Errorf("Get(age) failed, got %s", val)
	}

	// 测试更新
	sl.Put("name", []byte("李四"))
	if val, ok := sl.Get("name"); !ok || string(val) != "李四" {
		t.Errorf("Update failed, got %s", val)
	}

	// 测试删除
	sl.Delete("age")
	if _, ok := sl.Get("age"); ok {
		t.Error("Delete failed, key still exists")
	}

	// 测试大小
	if sl.Size() != 3 {
		t.Errorf("Size() failed, got %d, want 3", sl.Size())
	}
}

func TestSkipListForEach(t *testing.T) {
	sl := NewSkipList()

	// 插入乱序数据
	keys := []string{"banana", "apple", "cherry", "date"}
	for _, k := range keys {
		sl.Put(k, []byte(k))
	}

	// 遍历应该按顺序输出
	var result []string
	sl.ForEach(func(key string, value []byte, deleted bool) bool {
		if !deleted {
			result = append(result, key)
		}
		return true
	})

	expected := []string{"apple", "banana", "cherry", "date"}
	if len(result) != len(expected) {
		t.Errorf("ForEach length mismatch, got %v, want %v", result, expected)
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("ForEach order mismatch at %d, got %s, want %s", i, v, expected[i])
		}
	}
}

func TestSkipListConcurrency(t *testing.T) {
	sl := NewSkipList()
	done := make(chan bool)

	// 并发写入
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				sl.Put(key, []byte(key))
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证大小
	if sl.Size() != 1000 {
		t.Errorf("Concurrency test failed, size = %d, want 1000", sl.Size())
	}
}