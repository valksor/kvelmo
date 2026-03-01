//! kvelmo Desktop Application Library
//!
//! This library provides the core functionality for the kvelmo desktop app:
//! - Server management (spawning/stopping the Go backend)
//! - Platform-specific handling (WSL on Windows, sidecar on macOS/Linux)
//! - Window state persistence

pub mod server_manager;

#[cfg(target_os = "windows")]
pub mod wsl;

pub mod window_state;

use tauri::{Emitter, Manager};

/// Application version (should match Cargo.toml and Go binary)
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Initialize the Tauri application
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_process::init())
        .setup(|app| {
            let app_handle = app.handle().clone();

            // Always spawn the Go server
            // In dev mode: Vite proxies to it, no navigation needed
            // In production: Navigate to the server URL
            println!("[kvelmo-desktop] Starting server manager...");
            tauri::async_runtime::spawn(async move {
                match server_manager::start_server(&app_handle).await {
                    Ok(()) => println!("[kvelmo-desktop] Server started successfully"),
                    Err(e) => {
                        println!("[kvelmo-desktop] Failed to start server: {}", e);
                        let _ = app_handle.emit("server-error", e.to_string());
                    }
                }
            });

            // Restore window state
            if let Some(window) = app.get_webview_window("main") {
                window_state::restore(&window);
            }

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { .. } = event {
                // Save window state before closing
                window_state::save(window);
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
