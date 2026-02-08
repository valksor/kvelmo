package com.valksor.mehrhof.testutil

import com.valksor.mehrhof.api.MehrhofApiClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import org.junit.jupiter.api.extension.AfterEachCallback
import org.junit.jupiter.api.extension.BeforeEachCallback
import org.junit.jupiter.api.extension.ExtensionContext

/**
 * JUnit 5 extension that manages a [MockWebServer] lifecycle.
 *
 * Provides helper methods for common response patterns and request verification.
 *
 * Usage:
 * ```kotlin
 * class MyApiTest {
 *     @RegisterExtension
 *     val mockServer = MockServerExtension()
 *
 *     @Test
 *     fun `test api call`() {
 *         mockServer.enqueueSuccess("""{"status": "ok"}""")
 *
 *         val client = mockServer.createClient()
 *         val result = client.getStatus()
 *
 *         assertTrue(result.isSuccess)
 *         mockServer.assertRequest("GET", "/api/v1/status")
 *     }
 * }
 * ```
 */
class MockServerExtension :
    BeforeEachCallback,
    AfterEachCallback {
    private lateinit var server: MockWebServer

    /**
     * Get the underlying MockWebServer instance.
     */
    val mockServer: MockWebServer
        get() = server

    /**
     * Get the base URL of the mock server.
     */
    val baseUrl: String
        get() = server.url("/").toString().trimEnd('/')

    override fun beforeEach(context: ExtensionContext?) {
        server = MockWebServer()
        server.start()
    }

    override fun afterEach(context: ExtensionContext?) {
        server.shutdown()
    }

    /**
     * Create a [MehrhofApiClient] configured to use this mock server.
     */
    fun createClient(): MehrhofApiClient = MehrhofApiClient(baseUrl)

    /**
     * Enqueue a successful JSON response (HTTP 200).
     */
    fun enqueueSuccess(jsonBody: String) {
        server.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(jsonBody),
        )
    }

    /**
     * Enqueue an error response.
     */
    fun enqueueError(
        statusCode: Int,
        body: String = "",
    ) {
        server.enqueue(
            MockResponse()
                .setResponseCode(statusCode)
                .setBody(body),
        )
    }

    /**
     * Enqueue a JSON error response.
     */
    fun enqueueJsonError(
        statusCode: Int,
        jsonBody: String,
    ) {
        server.enqueue(
            MockResponse()
                .setResponseCode(statusCode)
                .setHeader("Content-Type", "application/json")
                .setBody(jsonBody),
        )
    }

    /**
     * Enqueue a network failure (connection refused).
     */
    fun enqueueNetworkError() {
        server.enqueue(
            MockResponse()
                .setSocketPolicy(okhttp3.mockwebserver.SocketPolicy.DISCONNECT_AFTER_REQUEST),
        )
    }

    /**
     * Take the next request from the queue.
     */
    fun takeRequest(): RecordedRequest = server.takeRequest()

    /**
     * Assert that the next request matches the expected method and path.
     */
    fun assertRequest(
        method: String,
        path: String,
    ): RecordedRequest {
        val request = server.takeRequest()
        assert(request.method == method) {
            "Expected method $method but was ${request.method}"
        }
        assert(request.path == path) {
            "Expected path $path but was ${request.path}"
        }
        return request
    }

    /**
     * Assert that the next request matches the expected method and path prefix.
     * Useful for paths with query parameters.
     */
    fun assertRequestStartsWith(
        method: String,
        pathPrefix: String,
    ): RecordedRequest {
        val request = server.takeRequest()
        assert(request.method == method) {
            "Expected method $method but was ${request.method}"
        }
        assert(request.path?.startsWith(pathPrefix) == true) {
            "Expected path starting with $pathPrefix but was ${request.path}"
        }
        return request
    }

    /**
     * Assert that the next POST request has the expected JSON body.
     */
    fun assertPostBody(
        path: String,
        expectedBodyContains: String,
    ): RecordedRequest {
        val request = server.takeRequest()
        assert(request.method == "POST") {
            "Expected POST but was ${request.method}"
        }
        assert(request.path == path) {
            "Expected path $path but was ${request.path}"
        }
        val body = request.body.readUtf8()
        assert(body.contains(expectedBodyContains)) {
            "Expected body to contain '$expectedBodyContains' but was '$body'"
        }
        return request
    }

    /**
     * Get the number of requests that have been made.
     */
    val requestCount: Int
        get() = server.requestCount
}
