# SCHEMA_RULES — lurus 事件契约演进铁律

> 适用范围:`proto/lurus/events/**` 全部事件类型与 `EventEnvelope`。
> 执法工具:`buf lint`(STANDARD)+ `buf breaking`(FILE,CI 对 main 跑)。
> 工具挡不住的语义规则(下表 R3/R5/R6)靠 code review 执行,违例 = 拒合。

## 规则

| # | 规则 | 原因 |
|---|------|------|
| R1 | **字段只增不删**。要淘汰一个字段:标注 `[deprecated = true]`,停止写入,永不移除 | 事件日志 append-only,历史事件永远要能解码 |
| R2 | **字段号与字段名永不复用**。若确需移除(仅限从未发布的分支),必须 `reserved` 号段与名字 | 复用字段号 = 旧数据解码成新语义,静默数据腐蚀 |
| R3 | **语义变更 = 新 event_type**。字段含义、单位、必填性的任何改变都不是"改 schema",而是发布一个新的 NSID 类型(如 `...stock.received_v2` 或更准确的新动词);旧类型继续可解码 | 读视图按 event_type 分派;同名异义会污染所有历史重放 |
| R4 | **payload 加字段 → `schema_ver` +1**(EventEnvelope.schema_ver),字段必须向后兼容(proto3 默认值语义安全) | 消费方可按 schema_ver 决定是否启用新字段 |
| R5 | **PII 字段必须标注 `(lurus.events.v1.pii) = true`**(定义见 `proto/lurus/events/v1/annotations.proto`)。姓名、手机号、邮箱、地址、用户输入的自由文本、可含人名的设备名等一律算 PII | 为将来 crypto-shredding 预留:PII 字段按租户密钥加密,删密钥 = 在不可变日志上实现擦除(PIPL) |
| R6 | **event_type 命名 = NSID 风格** `cn.lurus.<product>.<domain>.<past_tense_fact>`,动词过去式、陈述已发生事实,禁止命令式(`receive_stock` ✗ / `stock.received` ✓) | 事件是事实不是指令;命名空间防跨产品撞名 |
| R7 | **每个事件域首批类型 ≤5 个,宁缺毋滥**。新增类型须先在本文件的映射表登记 | 契约面积即维护面积 |
| R8 | **金额一律 int64 分(fen)**,数量一律 int64 最小计量单位;禁 float/double 进任何事件 payload | 与 platform ledger int64-money 约定对齐;浮点不可对账 |

## 已注册 event_type 总表

| NSID | payload message | schema_ver | 状态 |
|------|-----------------|-----------|------|
| `cn.lurus.tally.stock.received` | `lurus.events.tally.v1.StockReceived` | 1 | active |
| `cn.lurus.tally.stock.issued` | `lurus.events.tally.v1.StockIssued` | 1 | active |
| `cn.lurus.tally.stock.transferred` | `lurus.events.tally.v1.StockTransferred` | 1 | active |
| `cn.lurus.tally.stock.counted` | `lurus.events.tally.v1.StockCounted` | 1 | active |
| `cn.lurus.tally.stock.written_off` | `lurus.events.tally.v1.StockWrittenOff` | 1 | active |
| `cn.lurus.lugo.billing.charged` | `lurus.events.lugo.v1.BillingCharged` | 1 | schema-only |
| `cn.lurus.lugo.billing.refunded` | `lurus.events.lugo.v1.BillingRefunded` | 1 | schema-only |
| `cn.lurus.lugo.billing.quota_deducted` | `lurus.events.lugo.v1.QuotaDeducted` | 1 | schema-only |
| `cn.lurus.sync.device_registered` | `lurus.events.sync.v1.DeviceRegistered` | 1 | active |
| `cn.lurus.sync.cursor_advanced` | `lurus.events.sync.v1.CursorAdvanced` | 1 | active |

> `schema-only`:类型已定稿但无任何生产者/消费者实现;资金执行路径仍以
> platform ledger 为唯一真源(见 billing.proto 头注)。

## 本仓布局约定

- `.proto` 源:`proto/lurus/...`(buf module root = `proto/`)
- Go 生成物:仓根 `<domain>/<version>/*.pb.go`(与既有 `identity/v1/` 布局一致);
  重新生成:`buf generate`
- `identity/v1` 是先于本规则的存量(仅生成物、无 .proto 源),不受 R1-R8 追溯约束,
  其 .proto 源回填是开放事项
