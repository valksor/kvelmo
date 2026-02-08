package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.project.Project
import io.mockk.mockk
import io.mockk.unmockkAll
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for [MehrhofToolWindowFactory].
 *
 * Note: Full integration tests of createToolWindowContent require the IntelliJ
 * test framework. These tests verify basic factory properties.
 */
class MehrhofToolWindowFactoryTest {
    private lateinit var project: Project

    @BeforeEach
    fun setUp() {
        project = mockk(relaxed = true)
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Factory Tests
    // ========================================================================

    @Test
    fun `shouldBeAvailable returns true for any project`() {
        val factory = MehrhofToolWindowFactory()

        assertTrue(factory.shouldBeAvailable(project))
    }

    @Test
    fun `factory implements ToolWindowFactory`() {
        val factory = MehrhofToolWindowFactory()

        assertTrue(factory is com.intellij.openapi.wm.ToolWindowFactory)
    }

    @Test
    fun `factory implements DumbAware`() {
        val factory = MehrhofToolWindowFactory()

        assertTrue(factory is com.intellij.openapi.project.DumbAware)
    }
}
