package orchestrator

func performOperation(arg1, arg2 float64, op string) float64 {
	switch op {
	case "+":
		return arg1 + arg2
	case "-":
		return arg1 - arg2
	case "*":
		return arg1 * arg2
	case "/":
		if arg2 == 0 {
			panic("division by zero")
		}
		return arg1 / arg2
	default:
		panic("unknown operator")
	}
}
