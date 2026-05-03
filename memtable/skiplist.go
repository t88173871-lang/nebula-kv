package memtable

import (
	"math/rand"
	"sync"
)

const (
	maxLevel = 32  // 最大层数
	p        = 0.25 // 每层晋升概率
)

// Node 跳表节点
type Node struct {
	key     string
	value   []byte
	deleted bool       // 标记是否已删除（支持 MVCC）
	version uint64     // 版本号（支持 MVCC）
	next    []*Node    // 多层指针
}

// SkipList 跳表
type SkipList struct {
	head  *Node
	level int      // 当前最大层数
	size  int      // 元素数量
	mu    sync.RWMutex
}

// NewSkipList 创建新的跳表
func NewSkipList() *SkipList {
	return &SkipList{
		head: &Node{
			next: make([]*Node, maxLevel),
		},
		level: 1,
		size:  0,
	}
}

// randomLevel 随机生成层数
func (sl *SkipList) randomLevel() int {
	level := 1
	for rand.Float64() < p && level < maxLevel {
		level++
	}
	return level
}

// Put 插入或更新键值对
func (sl *SkipList) Put(key string, value []byte) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// 记录每层的前驱节点
	update := make([]*Node, maxLevel)
	current := sl.head

	// 从最高层开始查找插入位置
	for i := sl.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		update[i] = current
	}

	// 检查是否已存在
	current = current.next[0]
	if current != nil && current.key == key {
		// 更新现有节点
		current.value = value
		current.deleted = false
		return
	}

	// 生成随机层数
	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level; i < newLevel; i++ {
			update[i] = sl.head
		}
		sl.level = newLevel
	}

	// 创建新节点
	newNode := &Node{
		key:   key,
		value: value,
		next:  make([]*Node, newLevel),
	}

	// 更新指针
	for i := 0; i < newLevel; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}

	sl.size++
}

// Get 获取键对应的值
func (sl *SkipList) Get(key string) ([]byte, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.head

	// 从最高层开始查找
	for i := sl.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
	}

	// 检查是否找到
	current = current.next[0]
	if current != nil && current.key == key && !current.deleted {
		return current.value, true
	}

	return nil, false
}

// Delete 删除键（标记删除）
func (sl *SkipList) Delete(key string) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	update := make([]*Node, maxLevel)
	current := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		update[i] = current
	}

	current = current.next[0]
	if current != nil && current.key == key {
		current.deleted = true // 标记删除，不真正移除（支持 MVCC）
		return true
	}

	return false
}

// Size 返回元素数量
func (sl *SkipList) Size() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.size
}

// ForEach 遍历所有键值对（按顺序）
func (sl *SkipList) ForEach(fn func(key string, value []byte, deleted bool) bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.head.next[0]
	for current != nil {
		if !fn(current.key, current.value, current.deleted) {
			break
		}
		current = current.next[0]
	}
}