package com.valksor.mehrhof.settings

import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.components.service
import com.intellij.util.xmlb.XmlSerializerUtil

/**
 * Application-level settings for the Mehrhof plugin.
 * Persisted across IDE restarts.
 */
@State(
    name = "MehrhofSettings",
    storages = [Storage("mehrhof.xml")]
)
@Service(Service.Level.APP)
class MehrhofSettings : PersistentStateComponent<MehrhofSettings.State> {

    private var myState = State()

    data class State(
        var serverUrl: String = "",  // Empty = plugin manages server
        var mehrExecutable: String = "",  // Empty = auto-detect from default locations
        var showNotifications: Boolean = true,
        var defaultAgent: String = "",
        var autoReconnect: Boolean = true,
        var reconnectDelaySeconds: Int = 5,
        var maxReconnectAttempts: Int = 10
    )

    override fun getState(): State = myState

    override fun loadState(state: State) {
        XmlSerializerUtil.copyBean(state, myState)
    }

    var serverUrl: String
        get() = myState.serverUrl
        set(value) {
            myState.serverUrl = value
        }

    var mehrExecutable: String
        get() = myState.mehrExecutable
        set(value) {
            myState.mehrExecutable = value
        }

    var showNotifications: Boolean
        get() = myState.showNotifications
        set(value) {
            myState.showNotifications = value
        }

    var defaultAgent: String
        get() = myState.defaultAgent
        set(value) {
            myState.defaultAgent = value
        }

    var autoReconnect: Boolean
        get() = myState.autoReconnect
        set(value) {
            myState.autoReconnect = value
        }

    var reconnectDelaySeconds: Int
        get() = myState.reconnectDelaySeconds
        set(value) {
            myState.reconnectDelaySeconds = value
        }

    var maxReconnectAttempts: Int
        get() = myState.maxReconnectAttempts
        set(value) {
            myState.maxReconnectAttempts = value
        }

    companion object {
        fun getInstance(): MehrhofSettings = service()
    }
}
