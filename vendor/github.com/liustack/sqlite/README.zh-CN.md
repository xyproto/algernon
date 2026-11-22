<h1 align="center"><img src="assets/brand/sqlite-banner.png" alt="SQLite driver for Go"></h1>

<p align="center"><b>全网最快的 cgo-free SQLite 驱动，完全兼容 Go 的 <code>database/sql</code>。<br>自带数据实时备到 Cloudflare R2，让你的应用在单台 VPS 上也能稳定可靠。<br>并发写按顺序排队，<code>database is locked</code> 只代表真过载，不再是运气不好。</b></p>

<p align="center">
  <a href="README.md">English</a> · 简体中文 ·
  <a href="docs/benchmarks.zh-CN.md">基准测试</a> ·
  <a href="docs/architecture.zh-CN.md">工作原理</a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/liustack/sqlite"><img src="https://pkg.go.dev/badge/github.com/liustack/sqlite.svg" alt="Go Reference"></a>
  <a href="https://github.com/liustack/sqlite/actions/workflows/test.yml"><img src="https://github.com/liustack/sqlite/actions/workflows/test.yml/badge.svg?branch=main" alt="test"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License: MIT"></a>
  <a href="https://x.com/liustack"><img src="https://img.shields.io/badge/follow-%40liustack-black?style=flat-square&logo=x&logoColor=white" alt="Follow @liustack on X"></a>
  <img src="https://img.shields.io/badge/CGO-off-success?style=flat-square" alt="CGO off">
  <img src="https://img.shields.io/badge/Not%20backed%20by-Y%20Combinator-FF6600?style=flat-square&logo=ycombinator&logoColor=white" alt="Not backed by Y Combinator">
  <img src="https://img.shields.io/badge/users-unknown-lightgrey?style=flat-square" alt="Users unknown">
</p>

给数据放在 SQLite 里、又想保持纯 Go 的 Go 程序用：一个静态二进制，在哪台机器上都能交叉编译，构建和容器里都不需要 C 编译器。以前这样做要牺牲速度，现在落在后面的反而是 cgo 驱动。

> liustack/sqlite 还在 v0.1，到 v1 之前 API 可能还会变。

## 为什么有它

