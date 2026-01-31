package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class CommandParserTest {
    // ========================================================================
    // Empty/whitespace input tests
    // ========================================================================

    @Test
    fun `empty input returns error`() {
        val result = CommandParser.parse("")
        assertTrue(result is ParsedInput.Error)
        assertEquals("Empty input", (result as ParsedInput.Error).message)
    }

    @Test
    fun `whitespace only input returns error`() {
        val result = CommandParser.parse("   ")
        assertTrue(result is ParsedInput.Error)
    }

    @Test
    fun `tabs only input returns error`() {
        val result = CommandParser.parse("\t\t")
        assertTrue(result is ParsedInput.Error)
    }

    // ========================================================================
    // Help command tests
    // ========================================================================

    @Test
    fun `parses help command`() {
        val result = CommandParser.parse("help")
        assertTrue(result is ParsedInput.Help)
    }

    @Test
    fun `parses HELP command case insensitive`() {
        val result = CommandParser.parse("HELP")
        assertTrue(result is ParsedInput.Help)
    }

    @Test
    fun `parses question mark as help`() {
        val result = CommandParser.parse("?")
        assertTrue(result is ParsedInput.Help)
    }

    // ========================================================================
    // Clear command tests
    // ========================================================================

    @Test
    fun `parses clear command`() {
        val result = CommandParser.parse("clear")
        assertTrue(result is ParsedInput.Clear)
    }

    @Test
    fun `parses CLEAR command case insensitive`() {
        val result = CommandParser.parse("CLEAR")
        assertTrue(result is ParsedInput.Clear)
    }

    // ========================================================================
    // Answer command tests
    // ========================================================================

    @Test
    fun `parses answer with response`() {
        val result = CommandParser.parse("answer yes")
        assertTrue(result is ParsedInput.Answer)
        assertEquals("yes", (result as ParsedInput.Answer).response)
    }

    @Test
    fun `parses answer shortcut a`() {
        val result = CommandParser.parse("a no")
        assertTrue(result is ParsedInput.Answer)
        assertEquals("no", (result as ParsedInput.Answer).response)
    }

    @Test
    fun `parses answer with long response`() {
        val result = CommandParser.parse("answer This is a longer response with spaces")
        assertTrue(result is ParsedInput.Answer)
        assertEquals("This is a longer response with spaces", (result as ParsedInput.Answer).response)
    }

    @Test
    fun `answer without response returns error`() {
        val result = CommandParser.parse("answer")
        assertTrue(result is ParsedInput.Error)
        assertEquals("Usage: answer <response>", (result as ParsedInput.Error).message)
    }

    @Test
    fun `answer shortcut without response returns error`() {
        val result = CommandParser.parse("a")
        assertTrue(result is ParsedInput.Error)
    }

    // ========================================================================
    // Chat command tests
    // ========================================================================

    @Test
    fun `parses chat with message`() {
        val result = CommandParser.parse("chat Hello there")
        assertTrue(result is ParsedInput.Chat)
        assertEquals("Hello there", (result as ParsedInput.Chat).message)
    }

    @Test
    fun `parses chat shortcut c`() {
        val result = CommandParser.parse("c How are you?")
        assertTrue(result is ParsedInput.Chat)
        assertEquals("How are you?", (result as ParsedInput.Chat).message)
    }

    @Test
    fun `parses ask as chat`() {
        val result = CommandParser.parse("ask What is this?")
        assertTrue(result is ParsedInput.Chat)
        assertEquals("What is this?", (result as ParsedInput.Chat).message)
    }

    @Test
    fun `chat without message returns error`() {
        val result = CommandParser.parse("chat")
        assertTrue(result is ParsedInput.Error)
        assertEquals("Usage: chat <message>", (result as ParsedInput.Error).message)
    }

    // ========================================================================
    // Workflow command tests
    // ========================================================================

    @Test
    fun `parses start command with args`() {
        val result = CommandParser.parse("start github:123")
        assertTrue(result is ParsedInput.Command)
        val cmd = result as ParsedInput.Command
        assertEquals("start", cmd.name)
        assertEquals(listOf("github:123"), cmd.args)
    }

    @Test
    fun `parses plan command without args`() {
        val result = CommandParser.parse("plan")
        assertTrue(result is ParsedInput.Command)
        val cmd = result as ParsedInput.Command
        assertEquals("plan", cmd.name)
        assertTrue(cmd.args.isEmpty())
    }

    @Test
    fun `parses implement command`() {
        val result = CommandParser.parse("implement")
        assertTrue(result is ParsedInput.Command)
        assertEquals("implement", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses impl as implement alias`() {
        val result = CommandParser.parse("impl")
        assertTrue(result is ParsedInput.Command)
        assertEquals("impl", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses review command`() {
        val result = CommandParser.parse("review")
        assertTrue(result is ParsedInput.Command)
        assertEquals("review", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses continue command`() {
        val result = CommandParser.parse("continue")
        assertTrue(result is ParsedInput.Command)
        assertEquals("continue", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses cont as continue alias`() {
        val result = CommandParser.parse("cont")
        assertTrue(result is ParsedInput.Command)
        assertEquals("cont", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses finish command`() {
        val result = CommandParser.parse("finish")
        assertTrue(result is ParsedInput.Command)
        assertEquals("finish", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses abandon command`() {
        val result = CommandParser.parse("abandon")
        assertTrue(result is ParsedInput.Command)
        assertEquals("abandon", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses undo command`() {
        val result = CommandParser.parse("undo")
        assertTrue(result is ParsedInput.Command)
        assertEquals("undo", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses redo command`() {
        val result = CommandParser.parse("redo")
        assertTrue(result is ParsedInput.Command)
        assertEquals("redo", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses status command`() {
        val result = CommandParser.parse("status")
        assertTrue(result is ParsedInput.Command)
        assertEquals("status", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses st as status alias`() {
        val result = CommandParser.parse("st")
        assertTrue(result is ParsedInput.Command)
        assertEquals("st", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses cost command`() {
        val result = CommandParser.parse("cost")
        assertTrue(result is ParsedInput.Command)
        assertEquals("cost", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses list command`() {
        val result = CommandParser.parse("list")
        assertTrue(result is ParsedInput.Command)
        assertEquals("list", (result as ParsedInput.Command).name)
    }

    @Test
    fun `parses note command with message`() {
        val result = CommandParser.parse("note This is my note")
        assertTrue(result is ParsedInput.Command)
        val cmd = result as ParsedInput.Command
        assertEquals("note", cmd.name)
        assertEquals(listOf("This", "is", "my", "note"), cmd.args)
    }

    // ========================================================================
    // Default behavior (unrecognized input treated as chat)
    // ========================================================================

    @Test
    fun `unrecognized input treated as chat`() {
        val result = CommandParser.parse("Hello, how are you?")
        assertTrue(result is ParsedInput.Chat)
        assertEquals("Hello, how are you?", (result as ParsedInput.Chat).message)
    }

    @Test
    fun `sentence starting with unknown word treated as chat`() {
        val result = CommandParser.parse("Let me explain something")
        assertTrue(result is ParsedInput.Chat)
        assertEquals("Let me explain something", (result as ParsedInput.Chat).message)
    }

    // ========================================================================
    // normalizeCommand tests
    // ========================================================================

    @Test
    fun `normalizeCommand converts impl to implement`() {
        assertEquals("implement", CommandParser.normalizeCommand("impl"))
    }

    @Test
    fun `normalizeCommand converts st to status`() {
        assertEquals("status", CommandParser.normalizeCommand("st"))
    }

    @Test
    fun `normalizeCommand converts cont to continue`() {
        assertEquals("continue", CommandParser.normalizeCommand("cont"))
    }

    @Test
    fun `normalizeCommand converts spec to specification`() {
        assertEquals("specification", CommandParser.normalizeCommand("spec"))
    }

    @Test
    fun `normalizeCommand lowercases input`() {
        assertEquals("plan", CommandParser.normalizeCommand("PLAN"))
    }

    @Test
    fun `normalizeCommand keeps unknown commands lowercase`() {
        assertEquals("unknown", CommandParser.normalizeCommand("Unknown"))
    }

    // ========================================================================
    // workflowCommands set tests
    // ========================================================================

    @Test
    fun `workflowCommands contains expected commands`() {
        assertTrue("start" in CommandParser.workflowCommands)
        assertTrue("plan" in CommandParser.workflowCommands)
        assertTrue("implement" in CommandParser.workflowCommands)
        assertTrue("impl" in CommandParser.workflowCommands)
        assertTrue("review" in CommandParser.workflowCommands)
        assertTrue("finish" in CommandParser.workflowCommands)
        assertTrue("undo" in CommandParser.workflowCommands)
        assertTrue("redo" in CommandParser.workflowCommands)
    }

    @Test
    fun `workflowCommands does not contain chat commands`() {
        assertFalse("chat" in CommandParser.workflowCommands)
        assertFalse("answer" in CommandParser.workflowCommands)
        assertFalse("help" in CommandParser.workflowCommands)
        assertFalse("clear" in CommandParser.workflowCommands)
    }
}
