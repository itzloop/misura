package testsamples


type NamedParamsAndResults interface {
    Method1(a string, b *int, c []byte) (s string, err error)
}


type UnnamedAndNamedParamsAndResults interface {
    Method1(a string, b *int, c []byte) (s string, err error)
    Method2(a string, b *int, c []byte) (string, error)
    Method3(string, *int, []byte) (string, error)
    Method4(string, *int, []byte) (s string, err error)
}

type UnderscoreNames interface  {
    Method1(_ string, b *int, c []byte) (s string, err error)
    Method2(a string, _ *int, c []byte) (s string, err error)
    Method3(a string, b *int, _ []byte) (s string, err error)
    Method4(a string, b *int, c []byte) (_ string, err error)
    Method5(a string, b *int, c []byte) (s string, _ error)
    Method6(_ string, b *int, _ []byte) (s string, _ error)
    Method7(_ string, _ *int, _ []byte) (s string, _ error)
    Method8(_ string, _ *int, _ []byte) (_ string, _ error)
}


type NoParams interface {
    Method1() error
    Method2() (s string, err error)
    Method3() (string, error)
}

type NoResult interface {
    Method1(s string)
    Method2(n int)
    Method3(a, b, c int, s string)
    Method4(a, _, c int, _ string)
}

// TODO
// {filename: "test.go", target: "ConflictDuration"},
// {filename: "test.go", target: "ConflictTimePackage"},
