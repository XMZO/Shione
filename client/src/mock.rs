use crate::state::{AppPage, AppState, LogEntry, Member, Room};

pub fn initial_state() -> AppState {
    AppState {
        current_page: AppPage::Dashboard,
        connected: false,
        status_text: "未连接".to_owned(),
        virtual_ip: "--".to_owned(),
        node_latency: "--".to_owned(),
        upload_speed: "0 KB/s".to_owned(),
        download_speed: "0 KB/s".to_owned(),
        relay_mode: "离线".to_owned(),
        nat_type: "检测中".to_owned(),
        rooms: vec![
            Room {
                name: "星露谷 / 晚间小队".to_owned(),
                mode: "P2P 优先".to_owned(),
                members: "4 / 8".to_owned(),
                latency: "24 ms".to_owned(),
                active: true,
            },
            Room {
                name: "Minecraft 建筑服".to_owned(),
                mode: "Relay 备用".to_owned(),
                members: "9 / 20".to_owned(),
                latency: "38 ms".to_owned(),
                active: false,
            },
            Room {
                name: "联机测试房".to_owned(),
                mode: "QUIC".to_owned(),
                members: "2 / 6".to_owned(),
                latency: "31 ms".to_owned(),
                active: false,
            },
        ],
        members: vec![
            Member {
                name: "Akira".to_owned(),
                ip: "100.88.14.23".to_owned(),
                latency: "本机".to_owned(),
                online: true,
            },
            Member {
                name: "Mika".to_owned(),
                ip: "100.88.14.24".to_owned(),
                latency: "24 ms".to_owned(),
                online: true,
            },
            Member {
                name: "Northwind".to_owned(),
                ip: "100.88.14.25".to_owned(),
                latency: "42 ms".to_owned(),
                online: true,
            },
            Member {
                name: "Relay-Tokyo-01".to_owned(),
                ip: "100.88.255.1".to_owned(),
                latency: "57 ms".to_owned(),
                online: false,
            },
        ],
        logs: vec![
            LogEntry {
                time: "21:58:16".to_owned(),
                level: "INFO".to_owned(),
                message: "GUI 原型已启动，当前使用 mock 数据".to_owned(),
            },
            LogEntry {
                time: "21:58:12".to_owned(),
                level: "MOCK".to_owned(),
                message: "房间列表、成员、延迟与速率均为本地模拟".to_owned(),
            },
            LogEntry {
                time: "21:58:04".to_owned(),
                level: "READY".to_owned(),
                message: "等待用户点击连接按钮".to_owned(),
            },
        ],
        auto_reconnect: true,
        minimize_to_tray: true,
        launch_on_startup: false,
        tick: 0,
    }
}

pub fn tick(state: &mut AppState) {
    state.tick += 1;

    if !state.connected {
        return;
    }

    let wave = state.tick as i32;
    let latency = 24 + wave % 11;
    let upload_minor = (wave * 7) % 10;
    let download_minor = (wave * 3) % 10;

    state.node_latency = format!("{latency} ms");
    state.upload_speed = format!("1.{upload_minor} MB/s");
    state.download_speed = format!("8.{download_minor} MB/s");
    state.nat_type = if wave % 12 < 8 {
        "Full Cone NAT".to_owned()
    } else {
        "Port Restricted NAT".to_owned()
    };

    if let Some(room) = state.rooms.first_mut() {
        room.latency = format!("{} ms", latency + 2);
    }

    for (index, member) in state.members.iter_mut().enumerate() {
        if member.latency != "本机" {
            member.latency = format!("{} ms", latency + 10 + index as i32 * 4);
        }
    }

    if state.tick % 8 == 0 {
        state.push_log("NET", "mock 心跳正常，节点延迟与速率已刷新");
    }
}
