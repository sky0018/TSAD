# Golang日志库

[![Go Report Card](http://golang-report.byted.org/badge/code.byted.org/gopkg/logs)](http://golang-report.byted.org/report/code.byted.org/gopkg/logs)
[![build status](http://code.byted.org/ci/projects/10/status.png?ref=master)](http://code.byted.org/ci/projects/10?ref=master)


# 本日志库由于历史原因引入了较多的外部依赖，例如Metrics和databus，使用此日志库之前请周知

### 关于日志级别

比较有疑问的是以下几个日志级别的定位: Notice, Info, Trace。这里对日志级别的定义统一如下：

0. Trace
1. Debug
2. Info
3. Notice
4. Warn
5. Error
6. Fatal


### PushNotice
新版本中增加了对PushNotice的支持;

利用ctx缓存当前调用栈的kv信息, 因此需要提前调用NewNoticeCtx;

当栈返回时, 需要调用CtxFlushNotice;

下面是demo;

```
func handler(ctx context.Context, req interface{}) (interface{}, error) {
	ctx = logs.NewNoticeCtx(ctx)
	defer logs.CtxFlushNotice(ctx)
	logs.CtxPushNotice(ctx, "method", "handler")
	return method0(ctx, req.(int))
}

func method0(ctx context.Context, id int) (interface{}, error) {
	logs.CtxPushNotice(ctx, "id", id)
	return method1(ctx)
}

func method1(ctx context.Context) (interface{}, error) {
	logs.CtxPushNotice(ctx, "method1", "this is method1")
	return nil, nil
}
```

### 新版升级: 增加metrics
新的logs增加了Warn, Error, Fatal三个字段的metrics.

新增的metrics为toutiao.service.log.{PSM}.throughput. 

metrics的tag中包含一个"level"字段, 用于显示错误等级, 分别为"WARNING", "ERROR", "CRITICAL", 为了和PY做统一, 所以稍有差异.


### 设计思想

日志模块分为logger和logger provider两个不同的组件, logger provider实现log往哪个地方写的逻辑。通常会有console, file, scribe等。

logger模块拥有自己的level，用于判断是否往各个provider输出日志，provider模块也有各自的level

#### 前缀

    level date time code

*注*
如果该日志库无法满足需求，，请尽量在自己使用的时候封装一层，而不要直接修改基础库
