use std::cell::RefCell;
use std::rc::Rc;
use std::time::Duration;

use slint::{ComponentHandle, ModelRc, SharedString, Timer, TimerMode, VecModel};

use crate::mock;
use crate::state::{AppState, LogEntry as StateLogEntry, Member as StateMember, Room as StateRoom};

slint::include_modules!();

pub struct ClientApp {
    ui: AppWindow,
    state: Rc<RefCell<AppState>>,
    _timer: Timer,
}

impl ClientApp {
    pub fn new() -> Result<Self, slint::PlatformError> {
        let ui = AppWindow::new()?;
        let state = Rc::new(RefCell::new(mock::initial_state()));
        let timer = Timer::default();

        let app = Self {
            ui,
            state,
            _timer: timer,
        };

        app.install_handlers();
        app.install_mock_timer();
        app.sync();

        Ok(app)
    }

    pub fn run(self) -> Result<(), slint::PlatformError> {
        self.ui.run()
    }

    fn install_handlers(&self) {
        let ui_handle = self.ui.as_weak();
        let state = Rc::clone(&self.state);
        self.ui.on_toggle_connection(move || {
            let Some(ui) = ui_handle.upgrade() else {
                return;
            };

            {
                let mut state = state.borrow_mut();
                state.toggle_connection();
                mock::tick(&mut state);
            }

            sync_ui(&ui, &state.borrow());
        });

        let ui_handle = self.ui.as_weak();
        let state = Rc::clone(&self.state);
        self.ui.on_navigate(move |page| {
            let Some(ui) = ui_handle.upgrade() else {
                return;
            };

            state.borrow_mut().navigate(page);
            sync_ui(&ui, &state.borrow());
        });
    }

    fn install_mock_timer(&self) {
        let ui_handle = self.ui.as_weak();
        let state = Rc::clone(&self.state);

        self._timer.start(
            TimerMode::Repeated,
            Duration::from_millis(1000),
            move || {
                let Some(ui) = ui_handle.upgrade() else {
                    return;
                };

                {
                    let mut state = state.borrow_mut();
                    mock::tick(&mut state);
                }

                sync_ui(&ui, &state.borrow());
            },
        );
    }

    fn sync(&self) {
        sync_ui(&self.ui, &self.state.borrow());
    }
}

fn sync_ui(ui: &AppWindow, state: &AppState) {
    ui.set_current_page(state.current_page.as_i32());
    ui.set_connected(state.connected);
    ui.set_status_text(state.status_text.as_str().into());
    ui.set_virtual_ip(state.virtual_ip.as_str().into());
    ui.set_current_room(current_room_name(state).into());
    ui.set_node_latency(state.node_latency.as_str().into());
    ui.set_upload_speed(state.upload_speed.as_str().into());
    ui.set_download_speed(state.download_speed.as_str().into());
    ui.set_relay_mode(state.relay_mode.as_str().into());
    ui.set_nat_type(state.nat_type.as_str().into());
    ui.set_auto_reconnect(state.auto_reconnect);
    ui.set_minimize_to_tray(state.minimize_to_tray);
    ui.set_launch_on_startup(state.launch_on_startup);
    ui.set_rooms(model_from_vec(state.rooms.iter().map(map_room).collect()));
    ui.set_members(model_from_vec(
        state.members.iter().map(map_member).collect(),
    ));
    ui.set_logs(model_from_vec(state.logs.iter().map(map_log).collect()));
}

fn model_from_vec<T: Clone + 'static>(items: Vec<T>) -> ModelRc<T> {
    ModelRc::from(Rc::new(VecModel::from(items)))
}

fn map_room(room: &StateRoom) -> UiRoom {
    UiRoom {
        name: to_shared(&room.name),
        mode: to_shared(&room.mode),
        members: to_shared(&room.members),
        latency: to_shared(&room.latency),
        active: room.active,
    }
}

fn map_member(member: &StateMember) -> UiMember {
    UiMember {
        name: to_shared(&member.name),
        initials: to_shared(&member_initials(&member.name)),
        ip: to_shared(&member.ip),
        latency: to_shared(&member.latency),
        online: member.online,
    }
}

fn map_log(log: &StateLogEntry) -> UiLogEntry {
    UiLogEntry {
        time: to_shared(&log.time),
        level: to_shared(&log.level),
        message: to_shared(&log.message),
    }
}

fn to_shared(value: &str) -> SharedString {
    SharedString::from(value)
}

fn current_room_name(state: &AppState) -> &str {
    state
        .rooms
        .iter()
        .find(|room| room.active)
        .or_else(|| state.rooms.first())
        .map(|room| room.name.as_str())
        .unwrap_or("--")
}

fn member_initials(name: &str) -> String {
    let mut initials = name
        .split(|c: char| c == ' ' || c == '-' || c == '_')
        .filter_map(|part| part.chars().next())
        .take(2)
        .collect::<String>();

    if initials.is_empty() {
        initials = "?".to_owned();
    }

    initials.to_uppercase()
}
