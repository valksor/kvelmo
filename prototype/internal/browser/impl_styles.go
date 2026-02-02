package browser

import (
	"context"
	"fmt"

	"github.com/go-rod/rod/lib/proto"
)

// GetComputedStyles returns all computed CSS properties for an element matching the selector.
func (c *controller) GetComputedStyles(ctx context.Context, tabID, selector string) ([]ComputedStyle, error) {
	nodeID, err := c.resolveNodeID(ctx, tabID, selector)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()
	if err != nil {
		return nil, errNotFound("tab " + tabID)
	}

	ctxPage := page.Context(ctx)

	// Enable CSS domain
	_ = proto.CSSEnable{}.Call(ctxPage)

	result, err := proto.CSSGetComputedStyleForNode{NodeID: nodeID}.Call(ctxPage)
	if err != nil {
		return nil, fmt.Errorf("browser: get computed styles: %w", err)
	}

	styles := make([]ComputedStyle, 0, len(result.ComputedStyle))
	for _, prop := range result.ComputedStyle {
		styles = append(styles, ComputedStyle{
			Name:  prop.Name,
			Value: prop.Value,
		})
	}

	return styles, nil
}

// GetMatchedStyles returns the full CSS cascade for an element matching the selector.
func (c *controller) GetMatchedStyles(ctx context.Context, tabID, selector string) (*MatchedStyles, error) {
	nodeID, err := c.resolveNodeID(ctx, tabID, selector)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()
	if err != nil {
		return nil, errNotFound("tab " + tabID)
	}

	ctxPage := page.Context(ctx)

	// Enable CSS domain
	_ = proto.CSSEnable{}.Call(ctxPage)

	result, err := proto.CSSGetMatchedStylesForNode{NodeID: nodeID}.Call(ctxPage)
	if err != nil {
		return nil, fmt.Errorf("browser: get matched styles: %w", err)
	}

	matched := &MatchedStyles{}

	// Extract inline styles
	if result.InlineStyle != nil {
		matched.InlineStyles = convertCSSProperties(result.InlineStyle.CSSProperties)
	}

	// Extract matched CSS rules
	for _, ruleMatch := range result.MatchedCSSRules {
		if ruleMatch.Rule == nil {
			continue
		}
		matched.MatchedRules = append(matched.MatchedRules, convertRuleMatch(ruleMatch))
	}

	// Extract inherited styles
	for _, inherited := range result.Inherited {
		entry := InheritedStyleEntry{}
		if inherited.InlineStyle != nil {
			entry.InlineStyles = convertCSSProperties(inherited.InlineStyle.CSSProperties)
		}
		for _, ruleMatch := range inherited.MatchedCSSRules {
			if ruleMatch.Rule == nil {
				continue
			}
			entry.MatchedRules = append(entry.MatchedRules, convertRuleMatch(ruleMatch))
		}
		matched.InheritedStyles = append(matched.InheritedStyles, entry)
	}

	// Extract pseudo-element styles
	for _, pseudo := range result.PseudoElements {
		pe := PseudoElementStyles{
			PseudoType: string(pseudo.PseudoType),
		}
		for _, ruleMatch := range pseudo.Matches {
			if ruleMatch.Rule == nil {
				continue
			}
			pe.MatchedRules = append(pe.MatchedRules, convertRuleMatch(ruleMatch))
		}
		matched.PseudoElements = append(matched.PseudoElements, pe)
	}

	return matched, nil
}

// resolveNodeID finds a DOM element by CSS selector and returns its CDP NodeID.
func (c *controller) resolveNodeID(ctx context.Context, tabID, selector string) (proto.DOMNodeID, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()
	if err != nil {
		return 0, errNotFound("tab " + tabID)
	}

	elem, err := page.Context(ctx).Element(selector)
	if err != nil {
		return 0, fmt.Errorf("browser: element not found for selector %q: %w", selector, err)
	}

	desc, err := elem.Describe(0, false)
	if err != nil {
		return 0, fmt.Errorf("browser: describe element: %w", err)
	}

	return desc.NodeID, nil
}

// convertCSSProperties converts CDP CSS properties to our CSSProperty type.
func convertCSSProperties(props []*proto.CSSCSSProperty) []CSSProperty {
	result := make([]CSSProperty, 0, len(props))
	for _, p := range props {
		result = append(result, CSSProperty{
			Name:      p.Name,
			Value:     p.Value,
			Important: p.Important,
		})
	}

	return result
}

// convertRuleMatch converts a CDP rule match to our MatchedRule type.
func convertRuleMatch(rm *proto.CSSRuleMatch) MatchedRule {
	rule := rm.Rule
	mr := MatchedRule{
		Origin:     string(rule.Origin),
		Properties: convertCSSProperties(rule.Style.CSSProperties),
	}

	// Extract selector text
	if rule.SelectorList != nil {
		mr.Selector = rule.SelectorList.Text
	}

	// Extract source URL from stylesheet
	if rule.StyleSheetID != "" {
		mr.SourceURL = string(rule.StyleSheetID)
	}

	return mr
}
