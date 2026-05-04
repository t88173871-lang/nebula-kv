package wal

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

const (
	// Entry 格式: checksum(4) + keyLen(4) + valueLen(4) + key + value
	headerSize = 12
)

// Entry 日志条目
type Entry struct {
	Key     string
	Value   []byte
	Deleted bool
}

// WAL 预写日志
type WAL struct {
	file    *os.File
	mu      sync.Mutex
	dirPath string
}

// Open 打开或创建 WAL
func Open(dirPath string) (*WAL, error) {
	// 确保目录存在
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}

	// 打开日志文件（追加模式）
	filePath := dirPath + "/wal.log"
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &WAL{
		file:    file,
		dirPath: dirPath,
	}, nil
}

// Append 追加一条日志
func (w *WAL) Append(entry *Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 编码 entry
	data := encodeEntry(entry)

	// 写入文件
	_, err := w.file.Write(data)
	if err != nil {
		return err
	}

	// 强制刷盘，确保数据持久化
	return w.file.Sync()
}

// ReadAll 读取所有日志条目（用于恢复）
func (w *WAL) ReadAll() ([]*Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 重新打开文件用于读取
	file, err := os.Open(w.dirPath + "/wal.log")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在，返回空
		}
		return nil, err
	}
	defer file.Close()

	var entries []*Entry
	buf := make([]byte, 4096)

	for {
		// 读取 header
		header := make([]byte, headerSize)
		_, err := io.ReadFull(file, header)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// 解析 header
		checksum := binary.LittleEndian.Uint32(header[0:4])
		keyLen := binary.LittleEndian.Uint32(header[4:8])
		valueLen := binary.LittleEndian.Uint32(header[8:12])

		// 读取 key + value
		totalLen := keyLen + valueLen
		if uint32(len(buf)) < totalLen {
			buf = make([]byte, totalLen)
		}
		_, err = io.ReadFull(file, buf[:totalLen])
		if err != nil {
			return nil, err
		}

		// 校验 checksum
		data := append(header, buf[:totalLen]...)
		if crc32.ChecksumIEEE(data[4:]) != checksum {
			break // 校验失败，停止读取
		}

		// 解析 key 和 value
		key := string(buf[:keyLen])
		value := make([]byte, valueLen)
		copy(value, buf[keyLen:totalLen])

		entries = append(entries, &Entry{
			Key:   key,
			Value: value,
		})
	}

	return entries, nil
}

// Close 关闭 WAL
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Truncate 清空日志（在 MemTable 刷盘后调用）
func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 关闭当前文件
	if w.file != nil {
		w.file.Close()
	}

	// 重新创建空文件
	filePath := w.dirPath + "/wal.log"
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	w.file = file
	return nil
}

// encodeEntry 编码日志条目
func encodeEntry(entry *Entry) []byte {
	keyBytes := []byte(entry.Key)
	keyLen := uint32(len(keyBytes))
	valueLen := uint32(len(entry.Value))

	// 计算总长度
	totalSize := headerSize + keyLen + valueLen
	buf := make([]byte, totalSize)

	// 写入 keyLen 和 valueLen
	binary.LittleEndian.PutUint32(buf[4:8], keyLen)
	binary.LittleEndian.PutUint32(buf[8:12], valueLen)

	// 写入 key 和 value
	copy(buf[headerSize:headerSize+keyLen], keyBytes)
	copy(buf[headerSize+keyLen:], entry.Value)

	// 计算并写入 checksum
	checksum := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:4], checksum)

	return buf
}