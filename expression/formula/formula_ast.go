package formula

// abstract syntax tree for formulas

// An object that changes as the syntax tree is traversed.
type compilerEnv struct {
	lexEnv *lexEnv
}
