# figlet4go
     _______  __    _______  __       _______ .___________. _  _      _______   ______
    |   ____||  |  /  _____||  |     |   ____||           || || |    /  _____| /  __  \
    |  |__   |  | |  |  __  |  |     |  |__   `---|  |----`| || |_  |  |  __  |  |  |  |
    |   __|  |  | |  | |_ | |  |     |   __|      |  |     |__   _| |  | |_ | |  |  |  |
    |  |     |  | |  |__| | |  `----.|  |____     |  |        | |   |  |__| | |  `--'  |
    |__|     |__|  \______| |_______||_______|    |__|        |_|    \______|  \______/

A port of [figlet](http://www.figlet.org/) to golang.  
Make it easier to use,add some new feature such as colorized outputs.

## Usage


### Install

```
go get -u github.com/getwe/figlet4go
```

### Demo

```
cd demo/
go build
./demo -str="golang"
#Maybe you have to `brew install figlet` if you need 3D fond in mac osx.
```

see details in `demo/demo.go` .

![screenshot](./screenshot/demo1.jpg)
