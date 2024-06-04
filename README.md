# promwrapgen

> promwrapgen wraps an interface or a struct and adds prometheus metrics such as success count, error count, total count and duration


THIS IS A WIP

## TODO

- [x] Handle slice ... operator
- [x] Only work on targets passed not all interfaces in file
- [x] Pass method name and other method related information Total, Success and Error
- [x] Get list of targets not just one
    - [ ] ~~Check generated file exists, if yes append to it.~~
    - [x] Create a seperate file for each target.
- [x] Let users decided what metrics they want
- [ ] Handle `time` package conflict
- [ ] Add struct wrapping support?
    - Only methods in the same file will be included
    - Add an options to create an interface for the struct aswell
- [ ] Enable users to extend wrapping functionallity to add custom logic to their interfaces
- [x] ~~Custom metrics?~~ This is solved by accepting metrics interface.
- [ ] Per type method inclusion and exlusion
- [ ] Support both go:generate promwrapgen [args] and //promwrapgen:<target-name> [args]

## Usage

Assume we have the following interface in a `.go` file.

```golang
type IPUtil interface {
	PublicIP() (net.IP, error)
	LocalIPs() ([]net.IP, error)
}
```

Now to generate a wrapper for this interface we have 2 options:

1. Put a magic comment for the entire file and passing each interface name with the -t flag.

TODO: add multiple targets using multiple -t args

```golang
//go:generate promwrapgen -m all -t IPUtil
type IPUtil interface {
	PublicIP() (net.IP, error)
	LocalIPs() ([]net.IP, error)
}
```

2. Add a magic comment on top of the file then use `//promwrapgen:<taget-name> [args]` syntax. This method makes the file readable.

```golang
package main

//go:generate promwrapgen
import (
...
)

//promwrapgen:IPUtil -m all
type IPUtil interface {
	PublicIP() (net.IP, error)
	LocalIPs() ([]net.IP, error)
}

```

## Testing

```golang
go test ./...
```

- [x] Test generated wrappers for compliation
- [ ] Test generated code is as expected using `ast`, or maybe run them with a utilty program and run it that way.

## 

TODO
