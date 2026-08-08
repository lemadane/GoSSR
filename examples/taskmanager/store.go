package main

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	ID        string
	Title     string
	Priority  string // "HIGH", "MEDIUM", "LOW"
	Completed bool
	CreatedAt time.Time
}

type TaskStats struct {
	TotalCount     int
	CompletedCount int
	PendingCount   int
	HighPriority   int
}

type TaskStore struct {
	mutex     sync.RWMutex
	tasks     []Task
	idCounter uint64
}

func NewTaskStore() *TaskStore {
	store := &TaskStore{
		tasks: []Task{},
	}
	// Initialize with sample realistic data
	store.CreateTask("Initialize GoSSR template engine core", "HIGH", true)
	store.CreateTask("Integrate HTMX out-of-band updates & live search", "HIGH", false)
	store.CreateTask("Attach Alpine.js client dropdown & modal state", "MEDIUM", false)
	store.CreateTask("Verify XSS quote escaping & URL protocol sanitization", "LOW", true)
	return store
}

func (store *TaskStore) CreateTask(title string, priority string, completed bool) Task {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	idNumber := atomic.AddUint64(&store.idCounter, 1)
	if priority == "" {
		priority = "MEDIUM"
	}

	task := Task{
		ID:        fmt.Sprintf("%d", idNumber+100),
		Title:     strings.TrimSpace(title),
		Priority:  strings.ToUpper(priority),
		Completed: completed,
		CreatedAt: time.Now(),
	}

	store.tasks = append([]Task{task}, store.tasks...) // Insert at top
	return task
}

func (store *TaskStore) GetAllTasks() []Task {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	copied := make([]Task, len(store.tasks))
	copy(copied, store.tasks)
	return copied
}

func (store *TaskStore) SearchTasks(query string, filter string) []Task {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	filter = strings.ToLower(strings.TrimSpace(filter))

	var filtered []Task
	for _, task := range store.tasks {
		if query != "" && !strings.Contains(strings.ToLower(task.Title), query) {
			continue
		}
		if filter == "completed" && !task.Completed {
			continue
		}
		if filter == "pending" && task.Completed {
			continue
		}
		filtered = append(filtered, task)
	}
	return filtered
}

func (store *TaskStore) ToggleTask(id string) (Task, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for index := range store.tasks {
		if store.tasks[index].ID == id {
			store.tasks[index].Completed = !store.tasks[index].Completed
			return store.tasks[index], true
		}
	}
	return Task{}, false
}

func (store *TaskStore) DeleteTask(id string) bool {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for index, task := range store.tasks {
		if task.ID == id {
			store.tasks = append(store.tasks[:index], store.tasks[index+1:]...)
			return true
		}
	}
	return false
}

func (store *TaskStore) GetStats() TaskStats {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	stats := TaskStats{}
	stats.TotalCount = len(store.tasks)

	for _, task := range store.tasks {
		if task.Completed {
			stats.CompletedCount++
		} else {
			stats.PendingCount++
		}
		if task.Priority == "HIGH" {
			stats.HighPriority++
		}
	}
	return stats
}
