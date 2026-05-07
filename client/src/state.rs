use serde::{Deserialize, Serialize};

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum AppPage {
    Dashboard,
    Rooms,
    Settings,
    About,
}

impl AppPage {
    pub fn as_i32(self) -> i32 {
        match self {
            Self::Dashboard => 0,
            Self::Rooms => 1,
            Self::Settings => 2,
            Self::About => 3,
        }
    }

    pub fn from_i32(value: i32) -> Self {
        match value {
            1 => Self::Rooms,
            2 => Self::Settings,
            3 => Self::About,
            _ => Self::Dashboard,
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct Room {
    pub name: String,
    pub mode: String,
    pub members: String,
    pub latency: String,
    pub active: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct Member {
    pub name: String,
    pub ip: String,
    pub latency: String,
    pub online: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct LogEntry {
    pub time: String,
    pub level: String,
    pub message: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct AppState {
    pub current_page: AppPage,
    pub connected: bool,
    pub status_text: String,
    pub virtual_ip: String,
    pub node_latency: String,
    pub upload_speed: String,
    pub download_speed: String,
    pub relay_mode: String,
    pub nat_type: String,
    pub rooms: Vec<Room>,
    pub members: Vec<Member>,
    pub logs: Vec<LogEntry>,
    pub auto_reconnect: bool,
    pub minimize_to_tray: bool,
    pub launch_on_startup: bool,
    pub tick: u64,
}

impl AppState {
    pub fn navigate(&mut self, page: i32) {
        self.current_page = AppPage::from_i32(page);
    }

    pub fn toggle_connection(&mut self) {
        self.connected = !self.connected;

        if self.connected {
            self.status_text = "已连接".to_owned();
            self.virtual_ip = "100.88.14.23".to_owned();
            self.node_latency = "28 ms".to_owned();
            self.upload_speed = "1.4 MB/s".to_owned();
            self.download_speed = "8.7 MB/s".to_owned();
            self.relay_mode = "P2P / UDP".to_owned();
            self.push_log("INFO", "已建立 mock 虚拟局域网会话，当前使用 UDP 优先路径");
        } else {
            self.status_text = "未连接".to_owned();
            self.virtual_ip = "--".to_owned();
            self.node_latency = "--".to_owned();
            self.upload_speed = "0 KB/s".to_owned();
            self.download_speed = "0 KB/s".to_owned();
            self.relay_mode = "离线".to_owned();
            self.push_log("INFO", "连接已断开，所有 mock 节点状态保持为预览数据");
        }
    }

    pub fn push_log(&mut self, level: impl Into<String>, message: impl Into<String>) {
        let timestamp = format!("{:02}:{:02}:{:02}", 21, 59, self.tick % 60);
        self.logs.insert(
            0,
            LogEntry {
                time: timestamp,
                level: level.into(),
                message: message.into(),
            },
        );

        if self.logs.len() > 16 {
            self.logs.truncate(16);
        }
    }
}