做 [Tifa](https://github.com/liustack/tifa) 的时候，我想让每个应用都打成一个 Go 二进制，拷到一台小 VPS 上就能跑。数据要放在 SQLite 里，CGO 又必须关掉，不然交叉编译和 `scratch` 镜像就都没了。

现有的纯 Go 驱动为此付出了速度。modernc 把 SQLite 的 C 翻译成 Go，ncruces 把它当 WebAssembly 跑，两个都比 cgo 驱动慢。所以我让 LLVM 把 SQLite 原本的 C 直接编译成 Go 汇编，一直磨到落在后面的变成 cgo 驱动。

## 基准测试

每次操作的耗时，越低越好，取 6 轮的中位数。每行最好的那个，95% 置信区间和别的驱动都不重叠时才加粗。全部用 WAL、`synchronous=NORMAL`，每个驱动的设置都核对过一致。

<!-- bench:readme-linux-amd64 -->
**linux/amd64**（AMD EPYC 7B13，Go 1.27.4，提交 `91bfddb`）：

| 场景 | liustack/sqlite | [mattn](https://github.com/mattn/go-sqlite3)（cgo） | [modernc](https://gitlab.com/cznic/sqlite) | [ncruces](https://github.com/ncruces/go-sqlite3) |
| :-- | --: | --: | --: | --: |
| 插入一行 | 22.8 µs | 23.3 µs | 27.0 µs | 24.3 µs |
| 一个事务插入 1000 行 | **1.41 ms** | 1.68 ms | 2.25 ms | 1.89 ms |
| 按主键读一行 | **3.85 µs** | 7.54 µs | 9.61 µs | 7.22 µs |
| 读 10 万行 | **34.7 ms** | 81.4 ms | 79.7 ms | 56.0 ms |
| 连表加分组 | 2.83 ms | 2.89 ms | 4.91 ms | 4.88 ms |
| 查一个用户和他的 400 条评论 | **521 µs** | 876 µs | 1.13 ms | 841 µs |
| 读 100 行 200 KB 的文本 | 8.77 ms | **8.25 ms** | 9.69 ms | 9.49 ms |
| 按主键读，4 线程 | **1.58 µs** | 2.89 µs | 3.85 µs | 2.75 µs |
| 8 个 goroutine 同时插入 | **11.2 µs** | 21.7 µs | 26.2 µs | 24.7 µs |
| 64 个 goroutine 同时插入 | **3.72 µs** | 22.8 µs | 30.7 µs | 30.2 µs |
| 64 个 goroutine 同时插入，`synchronous=FULL` | **30.9 µs** | 4.60 ms | 7.15 ms | 1.44 ms |
<!-- /bench:readme-linux-amd64 -->

<!-- bench:readme-darwin-arm64 -->
**darwin/arm64**（Apple M4，Go 1.27.2，提交 `91bfddb`）：

| 场景 | liustack/sqlite | [mattn](https://github.com/mattn/go-sqlite3)（cgo） | [modernc](https://gitlab.com/cznic/sqlite) | [ncruces](https://github.com/ncruces/go-sqlite3) |
| :-- | --: | --: | --: | --: |
| 插入一行 | 5.98 µs | 5.90 µs | 7.28 µs | 6.83 µs |
| 一个事务插入 1000 行 | **470 µs** | 554 µs | 733 µs | 566 µs |
| 按主键读一行 | **1.07 µs** | 1.93 µs | 2.77 µs | 2.02 µs |
| 读 10 万行 | **13.7 ms** | 29.6 ms | 29.8 ms | 20.5 ms |
| 连表加分组 | 1.27 ms | **1.12 ms** | 2.64 ms | 1.85 ms |
| 查一个用户和他的 400 条评论 | **212 µs** | 363 µs | 556 µs | 339 µs |
| 读 100 行 200 KB 的文本 | 3.41 ms | 3.34 ms | 3.73 ms | 3.62 ms |
| 按主键读，10 线程 | **1.13 µs** | 1.51 µs | 1.78 µs | 2.32 µs |
| 8 个 goroutine 同时插入 | **2.98 µs** | 6.58 µs | 7.98 µs | 42.3 µs |
| 64 个 goroutine 同时插入 | **1.14 µs** | 9.36 µs | 6.35 ms | 78.7 µs |
<!-- /bench:readme-darwin-arm64 -->

和 C 写的 SQLite 在同一台机器上比：SQLite(C) 是直接调用 SQLite 的 C 程序，liustack/sqlite(直接调用) 不经过 `database/sql` 直接调驱动，两列 `database/sql` 像普通程序一样读写。多个写者同时写时更快，是因为驱动把它们的提交合成一次。

<!-- bench:readme-through-linux-amd64 -->
**linux/amd64**（AMD EPYC 7B13，Go 1.27.4，提交 `91bfddb`）：

**吞吐量**：所有线程加起来每秒完成多少次读或写，文件是 WAL 模式、10 万行，越高越好：

| 基准 | SQLite(C) | liustack/sqlite(直接调用) | liustack/sqlite(database/sql) | mattn(database/sql) |
| :-- | --: | --: | --: | --: |
| 1 个线程，每次按主键读一行 | 189k reads/s ±2% | **373k reads/s** ±4% | 284k reads/s ±4% | 102k reads/s ±2% |
| 8 个线程，每次按主键读一行 | 1075k reads/s ±5% | **1280k reads/s** ±4% | 955k reads/s ±5% | 421k reads/s ±2% |
| 64 个线程，每次按主键读一行 | 982k reads/s ±3% | **1289k reads/s** ±4% | 914k reads/s ±3% | 429k reads/s ±4% |
| 1 个线程，每次按主键读一行，整个文件在缓存里 | 235k reads/s ±4% | **719k reads/s** ±9% | 481k reads/s ±11% | 115k reads/s ±2% |
| 8 个线程，每次按主键读一行，整个文件在缓存里 | 1395k reads/s ±7% | **2228k reads/s** ±8% | 1430k reads/s ±7% | 443k reads/s ±4% |
| 64 个线程，每次按主键读一行，整个文件在缓存里 | 1204k reads/s ±13% | **2003k reads/s** ±5% | 1254k reads/s ±5% | 332k reads/s ±8% |
| 8 个线程照上面读，同时另有 8 个线程在写 | 968k reads/s ±7% | 975k reads/s ±5% | 593k reads/s ±12% | 164k reads/s ±4% |
| 8 个线程，每次插入一行，同时另有 8 个线程在读 | 2431 writes/s ±21% | 20551 writes/s ±8% | **27319 writes/s** ±14% | 14970 writes/s ±5% |
| 64 个线程，每次插入一行，`synchronous=FULL` | 845 writes/s ±3% | 56081 writes/s ±4% | 55955 writes/s ±3% | 851 writes/s ±5% |

<!-- /bench:readme-through-linux-amd64 -->

别的驱动还赢的地方：全在 SQLite 内部算的活（连表），本驱动跑的是同一份 C，只是变成了 Go 汇编，还有读大文本。读和多写为什么更快、各驱动的版本、怎么自己再跑一遍，见 [docs/benchmarks.zh-CN.md](docs/benchmarks.zh-CN.md)。

## 并发写不再报 `database is locked`

同一个进程里的写者按到达顺序排队，不去抢 SQLite 的锁。用 `_txlock=immediate` 打开数据库，一个写事务只有在排在它前面的事务加起来超过 busy timeout（默认 5 秒）时才会报 `database is locked`，也就是写入速度真的超过了磁盘能提交的速度。别的驱动里，抢锁一直输的写者会等上好几秒然后超时，而同样的负载下其他写者只等几毫秒。

<!-- bench:readme-contention-linux-amd64 -->
**linux/amd64**（AMD EPYC 7B13，Go 1.27.4，提交 `e5fe295`）：

**最慢的一次写**（一个事务，或一条自动提交的语句），以及每项 30 秒里有多少次写报 `database is locked`：

| 负载 | liustack/sqlite | [mattn](https://github.com/mattn/go-sqlite3) (cgo) | [modernc](https://gitlab.com/cznic/sqlite) | [ncruces](https://github.com/ncruces/go-sqlite3) |
| :-- | --: | --: | --: | --: |
| 64 个 goroutine 同时各插入一行 | 23.9 ms | 3.93 s | 3.63 s | 86.4 ms |
| 事务，64 个 goroutine，`_txlock=immediate` | 22.6 ms | 4.83 s，3 次锁错误 | 3.84 s | 252 ms |
| 事务，64 个 goroutine，immediate，每个事务里做 2 ms 的事 | 173 ms | 5.01 s，70 次锁错误 | 5.06 s，18 次锁错误 | 2.22 s |
| 事务，64 个 goroutine，immediate，`synchronous=FULL` | 215 ms | 5.01 s，77 次锁错误 | 5.04 s，125 次锁错误 | 1.41 s |
| 事务，256 个 goroutine，immediate，每个事务里做 1 ms 的事 | 329 ms | 5.01 s，273 次锁错误 | 5.11 s，232 次锁错误 | 5.01 s，7 次锁错误 |

<!-- /bench:readme-contention-linux-amd64 -->

这里的事务先读两行再改它们，事务里做的事拿着写锁。各项场景、中位数和每秒写入数见 [docs/benchmarks.zh-CN.md](docs/benchmarks.zh-CN.md#很多写者同时写)。

## 实时备份到 S3

打开数据库之前先启动复制器，之后每次提交约一秒就传到 S3、Cloudflare R2 或 MinIO：

```go
cfg := replica.Config{
	URL: "s3://my-bucket/app.db?endpoint=https://<account>.r2.cloudflarestorage.com&region=auto",
}
r, err := replica.New(ctx, "app.db", cfg)
if err != nil {
	return err
}
defer r.Close()

db, err := sql.Open("sqlite", "app.db")
```

换一台机器，一个调用就把数据库恢复回来，可以是最新的样子，也可以是之前某个时间的样子，默认至少能回到一天前：

```go
txid, err := replica.Restore(ctx, "app.db", cfg, replica.RestoreOptions{})
```

- **写入从不等上传。** 存储慢了或者连不上，写入不会变慢，也不会失败。改动先存在数据库旁边的磁盘上，存储一恢复就传上去。
- **不用多部署任何东西。** 它就是你程序里的一个包，部署的还是那一个二进制，旁边不用再跑 Litestream。
- **恢复时每个文件都核对**，再跑一遍 `PRAGMA integrity_check`，已经在那里的数据库从不被覆盖。
- **Litestream 也能读同一份副本。** 布局和 Litestream 一样，测试里让 `litestream restore` 在每个恢复点重建一遍，核对逐字节相同。
- **`r.Sync(ctx)` 会等到已提交的内容进了存储**，给那些必须先存好再往下走的写入用。

怎么配置、副本落后多少、有哪些限制：[复制](docs/replication.zh-CN.md)。

## 安装

```sh
go get github.com/liustack/sqlite
```

需要 Go 1.26.6 到 1.27（见[升级 Go](docs/upstream.zh-CN.md#升级-go)）。支持 linux、darwin、windows 的 amd64 和 arm64。

## 使用

```go
import (
	"database/sql"

	_ "github.com/liustack/sqlite"
)

db, err := sql.Open("sqlite", "file:app.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_txlock=immediate")
```

名字可以是文件名、`:memory:` 或 [SQLite URI](https://sqlite.org/uri.html)。`_pragma=name(value)` 在每个新连接上执行一条 PRAGMA，`_txlock=immediate` 让 `BEGIN` 先排队拿到写锁，这样先读后写的事务不会写到一半报 `database is locked`。只读的事务用 `db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})` 开，它不拿写锁。没有用 `busy_timeout` 另设的话，连接等锁最多 5 秒。其余见 [pkg.go.dev](https://pkg.go.dev/github.com/liustack/sqlite)。

每个连接在关闭前都留着 SQLite 的页缓存，默认最多 2 MB，而 `database/sql` 会给每个同时在跑的查询开一个连接。给连接池定个大小，并让它留着同样多的空闲连接，免得连接关了又开：

```go
db.SetMaxOpenConns(16)
db.SetMaxIdleConns(16)
```

SQLite 靠统计信息决定一条查询怎么执行。统计信息只有 `ANALYZE` 会收集，数据变了它也不会自己更新。交给 `PRAGMA optimize` 维护就行：程序启动时跑一次 `db.Exec("PRAGMA optimize=0x10002")`，之后定期跑 `db.Exec("PRAGMA optimize")`（比如每天一次，以及 `CREATE INDEX` 之后）。它只分析需要的表，每个索引只抽样读一部分，期间占着写锁，所以一个从没有过统计信息的大库，第一次跑可能让写入等上几秒。完整的 `ANALYZE` 统计最准，但要在写锁下读完整个库，适合在没有写入的时候跑。`_pragma=optimize` 会被拒绝，因为它会在每个连接打开时运行。

SQLite 给的错误是 `*sqlite.Error`。用 `errors.As` 取出来，`Code()` 和 `ExtendedCode()` 给出它的[错误码](https://sqlite.org/rescode.html)，比如 `sqlite.BUSY`、`sqlite.CONSTRAINT_UNIQUE`。

Go 写的 SQL 函数、聚合函数、窗口函数、排序规则和虚拟表放在 connector 上，这样连接池里的每个连接都有：

```go
c, err := sqlite.NewConnector("file:app.db",
	sqlite.WithFunction("reverse", 1, reverse, sqlite.Deterministic),
	sqlite.WithAggregate("mode", 1, newMode),
	sqlite.WithCollation("fold", fold),
	sqlite.WithModule("files", connectFiles))
db := sql.OpenDB(c)
```

`sqlite.Register` 把它们交给之后 `sql.Open` 创建的每个 connector。函数或虚拟表出错、panic 只让它所在的语句失败，连接不受影响。

常用函数在 `ext/` 下的包里现成可用：[`regexp`](https://pkg.go.dev/github.com/liustack/sqlite/ext/regexp)（`REGEXP` 运算符、`regexp_like`、`regexp_substr`、`regexp_replace`）、[`uuid`](https://pkg.go.dev/github.com/liustack/sqlite/ext/uuid)、[`hash`](https://pkg.go.dev/github.com/liustack/sqlite/ext/hash)（md5、sha1、sha256、sha512、crc32）、[`stats`](https://pkg.go.dev/github.com/liustack/sqlite/ext/stats)（方差和标准差）、[`unicode`](https://pkg.go.dev/github.com/liustack/sqlite/ext/unicode)（任何文字的大小写转换，和不分大小写的排序规则）、[`csv`](https://pkg.go.dev/github.com/liustack/sqlite/ext/csv)（把 CSV 文件当表查）、[`series`](https://pkg.go.dev/github.com/liustack/sqlite/ext/series)（`generate_series`）和 [`vec`](https://pkg.go.dev/github.com/liustack/sqlite/ext/vec)（向量距离，找出和一个 embedding 最近的行）。每个包是一个 option，`sqlite.Join` 能把几个合成一个交出去：

```go
sqlite.Register(sqlite.Join(regexp.Option(), uuid.Option()))
```

全文检索（[FTS5](https://sqlite.org/fts5.html)）、空间索引（[R*Tree](https://sqlite.org/rtree.html)）、[数学函数](https://sqlite.org/lang_mathfunc.html)，以及 `median` 和 `percentile`（[percentile](https://sqlite.org/percentile.html)）都已编进来，不用导入什么。数学函数在每个平台上算出的位都一样。

驱动注册的名字是 `sqlite`，和 modernc.org/sqlite、glebarez/go-sqlite 一样，所以同时导入其中一个的程序启动时会 panic。这时用 `sql.OpenDB(sqlite.NewConnector(dsn))` 打开数据库。

## 对你有什么不同

- **全程没有 C。** 在 Mac 上 `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build` 就行，出来的二进制能直接跑在 `scratch` 镜像里。
- **多个写入方不再互相等。** 几个 goroutine 同时写时，驱动把它们自动提交的 `INSERT`、`UPDATE`、`DELETE` 放进一个事务，只提交一次。每次调用照样拿到自己的错误、`LastInsertId` 和 `RowsAffected`，并且提交完才返回，持久性一点不打折。某条语句的失败让 SQLite 回滚了整个事务时（IO 错误、`ON CONFLICT ROLLBACK`、执行中 context 结束），排在它前面合并的语句一起失败，拿到 `ABORT_ROLLBACK`，没有哪条会跑第二遍。你自己开的事务从不合并。[规则](docs/group-commit.zh-CN.md)
- **写入方排队，不再动不动报 `database is locked`。** 驱动里同一个文件同一时刻只有你程序的一个连接在写，其余的排队等（最多等 `busy_timeout`），不在 SQLite 的忙等里睡。先读后写的事务，如果中间有别人写过，还是会失败，这种事务用 `_txlock=immediate` 开。别的进程照样遵守 SQLite 自己的锁。
- **读更省。** 调 SQLite 就是一次普通的 Go 函数调用，在 Go 1.27 上把数字读进你的变量不分配内存（字符串只分配它自己那一次）。
- **自带实时备份。** 每次提交约一秒后就到了 S3、R2 或 MinIO，在你的程序里完成。见[实时备份到 S3](#实时备份到-s3)。

## 凭什么信它

- **Tifa 建在它上面。** 用 `tifa new` 新建的应用都用这个驱动存数据（[Tifa](https://github.com/liustack/tifa) 是全栈 Go 框架）。
- **稳定和安全第一，速度第二，内存第三。** 更快但可能丢写入、返回错结果或削弱提交的改动，一律不做。合并提交给每条合并的语句都包一个 SAVEPOINT，就是这个原因，哪怕它拖慢速度。
- **C 是 SQLite 原版，一个字没改。** `engine/` 放的是 3.54.1 发布版的合并源码，按哈希核对（`engine/vendor.sh --verify`）。编译选项集中在一个文件里，每条都写了理由。
- **Go 汇编是生成的，从不手写。** LLVM 编译 C，生成器把 LLVM 的输出改写成 Go 汇编。遇到没有规则的东西就停下，还会核对链接后的二进制。仓库里的汇编和生成器按 `engine/` 现在生成的不一致时，`make check-gen` 会失败。
- **数字里留着输掉的地方。** 上面的表保留了别的驱动更快的行，[docs/benchmarks.zh-CN.md](docs/benchmarks.zh-CN.md) 写明了每次怎么测的、测的时候机器上还跑着什么。
- **每次推送都测**，linux/amd64、linux/arm64、darwin/arm64、windows/amd64、windows/arm64（除 windows 外开着竞态检测），还测 sqlite3 命令行同时写同一个库，以及写到一半杀掉进程。

## 还没有的

- 没有可加载扩展：SQLite 编译时去掉了 `load_extension`。
- 没有共享缓存：`cache=shared` 会报错。想让池里所有连接看到同一个内存数据库，用 `file:/name?vfs=memdb`。
- 双引号只表示标识符（`SQLITE_DQS=0`，SQLite 官方的建议）：字符串字面量请用单引号。
- 其他平台（freebsd、32 位、riscv64 等）构建不了：构建会报错，错误里写着支持哪些平台。

## 文档

| 文档 | 什么时候读 |
| :-- | :-- |
| [基准测试](docs/benchmarks.zh-CN.md) | 想看完整数据，或者在自己机器上跑一遍 |
| [架构](docs/architecture.zh-CN.md) | 想知道 C 怎么变成 Go 汇编，或者合并提交怎么工作 |
| [合并提交](docs/group-commit.zh-CN.md) | 想知道并发写入哪些会合并、失败时各自拿到什么错误 |
| [语句缓存](docs/query-cache.zh-CN.md) | 想知道驱动替你留下哪些语句，以及留下的语句为什么不会读到旧东西 |
| [复制](docs/replication.zh-CN.md) | 想把数据库复制到 S3、R2 或 MinIO，或者从副本恢复 |
| [所有核都忙的时候](docs/busy-cores.zh-CN.md) | 读或写偶尔要等几十毫秒以上，或者只在一两个核上跑 |
| [上游](docs/upstream.zh-CN.md) | 想知道用的是哪个 SQLite，或者怎么升级到新版 |
| [更新记录](CHANGELOG.md)（英文） | 升级前想知道这一版改了什么 |
| [Testing](docs/testing.md)、[Commits](docs/commit.md)（英文） | 准备改代码 |

## 联系我们

随时欢迎提 issue：[新建一个](https://github.com/liustack/sqlite/issues/new/choose)。用它做了什么、哪里不好用、下一步该做什么，都可以说。推荐你关注公众号 liustack，也欢迎你来 X 上聊：**[@liustack](https://x.com/liustack)**。驱动有什么变化、接下来做什么，都会在公众号和 X 同步更新。

关注微信公众号「liustack」：AI 创业机会、独立开发见解、AI 实战与工具，第一时间推送。微信扫码，或搜一搜「liustack」：

<p align="center">
  <img src="https://raw.githubusercontent.com/liustack/sqlite/main/assets/wechat-qrcode.png" width="420" alt="微信公众号 liustack" />
</p>

## 参与

liustack/sqlite 不接受 pull request。这里改一处，可能从 SQLite 的 C 一路牵动到生成器、合并提交和复制，做对需要大量上下文，所以每一行都由作者一个人写和审。可以这样帮忙：

- **[提 issue](https://github.com/liustack/sqlite/issues/new/choose)。** bug、结果不对、查询慢、看不懂的报错、写得不清楚的文档、想法。一个能复现的小程序比一个补丁更有用，issue 决定接下来做什么。
- **fork 它。** 按 MIT 许可，你的副本随你改、随你发布。在 fork 上开发的说明见 [CONTRIBUTING.md](CONTRIBUTING.md)。

安全问题请私下报告，不要提公开 issue，见 [SECURITY.md](SECURITY.md)。

## Star History

<a href="https://www.star-history.com/#liustack/sqlite&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=liustack/sqlite&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=liustack/sqlite&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=liustack/sqlite&type=Date" />
 </picture>
</a>

## 致谢

感谢 D. Richard Hipp 和 SQLite 的开发者写了 SQLite，感谢 musl 提供引擎链接的字符串函数。感谢 mattn/go-sqlite3、modernc.org/sqlite、ncruces/go-sqlite3 的作者，他们探出了 SQLite 在 Go 里的几条路，本驱动也是拿它们来比的。

## 许可

本仓库的代码用 MIT 许可。SQLite 属于公有领域，musl 的代码保留它的 MIT 许可。哪个路径归哪个许可，见 [NOTICE.md](NOTICE.md)。

SQLite 是 Hipp, Wyrick & Company, Inc. 的注册商标，本项目与其没有关联。横幅里的地鼠是重新绘制的 [Go gopher](https://go.dev/blog/gopher)，原作者 Renee French，按 [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) 授权使用。
