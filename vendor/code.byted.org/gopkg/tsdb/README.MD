# TSDB
该Lib用于拉取TSDB数据

# Feature
+ 自动重试
+ 超时设置
+ 并发控制
+ 批量提交任务

# Examples

同步的拉取数据
```
func sync() {
	cli, err := tsdb.NewClient(&tsdb.Options{
		MaxConcurrency: 1,
		DefaultTimeout: time.Second,
		DefaultRetry:   2,
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// 同步的访问TSDB
	var models []*tsdb.RespModel
	err = cli.Do(context.Background(), testMetrics, &models)
	if err != nil {
		panic(err)
	}

	for _, m := range models {
		fmt.Println(m.DPS.AVG())
	}
}
```

批量的异步提交任务
```
func async() {
	cli, err := tsdb.NewClient(&tsdb.Options{
		MaxConcurrency: 10,
		DefaultTimeout: time.Second,
		DefaultRetry:   2,
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// 异步提交TSDB任务
	taskNum := 20
	futures := make([]*tsdb.Future, 0, taskNum)
	for i := 0; i < taskNum; i++ {
		var models []*tsdb.RespModel
		ctx := context.Background()
		ctx = tsdb.WithRetry(ctx, i/2)
		ctx = tsdb.WithTimeout(ctx, time.Millisecond*time.Duration(100*i))
		futures = append(futures, cli.Go(ctx, testMetrics, &models))
	}

	// 等待任务的结果
	for i, f := range futures {
		<-f.Done
		fmt.Printf("task %v: \n", i)
		fmt.Println("url: ", f.QueryURL)
		fmt.Println("cost: ", f.Cost)
		if f.Err != nil {
			fmt.Println("err: ", f.Err)
		} else {
			models := f.Model.(*[]*tsdb.RespModel)
			for _, m := range *models {
				fmt.Println(m.DPS.AVG())
			}
		}
		fmt.Println()
	}
}

```