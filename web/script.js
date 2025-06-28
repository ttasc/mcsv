// File: script.js
document.addEventListener('DOMContentLoaded', function() {
    // Lấy các phần tử DOM
    const serverItems = document.querySelectorAll('.server-item');
    const noServer = document.querySelector('.no-server');
    const serverManagement = document.querySelector('.server-management');
    const serverTitle = document.querySelector('.server-title');
    const serverStatus = document.querySelector('.server-status');
    const toggleButton = document.querySelector('.toggle-button');
    const logContainer = document.querySelector('.log-container');
    const commandInput = document.querySelector('.command-input');
    const sendButton = document.querySelector('.send-button');
    const themeToggleBtn = document.getElementById('theme-toggle');
    const themeIcon = themeToggleBtn.querySelector('i');
    const themeText = themeToggleBtn.querySelector('span');

    // Dữ liệu giả lập cho các server
    const servers = {
        1: { name: "Survival World", status: "stopped" },
        2: { name: "Creative Paradise", status: "stopped" },
        3: { name: "SkyBlock Challenge", status: "running" },
        4: { name: "Hardcore Mode", status: "stopped" }
    };

    // Kiểm tra theme trong localStorage
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        document.body.className = savedTheme;
        updateThemeButton();
    }

    // Xử lý sự kiện chuyển đổi theme
    themeToggleBtn.addEventListener('click', function() {
        if (document.body.classList.contains('light-mode')) {
            document.body.classList.remove('light-mode');
            document.body.classList.add('dark-mode');
        } else {
            document.body.classList.remove('dark-mode');
            document.body.classList.add('light-mode');
        }
        localStorage.setItem('theme', document.body.className);
        updateThemeButton();
    });

    function updateThemeButton() {
        if (document.body.classList.contains('dark-mode')) {
            themeIcon.className = 'fas fa-sun';
            themeText.textContent = 'Chế độ sáng';
        } else {
            themeIcon.className = 'fas fa-moon';
            themeText.textContent = 'Chế độ tối';
        }
    }

    // Xử lý sự kiện khi chọn server
    serverItems.forEach(item => {
        item.addEventListener('click', function() {
            // Xóa active khỏi tất cả các item
            serverItems.forEach(i => i.classList.remove('active'));

            // Thêm active vào item được chọn
            this.classList.add('active');

            // Lấy ID server
            const serverId = this.getAttribute('data-id');
            const server = servers[serverId];

            // Cập nhật giao diện
            noServer.style.display = 'none';
            serverManagement.style.display = 'flex';
            serverTitle.textContent = server.name;

            // Cập nhật trạng thái
            if (server.status === 'running') {
                serverStatus.textContent = 'Running';
                serverStatus.className = 'server-status status-running';
                toggleButton.innerHTML = '<i class="fas fa-stop"></i> Dừng Server';
                toggleButton.classList.add('stop');
            } else {
                serverStatus.textContent = 'Stopped';
                serverStatus.className = 'server-status status-stopped';
                toggleButton.innerHTML = '<i class="fas fa-play"></i> Khởi động Server';
                toggleButton.classList.remove('stop');
            }
        });
    });

    // Xử lý nút khởi động/dừng server
    toggleButton.addEventListener('click', function() {
        if (this.classList.contains('stop')) {
            // Đang chạy -> dừng
            serverStatus.textContent = 'Stopped';
            serverStatus.className = 'server-status status-stopped';
            this.innerHTML = '<i class="fas fa-play"></i> Khởi động Server';
            this.classList.remove('stop');

            // Thêm log
            addLogLine('Server stopped by user');
        } else {
            // Đang dừng -> khởi động
            serverStatus.textContent = 'Running';
            serverStatus.className = 'server-status status-running';
            this.innerHTML = '<i class="fas fa-stop"></i> Dừng Server';
            this.classList.add('stop');

            // Kiểm tra có tạo thế giới mới không
            const newWorld = document.getElementById('new-world').checked;

            // Thêm log
            addLogLine('Starting server...');
            addLogLine(newWorld ? 'Creating new world...' : 'Loading existing world...');
            setTimeout(() => addLogLine('Server started successfully'), 1500);
        }
    });

    // Xử lý gửi lệnh
    sendButton.addEventListener('click', sendCommand);
    commandInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendCommand();
        }
    });

    function sendCommand() {
        const command = commandInput.value.trim();
        if (command) {
            addLogLine(`> ${command}`);

            // Giả lập phản hồi từ server
            if (command === 'help') {
                addLogLine('Available commands: help, stop, say, gamemode, etc.');
            } else if (command === 'stop') {
                addLogLine('Stopping the server...');
                setTimeout(() => {
                    addLogLine('Server stopped');
                    serverStatus.textContent = 'Stopped';
                    serverStatus.className = 'server-status status-stopped';
                    toggleButton.innerHTML = '<i class="fas fa-play"></i> Khởi động Server';
                    toggleButton.classList.remove('stop');
                }, 1000);
            } else {
                addLogLine(`Command executed: ${command}`);
            }

            // Xóa input
            commandInput.value = '';
        }
    }

    // Thêm dòng log mới
    function addLogLine(text) {
        const now = new Date();
        const timeString = `[${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}]`;

        const logLine = document.createElement('div');
        logLine.className = 'log-line';
        logLine.innerHTML = `<span class="log-time">${timeString}</span> ${text}`;

        logContainer.appendChild(logLine);

        // Tự động cuộn xuống dưới cùng
        logContainer.scrollTop = logContainer.scrollHeight;
    }

    // // Chọn server đầu tiên khi tải trang
    // serverItems[0].click();
});
