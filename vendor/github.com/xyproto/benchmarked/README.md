# Benchmarked

The quest to find a faster `bytes.Equal` function.

## Benchmark results

Tested on Arch Linux, using `go version go1.17.1 linux/amd64`.

`equal33` does better than `bytes.Equal` for many of the benchmarks.

Output from `go test -bench=.`, also using benchmark functions that comes with the Go compiler source code itself:

```
go version go1.17.1 linux/amd64
goos: linux
goarch: amd64
pkg: github.com/xyproto/benchmarked
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkEqual/equal33-12         	  513916	      2127 ns/op
BenchmarkEqual/equal33_0-12       	575770345	         2.025 ns/op
BenchmarkEqual/equal33_1-12       	274571619	         4.306 ns/op	 232.25 MB/s
BenchmarkEqual/equal33_6-12       	272037110	         4.311 ns/op	1391.75 MB/s
BenchmarkEqual/equal33_9-12       	263559217	         4.497 ns/op	2001.41 MB/s
BenchmarkEqual/equal33_15-12      	263189336	         4.495 ns/op	3336.80 MB/s
BenchmarkEqual/equal33_16-12      	261262440	         4.504 ns/op	3552.03 MB/s
BenchmarkEqual/equal33_20-12      	236804361	         5.025 ns/op	3979.79 MB/s
BenchmarkEqual/equal33_32-12      	202573218	         5.848 ns/op	5471.81 MB/s
BenchmarkEqual/equal33_4K-12      	16403617	        65.71 ns/op	62332.22 MB/s
BenchmarkEqual/equal33_4M-12      	   14845	     79608 ns/op	52686.90 MB/s
BenchmarkEqual/equal33_64M-12     	     930	   1287826 ns/op	52110.21 MB/s
BenchmarkEqual/equal33_128M-12    	     451	   2569812 ns/op	52228.62 MB/s
BenchmarkEqual/equal34-12         	  575106	      2024 ns/op
BenchmarkEqual/equal34_0-12       	645477108	         1.832 ns/op
BenchmarkEqual/equal34_1-12       	293812395	         4.047 ns/op	 247.12 MB/s
BenchmarkEqual/equal34_6-12       	261397843	         4.493 ns/op	1335.46 MB/s
BenchmarkEqual/equal34_9-12       	289246339	         4.117 ns/op	2186.14 MB/s
BenchmarkEqual/equal34_15-12      	262379502	         4.503 ns/op	3330.76 MB/s
BenchmarkEqual/equal34_16-12      	260349176	         4.627 ns/op	3458.02 MB/s
BenchmarkEqual/equal34_20-12      	224731081	         5.176 ns/op	3864.18 MB/s
BenchmarkEqual/equal34_32-12      	201397482	         5.959 ns/op	5369.63 MB/s
BenchmarkEqual/equal34_4K-12      	17627247	        70.87 ns/op	57792.85 MB/s
BenchmarkEqual/equal34_4M-12      	   13976	     88424 ns/op	47433.94 MB/s
BenchmarkEqual/equal34_64M-12     	     682	   1714232 ns/op	39148.06 MB/s
BenchmarkEqual/equal34_128M-12    	     334	   3605195 ns/op	37228.97 MB/s
BenchmarkEqual/bytes.Equal-12     	  546596	      2145 ns/op
BenchmarkEqual/bytes.Equal_0-12   	244680985	         4.805 ns/op
BenchmarkEqual/bytes.Equal_1-12   	275909860	         4.391 ns/op	 227.74 MB/s
BenchmarkEqual/bytes.Equal_6-12   	273729403	         4.365 ns/op	1374.45 MB/s
BenchmarkEqual/bytes.Equal_9-12   	257896288	         4.743 ns/op	1897.35 MB/s
BenchmarkEqual/bytes.Equal_15-12  	259227266	         4.696 ns/op	3194.15 MB/s
BenchmarkEqual/bytes.Equal_16-12  	253937388	         4.692 ns/op	3410.05 MB/s
BenchmarkEqual/bytes.Equal_20-12  	224784585	         5.408 ns/op	3697.92 MB/s
BenchmarkEqual/bytes.Equal_32-12  	191623539	         6.170 ns/op	5186.61 MB/s
BenchmarkEqual/bytes.Equal_4K-12  	18115466	        68.05 ns/op	60194.59 MB/s
BenchmarkEqual/bytes.Equal_4M-12  	   16230	     72845 ns/op	57578.60 MB/s
BenchmarkEqual/bytes.Equal_64M-12 	     918	   1478976 ns/op	45375.22 MB/s
BenchmarkEqual/bytes.Equal_128M-12         	     463	   2656537 ns/op	50523.56 MB/s
BenchmarkEqual/equal10-12                  	  473736	      2460 ns/op
BenchmarkEqual/equal10_0-12                	639111051	         1.885 ns/op
BenchmarkEqual/equal10_1-12                	491161402	         2.486 ns/op	 402.31 MB/s
BenchmarkEqual/equal10_6-12                	256543982	         4.650 ns/op	1290.30 MB/s
BenchmarkEqual/equal10_9-12                	226468648	         5.283 ns/op	1703.53 MB/s
BenchmarkEqual/equal10_15-12               	228609232	         5.274 ns/op	2844.14 MB/s
BenchmarkEqual/equal10_16-12               	223065722	         5.395 ns/op	2965.97 MB/s
BenchmarkEqual/equal10_20-12               	205139568	         5.799 ns/op	3448.76 MB/s
BenchmarkEqual/equal10_32-12               	179953700	         6.686 ns/op	4785.90 MB/s
BenchmarkEqual/equal10_4K-12               	17700243	        69.90 ns/op	58600.79 MB/s
BenchmarkEqual/equal10_4M-12               	   14479	     74694 ns/op	56153.19 MB/s
BenchmarkEqual/equal10_64M-12              	     902	   1277835 ns/op	52517.61 MB/s
BenchmarkEqual/equal10_128M-12             	     459	   3091076 ns/op	43421.04 MB/s
PASS
ok  	github.com/xyproto/benchmarked	83.209s
```

## Equal function

Here's the `equal33` function:

```go
func equal33(a, b []byte) bool {
    switch len(a) {
    case 0:
        return len(b) == 0
    case len(b):
        return !(string(a) != string(b))
    default:
        return false
    }
}
```

## bytes.Equal

For comparison, `bytes.Equal` looks like [this](https://cs.opensource.google/go/go/+/refs/tags/go1.16.7:src/bytes/bytes.go;l=18):

```go
func Equal(a, b []byte) bool {
    return string(a) == string(b)
}
```

## Accuracy

I am aware that perfect benchmarking is a tricky.

Please let me know if you have improvements to how the functions are benchmarked, or how the benchmarks are interpreted!


## General info

* Version: 0.3.0
* License: BSD
