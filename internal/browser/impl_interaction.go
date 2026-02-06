package browser

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Screenshot captures a screenshot of a tab.
func (c *controller) Screenshot(ctx context.Context, tabID string, opts ScreenshotOptions) ([]byte, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	var data []byte
	quality := opts.Quality
	ctxPage := page.Context(ctx)
	if opts.FullPage {
		data, err = ctxPage.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormat(opts.Format),
			Quality: &quality,
		})
	} else {
		data, err = ctxPage.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormat(opts.Format),
			Quality: &quality,
		})
	}

	if err != nil {
		return nil, errScreenshot(err)
	}

	return data, nil
}

// QuerySelector queries a single element.
func (c *controller) QuerySelector(ctx context.Context, tabID, selector string) (*DOMElement, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	elem, err := page.Context(ctx).Element(selector)
	if err != nil {
		return nil, errQuerySelector(err)
	}

	return c.elementToDOM(elem, page)
}

// QuerySelectorAll queries all matching elements.
func (c *controller) QuerySelectorAll(ctx context.Context, tabID, selector string) ([]DOMElement, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	elems, err := page.Context(ctx).Elements(selector)
	if err != nil {
		return nil, errQuerySelector(err)
	}

	result := make([]DOMElement, 0, len(elems))
	for _, elem := range elems {
		dom, err := c.elementToDOM(elem, page)
		if err != nil {
			continue
		}
		result = append(result, *dom)
	}

	return result, nil
}

// Click clicks an element.
func (c *controller) Click(ctx context.Context, tabID, selector string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return err
	}

	elem, err := page.Context(ctx).Element(selector)
	if err != nil {
		return errClick(err)
	}

	if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errClick(err)
	}

	return nil
}

// Type types text into an element.
func (c *controller) Type(ctx context.Context, tabID, selector, text string, clearField bool) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return err
	}

	elem, err := page.Context(ctx).Element(selector)
	if err != nil {
		return errType(err)
	}

	if clearField {
		if err := elem.SelectAllText(); err != nil {
			return errType(fmt.Errorf("select all: %w", err))
		}
		if err := elem.Input(""); err != nil {
			return errType(fmt.Errorf("clear: %w", err))
		}
	}

	if err := elem.Input(text); err != nil {
		return errType(err)
	}

	return nil
}

// Eval evaluates JavaScript.
// The expression can be a simple value (like "1 + 1") or a function.
// We use the RuntimeEvaluate CDP command directly to avoid Rod's
// function call mechanism which requires .apply().
func (c *controller) Eval(ctx context.Context, tabID, expression string) (any, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	// Use RuntimeEvaluate directly for simple expression evaluation.
	// This avoids Rod's Eval/Evaluate which wrap expressions in
	// function calls using .apply(), which fails for simple values.
	res, err := proto.RuntimeEvaluate{
		Expression:    expression,
		ReturnByValue: true,
		AwaitPromise:  true,
	}.Call(page.Context(ctx))
	if err != nil {
		return nil, errEval(err)
	}

	if res.ExceptionDetails != nil {
		return nil, errEval(fmt.Errorf("js exception: %s", res.ExceptionDetails.Exception.Description))
	}

	// The Value field is gson.JSON; use Val() to get the underlying Go value
	return res.Result.Value.Val(), nil
}

// elementToDOM converts a Rod element to our DOMElement type.
func (c *controller) elementToDOM(elem *rod.Element, _ *rod.Page) (*DOMElement, error) {
	// Get element info
	text, _ := elem.Text()
	visible, _ := elem.Visible()
	html, _ := elem.HTML()

	// Get full node description
	node, err := elem.Describe(1, false)
	if err != nil {
		// Fallback to basic info if Describe fails
		return &DOMElement{
			TagName:     "element",
			TextContent: text,
			OuterHTML:   html,
			Visible:     visible,
			X:           0,
			Y:           0,
		}, err
	}

	// Get attributes
	attributes := make(map[string]string)
	if node.Attributes != nil {
		for i := 0; i < len(node.Attributes)-1; i += 2 {
			attributes[node.Attributes[i]] = node.Attributes[i+1]
		}
	}

	// Get child count
	childCount := 0
	if node.ChildNodeCount != nil {
		childCount = *node.ChildNodeCount
	}

	return &DOMElement{
		NodeID:      int64(node.NodeID),
		BackendID:   int64(node.BackendNodeID),
		TagName:     node.NodeName,
		Attributes:  attributes,
		TextContent: text,
		OuterHTML:   html,
		ChildCount:  childCount,
		Visible:     visible,
		X:           0, // BoxModel requires separate call
		Y:           0,
	}, nil
}
