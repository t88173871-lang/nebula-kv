package sstable

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"sort"
)

// Entry 键值对条目
type Entry struct {
	Key     string
	Value   []byte
	Deleted bool
}

// SSTable 排序字符串表
type SSTable struct {
	filePath string
	file     *os.File
	index    *Index
}

// Index 索引块
type Index struct {
	Entries []IndexEntry
}

// IndexEntry 索引条目
type IndexEntry struct {
	Key    string
	Offset int64
	Size   int32
}

// Builder SSTable 构建器
type Builder struct {
	entries []Entry
}

// NewBuilder 创建构建器
func NewBuilder() *Builder {
	return &Builder{
		entries: make([]Entry, 0),
	}
}

// Add 添加键值对
func (b *Builder) Add(key string, value []byte, deleted bool) {
	b.entries = append(b.entries, Entry{
		Key:     key,
		Value:   value,
		Deleted: deleted,
	})
}

// Build 构建 SSTable 文件
func (b *Builder) Build(filePath string) (*SSTable, error) {
	// 按 key 排序
	sort.Slice(b.entries, func(i, j int) bool {
		return b.entries[i].Key < b.entries[j].Key
	})

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 写入数据块
	dataOffset := int64(0)
	indexEntries := make([]IndexEntry, 0, len(b.entries))

	for _, entry := range b.entries {
		// 编码 entry
		data := encodeEntry(&entry)
		n, err := file.Write(data)
		if err != nil {
			return nil, err
		}

		// 记录索引
		indexEntries = append(indexEntries, IndexEntry{
			Key:    entry.Key,
			Offset: dataOffset,
			Size:   int32(n),
		})

		dataOffset += int64(n)
	}

	// 写入索引块
	indexOffset := dataOffset
	indexData := encodeIndex(indexEntries)
	_, err = file.Write(indexData)
	if err != nil {
		return nil, err
	}

	// 写入 Footer（索引块的偏移量）
	footer := make([]byte, 8)
	binary.LittleEndian.PutUint64(footer, uint64(indexOffset))
	_, err = file.Write(footer)
	if err != nil {
		return nil, err
	}

	return &SSTable{
		filePath: filePath,
		index:    &Index{Entries: indexEntries},
	}, nil
}

// Open 打开 SSTable 文件
func Open(filePath string) (*SSTable, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	// 读取 Footer（最后 8 字节）
	footer := make([]byte, 8)
	_, err = file.ReadAt(footer, fileInfo.Size()-8)
	if err != nil {
		file.Close()
		return nil, err
	}

	indexOffset := int64(binary.LittleEndian.Uint64(footer))

	// 读取索引块
	indexSize := fileInfo.Size() - indexOffset - 8
	indexData := make([]byte, indexSize)
	_, err = file.ReadAt(indexData, indexOffset)
	if err != nil {
		file.Close()
		return nil, err
	}

	index := decodeIndex(indexData)

	return &SSTable{
		filePath: filePath,
		file:     file,
		index:    index,
	}, nil
}

// Get 查找键对应的值
func (s *SSTable) Get(key string) ([]byte, bool, error) {
	// 二分查找索引
	idx := sort.Search(len(s.index.Entries), func(i int) bool {
		return s.index.Entries[i].Key >= key
	})

	if idx >= len(s.index.Entries) || s.index.Entries[idx].Key != key {
		return nil, false, nil
	}

	// 读取对应的 data block
	indexEntry := s.index.Entries[idx]
	data := make([]byte, indexEntry.Size)
	_, err := s.file.ReadAt(data, indexEntry.Offset)
	if err != nil {
		return nil, false, err
	}

	// 解码 entry
	entry := decodeEntry(data)
	if entry.Deleted {
		return nil, false, nil
	}

	return entry.Value, true, nil
}

// ForEach 遍历所有键值对
func (s *SSTable) ForEach(fn func(key string, value []byte, deleted bool) bool) error {
	for _, indexEntry := range s.index.Entries {
		data := make([]byte, indexEntry.Size)
		_, err := s.file.ReadAt(data, indexEntry.Offset)
		if err != nil {
			return err
		}

		entry := decodeEntry(data)
		if !fn(entry.Key, entry.Value, entry.Deleted) {
			break
		}
	}
	return nil
}

// Close 关闭 SSTable
func (s *SSTable) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// encodeEntry 编码条目
func encodeEntry(entry *Entry) []byte {
	keyBytes := []byte(entry.Key)
	keyLen := uint32(len(keyBytes))
	valueLen := uint32(len(entry.Value))

	// deleted flag (1 byte) + key_len (4) + key + value_len (4) + value
	totalSize := 1 + 4 + len(keyBytes) + 4 + len(entry.Value)
	buf := make([]byte, totalSize)

	// 写入 deleted flag
	if entry.Deleted {
		buf[0] = 1
	} else {
		buf[0] = 0
	}

	// 写入 key_len 和 key
	binary.LittleEndian.PutUint32(buf[1:5], keyLen)
	copy(buf[5:5+keyLen], keyBytes)

	// 写入 value_len 和 value
	valueOffset := 5 + keyLen
	binary.LittleEndian.PutUint32(buf[valueOffset:valueOffset+4], valueLen)
	copy(buf[valueOffset+4:], entry.Value)

	return buf
}

// decodeEntry 解码条目
func decodeEntry(data []byte) *Entry {
	deleted := data[0] == 1
	keyLen := binary.LittleEndian.Uint32(data[1:5])
	key := string(data[5 : 5+keyLen])

	valueOffset := 5 + keyLen
	valueLen := binary.LittleEndian.Uint32(data[valueOffset : valueOffset+4])
	value := make([]byte, valueLen)
	copy(value, data[valueOffset+4:valueOffset+4+valueLen])

	return &Entry{
		Key:     key,
		Value:   value,
		Deleted: deleted,
	}
}

// encodeIndex 编码索引块
func encodeIndex(entries []IndexEntry) []byte {
	var buf bytes.Buffer

	// 写入条目数量
	count := uint32(len(entries))
	binary.Write(&buf, binary.LittleEndian, count)

	// 写入每个索引条目
	for _, entry := range entries {
		keyBytes := []byte(entry.Key)
		keyLen := uint32(len(keyBytes))

		binary.Write(&buf, binary.LittleEndian, keyLen)
		buf.Write(keyBytes)
		binary.Write(&buf, binary.LittleEndian, uint64(entry.Offset))
		binary.Write(&buf, binary.LittleEndian, entry.Size)
	}

	return buf.Bytes()
}

// decodeIndex 解码索引块
func decodeIndex(data []byte) *Index {
	reader := bytes.NewReader(data)

	var count uint32
	binary.Read(reader, binary.LittleEndian, &count)

	entries := make([]IndexEntry, 0, count)
	for i := uint32(0); i < count; i++ {
		var keyLen uint32
		binary.Read(reader, binary.LittleEndian, &keyLen)

		keyBytes := make([]byte, keyLen)
		io.ReadFull(reader, keyBytes)

		var offset uint64
		var size int32
		binary.Read(reader, binary.LittleEndian, &offset)
		binary.Read(reader, binary.LittleEndian, &size)

		entries = append(entries, IndexEntry{
			Key:    string(keyBytes),
			Offset: int64(offset),
			Size:   size,
		})
	}

	return &Index{Entries: entries}
}