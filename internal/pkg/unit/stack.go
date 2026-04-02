package unit

import (
	"fmt"
	"sync"
)

// SafeStack 是一个线程安全的后进先出（LIFO）栈数据结构
// 使用互斥锁保证并发安全，适用于多 goroutine 环境
// 元素类型限定为 string 类型
type SafeStack struct {
	// items 存储栈内元素的切片，0 索引为栈底，最后一个元素为栈顶
	items []string
	// mu 是读写互斥锁，用于保护 items 切片的并发访问
	// 使用 RWMutex 在读多写少场景下提升性能
	mu sync.RWMutex
}

// NewSafeStack 创建并初始化一个新的 SafeStack 实例
// 返回一个空栈，内部 items 切片初始容量为 0
// 使用示例:
//
//	stack := unit.NewSafeStack()
func NewSafeStack() *SafeStack {
	return &SafeStack{items: make([]string, 0)}
}

// Push 将元素压入栈顶
// 参数:
//
//	item: 要入栈的字符串元素
//
// 线程安全: 通过写锁保证并发安全
func (s *SafeStack) Push(item string) {
	s.mu.Lock()         // 加写锁，阻止其他读写操作
	defer s.mu.Unlock() // 函数退出时解锁

	// 使用 append 将新元素添加到切片末尾（栈顶）
	// 如果底层数组容量不足，Go 会自动进行扩容
	s.items = append(s.items, item)
}

// Pop 从栈顶弹出并返回元素（后进先出）
// 返回值:
//
//	string: 栈顶元素
//	error:  如果栈为空返回错误，否则为 nil
//
// 线程安全: 通过写锁保证并发安全
// 注意事项: 调用前建议先用 IsEmpty() 判断，或检查错误返回值
func (s *SafeStack) Pop() (string, error) {
	s.mu.Lock()         // 加写锁，阻止其他读写操作
	defer s.mu.Unlock() // 函数退出时解锁

	// 检查栈是否为空，避免索引越界
	if len(s.items) == 0 {
		return "", fmt.Errorf("stack is empty")
	}

	// 获取最后一个元素的索引（栈顶）
	lastIndex := len(s.items) - 1

	// 取出栈顶元素
	item := s.items[lastIndex]

	// 通过切片操作移除栈顶元素，释放内存
	// s.items[:lastIndex] 创建新切片，不包含最后一个元素
	s.items = s.items[:lastIndex]

	return item, nil
}

// Peek 查看栈顶元素但不弹出
// 返回值:
//
//	string: 栈顶元素，如果栈为空返回空字符串 ""
//
// 线程安全: 通过读锁保证并发安全，允许多个 goroutine 同时查看
// 使用场景: 当你只想查看栈顶而不想修改栈时使用
func (s *SafeStack) Peek() string {
	s.mu.RLock()         // 加读锁，阻止写操作但允许其他读操作
	defer s.mu.RUnlock() // 函数退出时解锁

	// 检查栈是否为空
	if len(s.items) == 0 {
		return ""
	}

	// 返回最后一个元素（栈顶），不修改原切片
	return s.items[len(s.items)-1]
}

// IsEmpty 判断栈是否为空
// 返回值:
//
//	bool: 栈为空返回 true，否则返回 false
//
// 线程安全: 通过读锁保证并发安全
func (s *SafeStack) IsEmpty() bool {
	s.mu.RLock()         // 加读锁
	defer s.mu.RUnlock() // 函数退出时解锁

	// 通过切片长度判断栈是否为空
	return len(s.items) == 0
}

// Size 返回栈中元素的数量
// 返回值:
//
//	int: 当前栈中元素的个数
//
// 线程安全: 通过读锁保证并发安全
func (s *SafeStack) Size() int {
	s.mu.RLock()         // 加读锁
	defer s.mu.RUnlock() // 函数退出时解锁

	// 返回切片长度，即栈中元素数量
	return len(s.items)
}
