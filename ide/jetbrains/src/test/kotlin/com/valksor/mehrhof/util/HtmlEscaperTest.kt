package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class HtmlEscaperTest {
    // ========================================================================
    // escape tests
    // ========================================================================

    @Test
    fun `escapes ampersand`() {
        assertEquals("foo &amp; bar", HtmlEscaper.escape("foo & bar"))
    }

    @Test
    fun `escapes less than`() {
        assertEquals("foo &lt; bar", HtmlEscaper.escape("foo < bar"))
    }

    @Test
    fun `escapes greater than`() {
        assertEquals("foo &gt; bar", HtmlEscaper.escape("foo > bar"))
    }

    @Test
    fun `converts newlines to br`() {
        assertEquals("line1<br>line2", HtmlEscaper.escape("line1\nline2"))
    }

    @Test
    fun `handles multiple special chars`() {
        assertEquals("a &amp; b &lt; c &gt; d", HtmlEscaper.escape("a & b < c > d"))
    }

    @Test
    fun `handles empty string`() {
        assertEquals("", HtmlEscaper.escape(""))
    }

    @Test
    fun `handles string with only special chars`() {
        assertEquals("&amp;&lt;&gt;<br>", HtmlEscaper.escape("&<>\n"))
    }

    @Test
    fun `handles multiple consecutive newlines`() {
        assertEquals("a<br><br>b", HtmlEscaper.escape("a\n\nb"))
    }

    @Test
    fun `preserves normal text`() {
        assertEquals("Hello World 123", HtmlEscaper.escape("Hello World 123"))
    }

    @Test
    fun `handles HTML-like input`() {
        assertEquals("&lt;div&gt;text&lt;/div&gt;", HtmlEscaper.escape("<div>text</div>"))
    }

    // ========================================================================
    // formatMessage tests
    // ========================================================================

    @Test
    fun `formatMessage wraps in div with class`() {
        val result = HtmlEscaper.formatMessage("user", "Hello")
        assertEquals("<div class=\"user\">Hello</div>", result)
    }

    @Test
    fun `formatMessage escapes content`() {
        val result = HtmlEscaper.formatMessage("error", "a < b")
        assertEquals("<div class=\"error\">a &lt; b</div>", result)
    }

    @Test
    fun `formatMessage handles empty content`() {
        val result = HtmlEscaper.formatMessage("system", "")
        assertEquals("<div class=\"system\"></div>", result)
    }

    @Test
    fun `formatMessage handles newlines in content`() {
        val result = HtmlEscaper.formatMessage("assistant", "line1\nline2")
        assertEquals("<div class=\"assistant\">line1<br>line2</div>", result)
    }

    // ========================================================================
    // CssClasses constants tests
    // ========================================================================

    @Test
    fun `CssClasses contains expected values`() {
        assertEquals("user", HtmlEscaper.CssClasses.USER)
        assertEquals("assistant", HtmlEscaper.CssClasses.ASSISTANT)
        assertEquals("system", HtmlEscaper.CssClasses.SYSTEM)
        assertEquals("error", HtmlEscaper.CssClasses.ERROR)
        assertEquals("command", HtmlEscaper.CssClasses.COMMAND)
    }

    // ========================================================================
    // Convenience method tests
    // ========================================================================

    @Test
    fun `formatUserMessage formats correctly`() {
        val result = HtmlEscaper.formatUserMessage("Hello")
        assertEquals("<div class=\"user\">You: Hello</div>", result)
    }

    @Test
    fun `formatUserMessage escapes special chars`() {
        val result = HtmlEscaper.formatUserMessage("a < b")
        assertEquals("<div class=\"user\">You: a &lt; b</div>", result)
    }

    @Test
    fun `formatAssistantMessage formats correctly`() {
        val result = HtmlEscaper.formatAssistantMessage("Response")
        assertEquals("<div class=\"assistant\">Agent: Response</div>", result)
    }

    @Test
    fun `formatSystemMessage formats correctly`() {
        val result = HtmlEscaper.formatSystemMessage("System info")
        assertEquals("<div class=\"system\">System info</div>", result)
    }

    @Test
    fun `formatErrorMessage formats correctly`() {
        val result = HtmlEscaper.formatErrorMessage("Something went wrong")
        assertEquals("<div class=\"error\">Error: Something went wrong</div>", result)
    }

    @Test
    fun `formatCommandMessage formats correctly`() {
        val result = HtmlEscaper.formatCommandMessage("plan")
        assertEquals("<div class=\"command\">&gt; plan</div>", result)
    }

    @Test
    fun `formatCommandMessage escapes the greater than in prefix`() {
        // The "> " prefix itself should be escaped
        val result = HtmlEscaper.formatCommandMessage("test")
        assertTrue(result.contains("&gt;"), "The > prefix should be escaped")
    }
}
