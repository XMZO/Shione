mod app;
mod mock;
mod state;

fn main() -> Result<(), slint::PlatformError> {
    silence_dependency_warnings();
    app::ClientApp::new()?.run()
}

fn silence_dependency_warnings() {
    log::set_max_level(log::LevelFilter::Error);
}
