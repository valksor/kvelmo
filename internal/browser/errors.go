package browser

import "fmt"

// BrowserError represents browser-specific errors.
type BrowserError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

// Error returns the error message.
func (e *BrowserError) Error() string {
	if e.Err == nil {
		return "browser: " + e.Op
	}

	return fmt.Sprintf("browser: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *BrowserError) Unwrap() error {
	return e.Err
}

// Common error constructors.
func errConnect(err error) error {
	return &BrowserError{Op: "connect", Err: err}
}

func errDisconnect(err error) error {
	return &BrowserError{Op: "disconnect", Err: err}
}

func errLaunch(err error) error {
	return &BrowserError{Op: "launch browser", Err: err}
}

func errListTabs(err error) error {
	return &BrowserError{Op: "list tabs", Err: err}
}

func errOpenTab(err error) error {
	return &BrowserError{Op: "open tab", Err: err}
}

func errCloseTab(err error) error {
	return &BrowserError{Op: "close tab", Err: err}
}

func errSwitchTab(err error) error {
	return &BrowserError{Op: "switch tab", Err: err}
}

func errNavigate(err error) error {
	return &BrowserError{Op: "navigate", Err: err}
}

func errScreenshot(err error) error {
	return &BrowserError{Op: "screenshot", Err: err}
}

func errQuerySelector(err error) error {
	return &BrowserError{Op: "query selector", Err: err}
}

func errClick(err error) error {
	return &BrowserError{Op: "click", Err: err}
}

func errType(err error) error {
	return &BrowserError{Op: "type", Err: err}
}

func errEval(err error) error {
	return &BrowserError{Op: "eval", Err: err}
}

func errNotFound(msg string) error {
	return &BrowserError{Op: "not found", Err: fmt.Errorf("%s", msg)}
}
