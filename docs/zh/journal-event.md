# Journal

`Journal` 通过注册事件的方式，记录系统中特定的状态，将其输出到指定存储系统中，目前仅实现了文件系统。

`Journal` 主要功能有两个：

- `Journal Event`：注册事件，跟踪事件；
- `Alerting`：依赖于`Journal Event` 的报警机制。

通过这两个功能，有助于辅助运维工作，跟踪系统的运行状况。


## Journal Event

通过事件方式记录需要的系统状态。可以是文件系统，也可以是其他存储系统，仅需实现接口：

```go
type EventTypeRegistry interface {

	// RegisterEventType introduces a new event type to a journal, and
	// returns an EventType token that components can later use to check whether
	// journalling for that type is enabled/suppressed, and to tag journal
	// entries appropriately.
	RegisterEventType(system, event string) EventType
}

type Journal interface {
	EventTypeRegistry

	// RecordEvent records this event to the journal, if and only if the
	// EventType is enabled. If so, it calls the supplier function to obtain
	// the payload to record.
	//
	// Implementations MUST recover from panics raised by the supplier function.
	RecordEvent(evtType EventType, supplier func() interface{})

	// Close closes this journal for further writing.
	Close() error
}
```

在 `venus-miner` 系统中注册了以下事件: `System: "miner", Event: "block_mined"`，用于记录矿工的出块。在日志中以 `json` 方式保存：
```json
{"System":"miner","Event":"block_mined","Timestamp":"2022-02-08T09:44:45.49744285+08:00","Data":{"cid":{"/":"bafy2bzacebur5zuktumidwj7hm6tv42f2cy6khwtk4zjz4434b5hfrdq2xcns"},"epoch":91552,"miner":"t01031","nulls":0,"parents":[{"/":"bafy2bzacect4kjtcml4uvgvmqtmbrdvh34zb73b5pqr7udjtzd5pnpa3lq5g4"}],"timestamp":1644284700}}
```

## Alerting

`Alerting` 封装了 `Journal` 用于跟踪事件，通过 `AddAlertType`注册报警类型，调用`Raise` 及 `Resolve` 发起或解决报警。

```go
type Alerting struct {
	j journal.Journal

	lk     sync.Mutex
	alerts map[AlertType]Alert
}
```
`Alerting` 可用于监控系统资源，辅助运维做出合理的资源分配。
