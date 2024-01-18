# promwrapgen

> promwrapgen wraps an interface or a struct and adds prometheus metrics such as success count, error count, total count and duration


THIS IS A WIP

## TODO

- [x] Handle slice ... operator
- [x] Only work on targets passed not all interfaces in file
- [x] Pass method name and other method related information Total, Success and Error
- [x] Get list of targets not just one
    - [ ] Check generated file exists, if yes append to it.
- [x] Let users decided what metrics they want
- [ ] Handle `time` package conflict
- [ ] Add struct wrapping support?? What if struct methods are in multiple files???
- [ ] Enable users to extend wrapping functionallity to add custom logic to their interfaces
- [ ] Custom metrics?
- [ ] Per type method inclusion and exlusion

## Usage

```golang

```

## Testing

- [ ] Test generated wrappers for compliation
- [ ] Test generated code is as expected using `ast`.

## 

TODO
