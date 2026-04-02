package unit

import (
	"sync"
)

// LinkNode 链表节点
type LinkNode struct {
	Value interface{}
	Next  *LinkNode
}

// LinkedQueue 并发安全的链表队列
type LinkedQueue struct {
	head *LinkNode // 队首
	tail *LinkNode // 队尾
	size int
	lock sync.Mutex
}

// NewLinkedQueue 创建新队列
func NewLinkedQueue() *LinkedQueue {
	return &LinkedQueue{}
}

// Enqueue 入队（追加到尾部）
func (q *LinkedQueue) Enqueue(v interface{}) {
	q.lock.Lock()
	defer q.lock.Unlock()

	newNode := &LinkNode{Value: v}

	if q.tail == nil { // 空队列
		q.head = newNode
		q.tail = newNode
	} else {
		q.tail.Next = newNode
		q.tail = newNode
	}
	q.size++
}

// Dequeue 出队（从头部移除）
func (q *LinkedQueue) Dequeue() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == nil {
		return nil
	}

	v := q.head.Value
	q.head = q.head.Next

	if q.head == nil { // 队列已空
		q.tail = nil
	}
	q.size--

	return v
}

// Size 获取队列大小
func (q *LinkedQueue) Size() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.size
}
