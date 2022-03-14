package blx

type JkError struct {
	message string
}

//JkError实现了 Error() 方法的对象都可以
func (e *JkError) Error() string {
	return e.message
}

func NewJkError(message string) *JkError {
	return &JkError{message: message}
}
