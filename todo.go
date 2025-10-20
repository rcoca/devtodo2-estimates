/*
  Copyright 2011 Alec Thomas

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ParseEstimate parses a string like "2h", "1d", "2w", "1m", "1y" into a time.Duration
func ParseEstimate(estimate string) (time.Duration, error) {
	if estimate == "" {
		return 0, nil
	}

	// Remove whitespace
	estimate = strings.TrimSpace(estimate)

	// Check if the string is valid
	if len(estimate) < 2 {
		return 0, fmt.Errorf("invalid estimate format: %s", estimate)
	}

	// Extract the numeric part and unit part
	numStr := ""
	unitStr := ""

	// Find where the numeric part ends
	for i, char := range estimate {
		if !unicode.IsDigit(char) && char != '.' && char != '-' && char != '+' {
			numStr = estimate[:i]
			unitStr = estimate[i:]
			break
		}
	}

	// If no unit found
	if unitStr == "" {
		return 0, fmt.Errorf("invalid estimate format: %s", estimate)
	}

	// Parse the numeric part
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in estimate: %s", estimate)
	}

	// Convert to duration based on unit
	var duration time.Duration
	switch strings.ToLower(unitStr) {
	case "h", "hr", "hrs", "hour", "hours":
		duration = time.Duration(num * float64(time.Hour))
	case "m", "min", "mins", "minutes":
		duration = time.Duration(num * float64(time.Hour/60))
	case "d", "day", "days":
		duration = time.Duration(num * float64(24*time.Hour))
	case "w", "wk", "wks", "week", "weeks":
		duration = time.Duration(num * float64(7*24*time.Hour))
	case "mon", "mons", "month", "months":
		duration = time.Duration(num * float64(30*24*time.Hour))
	case "y", "yr", "yrs", "year", "years":
		duration = time.Duration(num * float64(365*24*time.Hour))
	default:
		return 0, fmt.Errorf("unknown time unit in estimate: %s", unitStr)
	}

	return duration, nil
}

// FormatEstimate formats a duration into a human-readable string
func FormatEstimate(duration time.Duration) string {
	if duration == 0 {
		return ""
	}

	// Convert to appropriate unit
	switch {
	case duration >= 365*24*time.Hour:
		return fmt.Sprintf("%.2fy", float64(duration.Hours()/(365.0*24)))
	case duration >= 30*24*time.Hour:
		return fmt.Sprintf("%.2fM", float64(duration.Hours()/(30.0*24)))
	case duration >= 7*24*time.Hour:
		return fmt.Sprintf("%.2fw", float64(duration.Hours()/(7.0*24)))
	case duration >= 24*time.Hour:
		return fmt.Sprintf("%.2fd", float64(duration.Hours()/(24.0)))
	case duration >= time.Hour:
		return fmt.Sprintf("%.2fh", duration.Hours())
	default:
		return fmt.Sprintf("%.2fm", float64(duration/time.Minute))
	}
}

type Priority int

// Priority constants.
const (
	VERYHIGH = Priority(iota)
	HIGH
	MEDIUM
	LOW
	VERYLOW
)

type Order int

// Order constants.
const (
	CREATED = Order(iota)
	COMPLETED
	TEXT
	PRIORITY
	DURATION
	DONE
	INDEX
	ESTIMATE
)

type TaskListIO interface {
	Deserialize(reader io.Reader) (TaskList, error)
	Serialize(writer io.Writer, tasks TaskList) error
}

type TaskNode interface {
	ID() int
	At(index int) Task
	Len() int
	Equal(other TaskNode) bool

	Parent() TaskNode
	SetParent(parent TaskNode)

	Append(child TaskNode)
	Create(title string, priority Priority) Task
	Delete()

	// Estimate methods for TaskNode
	SetEstimate(est time.Duration)
	Estimate() time.Duration
	SumDescendants() time.Duration
}

// Add Estimate method to taskNodeImpl
func (t *taskNodeImpl) Estimate() time.Duration {
	return t.estimate
}
func (t *taskNodeImpl) SetEstimate(est time.Duration) {
	t.estimate = est
}
func (t *taskNodeImpl) SumDescendants() time.Duration {
	var sum = time.Duration(0.0)
	for i := 0; i < t.Len(); i++ {
		sum += t.At(i).Estimate()
	}
	return sum
}

// Task interface additions
type Task interface {
	TaskNode

	Text() string
	SetText(text string)

	Priority() Priority
	SetPriority(priority Priority)

	SetCreationTime(time time.Time)
	CreationTime() time.Time

	SetCompleted()
	SetCompletionTime(time time.Time)
	CompletionTime() time.Time

	// Estimate methods
	Estimate() time.Duration
	SetEstimate(estimate time.Duration)

	// Extra attributes usable by extensions
	Attributes() map[string]string
}

type TaskList interface {
	TaskNode

	Title() string
	SetTitle(title string)

	Find(index string) Task
	FindAll(predicate func(node Task) bool) []Task
}

// Index referencing a task
type Index []int

// Implementation

var priorityMapFromString = map[string]Priority{
	"veryhigh": VERYHIGH,
	"high":     HIGH,
	"medium":   MEDIUM,
	"low":      LOW,
	"verylow":  VERYLOW,
}

var priorityToString = map[Priority]string{
	VERYHIGH: "veryhigh",
	HIGH:     "high",
	MEDIUM:   "medium",
	VERYLOW:  "verylow",
	LOW:      "low",
}

func (p Priority) String() string {
	return priorityToString[p]
}

func PriorityFromString(priority string) Priority {
	if p, ok := priorityMapFromString[priority]; ok {
		return p
	}
	return MEDIUM
}

var orderFromString = map[string]Order{
	"index":      INDEX,
	"started":    CREATED,
	"start":      CREATED,
	"creation":   CREATED,
	"created":    CREATED,
	"finish":     COMPLETED,
	"finished":   COMPLETED,
	"completion": COMPLETED,
	"completed":  COMPLETED,
	"text":       TEXT,
	"priority":   PRIORITY,
	"length":     DURATION,
	"lifetime":   DURATION,
	"duration":   DURATION,
	"done":       DONE,
}

var orderToString = map[Order]string{
	INDEX:     "index",
	CREATED:   "created",
	COMPLETED: "completed",
	TEXT:      "text",
	PRIORITY:  "priority",
	DURATION:  "duration",
	DONE:      "done",
}

func (t Order) String() string {
	return orderToString[t]
}

func OrderFromString(order string) (Order, bool) {
	reversed := false
	if len(order) >= 1 && order[0] == '-' {
		reversed = true
		order = order[1:]
	}
	if o, ok := orderFromString[order]; ok {
		return o, reversed
	}
	return PRIORITY, false
}

type taskNodeImpl struct {
	id         int
	tasks      []TaskNode
	parent     TaskNode
	estimate   time.Duration
	attributes map[string]string
}

func newTaskNode(id int) *taskNodeImpl {
	return &taskNodeImpl{
		id:         id,
		parent:     nil,
		attributes: make(map[string]string),
	}
}

func (t *taskNodeImpl) ID() int {
	return t.id
}

func (t *taskNodeImpl) Equal(other TaskNode) bool {
	return t == other
}

func (t *taskNodeImpl) Len() int {
	return len(t.tasks)
}

func (t *taskNodeImpl) At(index int) Task {
	if index >= len(t.tasks) {
		return nil
	}
	return t.tasks[index].(Task)
}

func (t *taskNodeImpl) Parent() TaskNode {
	return t.parent
}

func (t *taskNodeImpl) SetParent(parent TaskNode) {
	t.parent = parent
}

func (t *taskNodeImpl) Append(child TaskNode) {
	child.SetParent(t)
	t.tasks = append(t.tasks, child)
}

func (t *taskNodeImpl) Create(title string, priority Priority) Task {
	task := newTask(t.Len(), title, priority)
	t.Append(task)
	return task
}

func (t *taskNodeImpl) Delete() {
	parent := t.Parent().(*taskNodeImpl)
	if parent == nil {
		panic("can not delete root node")
	}
	for i := 0; i < parent.Len(); i++ {
		if parent.At(i).Equal(t) {
			parent.tasks = append(parent.tasks[:i], parent.tasks[i+1:]...)
			t.parent = nil
			return
		}
	}
	panic("couldn't find t in parent in order to delete")
}

type taskImpl struct {
	*taskNodeImpl
	text               string
	priority           Priority
	created, completed time.Time
	estimate           time.Duration
	attributes         map[string]string
}

func (t *taskImpl) Estimate() time.Duration {
	return t.estimate
}

func (t *taskImpl) SetEstimate(estimate time.Duration) {
	t.estimate = estimate
}

func newTask(id int, text string, priority Priority) Task {
	return &taskImpl{
		taskNodeImpl: newTaskNode(id),
		text:         text,
		priority:     priority,
		created:      time.Now().UTC(),
		completed:    time.Time{},
	}
}

func (t *taskImpl) ID() int {
	return t.id
}

func (t *taskImpl) SetCreationTime(time time.Time) {
	t.created = time
}

func (t *taskImpl) CreationTime() time.Time {
	return t.created
}

func (t *taskImpl) SetCompleted() {
	t.SetCompletionTime(time.Now().UTC())
}

func (t *taskImpl) SetCompletionTime(time time.Time) {
	t.completed = time
}

func (t *taskImpl) CompletionTime() time.Time {
	return t.completed
}

func (t *taskImpl) Text() string {
	return t.text
}

func (t *taskImpl) SetText(text string) {
	t.text = text
}

func (t *taskImpl) Priority() Priority {
	return t.priority
}

func (t *taskImpl) SetPriority(priority Priority) {
	t.priority = priority
}

func (t *taskImpl) Attributes() map[string]string {
	return t.attributes
}

type taskListImpl struct {
	*taskNodeImpl
	title string
}

func NewTaskList() TaskList {
	return &taskListImpl{
		taskNodeImpl: newTaskNode(-1),
		title:        "",
	}
}

// Convert "1.2.3" to int[]{0, 1, 2} ready for indexing into TaskNodes
func indexFromString(index string) Index {
	tokens := strings.Split(index, ".")
	numericIndex := make(Index, len(tokens))
	for i, token := range tokens {
		value, err := strconv.Atoi(token)
		if err != nil || value < 1 {
			return nil
		}
		numericIndex[i] = value - 1
	}
	return numericIndex
}

func (t *taskListImpl) ID() int {
	return -1
}

func (t *taskListImpl) Find(index string) Task {
	numericIndex := indexFromString(index)
	if numericIndex == nil {
		return nil
	}
	var node TaskNode = t // -golint
	for _, i := range numericIndex {
		if node = node.At(i); node == nil {
			return nil
		}
	}
	return node.(Task)
}

// FindAll recursively returns all matching nodes.
func (t *taskListImpl) FindAll(predicate func(task Task) bool) []Task {
	return findAll(t, predicate)
}

func findAll(node TaskNode, predicate func(task Task) bool) []Task {
	out := []Task{}
	if n, ok := node.(Task); ok && predicate(n) {
		out = append(out, n)
	}
	for i := 0; i < node.Len(); i++ {
		out = append(out, findAll(node.At(i), predicate)...)
	}
	return out
}

func (t *taskListImpl) Title() string {
	return t.title
}

func (t *taskListImpl) SetTitle(title string) {
	t.title = title
}

func ReparentTask(node TaskNode, below TaskNode) {
	node.Delete()
	below.Append(node)
}
