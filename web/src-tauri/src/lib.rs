//! kvelmo Desktop Application Library
//!
//! This library provides the core functionality for the kvelmo desktop app:
//! - Server management (spawning/stopping the Go backend)
//! - Platform-specific handling (WSL on Windows, sidecar on macOS/Linux)
//! - Window state persistence
//! - System tray with status indicator
//! - Native notifications for task completion/failure

pub mod server_manager;

#[cfg(target_os = "windows")]
pub mod wsl;

pub mod window_state;

use tauri::{
    Emitter, Manager,
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
};

/// Application version (should match Cargo.toml and Go binary)
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Initialize the Tauri application
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_notification::init())
        .setup(|app| {
            let app_handle = app.handle().clone();

            // Build system tray
            setup_tray(app)?;

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

/// Set up the system tray icon with menu items.
fn setup_tray(app: &tauri::App) -> Result<(), Box<dyn std::error::Error>> {
    let show = MenuItem::with_id(app, "show", "Show kvelmo", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&show, &quit])?;

    TrayIconBuilder::new()
        .tooltip("kvelmo - AI Task Orchestrator")
        .menu(&menu)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "show" => {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.show();
                    let _ = window.set_focus();
                }
            }
            "quit" => {
                app.exit(0);
            }
            _ => {}
        })
        .build(app)?;

    Ok(())
}
