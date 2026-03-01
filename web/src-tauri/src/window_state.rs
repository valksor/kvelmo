//! Window state persistence
//!
//! Saves and restores window position, size, and maximized state
//! to ~/.valksor/kvelmo/window-state.json

use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use tauri::{PhysicalPosition, PhysicalSize, WebviewWindow, Window};

/// Persisted window state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WindowState {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
    pub is_maximized: bool,
}

impl WindowState {
    /// Get the path to the window state file
    fn state_path() -> Option<PathBuf> {
        dirs::home_dir().map(|h| h.join(".valksor").join("kvelmo").join("window-state.json"))
    }

    /// Load window state from disk
    pub fn load() -> Option<Self> {
        let path = Self::state_path()?;
        let content = fs::read_to_string(path).ok()?;
        serde_json::from_str(&content).ok()
    }

    /// Save window state to disk
    pub fn save(&self) -> Result<(), String> {
        let path = Self::state_path().ok_or("Could not determine state path")?;

        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        }

        let content = serde_json::to_string_pretty(self).map_err(|e| e.to_string())?;
        fs::write(path, content).map_err(|e| e.to_string())
    }
}

/// Restore window state from saved file
pub fn restore(window: &WebviewWindow) {
    if let Some(state) = WindowState::load() {
        // Restore position
        let _ = window.set_position(PhysicalPosition::new(state.x, state.y));

        // Restore size
        let _ = window.set_size(PhysicalSize::new(state.width, state.height));

        // Restore maximized state
        if state.is_maximized {
            let _ = window.maximize();
        }
    }
}

/// Save current window state to file (from Window in event handler)
pub fn save(window: &Window) {
    let Ok(position) = window.outer_position() else {
        return;
    };
    let Ok(size) = window.outer_size() else {
        return;
    };

    let state = WindowState {
        x: position.x,
        y: position.y,
        width: size.width,
        height: size.height,
        is_maximized: window.is_maximized().unwrap_or(false),
    };

    if let Err(e) = state.save() {
        tracing::warn!("Failed to save window state: {}", e);
    }
}
