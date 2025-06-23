package cron

func (c *cron) List() []task {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	tasks := make([]task, 0, len(c.heap))
	for _, t := range c.heap {
		if t.Enable {
			tasks = append(tasks, *t)
		}
	}
	return tasks
}

func (h taskHeap) Len() int {
	return len(h)
}

func (h taskHeap) Less(i, j int) bool {
	return h[i].Next.Before(h[j].Next)
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *taskHeap) Push(x interface{}) {
	*h = append(*h, x.(*task))
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
