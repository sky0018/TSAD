# TaskQueue

Example

```
func inc(numI interface{}) (interface{}, error) {
	if num, ok := numI.(int); ok {
		return num + 1, nil
	}
	return nil, errors.New("invalid req")
}

func main() {
    cases := 1000
	results := make([]*Result, 0, cases)
    tq := NewTaskQueue(100, 1000)
    tq.Start()
    defer tq.Stop()

	for i := 0; i < cases; i++ {
		result, _ := queue.Submit(inc, rand.Int())
		results = append(results, result)
	}

    for i := 0; i < cases; i++ {
        <-results[i].Done
        fmt.Println(results[i].Resp)
    }
}
```
