# Risotto
Package risotto is a JavaScript to JavaScript compiler. Risotto's parser and AST is forked from [otto](https://github.com/robertkrimen/otto).
The main motivation behind Risotto is to be used by [Gonads](https://github.com/mamaar/gonads), a frontend toolkit that currently compiles JSX and SASS.

### Development
Because Risotto uses the stable and mature parser from otto, most of the development happens inside *generator*. Running `go test` there will run all input**n**.js in the *test* directory through Risotto. The output will be compared against the related output file.

# Example
### Input JavaScript
```
(function() {
    var i = <div />;
    console.log("Hello, world!")
})
```


### Output JavaScript
```
(function () {
    var i = React.createElement("div", null);
    console.log("Hello, world!");
});
```
