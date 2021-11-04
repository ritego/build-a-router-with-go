# RiteGo - Build a Router with Go

    The full source code for this tutorial is available at [Build a Router with Go](https//github.com/ritego/build-a-router-with-go). 

## Start with Mux
Mux is the short and nice way to say (multiplexer)[https://en.wikipedia.org/wiki/Multiplexer]. In electronics, the primary source of the word, a multiplexer is a device that produce a single signal output based on several possible signal inputs. The output is a function of the input signal and some additional login. 

Interestingly, the concept of a multiplexer fits into requests (inputs) and responses (outputs) pattern of the HTTP protocol. 

In this tutorial, we are going to build a custom HTTP multiplexer (or better still a mux or a router).


## Setup Module
First thing first, setup your environment
1. Run the following in your root directory: `$ go mod init github.com/ritego/build-a-router-with-go` to generate the `go.mod` file
```mod
// go. mod
module github.com/ritego/build-a-router-with-go

go 1.16
```

## Conclusion
The full source code for this tutorial is available at [Build a Router with Go](https://github.com/ritego/build-a-router-with-go).
