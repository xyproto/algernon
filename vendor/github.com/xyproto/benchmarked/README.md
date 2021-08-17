# Benchmarked

The quest to find a faster `bytes.Equal` function.

So far, this function is 28% faster than `bytes.Equal`:

## Code comparison

```go
func equal14(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }
    for i := 0; i < len(b); i++ {
        if i >= len(a) || a[i] != b[i] {
            return false
        }
    }
    return true
}
```

For comparison, `bytes.Equal` looks like this:

```go
func Equal(a, b []byte) bool {
    return string(a) == string(b)
}
```

## Benchmark results

```
goos: linux
goarch: amd64
pkg: github.com/xyproto/benchmarked
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkEqual/equal6-12         	 1302963	       900.7 ns/op
BenchmarkEqual/bytes.Equal-12    	 1627982	       735.7 ns/op
BenchmarkEqual/equal1-12         	 1623241	       713.0 ns/op
BenchmarkEqual/equal3-12         	 1744081	       696.8 ns/op
BenchmarkEqual/equal2-12         	 1711257	       694.3 ns/op
BenchmarkEqual/equal11-12        	 1713615	       692.9 ns/op
BenchmarkEqual/equal12-12        	 1819748	       638.0 ns/op
BenchmarkEqual/equal4-12         	 1930615	       614.8 ns/op
BenchmarkEqual/equal7-12         	 1963322	       603.2 ns/op
BenchmarkEqual/equal5-12         	 2042334	       590.3 ns/op
BenchmarkEqual/equal8-12         	 2036313	       578.2 ns/op
BenchmarkEqual/equal10-12        	 2060468	       576.4 ns/op
BenchmarkEqual/equal13-12        	 2168073	       542.9 ns/op
BenchmarkEqual/equal9-12         	 2222503	       540.5 ns/op
BenchmarkEqual/equal14-12        	 2218479	       532.1 ns/op
PASS
ok  	github.com/xyproto/benchmarked	28.312s
```

Currently, `equal14` is the function that is exported as `benchmarked.Equal`.

## Accuracy

I am aware that perfect benchmarking is a tricky.

Please let me know if you have improvements to how the functions are benchmarked, or how the benchmarks are interpreted!

## General info

* Version: 0.1.0
* License: BSD
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
