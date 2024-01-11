package types

type Method struct {
	MethodSigFull    string
	MethodName       string
	MethodParamNames string
	ResultNames      string
	NamedResults     bool
	HasError         bool
	HasCtx           bool
	Ctx              string
}
