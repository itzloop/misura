package testsamples


type NamedParamsAndResults interface {
    Method1(a string, b *int, c []byte) (s string, err error)
}
