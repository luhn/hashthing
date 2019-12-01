/*
 * It's unnecessary to implement this myself, it's been done many times before,
 * but I only need a simple interface and it's a good practice.
 */
package main

type FileQueue struct {
	items []File
	position int
	length int
	capacity int
}

func NewQueue(items []File) FileQueue {
	return FileQueue{items, 0, len(items), len(items)}
}

func (q FileQueue) Push(item File) {
	if q.length == q.capacity {
		panic("Queue cannot fit more items.")
	}
	index := (q.position + q.length) % q.capacity
	q.items[index] = item
	q.length += 1
}

func (q FileQueue) Empty() bool {
	return q.length == 0
}

func (q *FileQueue) Pop() File {
	if q.length == 0 {
		panic("Queue is empty!")
	}
	item := q.items[q.position]
	q.position = (q.position + 1) % q.capacity
	q.length -= 1
	return item
}
